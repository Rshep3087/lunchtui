package main

// Period types.
const (
	monthlyPeriodType = "month"
	annualPeriodType  = "year"
)

// Transaction status constants.
const (
	unclearedStatus = "uncleared"
	clearedStatus   = "cleared"
	pendingStatus   = "pending"
)

// Credit account type constants.
const (
	creditType        = "credit"
	creditCardSubtype = "credit card"
)

// Session states.
type sessionState int

const (
	overviewState sessionState = iota
	transactions
	detailedTransaction
	categorizeTransaction
	insertTransaction
	loading
	recurringExpenses
	budgets
	configView
	errorState
)

func (ss sessionState) String() string {
	switch ss {
	case overviewState:
		return "overview"
	case transactions:
		return "transactions"
	case detailedTransaction:
		return "transaction details"
	case categorizeTransaction:
		return "categorize transaction"
	case insertTransaction:
		return "insert transaction"
	case loading:
		return "loading"
	case recurringExpenses:
		return "recurring expenses"
	case budgets:
		return "budgets"
	case configView:
		return "configuration"
	case errorState:
		return "error"
	}

	return "unknown"
}
