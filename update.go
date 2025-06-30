package main

import (
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/log"
)

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// always check for quit key first
	if msg, ok := msg.(tea.KeyMsg); ok {
		if model, cmd := handleKeyPress(msg, &m); cmd != nil {
			log.Debug("key press handled, cmd returned")
			return model, cmd
		}
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.handleWindowSize(msg)

	case spinner.TickMsg:
		return m.handleSpinnerTick(msg)

	case getCategoriesMsg:
		return m.handleGetCategories(msg)

	case getAccountsMsg:
		return m.handleGetAccounts(msg)

	case getsTransactionsMsg:
		return m.handleGetTransactions(msg)

	case getUserMsg:
		return m.handleGetUser(msg)

	case getRecurringExpensesMsg:
		return m.handleGetRecurringExpenses(msg)

	case getTagsMsg:
		return m.handleGetTags(msg)

	case getBudgetsMsg:
		return m.handleGetBudgets(msg)

	case authErrorMsg:
		m.sessionState = errorState
		m.errorMsg = fmt.Sprintf("Check your API token: %s", msg.err.Error())
		return m, nil
	}

	var cmd tea.Cmd
	switch m.sessionState {
	case overviewState:
		m.overview, cmd = m.overview.Update(msg)
		return m, cmd

	case categorizeTransaction:
		return updateCategorizeTransaction(msg, &m)

	case detailedTransaction:
		return updateDetailedTransaction(msg, m)

	case transactions:
		return updateTransactions(msg, m)

	case recurringExpenses:
		m.recurringExpenses, cmd = m.recurringExpenses.Update(msg)
		return m, cmd

	case budgets:
		return updateBudgets(msg, m)

	case configView:
		m.configView, cmd = m.configView.Update(msg)
		return m, cmd

	case loading:
		m.loadingSpinner, cmd = m.loadingSpinner.Update(msg)
		return m, cmd
	}

	return m, nil
}
