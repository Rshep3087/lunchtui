package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func updateOverview(msg tea.Msg, m model) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		k := msg.String()
		if k == "t" {
			m.sessionState = transactions
			return m, tea.WindowSize()
		}
	}

	return m, nil
}

func overviewView(m model) string {
	if m.user == nil || len(m.transactions.Items()) == 0 {
		return "Loading..."
	}

	msg := fmt.Sprintf("Welcome %s!", m.user.UserName)
	msg += "\n\n"

	// show the user summary
	msg += m.summary.totalIncomeEarned.Display() + "\n"
	msg += m.summary.totalSpent.Display() + "\n"
	msg += m.summary.netIncome.Display() + "\n\n"
	msg += "Press 't' to view transactions."
	return lipgloss.NewStyle().Width(80).Render(msg)
}
