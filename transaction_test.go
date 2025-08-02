package main

import (
	"testing"

	"github.com/carlmjohnson/be"
	lm "github.com/icco/lunchmoney"
)

func TestTransactionItemTitle(t *testing.T) {
	tests := []struct {
		name     string
		trans    *lm.Transaction
		expected string
	}{
		{
			name: "basic transaction",
			trans: &lm.Transaction{
				ID:    123,
				Payee: "Coffee Shop",
			},
			expected: "Coffee Shop (123)",
		},
		{
			name: "empty payee",
			trans: &lm.Transaction{
				ID:    456,
				Payee: "",
			},
			expected: " (456)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := transactionItem{t: tt.trans}
			result := item.Title()
			be.Equal(t, tt.expected, result)
		})
	}
}

func TestTransactionItemDescription(t *testing.T) {
	tests := []struct {
		name         string
		trans        *lm.Transaction
		category     *lm.Category
		plaidAccount *lm.PlaidAccount
		asset        *lm.Asset
		tags         []*lm.Tag
		expected     string
	}{
		{
			name: "transaction with plaid account and tags",
			trans: &lm.Transaction{
				Date:   "2023-12-01",
				Amount: "10.50",
				Status: "cleared",
				Notes:  "Coffee break",
			},
			category:     &lm.Category{Name: "Food & Dining"},
			plaidAccount: &lm.PlaidAccount{Name: "Checking Account"},
			tags: []*lm.Tag{
				{Name: "work"},
				{Name: "coffee"},
			},
			expected: "2023-12-01 | Food & Dining | 10.50 | Checking Account | work,coffee, | cleared | Coffee break",
		},
		{
			name: "transaction with asset account",
			trans: &lm.Transaction{
				Date:   "2023-12-02",
				Amount: "25.00",
				Status: "pending",
				Notes:  "",
			},
			category: &lm.Category{Name: "Groceries"},
			asset:    &lm.Asset{Name: "Savings Account"},
			tags:     []*lm.Tag{},
			expected: "2023-12-02 | Groceries | 25.00 | Savings Account | no tags | pending | no notes",
		},
		{
			name: "transaction with long notes",
			trans: &lm.Transaction{
				Date:   "2023-12-03",
				Amount: "5.99",
				Status: "cleared",
				Notes:  "This is a very long note that should be truncated because it exceeds the maximum length",
			},
			category:     &lm.Category{Name: "Entertainment"},
			plaidAccount: &lm.PlaidAccount{Name: "Credit Card"},
			tags:         []*lm.Tag{},
			expected:     "2023-12-03 | Entertainment | 5.99 | Credit Card | no tags | cleared | This is a very long note that should be ...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := transactionItem{
				t:            tt.trans,
				category:     tt.category,
				plaidAccount: tt.plaidAccount,
				asset:        tt.asset,
				tags:         tt.tags,
			}
			result := item.Description()
			be.Equal(t, tt.expected, result)
		})
	}
}

func TestTransactionItemFilterValue(t *testing.T) {
	tests := []struct {
		name     string
		trans    *lm.Transaction
		category *lm.Category
		expected string
	}{
		{
			name: "basic filter value",
			trans: &lm.Transaction{
				Payee:  "Starbucks",
				Status: "cleared",
			},
			category: &lm.Category{Name: "Coffee"},
			expected: "Starbucks Coffee cleared",
		},
		{
			name: "empty values",
			trans: &lm.Transaction{
				Payee:  "",
				Status: "",
			},
			category: &lm.Category{Name: ""},
			expected: "  ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := transactionItem{
				t:        tt.trans,
				category: tt.category,
			}
			result := item.FilterValue()
			be.Equal(t, tt.expected, result)
		})
	}
}

func TestNewTransactionListKeyMap(t *testing.T) {
	keyMap := newTransactionListKeyMap()

	// Test that all key bindings are initialized
	be.Nonzero(t, keyMap.categorizeTransaction)
	be.Nonzero(t, keyMap.filterUncleared)
	be.Nonzero(t, keyMap.filterUncategorized)
	be.Nonzero(t, keyMap.refreshTransactions)
	be.Nonzero(t, keyMap.showDetailed)
	be.Nonzero(t, keyMap.insertTransaction)

	// Test key bindings have expected keys
	be.Equal(t, "c", keyMap.categorizeTransaction.Keys()[0])
	be.Equal(t, "u", keyMap.filterUncleared.Keys()[0])
}
