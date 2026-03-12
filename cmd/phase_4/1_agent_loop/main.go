package main

import (
	"agent_study/internal/agent"
	"agent_study/internal/config"
	"agent_study/internal/db"
	"agent_study/internal/log"
	llmModel "agent_study/pkg/llm_core/model"
	"agent_study/pkg/tools"
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

const defaultConfigPath = "conf/phase4/app.yaml"

const maxStepOutputChars = 300

type agentRunner interface {
	Run(ctx context.Context, task string) (*agent.State, error)
}

type stepCallbackSetter interface {
	SetStepCallback(callback agent.StepCallback)
}

// modelProvider/costProvider 让 CLI 在不绑定具体 Agent 实现的前提下，
// 也能展示当前模型名和累计费用。
type modelProvider interface {
	ModelName() string
}

type costProvider interface {
	TotalCostUSD() float64
}

func main() {
	ctx := context.Background()
	cfg, err := loadConfig(defaultConfigPath)
	if err != nil {
		panic(err)
	}

	if strings.TrimSpace(cfg.Log.Level) != "" {
		log.Init(&cfg.Log)
	}

	runner, err := newRunner(cfg)
	if err != nil {
		panic(err)
	}

	if err := runREPL(ctx, os.Stdin, os.Stdout, runner); err != nil {
		panic(err)
	}
}

func loadConfig(path string) (*config.Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return loadConfigFromBytes(raw)
}

func loadConfigFromBytes(raw []byte) (*config.Config, error) {
	expanded := os.ExpandEnv(string(raw))
	cfg := &config.Config{}
	if err := yaml.Unmarshal([]byte(expanded), cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func newRunner(cfg *config.Config) (*agent.Agent, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}

	var memoryOptions *agent.MemoryOptions
	if cfg.Sqlite.Name != "" {
		databaseCfg := cfg.Sqlite
		dbConn, err := db.InitSqlite(&databaseCfg)
		if err != nil {
			return nil, fmt.Errorf("init sqlite: %w", err)
		}
		memoryOptions = &agent.MemoryOptions{DB: dbConn}
	}

	buildinTools, _ := tools.NewBuiltinTools(tools.BuiltinOptions{})
	toolsReg := tools.NewRegistry()
	_ = toolsReg.Register(buildinTools...)

	return agent.NewAgent(agent.NewAgentOptions{
		Provider:      &cfg.LLM,
		MemoryOptions: memoryOptions,
		Tools:         toolsReg,
		Config: agent.Config{
			MaxSteps:     8,
			MaxBudgetUSD: 2,
		},
	})
}

func runREPL(ctx context.Context, in io.Reader, out io.Writer, runner agentRunner) error {
	if runner == nil {
		return fmt.Errorf("runner is nil")
	}
	if setter, ok := runner.(stepCallbackSetter); ok {
		setter.SetStepCallback(func(event agent.StepEvent) {
			printStep(out, event)
		})
	}
	streamingEnabled := false
	_, streamingEnabled = runner.(stepCallbackSetter)
	reader := bufio.NewReader(in)
	_, _ = fmt.Fprintln(out, "Agent ready. Type your question, or `exit` to quit.")
	_, _ = fmt.Fprintf(out, "Model: %s\n", runnerModelName(runner))
	_, _ = fmt.Fprintln(out, "----------------------------------------------------------------------------------------")
	for {
		_, _ = fmt.Fprint(out, "> ")
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				input := strings.TrimSpace(line)
				if input == "" {
					return nil
				}
				if shouldExit(input) {
					return nil
				}
				state, runErr := runner.Run(ctx, input)
				printRunResult(out, state, runErr, streamingEnabled)
				return nil
			}
			return err
		}

		input := strings.TrimSpace(line)
		if input == "" {
			continue
		}
		if shouldExit(input) {
			return nil
		}

		state, runErr := runner.Run(ctx, input)
		printRunResult(out, state, runErr, streamingEnabled)
		_, _ = fmt.Fprintf(out, "Cost: $%.4f\n", runnerTotalCost(runner))
	}
}

func shouldExit(input string) bool {
	switch strings.ToLower(strings.TrimSpace(input)) {
	case "exit", "quit":
		return true
	default:
		return false
	}
}

func printRunResult(out io.Writer, state *agent.State, err error, stepsAlreadyPrinted bool) {
	if err != nil {
		_, _ = fmt.Fprintf(out, "Agent error: %v\n", err)
		return
	}
	if !stepsAlreadyPrinted {
		for i, step := range state.Steps {
			printStep(out, agent.StepEvent{Index: i + 1, Step: step})
		}
	}
	_, _ = fmt.Fprintf(out, "Final Answer:\n%s\n", strings.TrimSpace(state.FinalAnswer))
}

func printStep(out io.Writer, event agent.StepEvent) {
	_, _ = fmt.Fprintf(out, "Step %d:\n", event.Index)
	if event.Step.Thought != "" {
		_, _ = fmt.Fprintf(out, "Thought: %s\n", truncateForTerminal(event.Step.Thought))
	}
	if reasoning := formatReasoningItems(event.Step.ReasoningItems); reasoning != "" {
		_, _ = fmt.Fprintf(out, "%s", reasoning)
	}
	_, _ = fmt.Fprintf(out, "Action: %s\n", event.Step.Action.Kind)
	if len(event.Step.Action.ToolCalls) > 0 {
		for _, call := range event.Step.Action.ToolCalls {
			_, _ = fmt.Fprintf(out, "Tool: %s %s\n", call.Name, truncateForTerminal(call.Arguments))
		}
	}
	if event.Step.Observation != "" {
		_, _ = fmt.Fprintf(out, "Observation: %s\n", truncateForTerminal(event.Step.Observation))
	}
	_, _ = fmt.Fprintln(out, "\n----------------------------------------------------------------------------------------")
}

func truncateForTerminal(content string) string {
	content = strings.TrimSpace(content)
	if len(content) <= maxStepOutputChars {
		return content
	}
	return content[:maxStepOutputChars] + "..."
}

func formatReasoningItems(items []llmModel.ReasoningItem) string {
	if len(items) == 0 {
		return ""
	}

	// Responses API 这类后端会返回结构化 reasoning item，
	// CLI 这里把它们摊平成可读文本，便于调试每一步的思考轨迹。
	var b strings.Builder
	for _, item := range items {
		_, _ = fmt.Fprintf(&b, "Reasoning Item: %s\n", truncateForTerminal(item.ID))
		if len(item.Summary) > 0 {
			parts := make([]string, 0, len(item.Summary))
			for _, summary := range item.Summary {
				if text := strings.TrimSpace(summary.Text); text != "" {
					parts = append(parts, text)
				}
			}
			if len(parts) > 0 {
				_, _ = fmt.Fprintf(&b, "Reasoning Summary: %s\n", truncateForTerminal(strings.Join(parts, " | ")))
			}
		}
		if strings.TrimSpace(item.EncryptedContent) != "" {
			_, _ = fmt.Fprintf(&b, "Reasoning Encrypted: %s\n", truncateForTerminal(item.EncryptedContent))
		}
	}
	return b.String()
}

func runnerModelName(runner agentRunner) string {
	if provider, ok := runner.(modelProvider); ok {
		return provider.ModelName()
	}
	if concrete, ok := runner.(*agent.Agent); ok {
		return concrete.Model
	}
	return ""
}

func runnerTotalCost(runner agentRunner) float64 {
	if provider, ok := runner.(costProvider); ok {
		return provider.TotalCostUSD()
	}
	if concrete, ok := runner.(*agent.Agent); ok && concrete.Cost != nil {
		return concrete.Cost.Totals().Cost.TotalCostUSD
	}
	return 0
}
