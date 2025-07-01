package overview

import (
	"strings"
	"testing"

	lm "github.com/icco/lunchmoney"
)

func TestUpdateAccountTree_CombinesDuplicateTypes(t *testing.T) {
	// Create a model with both plaid accounts and assets that have the same type
	m := New()
	m.SetCurrency("USD")

	// Create plaid accounts with investment type
	plaidAccounts := map[int64]*lm.PlaidAccount{
		1: {
			ID:     1,
			Name:   "Investment Account 1",
			Type:   "investment",
			ToBase: 1000.0,
		},
		2: {
			ID:     2,
			Name:   "Investment Account 2",
			Type:   "depository",
			ToBase: 2000.0,
		},
	}

	// Create assets with investment type (same as one of the plaid accounts)
	assets := map[int64]*lm.Asset{
		1: {
			ID:       1,
			Name:     "Asset Investment 1",
			TypeName: "investment",
			ToBase:   3000.0,
		},
		2: {
			ID:       2,
			Name:     "Real Estate",
			TypeName: "real estate",
			ToBase:   4000.0,
		},
	}

	// Set the accounts and assets
	m.SetAccounts(assets, plaidAccounts)

	// Get the tree string representation
	treeString := m.accountTree.String()

	// Check that there's only one "Investment" type node (not duplicated)
	// We'll look for the pattern where Investment appears as a branch node
	lines := strings.Split(treeString, "\n")
	investmentTypeLines := 0
	for _, line := range lines {
		// Look for lines that represent type headers (contain └── or ├── followed by just the type name)
		if (strings.Contains(line, "├── Investment") || strings.Contains(line, "└── Investment")) &&
			!strings.Contains(line, "(") { // Exclude account lines which contain amounts in parentheses
			investmentTypeLines++
		}
	}
	if investmentTypeLines != 1 {
		t.Errorf("Expected exactly one 'Investment' type node, but found %d", investmentTypeLines)
		t.Logf("Tree output:\n%s", treeString)
	}

	// Verify that both the plaid account and asset are under the same investment node
	// The tree should contain both "Investment Account 1" and "Asset Investment 1"
	if !strings.Contains(treeString, "Investment Account 1") {
		t.Error("Expected tree to contain 'Investment Account 1'")
	}
	if !strings.Contains(treeString, "Asset Investment 1") {
		t.Error("Expected tree to contain 'Asset Investment 1'")
	}

	// Verify other types are also present
	if !strings.Contains(treeString, "Depository") {
		t.Error("Expected tree to contain 'Depository'")
	}
	if !strings.Contains(treeString, "Real Estate") {
		t.Error("Expected tree to contain 'Real Estate'")
	}
}

func TestUpdateAccountTree_EmptyAccounts(t *testing.T) {
	// Test that the function doesn't crash with empty accounts
	m := New()
	m.SetCurrency("USD")
	m.SetAccounts(map[int64]*lm.Asset{}, map[int64]*lm.PlaidAccount{})

	treeString := m.accountTree.String()
	if !strings.Contains(treeString, "Accounts") {
		t.Error("Expected tree to contain root 'Accounts' node")
	}
}

func TestUpdateAccountTree_OnlyPlaidAccounts(t *testing.T) {
	// Test with only plaid accounts
	m := New()
	m.SetCurrency("USD")

	plaidAccounts := map[int64]*lm.PlaidAccount{
		1: {
			ID:     1,
			Name:   "Checking Account",
			Type:   "depository",
			ToBase: 1500.0,
		},
	}

	m.SetAccounts(map[int64]*lm.Asset{}, plaidAccounts)

	treeString := m.accountTree.String()
	if !strings.Contains(treeString, "Checking Account") {
		t.Error("Expected tree to contain 'Checking Account'")
	}
	if !strings.Contains(treeString, "Depository") {
		t.Error("Expected tree to contain 'Depository'")
	}
}

func TestUpdateAccountTree_OnlyAssets(t *testing.T) {
	// Test with only assets
	m := New()
	m.SetCurrency("USD")

	assets := map[int64]*lm.Asset{
		1: {
			ID:       1,
			Name:     "Stock Portfolio",
			TypeName: "investment",
			ToBase:   5000.0,
		},
	}

	m.SetAccounts(assets, map[int64]*lm.PlaidAccount{})

	treeString := m.accountTree.String()
	if !strings.Contains(treeString, "Stock Portfolio") {
		t.Error("Expected tree to contain 'Stock Portfolio'")
	}
	if !strings.Contains(treeString, "Investment") {
		t.Error("Expected tree to contain 'Investment'")
	}
}
