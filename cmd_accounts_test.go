package main

import (
	"testing"

	"github.com/carlmjohnson/be"
	lm "github.com/icco/lunchmoney"
)

func TestConvertAssetToAccount(t *testing.T) {
	tests := []struct {
		name     string
		asset    *lm.Asset
		expected Account
	}{
		{
			name: "basic asset conversion",
			asset: &lm.Asset{
				ID:              123,
				Name:            "Savings Account",
				TypeName:        "depository",
				SubtypeName:     "savings",
				Balance:         "1500.50",
				Currency:        "USD",
				InstitutionName: "Test Bank",
				Status:          "active",
			},
			expected: Account{
				ID:              123,
				Name:            "Savings Account",
				Type:            "depository",
				Subtype:         "savings",
				Balance:         "1500.50",
				Currency:        "USD",
				InstitutionName: "Test Bank",
				Status:          "active",
				AccountType:     "asset",
			},
		},
		{
			name: "empty asset conversion",
			asset: &lm.Asset{
				ID:              0,
				Name:            "",
				TypeName:        "",
				SubtypeName:     "",
				Balance:         "",
				Currency:        "",
				InstitutionName: "",
				Status:          "",
			},
			expected: Account{
				ID:              0,
				Name:            "",
				Type:            "",
				Subtype:         "",
				Balance:         "",
				Currency:        "",
				InstitutionName: "",
				Status:          "",
				AccountType:     "asset",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertAssetToAccount(tt.asset)
			be.Equal(t, tt.expected, result)
		})
	}
}

func TestConvertPlaidAccountToAccount(t *testing.T) {
	tests := []struct {
		name         string
		plaidAccount *lm.PlaidAccount
		expected     Account
	}{
		{
			name: "basic plaid account conversion",
			plaidAccount: &lm.PlaidAccount{
				ID:              456,
				Name:            "Checking Account",
				Type:            "depository",
				Subtype:         "checking",
				Balance:         "2500.75",
				Currency:        "USD",
				InstitutionName: "Chase Bank",
				Status:          "active",
			},
			expected: Account{
				ID:              456,
				Name:            "Checking Account",
				Type:            "depository",
				Subtype:         "checking",
				Balance:         "2500.75",
				Currency:        "USD",
				InstitutionName: "Chase Bank",
				Status:          "active",
				AccountType:     "plaid",
			},
		},
		{
			name: "credit card plaid account",
			plaidAccount: &lm.PlaidAccount{
				ID:              789,
				Name:            "Credit Card",
				Type:            "credit",
				Subtype:         "credit card",
				Balance:         "-500.25",
				Currency:        "USD",
				InstitutionName: "Capital One",
				Status:          "active",
			},
			expected: Account{
				ID:              789,
				Name:            "Credit Card",
				Type:            "credit",
				Subtype:         "credit card",
				Balance:         "-500.25",
				Currency:        "USD",
				InstitutionName: "Capital One",
				Status:          "active",
				AccountType:     "plaid",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertPlaidAccountToAccount(tt.plaidAccount)
			be.Equal(t, tt.expected, result)
		})
	}
}
