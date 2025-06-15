package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/charmbracelet/log"
	lm "github.com/icco/lunchmoney"
	"github.com/urfave/cli/v3"
)

// contextKey is used as a key for storing values in context.
type contextKey string

const (
	// clientContextKey is the key for storing the Lunch Money client in context.
	clientContextKey contextKey = "lunchMoneyClient"
)

// getClientFromContext retrieves the Lunch Money client from context.
func getClientFromContext(ctx context.Context) (*lm.Client, error) {
	client, ok := ctx.Value(clientContextKey).(*lm.Client)
	if !ok {
		return nil, errors.New("lunch money client not found in context")
	}
	return client, nil
}

// createRootCommand creates the root CLI command with subcommands.
func createRootCommand() *cli.Command {
	return &cli.Command{
		Name:                  "lunchtui",
		Usage:                 "A terminal UI and CLI for Lunch Money",
		EnableShellCompletion: true,
		Flags:                 globalFlags(),
		Action:                rootAction,
		Commands: []*cli.Command{
			createTransactionCommand(),
			createCategoriesCommand(),
			createAccountsCommand(),
		},
		Before: func(ctx context.Context, c *cli.Command) (context.Context, error) {
			// Setup logging if debug is enabled
			if c.Bool("debug") {
				log.SetLevel(log.DebugLevel)
				log.Debug("Debug logging enabled")
			} else {
				log.SetLevel(log.InfoLevel)
			}

			// Create Lunch Money client and store in context
			lmc, err := lm.NewClient(c.String("token"))
			if err != nil {
				return ctx, fmt.Errorf("failed to create Lunch Money client: %w", err)
			}

			// Store client in context
			ctx = context.WithValue(ctx, clientContextKey, lmc)

			return ctx, nil
		},
	}
}

// globalFlags returns the global flags available to all commands.
func globalFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:     "token",
			Usage:    "The API token for Lunch Money",
			Sources:  cli.EnvVars("LUNCHMONEY_API_TOKEN"),
			Required: true,
		},
		&cli.BoolFlag{
			Name:  "debits-as-negative",
			Usage: "Show debits as negative numbers",
		},
		&cli.BoolFlag{
			Name:  "debug",
			Usage: "Enable debug logging",
			Value: false,
		},
	}
}

func createTransactionCommand() *cli.Command {
	return &cli.Command{
		Name:  "transaction",
		Usage: "Transaction management commands",
		Commands: []*cli.Command{
			createTransactionInsertCommand(),
		},
	}
}

// createTransactionInsertCommand creates the transaction insert subcommand.
func createTransactionInsertCommand() *cli.Command {
	return &cli.Command{
		Name:  "insert",
		Usage: "Insert a new transaction",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "payee",
				Usage:    "The payee or merchant name",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "amount",
				Usage:    "Transaction amount (positive for expense, negative for income)",
				Required: true,
			},
			&cli.StringFlag{
				Name:  "date",
				Usage: "Transaction date (YYYY-MM-DD, defaults to today)",
				Value: time.Now().Format("2006-01-02"),
			},
			&cli.Int64Flag{
				Name:  "category",
				Usage: "Category ID for the transaction",
			},
			&cli.StringFlag{
				Name:  "status",
				Usage: "Transaction status (cleared, uncleared)",
				Value: "uncleared",
			},
			&cli.Int64Flag{
				Name:  "account",
				Usage: "Account ID (plaid account or asset)",
			},
			&cli.StringFlag{
				Name:  "currency",
				Usage: "Currency code",
				Value: "usd",
			},
			&cli.StringSliceFlag{
				Name:  "tags",
				Usage: "Tag IDs (can be specified multiple times)",
			},
			&cli.StringFlag{
				Name:  "notes",
				Usage: "Additional notes for the transaction",
			},
			&cli.BoolFlag{
				Name:  "apply-rules",
				Usage: "Apply rules to the transaction",
				Value: true,
			},
			&cli.BoolFlag{
				Name:  "skip-duplicates",
				Usage: "Skip duplicate transactions",
				Value: true,
			},
			&cli.BoolFlag{
				Name:  "check-for-recurring",
				Usage: "Check for recurring transactions",
				Value: true,
			},
			&cli.BoolFlag{
				Name:  "skip-balance-update",
				Usage: "Skip balance update",
				Value: false,
			},
		},
		Action: insertTransactionAction,
	}
}

// insertTransactionAction handles the transaction insert CLI command.
func insertTransactionAction(ctx context.Context, c *cli.Command) error {
	lmc, err := getClientFromContext(ctx)
	if err != nil {
		return err
	}

	// Validate and parse the amount
	amountStr := c.String("amount")
	if _, err = strconv.ParseFloat(amountStr, 64); err != nil {
		return fmt.Errorf("invalid amount: %s", amountStr)
	}

	// Validate date format
	dateStr := c.String("date")
	if _, err = time.Parse("2006-01-02", dateStr); err != nil {
		return fmt.Errorf("invalid date format: %s (expected YYYY-MM-DD)", dateStr)
	}

	// Validate status
	status := c.String("status")
	if status != "cleared" && status != "uncleared" {
		return fmt.Errorf("invalid status: %s (must be 'cleared' or 'uncleared')", status)
	}

	// Parse tag IDs
	var tagIDs []int
	if tagStrings := c.StringSlice("tags"); len(tagStrings) > 0 {
		tagIDs = make([]int, 0, len(tagStrings))
		for _, tagStr := range tagStrings {
			var tagID int
			tagID, err = strconv.Atoi(tagStr)
			if err != nil {
				return fmt.Errorf("invalid tag ID: %s", tagStr)
			}
			tagIDs = append(tagIDs, tagID)
		}
	}

	// Create the transaction
	transaction := lm.InsertTransaction{
		Date:     dateStr,
		Amount:   amountStr,
		Payee:    c.String("payee"),
		Currency: c.String("currency"),
		Notes:    c.String("notes"),
		Status:   status,
		TagsIDs:  tagIDs,
	}

	// Set category ID if provided
	if categoryID := c.Int64("category"); categoryID != 0 {
		transaction.CategoryID = &categoryID
	}

	// Set account ID if provided (we'll need to determine if it's plaid or asset)
	if accountID := c.Int64("account"); accountID != 0 {
		// For CLI, we'll assume it's a plaid account by default
		// In a more sophisticated implementation, we could check the account type
		transaction.PlaidAccountID = &accountID
	}

	// Create the request
	request := lm.InsertTransactionsRequest{
		ApplyRules:        c.Bool("apply-rules"),
		SkipDuplicates:    c.Bool("skip-duplicates"),
		CheckForRecurring: c.Bool("check-for-recurring"),
		DebitAsNegative:   c.Bool("debits-as-negative"),
		SkipBalanceUpdate: c.Bool("skip-balance-update"),
		Transactions:      []lm.InsertTransaction{transaction},
	}

	log.Debug("inserting transaction", "request", request)

	// Make the API call
	resp, err := lmc.InsertTransactions(ctx, request)
	if err != nil {
		return fmt.Errorf("failed to insert transaction: %w", err)
	}

	log.Debug("transaction inserted", "response", resp)

	if len(resp.IDs) == 0 {
		return errors.New("no transaction IDs returned")
	}

	// Success
	log.Infof("Transaction inserted successfully with ID: %d", resp.IDs[0])
	return nil
}

// createCategoriesCommand creates the categories command with subcommands.
func createCategoriesCommand() *cli.Command {
	return &cli.Command{
		Name:  "categories",
		Usage: "Category management commands",
		Commands: []*cli.Command{
			createCategoriesListCommand(),
		},
	}
}

// createCategoriesListCommand creates the categories list subcommand.
func createCategoriesListCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List all categories with their IDs and details",
		Flags: []cli.Flag{
			createOutputFormatFlag(),
		},
		Action: categoriesListAction,
	}
}

// categoriesListAction handles the categories list CLI command.
func categoriesListAction(ctx context.Context, c *cli.Command) error {
	lmc, err := getClientFromContext(ctx)
	if err != nil {
		return err
	}

	// Fetch categories
	categories, err := lmc.GetCategories(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch categories: %w", err)
	}

	// Sort categories by name for consistent output
	sort.Slice(categories, func(i, j int) bool {
		return categories[i].Name < categories[j].Name
	})

	// Add the special "Uncategorized" category
	categories = append(categories, &lm.Category{
		ID:          0,
		Name:        "uncategorized",
		Description: "Transactions without a category",
	})

	// Output based on format
	switch c.String("output") {
	case "json":
		return outputJSON(categories)
	case "table":
		return outputCategoriesTable(categories)
	default:
		return errors.New("unsupported output format")
	}
}

// outputJSON outputs data in JSON format.
func outputJSON(data any) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	fmt.Println(string(jsonData))
	return nil
}

// createStyledTable creates a table with the standard styling used across commands.
func createStyledTable(headers ...string) *table.Table {
	var (
		purple    = lipgloss.Color("99")
		gray      = lipgloss.Color("245")
		lightGray = lipgloss.Color("241")

		headerStyle  = lipgloss.NewStyle().Foreground(purple).Bold(true).Align(lipgloss.Center)
		cellStyle    = lipgloss.NewStyle().Padding(0, 1)
		oddRowStyle  = cellStyle.Foreground(gray)
		evenRowStyle = cellStyle.Foreground(lightGray)
	)

	return table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(purple)).
		StyleFunc(func(row, _ int) lipgloss.Style {
			switch {
			case row == table.HeaderRow:
				return headerStyle
			case row%2 == 0:
				return evenRowStyle
			default:
				return oddRowStyle
			}
		}).
		Headers(headers...)
}

// createOutputFormatFlag creates the standard output format flag used across list commands.
func createOutputFormatFlag() *cli.StringFlag {
	return &cli.StringFlag{
		Name:    "output",
		Aliases: []string{"o"},
		Usage:   "Output format: table or json",
		Value:   "table",
		Validator: func(s string) error {
			validFormats := []string{"table", "json"}
			if !slices.Contains(validFormats, s) {
				return fmt.Errorf("invalid output format: %s (must be one of %v)", s, validFormats)
			}
			return nil
		},
	}
}

// outputCategoriesTable outputs categories in table format.
func outputCategoriesTable(categories []*lm.Category) error {
	// Create table
	t := createStyledTable(
		"ID", "NAME", "DESCRIPTION", "IS INCOME", "EXCLUDE FROM BUDGET", "EXCLUDE FROM TOTALS", "IS GROUP",
	)

	// Add categories to table
	for _, category := range categories {
		description := category.Description
		if description == "" {
			description = "-"
		}
		t.Row(
			strconv.FormatInt(category.ID, 10),
			category.Name,
			description,
			strconv.FormatBool(category.IsIncome),
			strconv.FormatBool(category.ExcludeFromBudget),
			strconv.FormatBool(category.ExcludeFromTotals),
			strconv.FormatBool(category.IsGroup),
		)
	}

	// Print the table
	fmt.Println(t)

	return nil
}

// Account represents a unified account structure for both assets and plaid accounts.
type Account struct {
	ID              int64  `json:"id"`
	Name            string `json:"name"`
	Type            string `json:"type"`
	Subtype         string `json:"subtype"`
	Balance         string `json:"balance"`
	Currency        string `json:"currency"`
	InstitutionName string `json:"institution_name"`
	Status          string `json:"status"`
	AccountType     string `json:"account_type"` // "asset" or "plaid"
}

// convertAssetToAccount converts an Asset to the unified Account structure.
func convertAssetToAccount(asset *lm.Asset) Account {
	return Account{
		ID:              asset.ID,
		Name:            asset.Name,
		Type:            asset.TypeName,
		Subtype:         asset.SubtypeName,
		Balance:         asset.Balance,
		Currency:        asset.Currency,
		InstitutionName: asset.InstitutionName,
		Status:          asset.Status,
		AccountType:     "asset",
	}
}

// convertPlaidAccountToAccount converts a PlaidAccount to the unified Account structure.
func convertPlaidAccountToAccount(plaidAccount *lm.PlaidAccount) Account {
	return Account{
		ID:              plaidAccount.ID,
		Name:            plaidAccount.Name,
		Type:            plaidAccount.Type,
		Subtype:         plaidAccount.Subtype,
		Balance:         plaidAccount.Balance,
		Currency:        plaidAccount.Currency,
		InstitutionName: plaidAccount.InstitutionName,
		Status:          plaidAccount.Status,
		AccountType:     "plaid",
	}
}

// createAccountsCommand creates the accounts command with subcommands.
func createAccountsCommand() *cli.Command {
	return &cli.Command{
		Name:  "accounts",
		Usage: "Account management commands",
		Commands: []*cli.Command{
			createAccountsListCommand(),
		},
	}
}

// createAccountsListCommand creates the accounts list subcommand.
func createAccountsListCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List all accounts (assets and plaid accounts) with their IDs and details",
		Flags: []cli.Flag{
			createOutputFormatFlag(),
		},
		Action: accountsListAction,
	}
}

// accountsListAction handles the accounts list CLI command.
func accountsListAction(ctx context.Context, c *cli.Command) error {
	lmc, err := getClientFromContext(ctx)
	if err != nil {
		return fmt.Errorf("getting Lunch Money client from context: %w", err)
	}

	assetsChan := make(chan []*lm.Asset, 1)
	plaidAccountsChan := make(chan []*lm.PlaidAccount, 1)
	errorChan := make(chan error, 2)

	// Fetch assets
	go func() {
		assets, assetsError := lmc.GetAssets(ctx)
		if assetsError != nil {
			errorChan <- fmt.Errorf("failed to fetch assets: %w", assetsError)
			return
		}
		assetsChan <- assets
	}()

	// Fetch plaid accounts
	go func() {
		plaidAccounts, plaidAccountsErr := lmc.GetPlaidAccounts(ctx)
		if plaidAccountsErr != nil {
			errorChan <- fmt.Errorf("failed to fetch plaid accounts: %w", plaidAccountsErr)
			return
		}
		plaidAccountsChan <- plaidAccounts
	}()

	// Collect results
	var assets []*lm.Asset
	var plaidAccounts []*lm.PlaidAccount
	for range 2 {
		select {
		case assets = <-assetsChan:
		case plaidAccounts = <-plaidAccountsChan:
		case fetchError := <-errorChan:
			return fetchError
		}
	}

	// Convert to unified Account structure
	var accounts []Account
	for _, asset := range assets {
		accounts = append(accounts, convertAssetToAccount(asset))
	}
	for _, plaidAccount := range plaidAccounts {
		accounts = append(accounts, convertPlaidAccountToAccount(plaidAccount))
	}

	// Sort accounts by name for consistent output
	sort.Slice(accounts, func(i, j int) bool {
		return accounts[i].Name < accounts[j].Name
	})

	// Output based on format
	switch c.String("output") {
	case "json":
		return outputJSON(accounts)
	case "table":
		return outputAccountsTable(accounts)
	default:
		return errors.New("unsupported output format")
	}
}

// outputAccountsTable outputs accounts in table format.
func outputAccountsTable(accounts []Account) error {
	// Create table
	t := createStyledTable("ID", "NAME", "TYPE", "SUBTYPE", "BALANCE", "CURRENCY", "INSTITUTION", "STATUS", "ACCOUNT TYPE")

	// Add accounts to table
	for _, account := range accounts {
		subtype := account.Subtype
		if subtype == "" {
			subtype = "-"
		}
		institution := account.InstitutionName
		if institution == "" {
			institution = "-"
		}
		t.Row(
			strconv.FormatInt(account.ID, 10),
			account.Name,
			account.Type,
			subtype,
			account.Balance,
			account.Currency,
			institution,
			account.Status,
			account.AccountType,
		)
	}

	// Print the table
	fmt.Println(t)

	return nil
}
