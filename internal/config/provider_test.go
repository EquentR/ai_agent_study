package config

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestLLMProviderPricingUsesPerMillionTokens(t *testing.T) {
	provider := mustLoadLLMProvider(t, `
llmProvider:
  model: gpt-5.4
  type: openai
  cost:
    input: 1.25
    cachedInput: 0.125
    output: 10
`)

	pricing := provider.Pricing()
	if pricing == nil {
		t.Fatalf("pricing = nil, want configured pricing")
	}
	if pricing.Input.AmountUSD != 1.25 || pricing.Input.PerTokens != 1_000_000 {
		t.Fatalf("input pricing = %#v, want amount=1.25 perTokens=1_000_000", pricing.Input)
	}
	if pricing.CachedInput == nil {
		t.Fatalf("cached input pricing = nil, want non-nil")
	}
	if pricing.CachedInput.AmountUSD != 0.125 || pricing.CachedInput.PerTokens != 1_000_000 {
		t.Fatalf("cached pricing = %#v, want amount=0.125 perTokens=1_000_000", pricing.CachedInput)
	}
	if pricing.Output.AmountUSD != 10 || pricing.Output.PerTokens != 1_000_000 {
		t.Fatalf("output pricing = %#v, want amount=10 perTokens=1_000_000", pricing.Output)
	}
}

func TestLLMProviderPricingDistinguishesUnsetAndZeroCachedInput(t *testing.T) {
	withoutCached := mustLoadLLMProvider(t, `
llmProvider:
  cost:
    input: 1
    output: 2
`)
	pricingWithoutCached := withoutCached.Pricing()
	if pricingWithoutCached == nil {
		t.Fatalf("pricingWithoutCached = nil, want configured pricing")
	}
	if pricing := pricingWithoutCached; pricing.CachedInput != nil {
		t.Fatalf("pricing.CachedInput = %#v, want nil when cachedInput is omitted", pricing.CachedInput)
	}

	zeroCached := mustLoadLLMProvider(t, `
llmProvider:
  cost:
    input: 1
    cachedInput: 0
    output: 2
`)
	pricing := zeroCached.Pricing()
	if pricing == nil {
		t.Fatalf("pricing = nil, want configured pricing")
	}
	if pricing.CachedInput == nil {
		t.Fatalf("pricing.CachedInput = nil, want explicit zero price")
	}
	if pricing.CachedInput.AmountUSD != 0 || pricing.CachedInput.PerTokens != 1_000_000 {
		t.Fatalf("pricing.CachedInput = %#v, want zero-priced 1M unit", pricing.CachedInput)
	}
}

func TestLLMProviderPricingReturnsNilWhenRequiredPricesMissing(t *testing.T) {
	provider := mustLoadLLMProvider(t, `
llmProvider:
  cost:
    cachedInput: 0.1
`)

	if pricing := provider.Pricing(); pricing != nil {
		t.Fatalf("pricing = %#v, want nil when input/output prices are omitted", pricing)
	}
}

func TestLLMProviderContextWindowNormalizesDerivedLimits(t *testing.T) {
	provider := mustLoadLLMProvider(t, `
llmProvider:
  context:
    max: 128000
    output: 8000
`)

	ctx := provider.ContextWindow()
	if ctx.Max != 128000 {
		t.Fatalf("ctx.Max = %d, want 128000", ctx.Max)
	}
	if ctx.Input != 120000 {
		t.Fatalf("ctx.Input = %d, want 120000", ctx.Input)
	}
	if ctx.Output != 8000 {
		t.Fatalf("ctx.Output = %d, want 8000", ctx.Output)
	}
}

func TestBaseProviderSupportsAPIKeyAlias(t *testing.T) {
	provider := mustLoadLLMProvider(t, `
llmProvider:
  model: gpt-5.4
  type: openai_responses
  baseUrl: https://example.com/v1
  apiKey: test-key
`)

	if provider.AuthKey() != "test-key" {
		t.Fatalf("provider.AuthKey() = %q, want %q", provider.AuthKey(), "test-key")
	}
}

func mustLoadLLMProvider(t *testing.T, raw string) LLMProvider {
	t.Helper()

	var cfg struct {
		LLM LLMProvider `yaml:"llmProvider"`
	}
	if err := yaml.Unmarshal([]byte(raw), &cfg); err != nil {
		t.Fatalf("yaml.Unmarshal() error = %v", err)
	}
	return cfg.LLM
}
