package main

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	lm "github.com/rshep3087/lunchmoney"
)

type transactionItem struct {
	t        *lm.Transaction
	category *lm.Category
}

func (t transactionItem) Title() string {
	return t.t.Payee
}

func (t transactionItem) Description() string {
	amount, err := t.t.ParsedAmount()
	if err != nil {
		return fmt.Sprintf("error parsing amount: %v", err)
	}

	return fmt.Sprintf("%s %s %s %s", t.t.Date, t.category.Name, amount.Display(), t.t.Status)
}

func (t transactionItem) FilterValue() string {
	return t.t.Payee
}

type transactionListKeyMap struct {
	overview              key.Binding
	categorizeTransaction key.Binding
}

func newTransactionListKeyMap() *transactionListKeyMap {
	return &transactionListKeyMap{
		overview: key.NewBinding(
			key.WithKeys("o"),
			key.WithHelp("o", "overview"),
		),
		categorizeTransaction: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "categorize transaction"),
		),
	}
}

func updateTransactions(msg tea.Msg, m model) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.transactions.SetSize(msg.Width-h, msg.Height-v)
		return m, nil

	case updateTransactionStatusMsg:
		setItemCmd := m.transactions.SetItem(m.transactions.Index(), transactionItem{t: msg.t, category: m.categories[int(msg.t.CategoryID)]})
		statusCmd := m.transactions.NewStatusMessage(fmt.Sprintf("Updated %s for transaction: %s", msg.fieldUpdated, msg.t.Payee))
		return m, tea.Batch(setItemCmd, statusCmd)

	case tea.KeyMsg:
		// if the list is filtering, don't process key events
		if m.transactions.FilterState() == list.Filtering {
			break
		}

		if key.Matches(msg, m.transactionsListKeys.overview) {
			m.sessionState = overview
			return m, nil
		}

		if key.Matches(msg, m.transactionsListKeys.categorizeTransaction) {
			m.sessionState = categorizeTransaction
			return m, tea.WindowSize()
		}
	}

	var cmd tea.Cmd
	m.transactions, cmd = m.transactions.Update(msg)

	return m, cmd
}

func transactionsView(m model) string {
	return m.transactions.View()
}
