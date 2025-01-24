package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/tree"
	lm "github.com/rshep3087/lunchmoney"
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
	doc := strings.Builder{}
	if m.user == nil || len(m.transactions.Items()) == 0 {
		loadingStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ffd644"))

		return lipgloss.JoinHorizontal(lipgloss.Top,
			m.loadingSpinner.View(),
			loadingStyle.Render("Loading..."),
		)
	}

	doc.WriteString(fmt.Sprintf("Welcome %s!\n\n", m.user.UserName))

	// show the user summary
	layer := lipgloss.JoinVertical(lipgloss.Top,
		m.accountView,
		m.summary.View(),
	)

	doc.WriteString(layer)
	doc.WriteString("\n\n")
	doc.WriteString("Press 't' to view transactions.")

	return lipgloss.NewStyle().Render(doc.String())
}

func accountView(m model) string {
	t := tree.Root("Accounts").Enumerator(tree.RoundedEnumerator)

	// organize the assets by the type into a map
	assets := make(map[string][]lm.Asset)
	for _, a := range m.assets {
		assets[a.TypeName] = append(assets[a.TypeName], *a)
	}

	// add a child for each asset
	for typeName, assets := range assets {
		assetTree := tree.New().Root(typeName)
		for _, a := range assets {
			nameAndMoney := fmt.Sprintf("%s (%s)", a.Name, a.Balance)
			assetTree.Child(nameAndMoney)
		}

		t.Child(assetTree)
	}

	// // organize the plaid accounts by the type into a map
	plaidAccounts := make(map[string][]lm.PlaidAccount)
	for _, a := range m.plaidAccounts {
		plaidAccounts[a.Type] = append(plaidAccounts[a.Type], *a)
	}

	for typeName, accounts := range plaidAccounts {
		accountTree := tree.New().Root(typeName)
		for _, a := range accounts {
			nameAndMoney := fmt.Sprintf("%s (%s)", a.Name, a.Balance)
			accountTree.Child(nameAndMoney)
		}

		t.Child(accountTree)
	}

	return lipgloss.NewStyle().MarginRight(2).Render(t.String())
}
