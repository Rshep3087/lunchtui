package main

import (
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
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
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.loadingSpinner, cmd = m.loadingSpinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

func overviewView(m model) string {
	if m.user == nil || len(m.transactions.Items()) == 0 {
		loadingStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ffd644"))

		return lipgloss.JoinHorizontal(lipgloss.Top,
			m.loadingSpinner.View(),
			loadingStyle.Render("Loading..."),
		)
	}

	msg := fmt.Sprintf("Welcome %s!", m.user.UserName)
	msg += "\n\n"

	// show the user summary
	msg += m.summary.View() + "\n\n"
	msg += "Press 't' to view transactions."

	return lipgloss.NewStyle().Width(80).Render(msg)
}
