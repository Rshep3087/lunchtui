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

	// Reload data based on current session state
	switch m.previousSessionState {
	case budgets:
		// Budget loading doesn't block UI, stay in budgets state
		m.sessionState = budgets
		return m, m.getBudgets
	case overviewState,
		transactions,
		detailedTransaction,
		categorizeTransaction,
		insertTransaction,
		loading,
		recurringExpenses,
		configView,
		errorState:
		// All other states need to wait for transactions to load
		m.sessionState = loading
		m.loadingState.unset("transactions")
		return m, m.getTransactions
	}
	// Unreachable - all states are handled above
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

	// Reload data based on current session state
	switch m.previousSessionState {
	case budgets:
		// Budget loading doesn't block UI, stay in budgets state
		m.sessionState = budgets
		return m, m.getBudgets
	case overviewState,
		transactions,
		detailedTransaction,
		categorizeTransaction,
		insertTransaction,
		loading,
		recurringExpenses,
		configView,
		errorState:
		// All other states need to wait for transactions to load
		m.sessionState = loading
		m.loadingState.unset("transactions")
		return m, m.getTransactions
	}
	// Unreachable - all states are handled above
	return m, nil
}

func switchPeriodType(m *model) (tea.Model, tea.Cmd) {
	if m.periodType == monthlyPeriodType {
		m.periodType = annualPeriodType
	} else {
		m.periodType = monthlyPeriodType
	}

	m.previousSessionState = m.sessionState

	// Reload data based on current session state
	switch m.previousSessionState {
	case budgets:
		// Budget loading doesn't block UI, stay in budgets state
		m.sessionState = budgets
		return m, m.getBudgets
	case overviewState,
		transactions,
		detailedTransaction,
		categorizeTransaction,
		insertTransaction,
		loading,
		recurringExpenses,
		configView,
		errorState:
		// All other states need to wait for transactions to load
		m.sessionState = loading
		m.loadingState.unset("transactions")
		return m, m.getTransactions
	}
	// Unreachable - all states are handled above
	return m, nil
}
