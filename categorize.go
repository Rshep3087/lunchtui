package main

import (
	"maps"
	"slices"
	"sort"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	lm "github.com/rshep3087/lunchmoney"
)

func newCategorizeTransactionForm(categories []*lm.Category) *huh.Form { // Sort categories by Name
	sort.Slice(categories, func(i, j int) bool {
		return categories[i].Name < categories[j].Name
	})

	opts := make([]huh.Option[int], len(categories))
	for i, c := range categories {
		opts[i] = huh.NewOption(c.Name, c.ID)
	}

	return huh.NewForm(huh.NewGroup(
		huh.NewSelect[int]().
			Title("New category").
			Description("Select a new category for the transaction").
			Options(opts...).
			Key("category"),
	))
}

func updateCategorizeTransaction(msg tea.Msg, m *model) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	form, cmd := m.categoryForm.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.categoryForm = f
	}

	cmds = append(cmds, cmd)

	if m.categoryForm.State == huh.StateCompleted {
		m.sessionState = transactions
		m.categoryForm = newCategorizeTransactionForm(slices.Collect(maps.Values(m.categories)))
		cmds = append(cmds, m.categoryForm.Init(), tea.WindowSize())
	}

	return m, tea.Batch(cmds...)
}
func categorizeTransactionView(m model) string {
	return m.categoryForm.View()
}
