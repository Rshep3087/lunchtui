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
	config         key.Binding
	nextPeriod     key.Binding
	previousPeriod key.Binding
	switchPeriod   key.Binding
	escape         key.Binding
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
			km.config,
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
		config: key.NewBinding(
			key.WithKeys("g"),
			key.WithHelp("g", "configuration"),
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
		escape: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "escape"),
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
	if model, cmd := handleSpecialKeys(msg, m); cmd != nil {
		return model, cmd
	}

	// Check if input is blocked by active forms
	if isInputBlocked(m) {
		return m, nil
	}

	// Handle navigation keys
	if model, cmd := handleNavigationKeys(msg, m); cmd != nil {
		return model, cmd
	}

	// Handle session state changes
	if model, cmd := handleSessionStateKeys(msg, m); cmd != nil {
		return model, cmd
	}

	return m, nil
}

func handleSpecialKeys(msg tea.KeyMsg, m *model) (tea.Model, tea.Cmd) {
	if key.Matches(msg, m.keys.quit) {
		return m, tea.Quit
	}

	if key.Matches(msg, m.keys.escape) {
		return handleEscape(msg, m)
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

	if m.insertTransactionForm != nil && m.insertTransactionForm.State == huh.StateNormal {
		return true
	}

	if m.sessionState == loading {
		return true
	}

	// Block input when editing transaction notes (except for detailed transaction handling)
	if m.isEditingNotes {
		return true
	}

	return false
}

func handleNavigationKeys(msg tea.KeyMsg, m *model) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.nextPeriod):
		return advancePeriod(m)
	case key.Matches(msg, m.keys.previousPeriod):
		return retrievePreviousPeriod(m)
	case key.Matches(msg, m.keys.switchPeriod):
		return switchPeriodType(m)
	}

	return m, nil
}

func handleSessionStateKeys(msg tea.KeyMsg, m *model) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.transactions):
		if m.sessionState != transactions {
			m.previousSessionState = m.sessionState
			m.sessionState = transactions
			return m, tea.Batch(m.getTransactions)
		}

	case key.Matches(msg, m.keys.recurring):
		if m.sessionState != recurringExpenses {
			m.previousSessionState = m.sessionState
			m.recurringExpenses.SetFocus(true)
			m.sessionState = recurringExpenses
			return m, nil
		}

	case key.Matches(msg, m.keys.overview):
		if m.sessionState != overviewState {
			m.previousSessionState = m.sessionState
			m.sessionState = overviewState
			return m, tea.Batch(m.getTransactions, m.getAccounts)
		}

	case key.Matches(msg, m.keys.budgets):
		if m.sessionState != budgets {
			m.previousSessionState = budgets
			m.sessionState = budgets
			return m, m.getBudgets
		}

	case key.Matches(msg, m.keys.config):
		if m.sessionState != configView {
			m.previousSessionState = m.sessionState
			m.configView.SetFocus(true)
			m.sessionState = configView
			return m, nil
		}

	case key.Matches(msg, m.keys.fullHelp):
		if m.sessionState != transactions {
			m.help.ShowAll = !m.help.ShowAll
			return m, nil
		}
	}

	return m, nil
}

// handleEscape resets the session state to the overview state.
func handleEscape(msg tea.KeyMsg, m *model) (tea.Model, tea.Cmd) {
	if m.sessionState == categorizeTransaction {
		log.Debug("handling escape in categorize transaction state")
		m.previousSessionState = overviewState
		m.sessionState = transactions
		m.categoryForm.State = huh.StateAborted
		return m, m.getTransactions
	}

	if m.sessionState == insertTransaction {
		log.Debug("handling escape in insert transaction state")
		m.previousSessionState = overviewState
		m.sessionState = transactions
		m.insertTransactionForm.State = huh.StateAborted
		return m, m.getTransactions
	}

	// handle if user is filtering transactions and presses escape
	if m.sessionState == transactions && m.transactions.FilterState() == list.Filtering {
		log.Debug("handling escape in transactions filtering")
		var cmd tea.Cmd
		m.transactions, cmd = m.transactions.Update(msg)
		return m, cmd
	}

	if m.sessionState == detailedTransaction {
		// If editing notes, just exit edit mode without leaving detailed view
		if m.isEditingNotes {
			m.isEditingNotes = false
			m.notesInput.Blur()
			return m, nil
		}
		// Otherwise, exit detailed view
		m.currentTransaction = nil
		m.sessionState = transactions
		return m, m.getTransactions
	}

	m.previousSessionState = m.sessionState
	m.sessionState = overviewState
	return m, nil
}
