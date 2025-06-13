package main

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/log"
	lm "github.com/icco/lunchmoney"
)

func newInsertTransactionForm(categories []*lm.Category, plaidAccounts map[int64]*lm.PlaidAccount, assets map[int64]*lm.Asset, tags map[int]*lm.Tag) *huh.Form {
	// Sort categories alphabetically
	sort.Slice(categories, func(i, j int) bool {
		return categories[i].Name < categories[j].Name
	})

	// Create category options
	categoryOpts := make([]huh.Option[int64], len(categories))
	for i, c := range categories {
		categoryOpts[i] = huh.NewOption(c.Name, c.ID)
	}

	// Create account options (both plaid accounts and assets)
	accountOpts := make([]huh.Option[int64], 0)
	for _, acc := range plaidAccounts {
		accountOpts = append(accountOpts, huh.NewOption(fmt.Sprintf("[Plaid] %s", acc.Name), acc.ID))
	}
	for _, asset := range assets {
		accountOpts = append(accountOpts, huh.NewOption(fmt.Sprintf("[Asset] %s", asset.Name), asset.ID))
	}

	// Sort account options
	sort.Slice(accountOpts, func(i, j int) bool {
		return accountOpts[i].Key < accountOpts[j].Key
	})

	// Create tag options
	tagOpts := make([]huh.Option[int], 0)
	for _, tag := range tags {
		tagOpts = append(tagOpts, huh.NewOption(tag.Name, tag.ID))
	}

	// Sort tag options
	sort.Slice(tagOpts, func(i, j int) bool {
		return tagOpts[i].Key < tagOpts[j].Key
	})

	// Status options
	statusOpts := []huh.Option[string]{
		huh.NewOption("Cleared", "cleared"),
		huh.NewOption("Uncleared", "uncleared"),
	}

	// Default date to today
	today := time.Now().Format("2006-01-02")
	defaultCurrency := "USD"
	defaultStatus := "uncleared"

	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Payee").
				Description("The payee or merchant name").
				Key("payee").
				Placeholder("Enter payee name...").
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("payee is required")
					}
					return nil
				}),

			huh.NewInput().
				Title("Amount").
				Description("Transaction amount (positive for income, negative for expense)").
				Key("amount").
				Placeholder("Enter amount (e.g., -50.00 or 100.00)...").
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("amount is required")
					}
					if _, err := strconv.ParseFloat(s, 64); err != nil {
						return fmt.Errorf("amount must be a valid number")
					}
					return nil
				}),

			huh.NewInput().
				Title("Date").
				Description("Transaction date (YYYY-MM-DD)").
				Key("date").
				Value(&today).
				Placeholder("YYYY-MM-DD").
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("date is required")
					}
					if _, err := time.Parse("2006-01-02", s); err != nil {
						return fmt.Errorf("date must be in YYYY-MM-DD format")
					}
					return nil
				}),
		),
		huh.NewGroup(
			huh.NewSelect[int64]().
				Title("Category").
				Description("Select a category for the transaction").
				Options(categoryOpts...).
				Key("category"),

			huh.NewSelect[string]().
				Title("Status").
				Description("Transaction status").
				Options(statusOpts...).
				Key("status").
				Value(&defaultStatus), // Default to uncleared
		),
		huh.NewGroup(
			huh.NewSelect[int64]().
				Title("Account (Optional)").
				Description("Select an account for the transaction").
				Options(accountOpts...).
				Key("account"),

			huh.NewInput().
				Title("Currency (Optional)").
				Description("Currency code (defaults to USD)").
				Key("currency").
				Value(&defaultCurrency).
				Placeholder("USD"),
		),
		huh.NewGroup(
			huh.NewMultiSelect[int]().
				Title("Tags (Optional)").
				Description("Select tags for the transaction").
				Options(tagOpts...).
				Key("tags"),

			huh.NewText().
				Title("Notes (Optional)").
				Description("Additional notes for the transaction").
				Key("notes").
				Placeholder("Enter notes..."),
		),
	)
}

func insertNewTransaction(m *model) (tea.Model, tea.Cmd) {
	m.insertTransactionForm = newInsertTransactionForm(m.categories, m.plaidAccounts, m.assets, m.tags)
	m.insertTransactionForm.SubmitCmd = func() tea.Msg {
		return submitInsertTransactionForm(*m)
	}

	m.sessionState = insertTransaction
	return m, tea.Batch(m.insertTransactionForm.Init(), tea.WindowSize())
}

func updateInsertTransaction(msg tea.Msg, m *model) (tea.Model, tea.Cmd) {
	form, cmd := m.insertTransactionForm.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.insertTransactionForm = f
	}

	if m.insertTransactionForm.State == huh.StateCompleted {
		m.sessionState = transactions
		return m, m.getTransactions // Refresh transactions after insert
	}

	if m.insertTransactionForm.State == huh.StateAborted {
		m.sessionState = transactions
		return m, nil // Don't refresh if cancelled
	}

	return m, cmd
}

func insertTransactionView(m model) string {
	return m.insertTransactionForm.View()
}

func submitInsertTransactionForm(m model) tea.Msg {
	ctx := context.Background()

	// Extract form values
	payee := m.insertTransactionForm.Get("payee").(string)
	amountStr := m.insertTransactionForm.Get("amount").(string)
	date := m.insertTransactionForm.Get("date").(string)
	status := m.insertTransactionForm.Get("status").(string)
	currency := m.insertTransactionForm.Get("currency").(string)
	notes := m.insertTransactionForm.Get("notes").(string)

	// Handle optional category
	var categoryID *int64
	if catVal := m.insertTransactionForm.Get("category"); catVal != nil {
		if catID, ok := catVal.(int64); ok && catID != 0 {
			categoryID = &catID
		}
	}

	// Handle optional account
	var assetID *int64
	var plaidAccountID *int64
	if accVal := m.insertTransactionForm.Get("account"); accVal != nil {
		if accID, ok := accVal.(int64); ok && accID != 0 {
			// Check if it's a plaid account or asset
			if _, isPlaid := m.plaidAccounts[accID]; isPlaid {
				plaidAccountID = &accID
			} else {
				assetID = &accID
			}
		}
	}

	// Handle optional tags
	var tagIDs []int
	if tagsVal := m.insertTransactionForm.Get("tags"); tagsVal != nil {
		if tags, ok := tagsVal.([]int); ok {
			tagIDs = tags
		}
	}

	// Create the transaction
	transaction := lm.InsertTransaction{
		Date:           date,
		Amount:         amountStr,
		CategoryID:     categoryID,
		Payee:          payee,
		Currency:       currency,
		AssetID:        assetID,
		PlaidAccountID: plaidAccountID,
		Notes:          notes,
		Status:         status,
		TagsIDs:        tagIDs,
	}

	// Create the request
	request := lm.InsertTransactionsRequest{
		ApplyRules:        true, // Apply rules by default
		SkipDuplicates:    true, // Skip duplicates by default
		CheckForRecurring: true, // Check for recurring by default
		DebitAsNegative:   m.debitsAsNegative,
		SkipBalanceUpdate: false, // Don't skip balance update
		Transactions:      []lm.InsertTransaction{transaction},
	}

	log.Debug("inserting transaction", "request", request)

	// Make the API call
	resp, err := m.lmc.InsertTransactions(ctx, request)
	if err != nil {
		log.Error("failed to insert transaction", "error", err)
		return insertTransactionErrorMsg{error: err}
	}

	log.Debug("transaction inserted", "response", resp)

	if len(resp.IDs) == 0 {
		return insertTransactionErrorMsg{error: fmt.Errorf("no transaction IDs returned")}
	}

	return insertTransactionSuccessMsg{transactionID: resp.IDs[0]}
}

// Message types for insert transaction responses
type insertTransactionSuccessMsg struct {
	transactionID int64
}

type insertTransactionErrorMsg struct {
	error error
}
