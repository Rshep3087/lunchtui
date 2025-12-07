package recurring

import (
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/icco/lunchmoney"
)

type Colors struct {
	Primary string
}

type Model struct {
	recurringExpenses table.Model
}

func New(colors Colors) Model {
	recurringExpenses := table.New(
		table.WithColumns([]table.Column{
			{Title: "Merchant", Width: 20},
			{Title: "Description", Width: 30},
			{Title: "Repeats", Width: 10},
			{Title: "Billing Day", Width: 12},
			{Title: "Amount", Width: 10},
		}),
	)

	tableStyle := table.DefaultStyles()
	tableStyle.Selected = tableStyle.Selected.
		Foreground(lipgloss.Color(colors.Primary))

	recurringExpenses.SetStyles(tableStyle)

	return Model{recurringExpenses: recurringExpenses}
}

func (m *Model) SetFocus(focus bool) {
	if focus {
		m.recurringExpenses.Focus()
	} else {
		m.recurringExpenses.Blur()
	}
}

func (m *Model) SetSize(width, height int) {
	m.recurringExpenses.SetHeight(height)
	m.recurringExpenses.SetWidth(width)
}

func (m *Model) SetRecurringExpenses(re []*lunchmoney.RecurringExpense) {
	rows := make([]table.Row, 0)
	for _, r := range re {
		money, err := r.ParsedAmount()
		if err != nil {
			continue
		}
		rows = append(rows, table.Row{
			r.Payee,
			r.Description,
			r.Cadence,
			r.BillingDate,
			money.Display(),
		})
	}

	m.recurringExpenses.SetRows(rows)
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	m.recurringExpenses, cmd = m.recurringExpenses.Update(msg)
	return *m, cmd
}

func (m *Model) View() string {
	return m.recurringExpenses.View()
}
