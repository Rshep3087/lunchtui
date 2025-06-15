package main

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/log"
)

type keyMap struct {
	transactions   key.Binding
	overview       key.Binding
	recurring      key.Binding
	budgets        key.Binding
	nextPeriod     key.Binding
	previousPeriod key.Binding
	switchPeriod   key.Binding
	fullHelp       key.Binding
	quit           key.Binding
}

func (km keyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		km.overview,
		km.transactions,
		km.budgets,
		km.recurring,
		km.switchPeriod,
		km.quit,
		km.fullHelp,
	}
}

func (km keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{
			km.overview,
			km.transactions,
			km.budgets,
			km.recurring,
			km.quit,
			km.fullHelp,
		},
		{
			km.nextPeriod,
			km.previousPeriod,
			km.switchPeriod,
		},
	}
}

func initializeKeyMap() keyMap {
	keys := keyMap{
		transactions: key.NewBinding(
			key.WithKeys("t"),
			key.WithHelp("t", "transactions"),
		),
		overview: key.NewBinding(
			key.WithKeys("o"),
			key.WithHelp("o", "overview"),
		),
		recurring: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "recurring expenses"),
		),
		budgets: key.NewBinding(
			key.WithKeys("b"),
			key.WithHelp("b", "budgets"),
		),
		nextPeriod: key.NewBinding(
			key.WithKeys("]"),
			key.WithHelp("]", "next month"),
		),
		previousPeriod: key.NewBinding(
			key.WithKeys("["),
			key.WithHelp("[", "previous month"),
		),
		switchPeriod: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "switch range"),
		),
		fullHelp: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}
	return keys
}

func handleKeyPress(msg tea.KeyMsg, m *model) (tea.Model, tea.Cmd) {
	k := msg.String()
	log.Debug("key pressed", "key", k)

	// Handle special keys first
	if model, cmd := handleSpecialKeys(k, m); cmd != nil {
		return model, cmd
	}

	// Check if input is blocked by active forms
	if isInputBlocked(m) {
		return m, nil
	}

	// Handle navigation keys
	if model, cmd := handleNavigationKeys(k, m); cmd != nil {
		return model, cmd
	}

	// Handle session state changes
	if model, cmd := handleSessionStateKeys(k, m); cmd != nil {
		return model, cmd
	}

	return m, nil
}

func handleSpecialKeys(k string, m *model) (tea.Model, tea.Cmd) {
	if k == "ctrl+c" || k == "q" {
		return m, tea.Quit
	}

	if k == "esc" {
		return handleEscape(m)
	}

	return m, nil
}

func isInputBlocked(m *model) bool {
	if m.transactions.FilterState() == list.Filtering {
		return true
	}

	if m.categoryForm != nil && m.categoryForm.State == huh.StateNormal {
		return true
	}

	if m.insertTransactionForm != nil && m.insertTransactionForm.State == huh.StateNormal {
		return true
	}

	if m.sessionState == loading {
		return true
	}

	return false
}

func handleNavigationKeys(k string, m *model) (tea.Model, tea.Cmd) {
	switch k {
	case "]":
		return advancePeriod(m)
	case "[":
		return retrievePreviousPeriod(m)
	case "s":
		return switchPeriodType(m)
	}

	return m, nil
}

func handleSessionStateKeys(k string, m *model) (tea.Model, tea.Cmd) {
	switch k {
	case "t":
		if m.sessionState != transactions {
			m.previousSessionState = m.sessionState
			m.sessionState = transactions
			return m, nil
		}
	case "r":
		if m.sessionState != recurringExpenses {
			m.previousSessionState = m.sessionState
			m.recurringExpenses.SetFocus(true)
			m.sessionState = recurringExpenses
			return m, nil
		}
	case "o":
		if m.sessionState != overviewState {
			m.previousSessionState = m.sessionState
			m.sessionState = overviewState
			return m, nil
		}
	case "b":
		if m.sessionState != budgets {
			m.previousSessionState = budgets
			m.loadingState.unset("budgets")
			m.sessionState = loading
			return m, m.getBudgets
		}
	case "?":
		if m.sessionState != transactions {
			m.help.ShowAll = !m.help.ShowAll
			return m, nil
		}
	}

	return m, nil
}

// handleEscape resets the session state to the overview state.
func handleEscape(m *model) (tea.Model, tea.Cmd) {
	if m.sessionState == categorizeTransaction {
		m.previousSessionState = overviewState
		m.sessionState = transactions
		m.categoryForm.State = huh.StateAborted
		return m, m.getTransactions
	}

	if m.sessionState == insertTransaction {
		m.previousSessionState = overviewState
		m.sessionState = transactions
		m.insertTransactionForm.State = huh.StateAborted
		return m, nil // Don't refresh transactions on cancel
	}

	if m.sessionState == detailedTransaction {
		m.currentTransaction = nil
		m.sessionState = transactions
		return m, m.getTransactions
	}

	m.previousSessionState = m.sessionState
	m.sessionState = overviewState
	return m, nil
}
