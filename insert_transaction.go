package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
)

func updateInsertTransaction(msg tea.Msg, m *model) (tea.Model, tea.Cmd) {
	form, cmd := m.insertTransactionForm.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.insertTransactionForm = f
	}

	if m.insertTransactionForm.State == huh.StateCompleted {
		m.sessionState = transactions
		return m, m.getTransactions // Refresh transactions after insert
	}

	if m.insertTransactionForm.State == huh.StateAborted {
		m.sessionState = transactions
		return m, nil // Don't refresh if cancelled
	}

	return m, cmd
}
