package main

import (
	"github.com/Rhymond/go-money"
)

// NetWorthData represents the complete net worth calculation results.
type NetWorthData struct {
	NetWorth         *money.Money
	TotalAssets      *money.Money
	TotalLiabilities *money.Money
	Currency         string
	Breakdown        *NetWorthBreakdown
}

// NetWorthBreakdown provides detailed account breakdown by type.
type NetWorthBreakdown struct {
	Assets      map[string][]*AccountSummary
	Liabilities map[string][]*AccountSummary
}

// AccountSummary represents an individual account in the net worth calculation.
// This includes both the money.Money object for calculations and display formatting.
type AccountSummary struct {
	ID              int64        `json:"id"`
	Name            string       `json:"name"`
	DisplayName     string       `json:"display_name,omitempty"`
	Type            string       `json:"type"`
	Subtype         string       `json:"subtype,omitempty"`
	Amount          *money.Money `json:"-"` // Not exported in JSON, used for calculations
	InstitutionName string       `json:"institution_name,omitempty"`
	AccountType     string       `json:"account_type"` // "asset" or "plaid"
}

// NetWorthJSONSummary converts NetWorthData to a JSON-friendly format for CLI output.
type NetWorthJSONSummary struct {
	NetWorth         string                 `json:"net_worth"`
	Currency         string                 `json:"currency"`
	TotalAssets      string                 `json:"total_assets"`
	TotalLiabilities string                 `json:"total_liabilities"`
	Breakdown        *NetWorthJSONBreakdown `json:"breakdown,omitempty"`
}

// NetWorthJSONBreakdown is the JSON-friendly version of NetWorthBreakdown.
type NetWorthJSONBreakdown struct {
	Assets      map[string][]*AccountJSONSummary `json:"assets"`
	Liabilities map[string][]*AccountJSONSummary `json:"liabilities"`
}

// AccountJSONSummary is the JSON-friendly version of AccountSummary.
type AccountJSONSummary struct {
	ID              int64  `json:"id"`
	Name            string `json:"name"`
	DisplayName     string `json:"display_name,omitempty"`
	Type            string `json:"type"`
	Subtype         string `json:"subtype,omitempty"`
	Amount          string `json:"amount"`
	InstitutionName string `json:"institution_name,omitempty"`
	AccountType     string `json:"account_type"`
}

// ToJSON converts NetWorthData to JSON-friendly format.
func (nw *NetWorthData) ToJSON() *NetWorthJSONSummary {
	summary := &NetWorthJSONSummary{
		NetWorth:         nw.NetWorth.Display(),
		Currency:         nw.Currency,
		TotalAssets:      nw.TotalAssets.Display(),
		TotalLiabilities: nw.TotalLiabilities.Display(),
	}

	if nw.Breakdown != nil {
		summary.Breakdown = &NetWorthJSONBreakdown{
			Assets:      make(map[string][]*AccountJSONSummary),
			Liabilities: make(map[string][]*AccountJSONSummary),
		}

		for category, accounts := range nw.Breakdown.Assets {
			for _, account := range accounts {
				summary.Breakdown.Assets[category] = append(summary.Breakdown.Assets[category], &AccountJSONSummary{
					ID:              account.ID,
					Name:            account.Name,
					DisplayName:     account.DisplayName,
					Type:            account.Type,
					Subtype:         account.Subtype,
					Amount:          account.Amount.Display(),
					InstitutionName: account.InstitutionName,
					AccountType:     account.AccountType,
				})
			}
		}

		for category, accounts := range nw.Breakdown.Liabilities {
			for _, account := range accounts {
				summary.Breakdown.Liabilities[category] = append(
					summary.Breakdown.Liabilities[category],
					&AccountJSONSummary{
						ID:              account.ID,
						Name:            account.Name,
						DisplayName:     account.DisplayName,
						Type:            account.Type,
						Subtype:         account.Subtype,
						Amount:          "-" + account.Amount.Display(),
						InstitutionName: account.InstitutionName,
						AccountType:     account.AccountType,
					},
				)
			}
		}
	}

	return summary
}

// GetDisplayName returns the display name if available, otherwise the regular name.
func (a *AccountSummary) GetDisplayName() string {
	return unescapeDisplayName(a.Name, a.DisplayName)
}

// IsLiability determines if this account is a liability based on type and subtype.
func (a *AccountSummary) IsLiability() bool {
	return a.Type == creditType && a.Subtype == creditCardSubtype
}
