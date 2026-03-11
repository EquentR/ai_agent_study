package agent

import (
	"errors"
	"math"
	"testing"

	llmModel "agent_study/pkg/llm_core/model"
	sharedTypes "agent_study/pkg/types"
)

func TestCostTrackerAddUsageCalculatesBreakdownAndTotals(t *testing.T) {
	tracker, err := NewCostTracker(sharedTypes.ModelPricing{
		Input:       sharedTypes.TokenPrice{AmountUSD: 1.2, PerTokens: 1000},
		CachedInput: &sharedTypes.TokenPrice{AmountUSD: 0.3, PerTokens: 1000},
		Output:      sharedTypes.TokenPrice{AmountUSD: 2.4, PerTokens: 1000},
	}, 0.2)
	if err != nil {
		t.Fatalf("NewCostTracker() error = %v", err)
	}

	breakdown, err := tracker.AddUsage(llmModel.TokenUsage{
		PromptTokens:       100,
		CachedPromptTokens: 40,
		CompletionTokens:   25,
		TotalTokens:        125,
	})
	if err != nil {
		t.Fatalf("AddUsage() error = %v", err)
	}

	if breakdown.UncachedPromptTokens != 60 {
		t.Fatalf("UncachedPromptTokens = %d, want 60", breakdown.UncachedPromptTokens)
	}
	if breakdown.CachedPromptTokens != 40 {
		t.Fatalf("CachedPromptTokens = %d, want 40", breakdown.CachedPromptTokens)
	}
	if breakdown.CompletionTokens != 25 {
		t.Fatalf("CompletionTokens = %d, want 25", breakdown.CompletionTokens)
	}
	assertFloatEquals(t, breakdown.InputCostUSD, 0.072)
	assertFloatEquals(t, breakdown.CachedInputCostUSD, 0.012)
	assertFloatEquals(t, breakdown.OutputCostUSD, 0.06)
	assertFloatEquals(t, breakdown.TotalCostUSD, 0.144)

	totals := tracker.Totals()
	if totals.Usage.PromptTokens != 100 || totals.Usage.CachedPromptTokens != 40 || totals.Usage.CompletionTokens != 25 {
		t.Fatalf("Totals().Usage = %#v, want accumulated usage", totals.Usage)
	}
	assertFloatEquals(t, totals.Cost.TotalCostUSD, 0.144)
	if tracker.OverBudget() {
		t.Fatalf("OverBudget() = true, want false")
	}
	assertFloatEquals(t, tracker.RemainingBudgetUSD(), 0.056)
}

func TestCostTrackerAddUsageReturnsBudgetExceededAfterCrossingLimit(t *testing.T) {
	tracker, err := NewCostTracker(sharedTypes.ModelPricing{
		Input:  sharedTypes.TokenPrice{AmountUSD: 1, PerTokens: 1000},
		Output: sharedTypes.TokenPrice{AmountUSD: 1, PerTokens: 1000},
	}, 0.1)
	if err != nil {
		t.Fatalf("NewCostTracker() error = %v", err)
	}

	if _, err := tracker.AddUsage(llmModel.TokenUsage{PromptTokens: 40, CompletionTokens: 40, TotalTokens: 80}); err != nil {
		t.Fatalf("first AddUsage() error = %v, want nil", err)
	}

	_, err = tracker.AddUsage(llmModel.TokenUsage{PromptTokens: 30, CompletionTokens: 30, TotalTokens: 60})
	if !errors.Is(err, ErrBudgetExceeded) {
		t.Fatalf("second AddUsage() error = %v, want ErrBudgetExceeded", err)
	}
	if !tracker.OverBudget() {
		t.Fatalf("OverBudget() = false, want true")
	}
	assertFloatEquals(t, tracker.RemainingBudgetUSD(), -0.04)
}

func TestCostTrackerAddUsageRejectsImpossibleCacheCounts(t *testing.T) {
	tracker, err := NewCostTracker(sharedTypes.ModelPricing{
		Input: sharedTypes.TokenPrice{AmountUSD: 1, PerTokens: 1000},
	}, 0)
	if err != nil {
		t.Fatalf("NewCostTracker() error = %v", err)
	}

	_, err = tracker.AddUsage(llmModel.TokenUsage{PromptTokens: 10, CachedPromptTokens: 11, TotalTokens: 10})
	if !errors.Is(err, ErrInvalidTokenUsage) {
		t.Fatalf("AddUsage() error = %v, want ErrInvalidTokenUsage", err)
	}
}

func TestCalculateUsageCostFallsBackToInputPricingWhenCachedPricingUnset(t *testing.T) {
	breakdown, err := CalculateUsageCost(llmModel.TokenUsage{
		PromptTokens:       100,
		CachedPromptTokens: 40,
		CompletionTokens:   0,
		TotalTokens:        100,
	}, sharedTypes.ModelPricing{
		Input: sharedTypes.TokenPrice{AmountUSD: 1, PerTokens: 1000},
	})
	if err != nil {
		t.Fatalf("CalculateUsageCost() error = %v", err)
	}

	assertFloatEquals(t, breakdown.InputCostUSD, 0.06)
	assertFloatEquals(t, breakdown.CachedInputCostUSD, 0.04)
	assertFloatEquals(t, breakdown.TotalCostUSD, 0.10)
}

func TestCalculateUsageCostExplicitZeroCachedPricingMakesCacheFree(t *testing.T) {
	breakdown, err := CalculateUsageCost(llmModel.TokenUsage{
		PromptTokens:       100,
		CachedPromptTokens: 40,
		CompletionTokens:   0,
		TotalTokens:        100,
	}, sharedTypes.ModelPricing{
		Input:       sharedTypes.TokenPrice{AmountUSD: 1, PerTokens: 1000},
		CachedInput: &sharedTypes.TokenPrice{},
	})
	if err != nil {
		t.Fatalf("CalculateUsageCost() error = %v", err)
	}

	assertFloatEquals(t, breakdown.InputCostUSD, 0.06)
	assertFloatEquals(t, breakdown.CachedInputCostUSD, 0)
	assertFloatEquals(t, breakdown.TotalCostUSD, 0.06)
}

func assertFloatEquals(t *testing.T, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("got %.12f, want %.12f", got, want)
	}
}
