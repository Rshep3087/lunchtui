package main

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strconv"

	lm "github.com/icco/lunchmoney"
	"github.com/spf13/cobra"
)

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

// accountsCmd represents the accounts command.
var accountsCmd = &cobra.Command{
	Use:   "accounts",
	Short: "Account management commands",
	Long:  `Commands for managing accounts in Lunch Money.`,
}

// accountsListCmd represents the accounts list command.
var accountsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all accounts",
	Long:  `List all accounts (assets and plaid accounts) with their IDs and details.`,
	RunE:  accountsListRun,
}

func init() {
	// Add accounts list subcommand
	accountsCmd.AddCommand(accountsListCmd)

	// Accounts list flags
	accountsListCmd.Flags().StringP("output", "o", tableOutputFormat, "Output format: table or json")
}

func accountsListRun(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Get output format
	outputFormat, _ := cmd.Flags().GetString("output")

	// Validate output format
	validFormats := []string{tableOutputFormat, jsonOutputFormat}
	if !slices.Contains(validFormats, outputFormat) {
		return fmt.Errorf("invalid output format: %s (must be one of %v)", outputFormat, validFormats)
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
	switch outputFormat {
	case jsonOutputFormat:
		return outputJSON(accounts)
	case tableOutputFormat:
		return outputAccountsTable(accounts)
	default:
		return errors.New("unsupported output format")
	}
}

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
