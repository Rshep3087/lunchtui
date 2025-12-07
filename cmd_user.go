package main

import (
	"errors"
	"fmt"
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
	ctx := cmd.Context()

	// Get and validate output format
	outputFormat, err := validateOutputFormat(cmd)
	if err != nil {
		return err
	}

	// Fetch user information
	user, err := lmc.GetUser(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch user information: %w", err)
	}

	// Output based on format
	switch outputFormat {
	case jsonOutputFormat:
		return outputJSON(cmd, user)
	case tableOutputFormat:
		return outputUserTable(cmd, user)
	default:
		return errors.New("unsupported output format")
	}
}

func outputUserTable(cmd *cobra.Command, user *lm.User) error {
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
	fmt.Fprintln(cmd.OutOrStdout(), t)

	return nil
}
