package main

import (
	"context"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	lm "github.com/rshep3087/lunchmoney"
)

func newCategorizeTransactionModel() list.Model {
	m := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	m.Title = "Categorize Transaction"
	// on the esc we want to go back to the transactions view
	m.DisableQuitKeybindings()
	return m
}

type categoryItem struct {
	c *lm.Category
}

func (c categoryItem) FilterValue() string { return c.c.Name }
func (c categoryItem) Title() string       { return c.c.Name }
func (c categoryItem) Description() string { return c.c.Description }

func updateCategorizeTransaction(msg tea.Msg, m *model) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.categorizeTransactions.SetSize(msg.Width-h, msg.Height-v)
		return m, nil

	case tea.KeyMsg:
		// if the list is filtering, don't process key events
		if m.transactions.FilterState() == list.Filtering {
			break
		}

		k := msg.String()
		if k == "enter" {
			// when the user presses enter, we want to categorize the transaction
			return m, func() tea.Msg {
				// get the selected transaction
				ti, ok := m.transactions.SelectedItem().(transactionItem)
				if !ok {
					return nil
				}

				// get the selected category
				ci, ok := m.categorizeTransactions.SelectedItem().(categoryItem)
				if !ok {
					return nil
				}

				resp, err := m.lmc.UpdateTransaction(context.TODO(), ti.t.ID, &lm.UpdateTransaction{CategoryID: &ci.c.ID})
				if err != nil {
					return err
				}

				if !resp.Updated {
					return nil
				}

				m.sessionState = transactions
				ti.t.CategoryID = int64(ci.c.ID)
				return updateTransactionStatusMsg{t: ti.t, fieldUpdated: "category"}
			}
		}

		if k == "esc" {
			m.sessionState = transactions
			return m, nil
		}

	}

	var cmd tea.Cmd
	m.categorizeTransactions, cmd = m.categorizeTransactions.Update(msg)

	return m, cmd
}
func categorizeTransactionView(m model) string {
	return m.categorizeTransactions.View()
}
