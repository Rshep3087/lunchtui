package main

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	lm "github.com/rshep3087/lunchmoney"
)

type transactionItem struct {
	t            *lm.Transaction
	category     *lm.Category
	plaidAccount *lm.PlaidAccount
	asset        *lm.Asset
}

func (t transactionItem) Title() string {
	return t.t.Payee
}

func (t transactionItem) Description() string {
	amount, err := t.t.ParsedAmount()
	if err != nil {
		return fmt.Sprintf("error parsing amount: %v", err)
	}

	var account string
	if t.plaidAccount != nil {
		account = t.plaidAccount.Name
	} else if t.asset != nil {
		account = t.asset.Name
	}

	return fmt.Sprintf("%s | %s | %s | %s | %s", t.t.Date, t.category.Name, amount.Display(), account, t.t.Status)
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
		// create a copy of the transaction and update the status
		// this keep the category, assets, plaidAccount, etc. intact
		t, ok := m.transactions.SelectedItem().(transactionItem)
		if !ok {
			return m, nil
		}

		t.t = msg.t

		setItemCmd := m.transactions.SetItem(m.transactions.Index(), t)
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
