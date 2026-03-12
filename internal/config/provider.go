package config

import sharedTypes "agent_study/pkg/types"

const tokensPerMillion int64 = 1_000_000

type Provider interface {
	ModelName() string
	BaseURL() string
	Type() string
	AuthKey() string
}

type BaseProvider struct {
	Model   string `yaml:"model"`
	BaseUrl string `yaml:"baseUrl"`
	Typ     string `yaml:"type"`
	Key     string `yaml:"authKey"`
}

func (p BaseProvider) ModelName() string {
	return p.Model
}

func (p BaseProvider) BaseURL() string {
	return p.BaseUrl
}

func (p BaseProvider) Type() string {
	return p.Typ
}

func (p BaseProvider) AuthKey() string {
	return p.Key
}

// LLMProvider 在通用模型提供方配置的基础上，补充运行时会用到的价格与上下文窗口信息。
type LLMProvider struct {
	BaseProvider `yaml:",inline"`
	Cost         LLMCostConfig    `yaml:"cost"`
	Context      LLMContextConfig `yaml:"context"`
}

// LLMCostConfig 把价格字段设计成指针，目的是让 YAML 能区分“没配”与“显式配置为 0”。
type LLMCostConfig struct {
	Input       *float64 `yaml:"input,omitempty"`
	CachedInput *float64 `yaml:"cachedInput,omitempty"`
	Output      *float64 `yaml:"output,omitempty"`
}

// LLMContextConfig 描述模型总上下文窗口，以及输入和输出各自可占用的 token 配额。
type LLMContextConfig struct {
	Max    int64 `yaml:"max"`
	Input  int64 `yaml:"input"`
	Output int64 `yaml:"output"`
}

// Pricing 会把配置文件里按“每百万 token”书写的人类友好价格，转换成运行时费用统计
// 所使用的统一价格结构。
func (p *LLMProvider) Pricing() *sharedTypes.ModelPricing {
	if p.Cost.Input == nil || p.Cost.Output == nil {
		return nil
	}
	pricing := &sharedTypes.ModelPricing{
		Input: sharedTypes.TokenPrice{
			AmountUSD: *p.Cost.Input,
			PerTokens: tokensPerMillion,
		},
		Output: sharedTypes.TokenPrice{
			AmountUSD: *p.Cost.Output,
			PerTokens: tokensPerMillion,
		},
	}
	if p.Cost.CachedInput != nil {
		pricing.CachedInput = &sharedTypes.TokenPrice{
			AmountUSD: *p.Cost.CachedInput,
			PerTokens: tokensPerMillion,
		}
	}
	return pricing
}

// ContextWindow 返回归一化后的上下文窗口配置，方便下游直接使用已经补齐的限制值。
func (p *LLMProvider) ContextWindow() LLMContextConfig {
	return p.Context.Normalized()
}

// Normalized 会在信息足够时推导缺失的上下文维度；如果现有数据不足以安全推导，
// 就尽量保留调用方原始配置不做过度猜测。
func (c LLMContextConfig) Normalized() LLMContextConfig {
	normalized := c
	if normalized.Max <= 0 && normalized.Input > 0 && normalized.Output > 0 {
		normalized.Max = normalized.Input + normalized.Output
	}
	if normalized.Max <= 0 {
		return normalized
	}
	if normalized.Input <= 0 && normalized.Output >= 0 && normalized.Max >= normalized.Output {
		normalized.Input = normalized.Max - normalized.Output
	}
	if normalized.Output <= 0 && normalized.Input >= 0 && normalized.Max >= normalized.Input {
		normalized.Output = normalized.Max - normalized.Input
	}
	return normalized
}

type EmbeddingProvider struct {
	BaseProvider `yaml:",inline"`
	Dimension    int `yaml:"dimension"`
}

type RerankingProvider struct {
	BaseProvider `yaml:",inline"`
}
