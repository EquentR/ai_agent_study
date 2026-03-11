package agent

import (
	"errors"
	"sync"

	llmModel "agent_study/pkg/llm_core/model"
	sharedTypes "agent_study/pkg/types"
)

var (
	ErrInvalidPricing    = errors.New("invalid token pricing")
	ErrInvalidTokenUsage = errors.New("invalid token usage")
	ErrBudgetExceeded    = errors.New("agent budget exceeded")
)

// CostTracker 用于在一次 agent 运行期间累计每次请求的 token 使用量与费用，
// 并在需要时执行最大预算限制。
type CostTracker struct {
	mu           sync.RWMutex
	pricing      sharedTypes.ModelPricing
	maxBudgetUSD float64
	totalUsage   llmModel.TokenUsage
	totalCost    sharedTypes.CostBreakdown
}

// CostTotals 表示当前累计用量与累计费用的一份只读快照。
type CostTotals struct {
	Usage llmModel.TokenUsage
	Cost  sharedTypes.CostBreakdown
}

// NewCostTracker 会先校验价格配置，确保后续请求处理过程中的费用统计保持快速失败
// 且结果可预测。
func NewCostTracker(pricing sharedTypes.ModelPricing, maxBudgetUSD float64) (*CostTracker, error) {
	if maxBudgetUSD < 0 {
		return nil, ErrInvalidPricing
	}
	if err := validateModelPricing(pricing); err != nil {
		return nil, err
	}
	return &CostTracker{
		pricing:      pricing,
		maxBudgetUSD: maxBudgetUSD,
	}, nil
}

// CalculateUsageCost 会把 prompt token 拆成缓存命中和未命中两部分，因为不同
// 模型提供方可能对它们采用不同计费单价。
func CalculateUsageCost(usage llmModel.TokenUsage, pricing sharedTypes.ModelPricing) (sharedTypes.CostBreakdown, error) {
	if err := validateModelPricing(pricing); err != nil {
		return sharedTypes.CostBreakdown{}, err
	}

	normalizedUsage, err := normalizeUsage(usage)
	if err != nil {
		return sharedTypes.CostBreakdown{}, err
	}

	// CachedPromptTokens 一定是 PromptTokens 的子集，因此剩余部分就是需要按
	// 常规输入单价计费的未缓存 token。
	uncachedPromptTokens := normalizedUsage.PromptTokens - normalizedUsage.CachedPromptTokens
	breakdown := sharedTypes.CostBreakdown{
		UncachedPromptTokens: uncachedPromptTokens,
		CachedPromptTokens:   normalizedUsage.CachedPromptTokens,
		CompletionTokens:     normalizedUsage.CompletionTokens,
		InputCostUSD:         calculateTokenPrice(uncachedPromptTokens, pricing.Input),
		CachedInputCostUSD:   calculateTokenPrice(normalizedUsage.CachedPromptTokens, resolveCachedInputPricing(pricing)),
		OutputCostUSD:        calculateTokenPrice(normalizedUsage.CompletionTokens, pricing.Output),
	}
	breakdown.TotalCostUSD = breakdown.InputCostUSD + breakdown.CachedInputCostUSD + breakdown.OutputCostUSD
	return breakdown, nil
}

// AddUsage 在返回单次请求费用拆分的同时，也会把累计用量和累计费用一并更新。
func (c *CostTracker) AddUsage(usage llmModel.TokenUsage) (sharedTypes.CostBreakdown, error) {
	breakdown, err := CalculateUsageCost(usage, c.pricing)
	if err != nil {
		return sharedTypes.CostBreakdown{}, err
	}
	normalizedUsage, err := normalizeUsage(usage)
	if err != nil {
		return sharedTypes.CostBreakdown{}, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// 即使本次累计后超预算，也先把 totals 写进去，再返回 ErrBudgetExceeded，
	// 这样调用方仍然能看到触发超限时的精确累计状态。
	c.totalUsage.PromptTokens += normalizedUsage.PromptTokens
	c.totalUsage.CachedPromptTokens += normalizedUsage.CachedPromptTokens
	c.totalUsage.CompletionTokens += normalizedUsage.CompletionTokens
	c.totalUsage.TotalTokens += normalizedUsage.TotalTokens

	c.totalCost.UncachedPromptTokens += breakdown.UncachedPromptTokens
	c.totalCost.CachedPromptTokens += breakdown.CachedPromptTokens
	c.totalCost.CompletionTokens += breakdown.CompletionTokens
	c.totalCost.InputCostUSD += breakdown.InputCostUSD
	c.totalCost.CachedInputCostUSD += breakdown.CachedInputCostUSD
	c.totalCost.OutputCostUSD += breakdown.OutputCostUSD
	c.totalCost.TotalCostUSD += breakdown.TotalCostUSD

	if c.maxBudgetUSD > 0 && c.totalCost.TotalCostUSD > c.maxBudgetUSD {
		return breakdown, ErrBudgetExceeded
	}

	return breakdown, nil
}

func (c *CostTracker) Totals() CostTotals {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return CostTotals{
		Usage: c.totalUsage,
		Cost:  c.totalCost,
	}
}

func (c *CostTracker) OverBudget() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.maxBudgetUSD <= 0 {
		return false
	}
	return c.totalCost.TotalCostUSD > c.maxBudgetUSD
}

func (c *CostTracker) RemainingBudgetUSD() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.maxBudgetUSD <= 0 {
		return 0
	}
	return c.maxBudgetUSD - c.totalCost.TotalCostUSD
}

func validateModelPricing(pricing sharedTypes.ModelPricing) error {
	if err := validateTokenPrice(pricing.Input); err != nil {
		return err
	}
	if err := validateOptionalTokenPrice(pricing.CachedInput); err != nil {
		return err
	}
	if err := validateTokenPrice(pricing.Output); err != nil {
		return err
	}
	return nil
}

func validateOptionalTokenPrice(price *sharedTypes.TokenPrice) error {
	if price == nil {
		return nil
	}
	return validateTokenPrice(*price)
}

func validateTokenPrice(price sharedTypes.TokenPrice) error {
	if price.AmountUSD < 0 || price.PerTokens < 0 {
		return ErrInvalidPricing
	}
	// 全 0 价格被视为显式配置的“免费档”；只要金额非 0，就必须同时声明它对应的
	// 计费 token 单位。
	if price.AmountUSD == 0 && price.PerTokens == 0 {
		return nil
	}
	if price.PerTokens <= 0 {
		return ErrInvalidPricing
	}
	return nil
}

func resolveCachedInputPricing(pricing sharedTypes.ModelPricing) sharedTypes.TokenPrice {
	// 如果没有单独配置缓存输入价格，就回退到普通输入价格，避免调用方在
	// “两者同价”这个常见场景下重复填写配置。
	if pricing.CachedInput != nil {
		return *pricing.CachedInput
	}
	return pricing.Input
}

func calculateTokenPrice(tokens int64, price sharedTypes.TokenPrice) float64 {
	if tokens <= 0 || price.AmountUSD == 0 || price.PerTokens <= 0 {
		return 0
	}
	return float64(tokens) * price.AmountUSD / float64(price.PerTokens)
}

func normalizeUsage(usage llmModel.TokenUsage) (llmModel.TokenUsage, error) {
	if usage.PromptTokens < 0 || usage.CachedPromptTokens < 0 || usage.CompletionTokens < 0 || usage.TotalTokens < 0 {
		return llmModel.TokenUsage{}, ErrInvalidTokenUsage
	}
	// 缓存命中的 prompt token 不可能超过总 prompt token；否则后续拆分未缓存
	// token 时就会出现负数。
	if usage.CachedPromptTokens > usage.PromptTokens {
		return llmModel.TokenUsage{}, ErrInvalidTokenUsage
	}
	// 有些模型提供方不会回填总 token 数，这时就根据已经校验过的 prompt 和
	// 输出 token 数补出来。
	if usage.TotalTokens == 0 {
		usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
	}
	return usage, nil
}
