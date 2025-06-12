package main

import (
	"context"
	"fmt"
	"strings"

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
	filterUncategorized   key.Binding
	refreshTransactions   key.Binding
	showDetailed          key.Binding
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
		filterUncategorized: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "filter uncategorized transactions"),
		),
		refreshTransactions: key.NewBinding(
			key.WithKeys("f5"),
			key.WithHelp("f5", "refresh transactions"),
		),
		showDetailed: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "show transaction details"),
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

		if key.Matches(msg, m.transactionsListKeys.filterUncategorized) {
			return filterUncategorizedTransactions(m)
		}

		if key.Matches(msg, m.transactionsListKeys.categorizeTransaction) {
			return categorizeTrans(&m)
		}

		if key.Matches(msg, m.transactionsListKeys.refreshTransactions) {
			return refreshTransactions(m)
		}

		if key.Matches(msg, m.transactionsListKeys.showDetailed) {
			return showDetailedTransaction(m)
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

func filterUncategorizedTransactions(m model) (tea.Model, tea.Cmd) {
	uncategorizedItems := make([]list.Item, 0)
	for _, item := range m.transactions.Items() {
		if t, ok := item.(transactionItem); ok && t.t.CategoryID == 0 {
			uncategorizedItems = append(uncategorizedItems, item)
		}
	}
	m.transactions.SetItems(uncategorizedItems)

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

func showDetailedTransaction(m model) (tea.Model, tea.Cmd) {
	t, ok := m.transactions.SelectedItem().(transactionItem)
	if !ok {
		return m, nil
	}

	m.currentTransaction = &t
	m.previousSessionState = m.sessionState
	m.sessionState = detailedTransaction
	return m, nil
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

func updateDetailedTransaction(msg tea.Msg, m model) (tea.Model, tea.Cmd) {
	// Handle key messages for detailed transaction view
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.currentTransaction = nil
			m.sessionState = transactions
			return m, nil
		}
	}

	return m, nil
}

func detailedTransactionView(m model) string {
	if m.currentTransaction == nil {
		return "No transaction selected"
	}

	t := m.currentTransaction

	// Define styles for the detailed view
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#7D56F4")).
		Padding(0, 1).
		MarginBottom(1).
		Bold(true)

	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7D56F4")).
		Bold(true).
		Width(20).
		Align(lipgloss.Right)

	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FAFAFA")).
		MarginLeft(2)

	containerStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7D56F4")).
		Padding(1, 2).
		Margin(1)

	statusStyle := lipgloss.NewStyle().
		Padding(0, 1).
		Bold(true)

	// Status-specific styling
	switch t.t.Status {
	case "cleared":
		statusStyle = statusStyle.Foreground(lipgloss.Color("#FAFAFA")).Background(lipgloss.Color("#22ba46"))
	case "uncleared":
		statusStyle = statusStyle.Foreground(lipgloss.Color("#FAFAFA")).Background(lipgloss.Color("#e05951"))
	case "pending":
		statusStyle = statusStyle.Foreground(lipgloss.Color("#000000")).Background(lipgloss.Color("#7f7d78"))
	default:
		statusStyle = statusStyle.Foreground(lipgloss.Color("#FAFAFA")).Background(lipgloss.Color("#666666"))
	}

	// Parse amount
	amount, err := t.t.ParsedAmount()
	amountStr := "Error parsing amount"
	if err == nil {
		amountStr = amount.Display()
	}

	// Get account name
	accountName := "Unknown"
	if t.plaidAccount != nil {
		accountName = t.plaidAccount.Name
	} else if t.asset != nil {
		accountName = t.asset.Name
	}

	// Format tags
	tagsStr := "None"
	if len(t.tags) > 0 {
		tagNames := make([]string, len(t.tags))
		for i, tag := range t.tags {
			tagNames[i] = tag.Name
		}
		tagsStr = strings.Join(tagNames, ", ")
	}

	// Format notes
	notesStr := t.t.Notes
	if notesStr == "" {
		notesStr = "None"
	}

	// Format currency
	currencyStr := t.t.Currency
	if currencyStr == "" {
		currencyStr = "USD" // Default currency
	}

	// Create header
	header := headerStyle.Render("Transaction Details")

	// Create detail rows
	details := []string{
		lipgloss.JoinHorizontal(lipgloss.Left,
			labelStyle.Render("ID:"),
			valueStyle.Render(fmt.Sprintf("%d", t.t.ID)),
		),
		lipgloss.JoinHorizontal(lipgloss.Left,
			labelStyle.Render("Payee:"),
			valueStyle.Render(t.t.Payee),
		),
		lipgloss.JoinHorizontal(lipgloss.Left,
			labelStyle.Render("Amount:"),
			valueStyle.Render(amountStr),
		),
		lipgloss.JoinHorizontal(lipgloss.Left,
			labelStyle.Render("Currency:"),
			valueStyle.Render(currencyStr),
		),
		lipgloss.JoinHorizontal(lipgloss.Left,
			labelStyle.Render("Date:"),
			valueStyle.Render(t.t.Date),
		),
		lipgloss.JoinHorizontal(lipgloss.Left,
			labelStyle.Render("Status:"),
			statusStyle.Render(strings.ToUpper(t.t.Status)),
		),
		lipgloss.JoinHorizontal(lipgloss.Left,
			labelStyle.Render("Category:"),
			valueStyle.Render(t.category.Name),
		),
		lipgloss.JoinHorizontal(lipgloss.Left,
			labelStyle.Render("Account:"),
			valueStyle.Render(accountName),
		),
		lipgloss.JoinHorizontal(lipgloss.Left,
			labelStyle.Render("Tags:"),
			valueStyle.Render(tagsStr),
		),
	}

	// Add optional fields if they exist
	if t.t.RecurringID != 0 {
		details = append(details, lipgloss.JoinHorizontal(lipgloss.Left,
			labelStyle.Render("Recurring ID:"),
			valueStyle.Render(fmt.Sprintf("%d", t.t.RecurringID)),
		))
	}

	if t.t.GroupID != 0 {
		details = append(details, lipgloss.JoinHorizontal(lipgloss.Left,
			labelStyle.Render("Group ID:"),
			valueStyle.Render(fmt.Sprintf("%d", t.t.GroupID)),
		))
	}

	if t.t.ParentID != 0 {
		details = append(details, lipgloss.JoinHorizontal(lipgloss.Left,
			labelStyle.Render("Parent ID:"),
			valueStyle.Render(fmt.Sprintf("%d", t.t.ParentID)),
		))
	}

	if t.t.ExternalID != 0 {
		details = append(details, lipgloss.JoinHorizontal(lipgloss.Left,
			labelStyle.Render("External ID:"),
			valueStyle.Render(fmt.Sprintf("%d", t.t.ExternalID)),
		))
	}

	if t.t.IsGroup {
		details = append(details, lipgloss.JoinHorizontal(lipgloss.Left,
			labelStyle.Render("Group Transaction:"),
			valueStyle.Render("Yes"),
		))
	}

	// Add notes section
	notesSection := lipgloss.JoinHorizontal(lipgloss.Left,
		labelStyle.Render("Notes:"),
		valueStyle.Render(notesStr),
	)

	// Combine all details
	allDetails := append(details, notesSection)
	content := lipgloss.JoinVertical(lipgloss.Left, allDetails...)

	// Add instructions
	instructionsStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666")).
		Italic(true).
		MarginTop(2)

	instructions := instructionsStyle.Render("Press 'esc' to return to transaction list")

	// Final layout
	finalContent := lipgloss.JoinVertical(lipgloss.Left,
		header,
		containerStyle.Render(content),
		instructions,
	)

	return finalContent
}
