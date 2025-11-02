package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Rshep3087/lunchtui/config"
	"github.com/charmbracelet/fang"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/charmbracelet/log"
	lm "github.com/icco/lunchmoney"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	jsonOutputFormat  = "json"
	tableOutputFormat = "table"
)

// Global variables for configuration.
var (
	cfgFile string
	lmc     *lm.Client

	// local variables for root command.
	showUserInfo bool
)

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.lunchtui.toml)")
	rootCmd.PersistentFlags().Bool("debug", false, "enable debug logging")
	rootCmd.PersistentFlags().String("token", "", "the API token for Lunch Money")
	rootCmd.PersistentFlags().Bool("debits-as-negative", false, "show debits as negative numbers")
	rootCmd.PersistentFlags().Bool("hide-pending-transactions", false,
		"hide pending transactions from all transaction lists")
	rootCmd.PersistentFlags().String("anthropic-api-key", "", "Anthropic API key for AI-powered category recommendations")

	// root comand flags
	rootCmd.Flags().BoolVar(&showUserInfo, "show-user-info", true, "show user information in the overview")
	_ = viper.BindPFlag("show_user_info", rootCmd.Flags().Lookup("show-user-info"))

	// Bind flags to viper
	_ = viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug"))
	_ = viper.BindPFlag("token", rootCmd.PersistentFlags().Lookup("token"))
	_ = viper.BindPFlag("debits_as_negative", rootCmd.PersistentFlags().Lookup("debits-as-negative"))
	_ = viper.BindPFlag("hide_pending_transactions", rootCmd.PersistentFlags().Lookup("hide-pending-transactions"))
	_ = viper.BindPFlag("ai.anthropic_api_key", rootCmd.PersistentFlags().Lookup("anthropic-api-key"))

	// Bind environment variables
	_ = viper.BindEnv("token", "LUNCHMONEY_API_TOKEN")
	_ = viper.BindEnv("ai.anthropic_api_key", "ANTHROPIC_API_KEY")

	rootCmd.AddCommand(transactionCmd)
	rootCmd.AddCommand(accountsCmd)
	rootCmd.AddCommand(userCmd)
	rootCmd.AddCommand(networthCmd)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		if err != nil {
			log.Error("Error finding home directory", "error", err)
			os.Exit(1)
		}

		// Search config in multiple locations (in order of precedence)
		// Current directory (highest precedence)
		viper.AddConfigPath(".")
		viper.SetConfigName("lunchtui")
		viper.SetConfigType("toml")

		// User config directory
		if configDir, configErr := os.UserConfigDir(); configErr == nil {
			viper.AddConfigPath(filepath.Join(configDir, "lunchtui"))
		}

		// User home directory
		viper.AddConfigPath(home)
		viper.AddConfigPath(filepath.Join(home, ".config", "lunchtui"))

		// System-wide config directory (lowest precedence)
		viper.AddConfigPath("/etc/lunchtui")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err != nil {
		log.Debug("Config file not found or error reading", "error", err)
		return
	}
}

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:   "lunchtui",
	Short: "A terminal UI and CLI for Lunch Money",
	Long:  `A comprehensive terminal-based interface and CLI for managing your Lunch Money financial data.`,
	PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
		// Validate token
		if viper.GetString("token") == "" {
			return errors.New("API token is required (set via --token flag, " +
				"LUNCHMONEY_API_TOKEN environment variable, or config file)")
		}

		// Create Lunch Money client
		var err error
		lmc, err = lm.NewClient(viper.GetString("token"))
		if err != nil {
			return fmt.Errorf("failed to create Lunch Money client: %w", err)
		}

		loggingTransport := newLoggingTransport(lmc.HTTP.Transport, log.Default())
		lmc.HTTP.Transport = loggingTransport

		if viper.GetBool("debug") {
			log.SetLevel(log.DebugLevel)
		}

		categoryService := NewCategoryService(lmc)
		cmd.Root().AddCommand(newCategoriesCmd(categoryService))

		return nil
	},
	RunE: func(c *cobra.Command, _ []string) error {
		// Start TUI when no subcommands are provided
		config := Config{
			Debug:                   viper.GetBool("debug"),
			Token:                   viper.GetString("token"),
			DebitsAsNegative:        viper.GetBool("debits_as_negative"),
			HidePendingTransactions: viper.GetBool("hide_pending_transactions"),
			ShowUserInfo:            viper.GetBool("show_user_info"),
			Colors: config.Colors{
				Primary:       viper.GetString("colors.primary"),
				Error:         viper.GetString("colors.error"),
				Success:       viper.GetString("colors.success"),
				Warning:       viper.GetString("colors.warning"),
				Muted:         viper.GetString("colors.muted"),
				Income:        viper.GetString("colors.income"),
				Expense:       viper.GetString("colors.expense"),
				Border:        viper.GetString("colors.border"),
				Background:    viper.GetString("colors.background"),
				Text:          viper.GetString("colors.text"),
				SecondaryText: viper.GetString("colors.secondary_text"),
			},
			AI: AIConfig{
				AnthropicAPIKey: viper.GetString("ai.anthropic_api_key"),
			},
		}

		return rootAction(c.Context(), config, lmc)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := fang.Execute(context.Background(), rootCmd); err != nil {
		os.Exit(1)
	}
}

// Utility functions for output formatting.
func outputJSON(data any) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	fmt.Println(string(jsonData))
	return nil
}

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
