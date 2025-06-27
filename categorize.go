package main

import (
	"sort"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/log"
)

func (m *model) newCategorizeTransactionForm(t transactionItem) *huh.Form {
	sort.Slice(m.categories, func(i, j int) bool {
		return m.categories[i].Name < m.categories[j].Name
	})

	opts := make([]huh.Option[int64], len(m.categories))
	for i, c := range m.categories {
		opts[i] = huh.NewOption(c.Name, c.ID)
	}

	form := huh.NewForm(huh.NewGroup(
		huh.NewSelect[int64]().
			Title("New category").
			Description("Select a new category for the transaction").
			Options(opts...).
			Key("category"),
	))

	form.SubmitCmd = func() tea.Msg { return submitCategoryForm(*m, t) }

	return form
}

func updateCategorizeTransaction(msg tea.Msg, m *model) (tea.Model, tea.Cmd) {
	form, cmd := m.categoryForm.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.categoryForm = f
	}

	if m.categoryForm.State == huh.StateCompleted {
		m.sessionState = m.previousSessionState
		log.Debug("categorize transaction form completed", "new_state", m.sessionState)
	}

	return m, cmd
}
func categorizeTransactionView(m model) string {
	return m.categoryForm.View()
}
