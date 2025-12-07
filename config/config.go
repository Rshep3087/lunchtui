package config

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	settingColumnWidth     = 30
	valueColumnWidth       = 40
	descriptionColumnWidth = 50
	minMaskLength          = 4
)

// Colors represents the customizable color configuration.
type Colors struct {
	// Primary accent color used for highlights, selected items, and key bindings
	Primary string `toml:"primary"`
	// Error color used for error messages and failed transactions
	Error string `toml:"error"`
	// Success color used for successful/cleared transactions and positive values
	Success string `toml:"success"`
	// Warning color used for uncleared transactions and warnings
	Warning string `toml:"warning"`
	// Muted color used for pending transactions and secondary text
	Muted string `toml:"muted"`
	// Income color used for positive income values
	Income string `toml:"income"`
	// Expense color used for negative expense values
	Expense string `toml:"expense"`
	// Border color used for borders and separators
	Border string `toml:"border"`
	// Background color used for highlighted backgrounds
	Background string `toml:"background"`
	// Text color used for primary text
	Text string `toml:"text"`
	// SecondaryText color used for less important text
	SecondaryText string `toml:"secondary_text"`
}

// Config represents the application configuration structure.
type Config struct {
	// Debug enables debug logging
	Debug bool `toml:"debug"`
	// Token is the Lunch Money API token
	Token string `toml:"token"`
	// DebitsAsNegative shows debits as negative numbers
	DebitsAsNegative bool `toml:"debits_as_negative"`
	// HidePendingTransactions hides pending transactions from all transaction lists
	HidePendingTransactions bool `toml:"hide_pending_transactions"`
	// Colors contains customizable color settings
	Colors Colors `toml:"colors"`
}

// Model represents the config view model.
type Model struct {
	configTable table.Model
}

// New creates a new config view model.
func New(colors Colors) Model {
	configTable := table.New(
		table.WithColumns([]table.Column{
			{Title: "Setting", Width: settingColumnWidth},
			{Title: "Value", Width: valueColumnWidth},
			{Title: "Description", Width: descriptionColumnWidth},
		}),
	)

	tableStyle := table.DefaultStyles()
	tableStyle.Selected = tableStyle.Selected.
		Foreground(lipgloss.Color(colors.Primary))

	configTable.SetStyles(tableStyle)

	return Model{configTable: configTable}
}

// SetFocus sets the focus state of the config table.
func (m *Model) SetFocus(focus bool) {
	if focus {
		m.configTable.Focus()
	} else {
		m.configTable.Blur()
	}
}

// SetSize sets the size of the config table.
func (m *Model) SetSize(width, height int) {
	m.configTable.SetHeight(height)
	m.configTable.SetWidth(width)
}

func maskSensitiveValue(value string) string {
	if value == "" {
		return "(not set)"
	}

	if len(value) <= minMaskLength {
		return strings.Repeat("*", len(value))
	}

	return value[:minMaskLength] + strings.Repeat("*", len(value)-minMaskLength)
}

// SetConfig sets the configuration data for the view.
func (m *Model) SetConfig(config Config) {
	rows := []table.Row{
		{
			"Debug",
			strconv.FormatBool(config.Debug),
			"Enable debug logging",
		},
		{
			"Token",
			maskSensitiveValue(config.Token),
			"Lunch Money API token",
		},
		{
			"Debits as Negative",
			strconv.FormatBool(config.DebitsAsNegative),
			"Show debits as negative numbers",
		},
		{
			"Hide Pending Transactions",
			strconv.FormatBool(config.HidePendingTransactions),
			"Hide pending transactions from all transaction lists",
		},
		{
			"Primary Color",
			config.Colors.Primary,
			"Primary accent color",
		},
		{
			"Success Color",
			config.Colors.Success,
			"Success/cleared transaction color",
		},
		{
			"Error Color",
			config.Colors.Error,
			"Error message color",
		},
	}

	m.configTable.SetRows(rows)
}

// Init initializes the config view.
func (m *Model) Init() tea.Cmd {
	return nil
}

// Update handles updates to the config view.
func (m *Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	m.configTable, cmd = m.configTable.Update(msg)
	return *m, cmd
}

// View renders the config view.
func (m *Model) View() string {
	return m.configTable.View()
}
