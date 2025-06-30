package config

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

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
}

// Model represents the config view model.
type Model struct {
	configTable table.Model
}

// New creates a new config view model.
func New() Model {
	configTable := table.New(
		table.WithColumns([]table.Column{
			{Title: "Setting", Width: 30},
			{Title: "Value", Width: 40},
			{Title: "Description", Width: 50},
		}),
	)

	tableStyle := table.DefaultStyles()
	tableStyle.Selected = tableStyle.Selected.
		Foreground(lipgloss.Color("#ffd644"))

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

	if len(value) <= 4 {
		return strings.Repeat("*", len(value))
	}

	return value[:4] + strings.Repeat("*", len(value)-4)
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
	}

	m.configTable.SetRows(rows)
}

// Init initializes the config view.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles updates to the config view.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	m.configTable, cmd = m.configTable.Update(msg)
	return m, cmd
}

// View renders the config view.
func (m Model) View() string {
	return m.configTable.View()
}
