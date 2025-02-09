package main

import (
	"sort"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	lm "github.com/icco/lunchmoney"
)

func newCategorizeTransactionForm(categories []*lm.Category) *huh.Form {
	sort.Slice(categories, func(i, j int) bool {
		return categories[i].Name < categories[j].Name
	})

	opts := make([]huh.Option[int64], len(categories))
	for i, c := range categories {
		opts[i] = huh.NewOption(c.Name, c.ID)
	}

	return huh.NewForm(huh.NewGroup(
		huh.NewSelect[int64]().
			Title("New category").
			Description("Select a new category for the transaction").
			Options(opts...).
			Key("category"),
	))
}

func updateCategorizeTransaction(msg tea.Msg, m *model) (tea.Model, tea.Cmd) {
	form, cmd := m.categoryForm.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.categoryForm = f
	}

	if m.categoryForm.State == huh.StateCompleted {
		m.sessionState = transactions
	}

	return m, cmd
}
func categorizeTransactionView(m model) string {
	return m.categoryForm.View()
}
