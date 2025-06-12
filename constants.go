package main

// Period types
const (
	monthlyPeriodType = "month"
	annualPeriodType  = "year"
)

// Session states
type sessionState int

const (
	overviewState sessionState = iota
	transactions
	detailedTransaction
	categorizeTransaction
	loading
	recurringExpenses
	budgets
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
	case loading:
		return "loading"
	case recurringExpenses:
		return "recurring expenses"
	case budgets:
		return "budgets"
	}

	return "unknown"
}
