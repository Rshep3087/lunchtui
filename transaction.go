package main

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	lm "github.com/icco/lunchmoney"
)

type transactionItem struct {
	t            *lm.Transaction
	category     *lm.Category
	plaidAccount *lm.PlaidAccount
	asset        *lm.Asset
	tags         []*lm.Tag
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

	tags := ""
	for _, tag := range t.tags {
		tags += tag.Name + ","
	}

	if tags == "" {
		tags = "no tags"
	}

	return fmt.Sprintf("%s | %s | %s | %s | %s | %s",
		t.t.Date,
		t.category.Name,
		amount.Display(),
		account,
		tags,
		t.t.Status,
	)
}

func (t transactionItem) FilterValue() string {
	return fmt.Sprintf("%s %s %s", t.t.Payee, t.category.Name, t.t.Status)
}

type transactionListKeyMap struct {
	categorizeTransaction key.Binding
	filterUncleared       key.Binding
	refreshTransactions   key.Binding
}

func newTransactionListKeyMap() *transactionListKeyMap {
	return &transactionListKeyMap{
		categorizeTransaction: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "categorize transaction"),
		),
		filterUncleared: key.NewBinding(
			key.WithKeys("u"),
			key.WithHelp("u", "filter uncleared transactions"),
		),
		refreshTransactions: key.NewBinding(
			key.WithKeys("f5"),
			key.WithHelp("f5", "refresh transactions"),
		),
	}
}

func updateTransactions(msg tea.Msg, m model) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case updateTransactionMsg:
		log.Debug("updating transaction")
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
		t.category = m.idToCategory[t.t.CategoryID]

		setItemCmd := m.transactions.SetItem(m.transactions.Index(), t)
		statusCmd := m.transactions.NewStatusMessage(
			fmt.Sprintf("Updated %s for transaction: %s", msg.fieldUpdated, msg.t.Payee),
		)

		m.transactionsStats = newTransactionStats(m.transactions.Items())

		// move the cursor down to the next item automatically
		m.transactions.CursorDown()

		return m, tea.Batch(setItemCmd, statusCmd)

	case tea.KeyMsg:
		if m.transactions.FilterState() == list.Filtering {
			break
		}

		if key.Matches(msg, m.transactionsListKeys.filterUncleared) {
			return filterUnclearedTransactions(m)
		}

		if key.Matches(msg, m.transactionsListKeys.categorizeTransaction) {
			return categorizeTrans(&m)
		}

		if key.Matches(msg, m.transactionsListKeys.refreshTransactions) {
			return refreshTransactions(m)
		}
	}

	var cmd tea.Cmd
	m.transactions, cmd = m.transactions.Update(msg)

	return m, cmd
}

func categorizeTrans(m *model) (tea.Model, tea.Cmd) {
	// we know which transaction we're categorizing because we're
	// updating the category for the transaction at the current index
	t, ok := m.transactions.Items()[m.transactions.Index()].(transactionItem)
	if !ok {
		return m, nil
	}

	m.categoryForm = newCategorizeTransactionForm(m.categories)
	m.categoryForm.SubmitCmd = func() tea.Msg {
		return submitCategoryForm(*m, t)
	}

	m.sessionState = categorizeTransaction
	return m, tea.Batch(m.categoryForm.Init(), tea.WindowSize())
}

func submitCategoryForm(m model, t transactionItem) tea.Msg {
	ctx := context.Background()
	categoryValue := m.categoryForm.Get("category")
	cid64, isCategoryValueValid := categoryValue.(int64)
	if !isCategoryValueValid {
		log.Debug("invalid category value", "value", categoryValue)
		return nil
	}

	cid := int(cid64)

	log.Debug("updating transaction", "transaction", t.t.ID, "category", cid)

	status := clearedStatus
	resp, err := m.lmc.UpdateTransaction(ctx, t.t.ID, &lm.UpdateTransaction{CategoryID: &cid, Status: &status})
	if err != nil {
		log.Debug("updating transaction", "error", err)
		return err
	}

	if !resp.Updated {
		log.Debug("transaction not updated")
		return nil
	}

	newT, err := m.lmc.GetTransaction(ctx, t.t.ID, &lm.TransactionFilters{DebitAsNegative: &m.debitsAsNegative})
	if err != nil {
		log.Debug("getting transaction", "error", err)
		return err
	}

	// the transaction we get back from the API does not
	// respect the debitAsNegative setting, so we will use
	// the original transaction to update the category
	t.t.CategoryID = newT.CategoryID
	t.t.Status = newT.Status
	return updateTransactionMsg{t: t.t, fieldUpdated: "category"}
}

func filterUnclearedTransactions(m model) (tea.Model, tea.Cmd) {
	unclearedItems := make([]list.Item, 0)
	for _, item := range m.transactions.Items() {
		if t, ok := item.(transactionItem); ok && t.t.Status == "uncleared" {
			unclearedItems = append(unclearedItems, item)
		}
	}
	m.transactions.SetItems(unclearedItems)

	m.transactionsStats = newTransactionStats(m.transactions.Items())
	return m, nil
}

func refreshTransactions(m model) (tea.Model, tea.Cmd) {
	log.Debug("refreshing transactions")
	// Set loading state and switch to loading view
	m.loadingState.unset("transactions")
	m.previousSessionState = m.sessionState
	m.sessionState = loading

	// Show a status message to indicate refresh is starting
	statusCmd := m.transactions.NewStatusMessage("Refreshing transactions...")

	return m, tea.Batch(statusCmd, m.getTransactions)
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

// View renders the transactions stats in a single line.
func (t transactionsStats) View() string {
	pending := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7f7d78")).
		MarginRight(2).
		Render(fmt.Sprintf("%d pending", t.pending))

	uncleared := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#e05951")).
		MarginRight(2).
		Render(fmt.Sprintf("%d uncleared", t.uncleared))

	cleared := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#22ba46")).
		MarginRight(2).
		Render(fmt.Sprintf("%d cleared", t.cleared))

	transactionStatus := lipgloss.JoinHorizontal(lipgloss.Left, pending, uncleared, cleared)
	return lipgloss.NewStyle().
		MarginTop(1).
		MarginLeft(2).
		Render(transactionStatus)
}
