package main

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/charmbracelet/log"
	lm "github.com/icco/lunchmoney"
	"github.com/spf13/cobra"
)

// transactionCmd represents the transaction command.
var transactionCmd = &cobra.Command{
	Use:   "transaction",
	Short: "Transaction management commands",
	Long:  `Commands for managing transactions in Lunch Money.`,
}

// transactionInsertCmd represents the transaction insert command.
var transactionInsertCmd = &cobra.Command{
	Use:   "insert",
	Short: "Insert a new transaction",
	Long:  `Insert a new transaction into Lunch Money.`,
	RunE:  transactionInsertRun,
}

func init() {
	// Add transaction insert subcommand
	transactionCmd.AddCommand(transactionInsertCmd)

	// Transaction insert flags
	transactionInsertCmd.Flags().String("payee", "", "The payee or merchant name (required)")
	transactionInsertCmd.Flags().String("amount", "", "Transaction amount (positive for expense, negative for income) (required)")
	transactionInsertCmd.Flags().String("date", time.Now().Format("2006-01-02"), "Transaction date (YYYY-MM-DD, defaults to today)")
	transactionInsertCmd.Flags().Int64("category", 0, "Category ID for the transaction")
	transactionInsertCmd.Flags().String("status", unclearedStatus, "Transaction status (cleared, uncleared)")
	transactionInsertCmd.Flags().Int64("account", 0, "Account ID (plaid account or asset)")
	transactionInsertCmd.Flags().String("currency", "usd", "Currency code")
	transactionInsertCmd.Flags().StringSlice("tags", []string{}, "Tag IDs (can be specified multiple times)")
	transactionInsertCmd.Flags().String("notes", "", "Additional notes for the transaction")
	transactionInsertCmd.Flags().Bool("apply-rules", true, "Apply rules to the transaction")
	transactionInsertCmd.Flags().Bool("skip-duplicates", true, "Skip duplicate transactions")
	transactionInsertCmd.Flags().Bool("check-for-recurring", true, "Check for recurring transactions")
	transactionInsertCmd.Flags().Bool("skip-balance-update", false, "Skip balance update")

	// Mark required flags
	transactionInsertCmd.MarkFlagRequired("payee")
	transactionInsertCmd.MarkFlagRequired("amount")
}

func transactionInsertRun(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Get flag values
	payee, _ := cmd.Flags().GetString("payee")
	amountStr, _ := cmd.Flags().GetString("amount")
	dateStr, _ := cmd.Flags().GetString("date")
	categoryID, _ := cmd.Flags().GetInt64("category")
	status, _ := cmd.Flags().GetString("status")
	accountID, _ := cmd.Flags().GetInt64("account")
	currency, _ := cmd.Flags().GetString("currency")
	tagStrings, _ := cmd.Flags().GetStringSlice("tags")
	notes, _ := cmd.Flags().GetString("notes")
	applyRules, _ := cmd.Flags().GetBool("apply-rules")
	skipDuplicates, _ := cmd.Flags().GetBool("skip-duplicates")
	checkForRecurring, _ := cmd.Flags().GetBool("check-for-recurring")
	skipBalanceUpdate, _ := cmd.Flags().GetBool("skip-balance-update")

	// Validate and parse the amount
	if _, err := strconv.ParseFloat(amountStr, 64); err != nil {
		return fmt.Errorf("invalid amount: %s", amountStr)
	}

	// Validate date format
	if _, err := time.Parse("2006-01-02", dateStr); err != nil {
		return fmt.Errorf("invalid date format: %s (expected YYYY-MM-DD)", dateStr)
	}

	// Validate status
	if status != "cleared" && status != unclearedStatus {
		return fmt.Errorf("invalid status: %s (must be 'cleared' or 'uncleared')", status)
	}

	// Parse tag IDs
	var tagIDs []int
	if len(tagStrings) > 0 {
		tagIDs = make([]int, 0, len(tagStrings))
		for _, tagStr := range tagStrings {
			tagID, err := strconv.Atoi(tagStr)
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
		Payee:    payee,
		Currency: currency,
		Notes:    notes,
		Status:   status,
		TagsIDs:  tagIDs,
	}

	// Set category ID if provided
	if categoryID != 0 {
		transaction.CategoryID = &categoryID
	}

	// Set account ID if provided (we'll need to determine if it's plaid or asset)
	if accountID != 0 {
		// For CLI, we'll assume it's a plaid account by default
		// In a more sophisticated implementation, we could check the account type
		transaction.PlaidAccountID = &accountID
	}

	// Create the request
	request := lm.InsertTransactionsRequest{
		ApplyRules:        applyRules,
		SkipDuplicates:    skipDuplicates,
		CheckForRecurring: checkForRecurring,
		DebitAsNegative:   debitNeg,
		SkipBalanceUpdate: skipBalanceUpdate,
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
