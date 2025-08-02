package main

import (
	"testing"

	"github.com/carlmjohnson/be"
)

func TestSessionStateString(t *testing.T) {
	tests := []struct {
		name     string
		state    sessionState
		expected string
	}{
		{
			name:     "overview state",
			state:    overviewState,
			expected: "overview",
		},
		{
			name:     "transactions state",
			state:    transactions,
			expected: "transactions",
		},
		{
			name:     "detailed transaction state",
			state:    detailedTransaction,
			expected: "transaction details",
		},
		{
			name:     "categorize transaction state",
			state:    categorizeTransaction,
			expected: "categorize transaction",
		},
		{
			name:     "insert transaction state",
			state:    insertTransaction,
			expected: "insert transaction",
		},
		{
			name:     "loading state",
			state:    loading,
			expected: "loading",
		},
		{
			name:     "recurring expenses state",
			state:    recurringExpenses,
			expected: "recurring expenses",
		},
		{
			name:     "budgets state",
			state:    budgets,
			expected: "budgets",
		},
		{
			name:     "config view state",
			state:    configView,
			expected: "configuration",
		},
		{
			name:     "error state",
			state:    errorState,
			expected: "error",
		},
		{
			name:     "unknown state",
			state:    sessionState(999),
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.state.String()
			be.Equal(t, tt.expected, result)
		})
	}
}

func TestPeriodConstants(t *testing.T) {
	// Test that period constants have expected values
	be.Equal(t, "month", monthlyPeriodType)
	be.Equal(t, "year", annualPeriodType)
}

func TestSessionStateConstants(t *testing.T) {
	// Test that session state constants are defined and have different values
	be.True(t, overviewState != transactions)
	be.True(t, transactions != detailedTransaction)
	be.True(t, detailedTransaction != categorizeTransaction)
	be.True(t, categorizeTransaction != insertTransaction)
	be.True(t, insertTransaction != loading)
	be.True(t, loading != recurringExpenses)
	be.True(t, recurringExpenses != budgets)
	be.True(t, budgets != configView)
	be.True(t, configView != errorState)

	// Test that overviewState is 0 (first iota value)
	be.Equal(t, sessionState(0), overviewState)
}
