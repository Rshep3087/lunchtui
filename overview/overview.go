package overview

import (
	"fmt"

	"slices"
	"strings"

	"github.com/Rhymond/go-money"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/tree"
	"github.com/charmbracelet/log"
	lm "github.com/icco/lunchmoney"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var titleCaser = cases.Title(language.English)

// Model deines the state for the overview widget for LunchTUI
type Model struct {
	Styles            Styles
	Viewport          viewport.Model
	summary           Summary
	transactions      []*lm.Transaction
	categories        map[int64]*lm.Category
	assets            map[int64]*lm.Asset
	plaidAccounts     map[int64]*lm.PlaidAccount
	accountTree       *tree.Tree
	spendingBreakdown table.Model
	currency          string
}

type categoryTotal struct {
	category string
	total    *money.Money
}

func (m *Model) calculateSpendingBreakdown() []table.Row {
	var rows []table.Row

	categoryTotals := make(map[string]*money.Money)

	for _, t := range m.transactions {
		category := m.categories[t.CategoryID]
		if category.ExcludeFromTotals || category.IsIncome {
			continue
		}

		amount, err := t.ParsedAmount()
		if err != nil {
			continue
		}

		amount = amount.Absolute()

		if _, exists := categoryTotals[category.Name]; !exists {
			categoryTotals[category.Name] = money.New(0, "USD")
		}

		categoryTotals[category.Name], _ = categoryTotals[category.Name].Add(amount)
	}

	var sortedTotals []categoryTotal
	for category, total := range categoryTotals {
		sortedTotals = append(sortedTotals, categoryTotal{category: category, total: total})
	}

	// sort the categories by the total spent
	slices.SortFunc(sortedTotals, func(a categoryTotal, b categoryTotal) int {
		x, _ := a.total.Compare(b.total)
		return -x
	})

	for _, total := range sortedTotals {
		rows = append(rows, table.Row{
			total.category,
			total.total.Display(),
		})
	}

	return rows
}

func (m *Model) calculateNetWorth() *money.Money {
	if m.currency == "" {
		return money.New(0, "USD")
	}

	netWorth := money.New(0, m.currency)

	for _, asset := range m.assets {
		amount := money.NewFromFloat(asset.ToBase, m.currency)
		if asset.TypeName == "credit" && asset.SubtypeName == "credit card" {
			amount = amount.Absolute()
		}

		nwa, err := netWorth.Add(amount)
		if err != nil {
			continue
		}

		netWorth = nwa
	}

	for _, account := range m.plaidAccounts {
		amount := money.NewFromFloat(account.ToBase, m.currency)

		if account.Type == "credit" && account.Subtype == "credit card" {
			// if the account is a credit card, we want to show the amount as a positive number
			amount = amount.Absolute()
		}

		nwa, err := netWorth.Add(amount)
		if err != nil {
			continue
		}

		netWorth = nwa
	}

	return netWorth
}

type Summary struct {
	totalIncomeEarned money.Money
	totalSpent        money.Money
	netIncome         money.Money
}

type Styles struct {
	IncomeStyle    lipgloss.Style
	SpentStyle     lipgloss.Style
	TreeRootStyle  lipgloss.Style
	AssetTypeStyle lipgloss.Style
	AccountStyle   lipgloss.Style
	SummaryStyle   lipgloss.Style
}

func defaultStyles() Styles {
	return Styles{
		IncomeStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("#00ff00")),
		SpentStyle:     lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000")),
		TreeRootStyle:  lipgloss.NewStyle().Foreground(lipgloss.Color("#828282")),
		AssetTypeStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("#bbbbbb")),
		AccountStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("#d29b1d")),

		SummaryStyle: lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1),
	}
}

type Option func(*Model)

func (m *Model) SetTransactions(transactions []*lm.Transaction) {
	m.transactions = transactions
	m.updateSummary()
	m.UpdateViewport()
}

func (m *Model) SetCategories(categories map[int64]*lm.Category) {
	m.categories = categories
	m.updateSummary()
	m.UpdateViewport()
}

func (m *Model) SetAccounts(assets map[int64]*lm.Asset, plaidAccounts map[int64]*lm.PlaidAccount) {
	m.assets = assets
	m.plaidAccounts = plaidAccounts
	m.updateAccountTree()
	m.UpdateViewport()
}

func (m *Model) SetCurrency(currency string) {
	m.currency = currency
	m.updateAccountTree()
	m.UpdateViewport()
}

func New(opts ...Option) Model {
	m := Model{
		Styles:      defaultStyles(),
		Viewport:    viewport.New(0, 20),
		summary:     Summary{},
		accountTree: tree.New(),
	}

	m.accountTree.Root(m.Styles.TreeRootStyle.Render("Accounts"))

	tableStyle := table.DefaultStyles()
	tableStyle.Selected = lipgloss.NewStyle()
	m.spendingBreakdown = table.New(
		table.WithColumns([]table.Column{
			{Title: "Category", Width: 20},
			{Title: "Total Spent", Width: 15},
		}),
		table.WithFocused(false),
		table.WithStyles(tableStyle),
	)

	for _, opt := range opts {
		opt(&m)
	}

	m.UpdateViewport()

	return m
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	m.Viewport, cmd = m.Viewport.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	return m.Viewport.View()
}
func (m *Model) SetSize(width, height int) {
	m.setSize(width, height)
}

func (m *Model) setSize(width, height int) {
	m.Viewport.Width = width
	m.Viewport.Height = height
}

func (m *Model) UpdateViewport() {
	netWorth := m.calculateNetWorth()
	accountTreeContent := lipgloss.JoinVertical(lipgloss.Top,
		lipgloss.NewStyle().Bold(true).Render("Accounts Overview"),
		lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(0, 1).
			Render(
				lipgloss.JoinVertical(lipgloss.Top,
					lipgloss.NewStyle().MarginBottom(1).Render(m.accountTree.String()),
					lipgloss.NewStyle().MarginTop(1).Render(fmt.Sprintf("Estimated Net Worth: %s", m.Styles.IncomeStyle.Render(netWorth.Display()))),
				),
			),
	)

	var spendingBreakdownData string
	rows := m.calculateSpendingBreakdown()
	if len(rows) == 0 {
		spendingBreakdownData = "No data available"
	} else {
		m.spendingBreakdown.SetRows(rows)
		m.spendingBreakdown.SetHeight(len(rows))
		spendingBreakdownData = m.spendingBreakdown.View()
	}

	spendingBreakdown := lipgloss.JoinVertical(lipgloss.Top,
		lipgloss.NewStyle().Bold(true).Render("Spending Breakdown"),
		lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(0, 1).
			Render(spendingBreakdownData),
	)

	mainContent := lipgloss.JoinHorizontal(lipgloss.Top,
		accountTreeContent,
		lipgloss.JoinVertical(lipgloss.Top,
			m.summaryView(),
			spendingBreakdown,
		),
	)

	m.Viewport.SetContent(mainContent)
}

func (m Model) summaryView() string {
	if m.summary.totalIncomeEarned.Currency() == nil || m.summary.totalSpent.Currency() == nil || m.summary.netIncome.Currency() == nil {
		return lipgloss.JoinVertical(lipgloss.Top,
			lipgloss.NewStyle().Bold(true).Render("Period Summary"),
			m.Styles.SummaryStyle.Render("No data available"),
		)
	}

	var b strings.Builder

	b.WriteString(fmt.Sprintf("Income: %s\n", m.Styles.IncomeStyle.Render(m.summary.totalIncomeEarned.Display())))
	b.WriteString(fmt.Sprintf("Spent: %s\n", m.Styles.SpentStyle.Render(m.summary.totalSpent.Display())))
	if m.summary.netIncome.IsNegative() {
		b.WriteString(fmt.Sprintf("Net Income: %s", m.Styles.SpentStyle.Render(m.summary.netIncome.Display())))
	} else {
		b.WriteString(fmt.Sprintf("Net Income: %s", m.Styles.IncomeStyle.Render(m.summary.netIncome.Display())))
	}

	return lipgloss.JoinVertical(lipgloss.Top,
		lipgloss.NewStyle().Bold(true).Render("Period Summary"),
		m.Styles.SummaryStyle.Render(b.String()),
	)
}

func (m *Model) updateSummary() {
	if m.categories == nil {
		return
	}

	if len(m.transactions) == 0 {
		return
	}

	var totalIncomeEarned, totalSpent = money.New(0, m.currency), money.New(0, m.currency)

	for _, t := range m.transactions {
		category := m.categories[t.CategoryID]
		if category.ExcludeFromTotals {
			continue
		}

		amount, err := t.ParsedAmount()
		if err != nil {
			log.Debug("parsing amount", "error", err)
			continue
		}

		if m.categories[t.CategoryID].IsIncome {
			tie, err := totalIncomeEarned.Add(amount)
			if err != nil {
				log.Debug("adding amount to total income earned", "error", err)
				continue
			}

			totalIncomeEarned = tie
		} else {
			tsa, err := totalSpent.Add(amount)
			if err != nil {
				log.Debug("adding amount to total spent", "error", err)
				continue
			}

			totalSpent = tsa
		}

	}

	netIncome, _ := totalIncomeEarned.Add(totalSpent)

	m.summary = Summary{totalIncomeEarned: *totalIncomeEarned, totalSpent: *totalSpent, netIncome: *netIncome}
}

func (m *Model) updateAccountTree() {
	// organize the assets by the type into a map
	assets := make(map[string][]lm.Asset)
	for _, a := range m.assets {
		assets[a.TypeName] = append(assets[a.TypeName], *a)
	}

	// add a child for each asset
	for typeName, assets := range assets {
		assetTree := tree.New().Root(titleCaser.String(m.Styles.AssetTypeStyle.Render(typeName)))
		for _, a := range assets {
			pa := money.NewFromFloat(a.ToBase, m.currency)
			text := fmt.Sprintf("%s (%s)", a.Name, pa.Display())
			assetTree.Child(m.Styles.AccountStyle.Render(text))
		}

		m.accountTree.Child(assetTree)
	}

	// // organize the plaid accounts by the type into a map
	plaidAccounts := make(map[string][]lm.PlaidAccount)
	for _, a := range m.plaidAccounts {
		plaidAccounts[a.Type] = append(plaidAccounts[a.Type], *a)
	}

	for typeName, accounts := range plaidAccounts {
		accountTree := tree.New().Root(titleCaser.String(m.Styles.AssetTypeStyle.Render(typeName)))
		for _, a := range accounts {
			pa := money.NewFromFloat(a.ToBase, m.currency)
			text := fmt.Sprintf("%s (%s)", a.Name, pa.Display())
			accountTree.Child(m.Styles.AccountStyle.Render(text))
		}

		m.accountTree.Child(accountTree)
	}
}
