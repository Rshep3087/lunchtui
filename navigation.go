package main

import (
	tea "github.com/charmbracelet/bubbletea"
)

// advancePeriod advances the current period by one month or year depending on the period type.
func advancePeriod(m *model) (tea.Model, tea.Cmd) {
	if m.periodType == monthlyPeriodType {
		m.currentPeriod = m.currentPeriod.AddDate(0, 1, 0)
	}

	if m.periodType == annualPeriodType {
		m.currentPeriod = m.currentPeriod.AddDate(1, 0, 0)
	}

	m.previousSessionState = m.sessionState
	m.sessionState = loading

	// Reload data based on current session state
	switch m.previousSessionState {
	case budgets:
		m.loadingState.unset("budgets")
		return m, m.getBudgets
	case overviewState:
		m.loadingState.unset("transactions")
		return m, m.getTransactions
	case transactions:
		m.loadingState.unset("transactions")
		return m, m.getTransactions
	case detailedTransaction:
		m.loadingState.unset("transactions")
		return m, m.getTransactions
	case categorizeTransaction:
		m.loadingState.unset("transactions")
		return m, m.getTransactions
	case loading:
		m.loadingState.unset("transactions")
		return m, m.getTransactions
	case recurringExpenses:
		m.loadingState.unset("transactions")
		return m, m.getTransactions
	}

	return m, nil
}

// retrievePreviousPeriod retrieves the previous period by one month or year depending on the period type.
func retrievePreviousPeriod(m *model) (tea.Model, tea.Cmd) {
	if m.periodType == monthlyPeriodType {
		m.currentPeriod = m.currentPeriod.AddDate(0, -1, 0)
	}

	if m.periodType == annualPeriodType {
		m.currentPeriod = m.currentPeriod.AddDate(-1, 0, 0)
	}

	m.previousSessionState = m.sessionState
	m.sessionState = loading

	// Reload data based on current session state
	switch m.previousSessionState {
	case budgets:
		m.loadingState.unset("budgets")
		return m, m.getBudgets
	case overviewState:
		m.loadingState.unset("transactions")
		return m, m.getTransactions
	case transactions:
		m.loadingState.unset("transactions")
		return m, m.getTransactions
	case detailedTransaction:
		m.loadingState.unset("transactions")
		return m, m.getTransactions
	case categorizeTransaction:
		m.loadingState.unset("transactions")
		return m, m.getTransactions
	case loading:
		m.loadingState.unset("transactions")
		return m, m.getTransactions
	case recurringExpenses:
		m.loadingState.unset("transactions")
		return m, m.getTransactions
	}

	return m, nil
}

func switchPeriodType(m *model) (tea.Model, tea.Cmd) {
	if m.periodType == monthlyPeriodType {
		m.periodType = annualPeriodType
	} else {
		m.periodType = monthlyPeriodType
	}

	m.previousSessionState = m.sessionState
	m.sessionState = loading

	// Reload data based on current session state
	switch m.previousSessionState {
	case budgets:
		m.loadingState.unset("budgets")
		return m, m.getBudgets
	case overviewState:
		m.loadingState.unset("transactions")
		return m, m.getTransactions
	case transactions:
		m.loadingState.unset("transactions")
		return m, m.getTransactions
	case detailedTransaction:
		m.loadingState.unset("transactions")
		return m, m.getTransactions
	case categorizeTransaction:
		m.loadingState.unset("transactions")
		return m, m.getTransactions
	case loading:
		m.loadingState.unset("transactions")
		return m, m.getTransactions
	case recurringExpenses:
		m.loadingState.unset("transactions")
		return m, m.getTransactions
	}

	return m, nil
}
