package main

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"

	lm "github.com/icco/lunchmoney"
	"github.com/spf13/cobra"
)

// userCmd represents the user command.
var userCmd = &cobra.Command{
	Use:   "user",
	Short: "User management commands",
	Long:  `Commands for managing user information in Lunch Money.`,
}

// userGetCmd represents the user get command.
var userGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get user information",
	Long:  `Get user information from Lunch Money.`,
	RunE:  userGetRun,
}

func init() {
	// Add user get subcommand
	userCmd.AddCommand(userGetCmd)

	// User get flags
	userGetCmd.Flags().StringP("output", "o", tableOutputFormat, "Output format: table or json")
}

func userGetRun(cmd *cobra.Command, _ []string) error {
	ctx := context.Background()

	// Get output format
	outputFormat, _ := cmd.Flags().GetString("output")

	// Validate output format
	validFormats := []string{tableOutputFormat, jsonOutputFormat}
	if !slices.Contains(validFormats, outputFormat) {
		return fmt.Errorf("invalid output format: %s (must be one of %v)", outputFormat, validFormats)
	}

	// Fetch user information
	user, err := lmc.GetUser(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch user information: %w", err)
	}

	// Output based on format
	switch outputFormat {
	case jsonOutputFormat:
		return outputJSON(user)
	case tableOutputFormat:
		return outputUserTable(user)
	default:
		return errors.New("unsupported output format")
	}
}

func outputUserTable(user *lm.User) error {
	// Create table
	t := createStyledTable("FIELD", "VALUE")

	// Add user information to table
	if user.UserID != 0 {
		t.Row("User ID", strconv.Itoa(user.UserID))
	}
	if user.UserName != "" {
		t.Row("Username", user.UserName)
	}
	if user.UserEmail != "" {
		t.Row("Email", user.UserEmail)
	}
	if user.PrimaryCurrency != "" {
		t.Row("Primary Currency", user.PrimaryCurrency)
	}
	if user.APIKeyLabel != "" {
		t.Row("API Key Label", user.APIKeyLabel)
	}
	if user.BudgetName != "" {
		t.Row("Budget Name", user.BudgetName)
	}
	if user.AccountID != 0 {
		t.Row("Account ID", strconv.Itoa(user.AccountID))
	}

	// Print the table
	fmt.Println(t)

	return nil
}
