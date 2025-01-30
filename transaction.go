package main

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	return fmt.Sprintf("%s %s %s", t.t.Payee, t.category.Name, t.t.Status)
}

type transactionListKeyMap struct {
	categorizeTransaction key.Binding
}

func newTransactionListKeyMap() *transactionListKeyMap {
	return &transactionListKeyMap{
		categorizeTransaction: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "categorize transaction"),
		),
	}
}

func updateTransactions(msg tea.Msg, m model) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case updateTransactionMsg:
		// create a copy of the transaction and update the status
		// this keep the category, assets, plaidAccount, etc. intact
		t, ok := m.transactions.SelectedItem().(transactionItem)
		if !ok {
			return m, nil
		}

		t.t = msg.t
		// must set the new category on the transaction item
		// in case that is what changed
		// in future, we could check the fieldUpdated to see what changed
		t.category = m.categories[int(t.t.CategoryID)]

		setItemCmd := m.transactions.SetItem(m.transactions.Index(), t)
		statusCmd := m.transactions.NewStatusMessage(fmt.Sprintf("Updated %s for transaction: %s", msg.fieldUpdated, msg.t.Payee))

		m.transactionsStats = newTransactionStats(m.transactions.Items())

		return m, tea.Batch(setItemCmd, statusCmd)

	case tea.KeyMsg:
		// if the list is filtering, don't process key events
		if m.transactions.FilterState() == list.Filtering {
			break
		}

		if key.Matches(msg, m.transactionsListKeys.categorizeTransaction) {
			// we know which transaction we're categorizing because we're
			// updating the category for the transaction at the current index
			t := m.transactions.Items()[m.transactions.Index()].(transactionItem).t
			m.categoryForm.SubmitCmd = func() tea.Msg {
				cid := m.categoryForm.GetInt("category")

				resp, err := m.lmc.UpdateTransaction(context.TODO(), t.ID, &lm.UpdateTransaction{CategoryID: &cid})
				if err != nil {
					return err
				}

				if !resp.Updated {
					return nil
				}

				newT, err := m.lmc.GetTransaction(context.TODO(), t.ID, &lm.TransactionFilters{
					DebitAsNegative: &m.debitsAsNegative,
				})
				if err != nil {
					return err
				}

				// the transaction we get back from the API does not
				// respect the debitAsNegative setting, so we will use
				// the original transaction to update the category
				t.CategoryID = newT.CategoryID
				return updateTransactionMsg{t: t, fieldUpdated: "category"}
			}

			m.sessionState = categorizeTransaction
			return m, tea.WindowSize()
		}
	}

	var cmd tea.Cmd
	m.transactions, cmd = m.transactions.Update(msg)

	return m, cmd
}

func transactionsView(m model) string {
	return lipgloss.JoinVertical(lipgloss.Left,
		m.transactions.View(),
		m.transactionsStats.View(),
	)
}

func newTransactionStats(ts []list.Item) *transactionsStats {
	stats := transactionsStats{}

	for _, t := range ts {
		ti, ok := t.(transactionItem)
		if !ok {
			continue
		}

		if ti.t == nil {
			continue
		}

		switch ti.t.Status {
		case "pending":
			stats.pending++
		case "uncleared":
			stats.uncleared++
		case "cleared":
			stats.cleared++
		}
	}

	return &stats
}

type transactionsStats struct {
	pending   int
	uncleared int
	cleared   int
}

// View renders the transactions stats in a single line
func (t transactionsStats) View() string {
	pending := lipgloss.NewStyle().Foreground(lipgloss.Color("#7f7d78")).MarginRight(2).Render(fmt.Sprintf("%d pending", t.pending))
	uncleared := lipgloss.NewStyle().Foreground(lipgloss.Color("#e05951")).MarginRight(2).Render(fmt.Sprintf("%d uncleared", t.uncleared))
	cleared := lipgloss.NewStyle().Foreground(lipgloss.Color("#22ba46")).MarginRight(2).Render(fmt.Sprintf("%d cleared", t.cleared))

	transactionStatus := lipgloss.JoinHorizontal(lipgloss.Left, pending, uncleared, cleared)
	return lipgloss.NewStyle().
		MarginTop(1).
		MarginLeft(2).
		Render(transactionStatus)
}
