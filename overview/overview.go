package overview

import (
	"fmt"
	"log"
	"slices"
	"strings"

	"github.com/Rhymond/go-money"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/tree"
	lm "github.com/rshep3087/lunchmoney"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var titleCaser = cases.Title(language.English)

// Model deines the state for the overview widget for LunchTUI
type Model struct {
	Styles        Styles
	Viewport      viewport.Model
	summary       Summary
	transactions  []*lm.Transaction
	categories    map[int]*lm.Category
	assets        map[int64]*lm.Asset
	plaidAccounts map[int64]*lm.PlaidAccount
	accountTree   *tree.Tree
	user          *lm.User
}

func (m *Model) calculateSpendingBreakdown() []table.Row {
	var rows []table.Row
	totalSpent := m.summary.totalSpent

	categoryTotals := make(map[string]*money.Money)

	for _, t := range m.transactions {
		category := m.categories[int(t.CategoryID)]
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

	for category, total := range categoryTotals {
		percentage := float64(total.Amount()) / float64(totalSpent.Amount()) * 100
		rows = append(rows, table.Row{category, total.Display(), fmt.Sprintf("%.2f%%", percentage)})
	}

	// Sort rows by total spent in descending order
	slices.SortFunc(rows, func(a, b table.Row) bool {
		amountA, _ := money.NewFromString(a[1].(string), "USD")
		amountB, _ := money.NewFromString(b[1].(string), "USD")
		return amountA.GreaterThan(amountB)
	})

	return rows
}

func (m *Model) calculateNetWorth() *money.Money {
	netWorth := money.New(0, "USD")

	for _, asset := range m.assets {
		amount, err := asset.ParsedAmount()
		if err != nil {
			log.Printf("error parsing asset amount: %v", err)
			continue
		}

		if asset.TypeName == "credit" && asset.SubtypeName == "credit card" {
			amount = amount.Negative()
		}

		netWorth, _ = netWorth.Add(amount)
	}

	for _, account := range m.plaidAccounts {
		amount, err := account.ParsedAmount()
		if err != nil {
			log.Printf("error parsing account amount: %v", err)
			continue
		}

		if account.Type == "credit" && account.Subtype == "credit card" {
			amount = amount.Negative()
		}

		netWorth, _ = netWorth.Add(amount)
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

		SummaryStyle: lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 2),
	}
}

type Option func(*Model)

func WithSummary(s Summary) Option {
	return func(m *Model) {
		m.summary = s
	}
}

func (m *Model) SetTransactions(transactions []*lm.Transaction) {
	log.Println("Setting transactions")
	m.transactions = transactions
	m.updateSummary()
	m.UpdateViewport()
}

func (m *Model) SetCategories(categories map[int]*lm.Category) {
	log.Println("Setting categories")
	m.categories = categories
	m.updateSummary()
	m.UpdateViewport()
}

func (m *Model) SetAccounts(assets map[int64]*lm.Asset, plaidAccounts map[int64]*lm.PlaidAccount) {
	log.Println("Setting accounts")
	m.assets = assets
	m.plaidAccounts = plaidAccounts
	m.updateAccountTree()
	m.UpdateViewport()
}

func (m *Model) SetUser(user *lm.User) {
	log.Println("Setting user")
	m.user = user
	m.UpdateViewport()
}

func New(opts ...Option) Model {
	m := Model{
		Styles:   defaultStyles(),
		Viewport: viewport.New(0, 20),
		summary: Summary{
			// setting them to 0 so that the currency is set,
			// otherwise it's nil and blows up
			totalIncomeEarned: *money.New(0, "USD"),
			totalSpent:        *money.New(0, "USD"),
			netIncome:         *money.New(0, "USD"),
		},
		accountTree: tree.New(),
	}

	m.accountTree.Root(m.Styles.TreeRootStyle.Render("Accounts"))

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
	accountTreeContent := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1, 2).
		Render(
			lipgloss.JoinVertical(lipgloss.Top,
				m.accountTree.String(),
				fmt.Sprintf("Estimated Net Worth: %s", m.Styles.IncomeStyle.Render(netWorth.Display())),
			),
		)

	spendingBreakdown := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1, 2).
		Render(
			lipgloss.JoinVertical(lipgloss.Top,
				lipgloss.NewStyle().Bold(true).Render("Spending Breakdown"),
				table.New(
					table.WithColumns([]table.Column{
						{Title: "Category", Width: 20},
						{Title: "Total Spent", Width: 15},
						{Title: "% of Total", Width: 10},
					}),
					table.WithRows(m.calculateSpendingBreakdown()),
				).View(),
			),
		)

	mainContent := lipgloss.JoinHorizontal(lipgloss.Top,
		m.summaryView(),
		accountTreeContent,
		spendingBreakdown,
	)

	m.Viewport.SetContent(
		lipgloss.JoinVertical(lipgloss.Top,
			m.headerView(),
			mainContent,
		),
	)
}

func (m *Model) headerView() string {
	if m.user == nil {
		return "Overview"
	}

	return fmt.Sprintf("Welcome - %s!", m.user.UserName)
}

func (m Model) summaryView() string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("Income: %s\n", m.Styles.IncomeStyle.Render(m.summary.totalIncomeEarned.Display())))
	b.WriteString(fmt.Sprintf("Spent: %s\n", m.Styles.SpentStyle.Render(m.summary.totalSpent.Display())))
	if m.summary.netIncome.IsNegative() {
		b.WriteString(fmt.Sprintf("Net Income: %s", m.Styles.SpentStyle.Render(m.summary.netIncome.Display())))
	} else {
		b.WriteString(fmt.Sprintf("Net Income: %s", m.Styles.IncomeStyle.Render(m.summary.netIncome.Display())))
	}

	return m.Styles.SummaryStyle.Render(b.String())
}

func (m *Model) updateSummary() {
	if m.categories == nil {
		return
	}

	if len(m.transactions) == 0 {
		return
	}

	var totalIncomeEarned, totalSpent = money.New(0, "USD"), money.New(0, "USD")

	for _, t := range m.transactions {
		category := m.categories[int(t.CategoryID)]
		if category.ExcludeFromTotals {
			continue
		}

		amount, err := t.ParsedAmount()
		if err != nil {
			continue
		}

		if m.categories[int(t.CategoryID)].IsIncome {
			totalIncomeEarned, _ = totalIncomeEarned.Add(amount)
		} else {
			totalSpent, _ = totalSpent.Add(amount)
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
			pa, err := a.ParsedAmount()
			if err != nil {
				log.Printf("error parsing amount: %v", err)
				continue
			}

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
			pa, err := a.ParsedAmount()
			if err != nil {
				log.Printf("error parsing amount: %v", err)
				continue
			}

			text := fmt.Sprintf("%s (%s)", a.Name, pa.Display())
			accountTree.Child(m.Styles.AccountStyle.Render(text))
		}

		m.accountTree.Child(accountTree)
	}
}
