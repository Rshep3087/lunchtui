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

// categoriesCmd represents the categories command.
var categoriesCmd = &cobra.Command{
	Use:   "categories",
	Short: "Category management commands",
	Long:  `Commands for managing categories in Lunch Money.`,
}

// categoriesListCmd represents the categories list command.
var categoriesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all categories",
	Long:  `List all categories with their IDs and details.`,
	RunE:  categoriesListRun,
}

func init() {
	// Add categories list subcommand
	categoriesCmd.AddCommand(categoriesListCmd)

	// Categories list flags
	categoriesListCmd.Flags().StringP("output", "o", tableOutputFormat, "Output format: table or json")
}

func categoriesListRun(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Get output format
	outputFormat, _ := cmd.Flags().GetString("output")

	// Validate output format
	validFormats := []string{tableOutputFormat, jsonOutputFormat}
	if !slices.Contains(validFormats, outputFormat) {
		return fmt.Errorf("invalid output format: %s (must be one of %v)", outputFormat, validFormats)
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
	case jsonOutputFormat:
		return outputJSON(categories)
	case tableOutputFormat:
		return outputCategoriesTable(categories)
	default:
		return errors.New("unsupported output format")
	}
}

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
