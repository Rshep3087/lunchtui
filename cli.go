package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/charmbracelet/log"
	lm "github.com/icco/lunchmoney"
	"github.com/urfave/cli/v3"
)

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
				Usage:    "Transaction amount (positive for income, negative for expense)",
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
				Value: "USD",
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
	// Setup logging if debug is enabled
	if c.Bool("debug") {
		log.SetLevel(log.DebugLevel)
	}

	// Create Lunch Money client
	lmc, err := lm.NewClient(c.String("token"))
	if err != nil {
		return fmt.Errorf("failed to create Lunch Money client: %w", err)
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
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "Output format: table or json",
				Value:   "table",
			},
		},
		Action: categoriesListAction,
	}
}

// categoriesListAction handles the categories list CLI command.
func categoriesListAction(ctx context.Context, c *cli.Command) error {
	// Setup logging if debug is enabled
	if c.Bool("debug") {
		log.SetLevel(log.DebugLevel)
	}

	// Validate output format
	outputFormat := c.String("output")
	if outputFormat != "table" && outputFormat != "json" {
		return fmt.Errorf("invalid output format: %s (must be 'table' or 'json')", outputFormat)
	}

	// Create Lunch Money client
	lmc, err := lm.NewClient(c.String("token"))
	if err != nil {
		return fmt.Errorf("failed to create Lunch Money client: %w", err)
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
	switch outputFormat {
	case "json":
		return outputJSON(categories)
	case "table":
		return outputCategoriesTable(categories)
	default:
		return fmt.Errorf("unsupported output format: %s", outputFormat)
	}
}

// outputCategoriesJSON outputs categories in JSON format.
func outputJSON(data any) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	fmt.Println(string(jsonData))
	return nil
}

// outputCategoriesTable outputs categories in table format.
func outputCategoriesTable(categories []*lm.Category) error {
	// Define table styles
	var (
		purple    = lipgloss.Color("99")
		gray      = lipgloss.Color("245")
		lightGray = lipgloss.Color("241")

		headerStyle  = lipgloss.NewStyle().Foreground(purple).Bold(true).Align(lipgloss.Center)
		cellStyle    = lipgloss.NewStyle().Padding(0, 1)
		oddRowStyle  = cellStyle.Foreground(gray)
		evenRowStyle = cellStyle.Foreground(lightGray)
	)

	// Create table
	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(purple)).
		StyleFunc(func(row, col int) lipgloss.Style {
			switch {
			case row == table.HeaderRow:
				return headerStyle
			case row%2 == 0:
				return evenRowStyle
			default:
				return oddRowStyle
			}
		}).
		Headers("ID", "NAME", "DESCRIPTION", "IS INCOME", "EXCLUDE FROM BUDGET", "EXCLUDE FROM TOTALS", "IS GROUP")

	// Add categories to table
	for _, category := range categories {
		description := category.Description
		if description == "" {
			description = "-"
		}
		t.Row(
			fmt.Sprintf("%d", category.ID),
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
