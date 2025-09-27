package main

import (
	"errors"
	"fmt"
	"slices"
	"sort"

	"github.com/Rhymond/go-money"
	lm "github.com/icco/lunchmoney"
	"github.com/spf13/cobra"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const (
	creditType        = "credit"
	creditCardSubtype = "credit card"
)

// networthCmd represents the networth command.
var networthCmd = &cobra.Command{
	Use:   "networth",
	Short: "Net worth calculation commands",
	Long:  `Commands for calculating and displaying net worth from Lunch Money data.`,
}

// networthGetCmd represents the networth get command.
var networthGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Calculate and display current net worth",
	Long:  `Calculate current net worth by fetching all assets and liabilities from Lunch Money.`,
	PreRunE: func(cmd *cobra.Command, _ []string) error {
		// Validate output format
		outputFormat, _ := cmd.Flags().GetString("output")
		validFormats := []string{tableOutputFormat, jsonOutputFormat}
		if !slices.Contains(validFormats, outputFormat) {
			return fmt.Errorf("invalid output format: %s (must be one of %v)", outputFormat, validFormats)
		}

		return nil
	},
	RunE: networthGetRun,
}

func init() {
	// Add networth get subcommand
	networthCmd.AddCommand(networthGetCmd)

	// Net worth get flags
	networthGetCmd.Flags().StringP("output", "o", tableOutputFormat, "Output format: table or json")
	networthGetCmd.Flags().Bool("breakdown", false, "Show detailed breakdown of assets and liabilities")
}

func networthGetRun(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()

	outputFormat, _ := cmd.Flags().GetString("output")
	showBreakdown, _ := cmd.Flags().GetBool("breakdown")

	// Fetch user info to get primary currency
	user, userErr := lmc.GetUser(ctx)
	if userErr != nil {
		return fmt.Errorf("failed to fetch user info: %w", userErr)
	}

	currency := user.PrimaryCurrency
	if currency == "" {
		currency = "USD"
	}

	// Parallel fetch of assets and plaid accounts
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

	// Calculate net worth using shared logic and types
	netWorthData := calculateNetWorthData(assets, plaidAccounts, currency, showBreakdown)

	switch outputFormat {
	case jsonOutputFormat:
		return outputJSON(netWorthData.ToJSON())
	case tableOutputFormat:
		return outputNetWorthTable(netWorthData)
	default:
		return errors.New("unsupported output format")
	}
}

// calculateNetWorthData reuses the existing net worth calculation logic from overview
// but returns the shared NetWorthData type for consistency between CLI and TUI.
func calculateNetWorthData(
	assets []*lm.Asset,
	plaidAccounts []*lm.PlaidAccount,
	currency string,
	includeBreakdown bool,
) *NetWorthData {
	netWorth := money.New(0, currency)
	totalAssets := money.New(0, currency)
	totalLiabilities := money.New(0, currency)

	var breakdown *NetWorthBreakdown
	if includeBreakdown {
		breakdown = &NetWorthBreakdown{
			Assets:      make(map[string][]*AccountSummary),
			Liabilities: make(map[string][]*AccountSummary),
		}
	}

	// Process assets and plaid accounts
	processAssets(assets, currency, &netWorth, &totalAssets, &totalLiabilities, breakdown, includeBreakdown)
	processPlaidAccounts(plaidAccounts, currency, &netWorth, &totalAssets, &totalLiabilities, breakdown, includeBreakdown)

	// Sort breakdown by amount (descending)
	if includeBreakdown {
		sortNetWorthBreakdown(breakdown)
	}

	return &NetWorthData{
		NetWorth:         netWorth,
		TotalAssets:      totalAssets,
		TotalLiabilities: totalLiabilities,
		Currency:         currency,
		Breakdown:        breakdown,
	}
}

func processAssets(
	assets []*lm.Asset,
	currency string,
	netWorth, totalAssets, totalLiabilities **money.Money,
	breakdown *NetWorthBreakdown,
	includeBreakdown bool,
) {
	for _, asset := range assets {
		amount := money.NewFromFloat(asset.ToBase, currency)
		*netWorth = updateNetWorthAmount(*netWorth, amount, asset.TypeName, asset.SubtypeName)

		if asset.TypeName == creditType && asset.SubtypeName == creditCardSubtype {
			*totalLiabilities, _ = (*totalLiabilities).Add(amount)
			if includeBreakdown {
				addAccountToBreakdown(breakdown.Liabilities, asset, amount)
			}
		} else {
			*totalAssets, _ = (*totalAssets).Add(amount)
			if includeBreakdown {
				addAccountToBreakdown(breakdown.Assets, asset, amount)
			}
		}
	}
}

func processPlaidAccounts(
	accounts []*lm.PlaidAccount,
	currency string,
	netWorth, totalAssets, totalLiabilities **money.Money,
	breakdown *NetWorthBreakdown,
	includeBreakdown bool,
) {
	for _, account := range accounts {
		amount := money.NewFromFloat(account.ToBase, currency)
		*netWorth = updateNetWorthAmount(*netWorth, amount, account.Type, account.Subtype)

		if account.Type == creditType && account.Subtype == creditCardSubtype {
			*totalLiabilities, _ = (*totalLiabilities).Add(amount)
			if includeBreakdown {
				addPlaidAccountToBreakdown(breakdown.Liabilities, account, amount)
			}
		} else {
			*totalAssets, _ = (*totalAssets).Add(amount)
			if includeBreakdown {
				addPlaidAccountToBreakdown(breakdown.Assets, account, amount)
			}
		}
	}
}

// updateNetWorthAmount reuses the exact logic from overview/overview.go.
func updateNetWorthAmount(netWorth, amount *money.Money, assetType, subtype string) *money.Money {
	var nwa *money.Money
	var err error

	if assetType == creditType && subtype == creditCardSubtype {
		nwa, err = netWorth.Subtract(amount)
	} else {
		nwa, err = netWorth.Add(amount)
	}

	if err != nil {
		return netWorth
	}

	return nwa
}

func addAccountToBreakdown(categoryMap map[string][]*AccountSummary, asset *lm.Asset, amount *money.Money) {
	category := formatAccountCategory(asset.TypeName, asset.SubtypeName)

	categoryMap[category] = append(categoryMap[category], &AccountSummary{
		ID:              asset.ID,
		Name:            asset.Name,
		DisplayName:     asset.DisplayName,
		Type:            asset.TypeName,
		Subtype:         asset.SubtypeName,
		Amount:          amount,
		InstitutionName: asset.InstitutionName,
		AccountType:     "asset",
	})
}

func addPlaidAccountToBreakdown(
	categoryMap map[string][]*AccountSummary,
	account *lm.PlaidAccount,
	amount *money.Money,
) {
	category := formatAccountCategory(account.Type, account.Subtype)

	categoryMap[category] = append(categoryMap[category], &AccountSummary{
		ID:              account.ID,
		Name:            account.Name,
		DisplayName:     account.DisplayName,
		Type:            account.Type,
		Subtype:         account.Subtype,
		Amount:          amount,
		InstitutionName: account.InstitutionName,
		AccountType:     "plaid",
	})
}

func formatAccountCategory(accountType, subtype string) string {
	caser := cases.Title(language.English)
	if subtype != "" && subtype != accountType {
		return caser.String(subtype)
	}
	return caser.String(accountType)
}

func sortNetWorthBreakdown(breakdown *NetWorthBreakdown) {
	// Sort each category by amount (descending)
	for category := range breakdown.Assets {
		sort.Slice(breakdown.Assets[category], func(i, j int) bool {
			cmp, _ := breakdown.Assets[category][i].Amount.Compare(breakdown.Assets[category][j].Amount)
			return cmp > 0
		})
	}
	for category := range breakdown.Liabilities {
		sort.Slice(breakdown.Liabilities[category], func(i, j int) bool {
			cmp, _ := breakdown.Liabilities[category][i].Amount.Compare(breakdown.Liabilities[category][j].Amount)
			return cmp > 0
		})
	}
}

func outputNetWorthTable(data *NetWorthData) error {
	fmt.Printf("Net Worth: %s\n\n", data.NetWorth.Display())

	if data.Breakdown != nil {
		if len(data.Breakdown.Assets) > 0 {
			fmt.Println("ASSETS:")
			printAccountCategoriesTable(data.Breakdown.Assets, false)
			fmt.Println()
		}

		if len(data.Breakdown.Liabilities) > 0 {
			fmt.Println("LIABILITIES:")
			printAccountCategoriesTable(data.Breakdown.Liabilities, true)
			fmt.Println()
		}

		fmt.Printf("Total Assets:      %s\n", data.TotalAssets.Display())
		fmt.Printf("Total Liabilities: %s\n", data.TotalLiabilities.Display())
		fmt.Printf("Net Worth:         %s\n", data.NetWorth.Display())
	}

	return nil
}

func printAccountCategoriesTable(accountsByType map[string][]*AccountSummary, isLiability bool) {
	var types []string
	for accountType := range accountsByType {
		types = append(types, accountType)
	}
	sort.Strings(types)

	for _, accountType := range types {
		accounts := accountsByType[accountType]
		if len(accounts) == 0 {
			continue
		}

		// Calculate total for this category
		total := calculateCategoryTotal(accounts)
		if isLiability {
			fmt.Printf("  %s: -%s\n", accountType, total.Display())
		} else {
			fmt.Printf("  %s: %s\n", accountType, total.Display())
		}

		// Print individual accounts if there are multiple
		if len(accounts) > 1 {
			for _, account := range accounts {
				displayAmount := account.Amount.Display()
				if isLiability {
					displayAmount = "-" + displayAmount
				}
				fmt.Printf("    %s: %s\n", account.GetDisplayName(), displayAmount)
			}
		}
	}
}

func calculateCategoryTotal(accounts []*AccountSummary) *money.Money {
	if len(accounts) == 0 {
		return money.New(0, "USD")
	}

	total := money.New(0, accounts[0].Amount.Currency().Code)
	for _, account := range accounts {
		total, _ = total.Add(account.Amount)
	}

	return total
}
