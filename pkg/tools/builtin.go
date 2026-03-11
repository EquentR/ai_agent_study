package tools

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"agent_study/pkg/types"
)

// BuiltinOptions 控制内置工具的工作目录与执行限制。
type BuiltinOptions struct {
	RootDir            string
	DefaultLSDepth     int
	DefaultExecTimeout time.Duration
	MaxExecTimeout     time.Duration
}

type builtinConfig struct {
	rootDir            string
	defaultLSDepth     int
	defaultExecTimeout time.Duration
	maxExecTimeout     time.Duration
}

// NewBuiltinTools 返回当前可用的全部内置工具，但不会自动注册。
func NewBuiltinTools(options BuiltinOptions) ([]Tool, error) {
	lsTool, err := NewLSTool(options)
	if err != nil {
		return nil, err
	}
	readTool, err := NewReadFileTool(options)
	if err != nil {
		return nil, err
	}
	writeTool, err := NewWriteFileTool(options)
	if err != nil {
		return nil, err
	}
	execTool, err := NewExecTool(options)
	if err != nil {
		return nil, err
	}

	return []Tool{lsTool, readTool, writeTool, execTool}, nil

}

// NewLSTool 创建目录树浏览工具。
func NewLSTool(options BuiltinOptions) (Tool, error) {
	cfg, err := normalizeBuiltinOptions(options)
	if err != nil {
		return Tool{}, err
	}

	return Tool{
		Name:        "ls",
		Description: "List files as a directory tree",
		Source:      "builtin",
		Parameters: types.JSONSchema{
			Type: "object",
			Properties: map[string]types.SchemaProperty{
				"path":      {Type: "string", Description: "Directory or file path relative to the tool root"},
				"max_depth": {Type: "integer", Description: "Maximum directory depth to render"},
			},
		},
		Handler: func(ctx context.Context, arguments map[string]interface{}) (string, error) {
			_ = ctx
			return runLS(cfg, arguments)
		},
	}, nil
}

// NewReadFileTool 创建文件读取工具。
func NewReadFileTool(options BuiltinOptions) (Tool, error) {
	cfg, err := normalizeBuiltinOptions(options)
	if err != nil {
		return Tool{}, err
	}

	return Tool{
		Name:        "read_file",
		Description: "Read a file from disk",
		Source:      "builtin",
		Parameters: types.JSONSchema{
			Type: "object",
			Properties: map[string]types.SchemaProperty{
				"path": {Type: "string", Description: "File path relative to the tool root"},
			},
			Required: []string{"path"},
		},
		Handler: func(ctx context.Context, arguments map[string]interface{}) (string, error) {
			_ = ctx
			return runReadFile(cfg, arguments)
		},
	}, nil
}

// NewWriteFileTool 创建文件写入工具。
func NewWriteFileTool(options BuiltinOptions) (Tool, error) {
	cfg, err := normalizeBuiltinOptions(options)
	if err != nil {
		return Tool{}, err
	}

	return Tool{
		Name:        "write_file",
		Description: "Write a file to disk",
		Source:      "builtin",
		Parameters: types.JSONSchema{
			Type: "object",
			Properties: map[string]types.SchemaProperty{
				"path":    {Type: "string", Description: "File path relative to the tool root"},
				"content": {Type: "string", Description: "File content to write"},
			},
			Required: []string{"path", "content"},
		},
		Handler: func(ctx context.Context, arguments map[string]interface{}) (string, error) {
			_ = ctx
			return runWriteFile(cfg, arguments)
		},
	}, nil
}

// NewExecTool 创建命令执行工具。
func NewExecTool(options BuiltinOptions) (Tool, error) {
	cfg, err := normalizeBuiltinOptions(options)
	if err != nil {
		return Tool{}, err
	}

	return Tool{
		Name:        "exec",
		Description: "Execute a shell command",
		Source:      "builtin",
		Parameters: types.JSONSchema{
			Type: "object",
			Properties: map[string]types.SchemaProperty{
				"command":    {Type: "string", Description: "Shell command to execute"},
				"dir":        {Type: "string", Description: "Working directory relative to the tool root"},
				"timeout_ms": {Type: "integer", Description: "Command timeout in milliseconds"},
			},
			Required: []string{"command"},
		},
		Handler: func(ctx context.Context, arguments map[string]interface{}) (string, error) {
			return runExec(ctx, cfg, arguments)
		},
	}, nil
}

func normalizeBuiltinOptions(options BuiltinOptions) (builtinConfig, error) {
	rootDir := options.RootDir
	if rootDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return builtinConfig{}, fmt.Errorf("get working directory: %w", err)
		}
		rootDir = cwd
	}

	absRoot, err := filepath.Abs(rootDir)
	if err != nil {
		return builtinConfig{}, fmt.Errorf("resolve root dir: %w", err)
	}

	// 这里统一补齐一组保守默认值，避免每个内置工具都重复处理相同的兜底逻辑。
	defaultLSDepth := options.DefaultLSDepth
	if defaultLSDepth <= 0 {
		defaultLSDepth = 3
	}

	defaultExecTimeout := options.DefaultExecTimeout
	if defaultExecTimeout <= 0 {
		defaultExecTimeout = 30 * time.Second
	}

	maxExecTimeout := options.MaxExecTimeout
	if maxExecTimeout <= 0 {
		maxExecTimeout = 2 * time.Minute
	}

	// 默认超时也必须被限制在全局最大值内，防止配置层误填出一个突破安全上限的
	// 默认执行时长。
	if defaultExecTimeout > maxExecTimeout {
		defaultExecTimeout = maxExecTimeout
	}

	return builtinConfig{
		rootDir:            absRoot,
		defaultLSDepth:     defaultLSDepth,
		defaultExecTimeout: defaultExecTimeout,
		maxExecTimeout:     maxExecTimeout,
	}, nil
}

func runLS(cfg builtinConfig, arguments map[string]interface{}) (string, error) {
	path, err := optionalStringArg(arguments, "path", ".")
	if err != nil {
		return "", err
	}
	maxDepth, err := optionalIntArg(arguments, "max_depth", cfg.defaultLSDepth)
	if err != nil {
		return "", err
	}
	if maxDepth < 0 {
		return "", fmt.Errorf("max_depth must be >= 0")
	}

	resolvedPath, displayPath, err := resolvePath(cfg.rootDir, path)
	if err != nil {
		return "", err
	}

	return renderTree(resolvedPath, displayPath, maxDepth)
}

func runReadFile(cfg builtinConfig, arguments map[string]interface{}) (string, error) {
	path, err := requiredStringArg(arguments, "path")
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("path cannot be empty")
	}

	resolvedPath, _, err := resolvePath(cfg.rootDir, path)
	if err != nil {
		return "", err
	}

	content, err := os.ReadFile(resolvedPath)
	if err != nil {
		return "", fmt.Errorf("read file %q: %w", path, err)
	}

	return string(content), nil
}

func runWriteFile(cfg builtinConfig, arguments map[string]interface{}) (string, error) {
	path, err := requiredStringArg(arguments, "path")
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("path cannot be empty")
	}
	content, err := requiredStringArg(arguments, "content")
	if err != nil {
		return "", err
	}

	resolvedPath, displayPath, err := resolvePath(cfg.rootDir, path)
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(filepath.Dir(resolvedPath), 0o755); err != nil {
		return "", fmt.Errorf("create parent directories for %q: %w", path, err)
	}
	if err := os.WriteFile(resolvedPath, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("write file %q: %w", path, err)
	}

	return fmt.Sprintf("wrote %d bytes to %s", len(content), displayPath), nil
}

func runExec(ctx context.Context, cfg builtinConfig, arguments map[string]interface{}) (string, error) {
	command, err := requiredStringArg(arguments, "command")
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(command) == "" {
		return "", fmt.Errorf("command cannot be empty")
	}

	dir, err := optionalStringArg(arguments, "dir", ".")
	if err != nil {
		return "", err
	}
	resolvedDir, displayDir, err := resolvePath(cfg.rootDir, dir)
	if err != nil {
		return "", err
	}

	timeoutMS, err := optionalIntArg(arguments, "timeout_ms", int(cfg.defaultExecTimeout/time.Millisecond))
	if err != nil {
		return "", err
	}
	if timeoutMS <= 0 {
		timeoutMS = int(cfg.defaultExecTimeout / time.Millisecond)
	}

	timeout := time.Duration(timeoutMS) * time.Millisecond
	// 调用方可以把超时设得更短，但不能突破内置工具统一配置的最大执行上限。
	if timeout > cfg.maxExecTimeout {
		timeout = cfg.maxExecTimeout
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// 统一走宿主平台的默认 shell，这样工具里执行的命令语法就和用户手动在
	// 当前系统终端里输入时保持一致。
	shell, shellArgs := shellCommand(command)
	cmd := exec.CommandContext(ctx, shell, shellArgs...)
	cmd.Dir = resolvedDir
	output, err := cmd.CombinedOutput()
	text := string(output)
	if text == "" && err == nil {
		text = fmt.Sprintf("command completed successfully in %s with no output", displayDir)
	}

	if ctx.Err() == context.DeadlineExceeded {
		if text == "" {
			text = fmt.Sprintf("command timed out after %s", timeout)
		}
		return text, fmt.Errorf("command timed out after %s", timeout)
	}
	if err != nil {
		if text == "" {
			text = err.Error()
		}
		return text, fmt.Errorf("execute command in %s: %w", displayDir, err)
	}

	return text, nil
}

func resolvePath(rootDir string, requested string) (string, string, error) {
	if strings.TrimSpace(requested) == "" {
		requested = "."
	}

	resolvedPath := requested
	if !filepath.IsAbs(resolvedPath) {
		resolvedPath = filepath.Join(rootDir, resolvedPath)
	}

	absPath, err := filepath.Abs(resolvedPath)
	if err != nil {
		return "", "", fmt.Errorf("resolve path %q: %w", requested, err)
	}

	relPath, err := filepath.Rel(rootDir, absPath)
	if err != nil {
		return "", "", fmt.Errorf("rel path %q: %w", requested, err)
	}
	// 只要相对 root 的结果出现越界，就拒绝这次请求；这样即便传进来的是绝对路径，
	// 读写和命令执行也依然被限制在工具沙箱目录内。
	if relPath == ".." || strings.HasPrefix(relPath, ".."+string(os.PathSeparator)) {
		return "", "", fmt.Errorf("path %q escapes root %q", requested, rootDir)
	}

	displayPath := filepath.ToSlash(relPath)
	if displayPath == "." || displayPath == "" {
		displayPath = "."
	}

	return absPath, displayPath, nil
}

func renderTree(resolvedPath string, displayPath string, maxDepth int) (string, error) {
	info, err := os.Stat(resolvedPath)
	if err != nil {
		return "", fmt.Errorf("stat %q: %w", displayPath, err)
	}

	var builder strings.Builder
	if info.IsDir() {
		builder.WriteString(displayPath + "/")
		if err := appendTree(&builder, resolvedPath, 0, maxDepth); err != nil {
			return "", err
		}
		return builder.String(), nil
	}

	builder.WriteString(displayPath)
	return builder.String(), nil
}

func appendTree(builder *strings.Builder, dir string, depth int, maxDepth int) error {
	if depth >= maxDepth {
		return nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read dir %q: %w", dir, err)
	}

	// 目录优先、名称升序，保证多次列目录时输出稳定，便于模型比较前后差异。
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir() != entries[j].IsDir() {
			return entries[i].IsDir()
		}
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		builder.WriteString("\n")
		builder.WriteString(strings.Repeat("  ", depth+1))
		builder.WriteString(entry.Name())
		if entry.IsDir() {
			builder.WriteString("/")
			if err := appendTree(builder, filepath.Join(dir, entry.Name()), depth+1, maxDepth); err != nil {
				return err
			}
		}
	}

	return nil
}

func shellCommand(command string) (string, []string) {
	if runtime.GOOS == "windows" {
		return "cmd", []string{"/C", command}
	}
	return "sh", []string{"-c", command}
}

func requiredStringArg(arguments map[string]interface{}, key string) (string, error) {
	value, ok := arguments[key]
	if !ok || value == nil {
		return "", fmt.Errorf("missing required argument %q", key)
	}
	text, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("argument %q must be a string", key)
	}
	return text, nil
}

func optionalStringArg(arguments map[string]interface{}, key string, fallback string) (string, error) {
	value, ok := arguments[key]
	if !ok || value == nil {
		return fallback, nil
	}
	text, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("argument %q must be a string", key)
	}
	if text == "" {
		return fallback, nil
	}
	return text, nil
}

func optionalIntArg(arguments map[string]interface{}, key string, fallback int) (int, error) {
	value, ok := arguments[key]
	if !ok || value == nil {
		return fallback, nil
	}

	switch typed := value.(type) {
	case int:
		return typed, nil
	case int8:
		return int(typed), nil
	case int16:
		return int(typed), nil
	case int32:
		return int(typed), nil
	case int64:
		return int(typed), nil
	case float32:
		return int(typed), nil
	case float64:
		return int(typed), nil
	default:
		return 0, fmt.Errorf("argument %q must be a number", key)
	}
}
