package main

import (
	"context"
	"errors"
	"fmt"
	"html"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
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
	return fmt.Sprintf("%s (%d)", t.t.Payee, t.t.ID)
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

	// Format notes with truncation
	notes := t.t.Notes
	if notes == "" {
		notes = "no notes"
	} else {
		// Truncate notes to 40 characters and add ellipsis if longer
		const maxNoteLength = 40
		if len(notes) > maxNoteLength {
			notes = notes[:maxNoteLength] + "..."
		}
	}

	return fmt.Sprintf("%s | %s | %s | %s | %s | %s | %s",
		t.t.Date,
		t.category.Name,
		amount.Display(),
		account,
		tags,
		t.t.Status,
		notes,
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
	insertTransaction     key.Binding
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
		insertTransaction: key.NewBinding(
			key.WithKeys("i"),
			key.WithHelp("i", "insert new transaction"),
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
		log.Debug("updating transaction", "transaction", msg.t.ID, "field", msg.fieldUpdated)

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

		if key.Matches(msg, m.transactionsListKeys.insertTransaction) {
			log.Debug("switching to insert transaction form")
			m.previousSessionState = m.sessionState
			m.sessionState = insertTransaction
			m.insertTransactionForm = m.newInsertTransactionForm()
			return m, m.insertTransactionForm.Init()
		}
	}

	var cmd tea.Cmd
	m.transactions, cmd = m.transactions.Update(msg)

	return m, cmd
}

type accountOpt struct {
	ID   int64
	Type string // "plaid" or "asset" or "cash"
}

func (m model) newInsertTransactionForm() *huh.Form {
	categoryOpts := m.generateCategoryOptions()
	tagOpts := m.generateTagOptions()
	accountOpts := m.generateAccountOptions()

	form := huh.NewForm(
		// first group contains the main transaction fields
		huh.NewGroup(
			huh.NewSelect[accountOpt]().Title("Account").Key("account").
				Height(6).Options(accountOpts...),
			huh.NewInput().Title("Payee").Key("payee").Description("The payee for the transaction").
				Validate(func(s string) error {
					if s == "" {
						return errors.New("payee cannot be empty")
					}
					return nil
				}),
			huh.NewInput().Title("Amount").Key("amount").Description("Enter the amount (e.g., 10.00)").
				Validate(func(s string) error {
					if s == "" {
						return errors.New("amount cannot be empty")
					}
					if _, err := strconv.ParseFloat(s, 64); err != nil {
						return fmt.Errorf("invalid amount: %w", err)
					}
					return nil
				}),
			huh.NewInput().Title("Date").Key("date").Description("Enter the date in YYYY-MM-DD format").
				Validate(func(s string) error {
					if len(s) != 10 {
						return errors.New("date must be in YYYY-MM-DD format")
					}
					if _, err := time.Parse("2006-01-02", s); err != nil {
						return fmt.Errorf("invalid date format: %w", err)
					}
					return nil
				}),
			huh.NewSelect[int64]().Title("Category").Key("category").
				Height(8).Options(categoryOpts...),
		),
		// second group contains optional fields
		huh.NewGroup(
			huh.NewSelect[string]().Options(
				huh.NewOption("Uncleared", unclearedStatus),
				huh.NewOption("Cleared", clearedStatus),
			).Key("status").Title("Status").Description("Select the transaction status"),
			huh.NewMultiSelect[int]().Options(tagOpts...).
				Title("Tags").Key("tags").Description("Select tag(s) for the transaction"),
			huh.NewText().Title("Notes").Key("notes").Description("Optional notes for the transaction"),
			huh.NewConfirm().Title("Create").Key("submit"),
		),
	).WithShowHelp(true).WithShowErrors(true)

	return form
}

func (m model) generateCategoryOptions() []huh.Option[int64] {
	categoryOpts := make([]huh.Option[int64], len(m.categories))
	for i, c := range m.categories {
		categoryOpts[i] = huh.NewOption(c.Name, c.ID)
	}
	return categoryOpts
}

func (m model) generateTagOptions() []huh.Option[int] {
	tagOpts := make([]huh.Option[int], 0, len(m.tags))
	for _, tag := range m.tags {
		tagOpts = append(tagOpts, huh.NewOption(html.UnescapeString(tag.Name), tag.ID))
	}
	return tagOpts
}

func (m model) generateAccountOptions() []huh.Option[accountOpt] {
	accountOpts := make([]huh.Option[accountOpt], 0, len(m.plaidAccounts)+len(m.assets)+1)
	accountOpts = append(accountOpts, huh.NewOption("Cash", accountOpt{}))
	for _, account := range m.plaidAccounts {
		accountOpts = append(accountOpts, huh.NewOption(html.UnescapeString(account.DisplayName), accountOpt{
			ID:   account.ID,
			Type: "plaid",
		}))
	}

	for _, asset := range m.assets {
		accountOpts = append(accountOpts, huh.NewOption(html.UnescapeString(asset.Name), accountOpt{
			ID:   asset.ID,
			Type: "asset",
		}))
	}
	return accountOpts
}

func categorizeTrans(m *model) (tea.Model, tea.Cmd) {
	// we know which transaction we're categorizing because we're
	// updating the category for the transaction at the current index
	t, ok := m.transactions.Items()[m.transactions.Index()].(transactionItem)
	if !ok {
		return m, nil
	}

	m.categoryForm = m.newCategorizeTransactionForm(t)

	m.previousSessionState = m.sessionState
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
	if m.isFilteredUncleared {
		// If already filtered, show all transactions
		m.transactions.SetItems(m.originalTransactions)
		m.isFilteredUncleared = false
	} else {
		// If not filtered, show only uncleared transactions
		unclearedItems := make([]list.Item, 0)
		for _, item := range m.originalTransactions {
			if t, ok := item.(transactionItem); ok && t.t.Status == unclearedStatus {
				unclearedItems = append(unclearedItems, item)
			}
		}
		m.transactions.SetItems(unclearedItems)
		m.isFilteredUncleared = true
	}

	m.transactionsStats = newTransactionStats(m.transactions.Items())
	return m, nil
}

func filterUncategorizedTransactions(m model) (tea.Model, tea.Cmd) {
	uncategorizedItems := make([]list.Item, 0)
	for _, item := range m.originalTransactions {
		if t, ok := item.(transactionItem); ok && t.t.CategoryID == 0 {
			uncategorizedItems = append(uncategorizedItems, item)
		}
	}
	m.transactions.SetItems(uncategorizedItems)
	// Reset the uncleared filter state since we're applying a different filter
	m.isFilteredUncleared = false

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
		m.transactionsStats.View(m.theme),
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
		case pendingStatus:
			stats.pending++
		case unclearedStatus:
			stats.uncleared++
		case clearedStatus:
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
func (t transactionsStats) View(theme Theme) string {
	pending := lipgloss.NewStyle().
		Foreground(theme.Muted).
		MarginRight(2).
		Render(fmt.Sprintf("%d pending", t.pending))

	uncleared := lipgloss.NewStyle().
		Foreground(theme.Warning).
		MarginRight(2).
		Render(fmt.Sprintf("%d uncleared", t.uncleared))

	cleared := lipgloss.NewStyle().
		Foreground(theme.Success).
		MarginRight(2).
		Render(fmt.Sprintf("%d cleared", t.cleared))

	transactionStatus := lipgloss.JoinHorizontal(lipgloss.Left, pending, uncleared, cleared)
	return lipgloss.NewStyle().
		MarginTop(1).
		MarginLeft(2).
		Render(transactionStatus)
}

func updateDetailedTransaction(msg tea.Msg, m model) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	// Handle key messages
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if m.isEditingNotes {
			// Handle keys while editing notes
			switch keyMsg.String() {
			case "enter":
				// Save the notes and exit edit mode
				newNotes := m.notesInput.Value()
				m.isEditingNotes = false
				m.notesInput.Blur()
				return m, m.updateTransactionNotes(newNotes)
			case "esc":
				// Cancel editing and revert to view mode
				m.isEditingNotes = false
				m.notesInput.Blur()
				return m, nil
			default:
				// Update the text input
				m.notesInput, cmd = m.notesInput.Update(msg)
				return m, cmd
			}
		}

		// Handle keys in view mode
		switch keyMsg.String() {
		case "n":
			// Enter edit mode for notes
			m.isEditingNotes = true
			m.notesInput.Focus()
			// Pre-fill with current notes if they exist
			if m.currentTransaction != nil && m.currentTransaction.t.Notes != "" {
				m.notesInput.SetValue(m.currentTransaction.t.Notes)
			} else {
				m.notesInput.SetValue("")
			}
			return m, nil
		case "c":
			m.previousSessionState = m.sessionState
			m.sessionState = categorizeTransaction

			m.categoryForm = m.newCategorizeTransactionForm(*m.currentTransaction)
			return m, tea.Batch(m.categoryForm.Init(), tea.WindowSize())
		}
	}

	// Update the text input if it's active
	if m.isEditingNotes {
		m.notesInput, cmd = m.notesInput.Update(msg)
		return m, cmd
	}

	if msg, ok := msg.(updateTransactionMsg); ok {
		log.Debug("updating detailed transaction", "transaction", msg.t.ID, "field", msg.fieldUpdated)
		// Update the current transaction with the new data
		if m.currentTransaction == nil {
			log.Error("current transaction is nil, cannot update")
			return m, nil
		}

		m.currentTransaction.t = msg.t
		// Update the category if it changed
		if category, catOK := m.idToCategory[msg.t.CategoryID]; catOK {
			m.currentTransaction.category = category
		}

		return m, nil
	}

	return m, nil
}

// updateTransactionNotes updates the notes for the current transaction.
func (m model) updateTransactionNotes(newNotes string) tea.Cmd {
	return func() tea.Msg {
		if m.currentTransaction == nil {
			return nil
		}

		ctx := context.Background()
		updateReq := &lm.UpdateTransaction{Notes: &newNotes}

		resp, err := m.lmc.UpdateTransaction(ctx, m.currentTransaction.t.ID, updateReq)
		if err != nil {
			log.Error("failed to update transaction notes", "error", err)
			return err
		}

		if !resp.Updated {
			log.Debug("transaction notes not updated")
			return nil
		}

		// Update the local transaction with new notes
		m.currentTransaction.t.Notes = newNotes
		log.Debug("transaction notes updated successfully", "notes", newNotes)
		return updateTransactionMsg{t: m.currentTransaction.t, fieldUpdated: "notes"}
	}
}

func detailedTransactionView(m model) string {
	if m.currentTransaction == nil {
		return "No transaction selected"
	}

	styles := createDetailedTransactionStyles(m.theme)
	data := extractTransactionData(m.currentTransaction, m)

	header := styles.headerStyle.Render("Transaction Details")
	details := buildTransactionDetailsWithNotes(data, styles, m)
	content := lipgloss.JoinVertical(lipgloss.Left, details...)
	instructions := createInstructionsWithNotes(styles, m.isEditingNotes)

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		styles.containerStyle.Render(content),
		instructions,
	)
}

type detailedTransactionStyles struct {
	headerStyle      lipgloss.Style
	labelStyle       lipgloss.Style
	valueStyle       lipgloss.Style
	containerStyle   lipgloss.Style
	statusStyle      lipgloss.Style
	instructionStyle lipgloss.Style
}

type transactionDisplayData struct {
	transaction *transactionItem
	amountStr   string
	accountName string
	tagsStr     string
	notesStr    string
	currencyStr string
	statusStyle lipgloss.Style
}

func createDetailedTransactionStyles(theme Theme) detailedTransactionStyles {
	return detailedTransactionStyles{
		headerStyle: lipgloss.NewStyle().
			Foreground(theme.Text).
			Background(theme.Background).
			Padding(0, 1).
			MarginBottom(1).
			Bold(true),
		labelStyle: lipgloss.NewStyle().
			Foreground(theme.Border).
			Bold(true).
			Width(20).
			Align(lipgloss.Right),
		valueStyle: lipgloss.NewStyle().
			Foreground(theme.Text).
			MarginLeft(2),
		containerStyle: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(theme.Border).
			Padding(1, 2).
			Margin(1),
		statusStyle: lipgloss.NewStyle().
			Padding(0, 1).
			Bold(true),
		instructionStyle: lipgloss.NewStyle().
			Foreground(theme.SecondaryText).
			Italic(true).
			MarginTop(2),
	}
}

func extractTransactionData(t *transactionItem, m model) transactionDisplayData {
	data := transactionDisplayData{transaction: t}

	// Parse amount
	amount, err := t.t.ParsedAmount()
	data.amountStr = "Error parsing amount"
	if err == nil {
		data.amountStr = amount.Display()
	}

	// Get account name
	data.accountName = "Unknown"
	if t.plaidAccount != nil {
		data.accountName = t.plaidAccount.Name
	} else if t.asset != nil {
		data.accountName = t.asset.Name
	}

	// Format tags
	data.tagsStr = formatTags(t.tags)

	// Format notes
	data.notesStr = t.t.Notes
	if data.notesStr == "" {
		data.notesStr = "None"
	}

	// Format currency
	data.currencyStr = t.t.Currency
	if data.currencyStr == "" {
		data.currencyStr = "USD"
	}

	// Create status-specific styling
	data.statusStyle = createStatusStyle(t.t.Status, m.theme)

	return data
}

func formatTags(tags []*lm.Tag) string {
	if len(tags) == 0 {
		return "None"
	}

	tagNames := make([]string, len(tags))
	for i, tag := range tags {
		tagNames[i] = tag.Name
	}
	return strings.Join(tagNames, ", ")
}

func createStatusStyle(status string, theme Theme) lipgloss.Style {
	baseStyle := lipgloss.NewStyle().Padding(0, 1).Bold(true)

	switch status {
	case "cleared":
		return baseStyle.Foreground(theme.Text).Background(theme.Success)
	case unclearedStatus:
		return baseStyle.Foreground(theme.Text).Background(theme.Warning)
	case pendingStatus:
		return baseStyle.Foreground(theme.Text).Background(theme.Muted)
	default:
		return baseStyle.Foreground(theme.Text).Background(theme.SecondaryText)
	}
}

// buildTransactionDetailsWithNotes builds transaction details with support for notes editing.
func buildTransactionDetailsWithNotes(data transactionDisplayData, styles detailedTransactionStyles, m model) []string {
	t := data.transaction

	details := []string{
		createDetailRow("ID:", strconv.FormatInt(t.t.ID, 10), styles),
		createDetailRow("Payee:", t.t.Payee, styles),
		createDetailRow("Amount:", data.amountStr, styles),
		createDetailRow("Category:", t.category.Name, styles),
		createDetailRow("Currency:", data.currencyStr, styles),
		createDetailRow("Date:", t.t.Date, styles),
		lipgloss.JoinHorizontal(lipgloss.Left,
			styles.labelStyle.Render("Status:"),
			data.statusStyle.Render(" "+t.t.Status),
		),
	}

	// Add optional fields
	details = appendOptionalFields(details, t, styles)

	// Add notes section with editing support
	if m.isEditingNotes {
		// Show text input for editing
		notesSection := lipgloss.JoinHorizontal(lipgloss.Left,
			styles.labelStyle.Render("Notes:"),
			" "+m.notesInput.View(),
		)
		details = append(details, notesSection)
	} else {
		// Show normal notes display
		notesSection := createDetailRow("Notes:", data.notesStr, styles)
		details = append(details, notesSection)
	}

	return details
}

func createDetailRow(label, value string, styles detailedTransactionStyles) string {
	return lipgloss.JoinHorizontal(lipgloss.Left,
		styles.labelStyle.Render(label),
		styles.valueStyle.Render(value),
	)
}

func appendOptionalFields(details []string, t *transactionItem, styles detailedTransactionStyles) []string {
	if t.t.RecurringID != 0 {
		details = append(details, createDetailRow("Recurring ID:", strconv.FormatInt(t.t.RecurringID, 10), styles))
	}
	if t.t.GroupID != 0 {
		details = append(details, createDetailRow("Group ID:", strconv.FormatInt(t.t.GroupID, 10), styles))
	}
	if t.t.ParentID != 0 {
		details = append(details, createDetailRow("Parent ID:", strconv.FormatInt(t.t.ParentID, 10), styles))
	}
	if t.t.ExternalID != "" {
		details = append(details, createDetailRow("External ID:", t.t.ExternalID, styles))
	}
	if t.t.IsGroup {
		details = append(details, createDetailRow("Group Transaction:", "Yes", styles))
	}
	return details
}

// createInstructionsWithNotes creates instructions with notes editing support.
func createInstructionsWithNotes(styles detailedTransactionStyles, isEditingNotes bool) string {
	if isEditingNotes {
		return styles.instructionStyle.Render("Press 'enter' to save notes, 'esc' to cancel")
	}
	return styles.instructionStyle.Render(
		"'n' to edit notes, 'c' to categorize transaction,\n'esc' to return to transaction list",
	)
}
