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
		nextPeriod: key.NewBinding(
			key.WithKeys("!"),
			key.WithHelp("shift+1", "next month"),
		),
		previousPeriod: key.NewBinding(
			key.WithKeys("@"),
			key.WithHelp("shift+2", "previous month"),
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

	// always quit on ctrl+c
	if k == "ctrl+c" {
		return m, tea.Quit
	}

	if k == "esc" {
		return handleEscape(m)
	}

	// check if any of the models that support filtering.
	if m.transactions.FilterState() == list.Filtering {
		return m, nil
	}

	if m.categoryForm != nil && m.categoryForm.State == huh.StateNormal {
		return m, nil
	}

	if k == "q" {
		return m, tea.Quit
	}

	if k == "!" {
		return advancePeriod(m)
	}

	if k == "@" {
		return retrievePreviousPeriod(m)
	}

	if k == "s" {
		return switchPeriodType(m)
	}

	// should this be deleted?
	if m.sessionState == loading {
		return m, nil
	}

	if k == "t" && m.sessionState != transactions {
		m.previousSessionState = m.sessionState
		m.sessionState = transactions
		return m, nil
	}

	if k == "r" && m.sessionState != recurringExpenses {
		m.previousSessionState = m.sessionState
		m.recurringExpenses.SetFocus(true)
		m.sessionState = recurringExpenses
		return m, nil
	}

	if k == "o" && m.sessionState != overviewState {
		m.previousSessionState = m.sessionState
		m.sessionState = overviewState
		return m, nil
	}

	if k == "?" && m.sessionState != transactions {
		m.help.ShowAll = !m.help.ShowAll
		return m, nil
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

	if m.sessionState == detailedTransaction {
		m.currentTransaction = nil
		m.sessionState = transactions
		return m, m.getTransactions
	}

	m.previousSessionState = m.sessionState
	m.sessionState = overviewState
	return m, nil
}
