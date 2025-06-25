package overview

import (
	"fmt"
	"strconv"

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

// Model deines the state for the overview widget for LunchTUI.
type Model struct {
	Styles             Styles
	Viewport           viewport.Model
	summary            Summary
	transactionMetrics TransactionMetrics
	transactions       []*lm.Transaction
	categories         map[int64]*lm.Category
	assets             map[int64]*lm.Asset
	plaidAccounts      map[int64]*lm.PlaidAccount
	accountTree        *tree.Tree
	spendingBreakdown  table.Model
	currency           string
	titleCaser         cases.Caser
	user               *lm.User
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
		if category == nil || category.ExcludeFromTotals || category.IsIncome {
			continue
		}

		amount, err := t.ParsedAmount()
		if err != nil {
			continue
		}

		amount = amount.Absolute()

		if _, exists := categoryTotals[category.Name]; !exists {
			categoryTotals[category.Name] = money.New(0, amount.Currency().Code)
		}

		if categoryTotals[category.Name] != nil {
			categoryTotals[category.Name], _ = categoryTotals[category.Name].Add(amount)
		}
	}

	var sortedTotals []categoryTotal
	for category, total := range categoryTotals {
		if total != nil {
			sortedTotals = append(sortedTotals, categoryTotal{category: category, total: total})
		}
	}

	// sort the categories by the total spent
	slices.SortFunc(sortedTotals, func(a categoryTotal, b categoryTotal) int {
		if a.total == nil || b.total == nil {
			return 0
		}
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
	netWorth = m.calculateAssetsNetWorth(netWorth, m.assets)
	netWorth = m.calculateAccountsNetWorth(netWorth, m.plaidAccounts)

	return netWorth
}

// calculateAssetsNetWorth calculates the net worth for assets.
func (m *Model) calculateAssetsNetWorth(netWorth *money.Money, assets map[int64]*lm.Asset) *money.Money {
	for _, asset := range assets {
		amount := money.NewFromFloat(asset.ToBase, m.currency)
		netWorth = m.updateNetWorth(netWorth, amount, asset.TypeName, asset.SubtypeName)
	}
	return netWorth
}

// calculateAccountsNetWorth calculates the net worth for plaid accounts.
func (m *Model) calculateAccountsNetWorth(netWorth *money.Money, accounts map[int64]*lm.PlaidAccount) *money.Money {
	for _, account := range accounts {
		amount := money.NewFromFloat(account.ToBase, m.currency)
		netWorth = m.updateNetWorth(netWorth, amount, account.Type, account.Subtype)
	}
	return netWorth
}

// updateNetWorth updates net worth based on the type and subtype of the asset or account.
func (m *Model) updateNetWorth(netWorth, amount *money.Money, assetType, subtype string) *money.Money {
	var nwa *money.Money
	var err error

	if assetType == "credit" && subtype == "credit card" {
		nwa, err = netWorth.Subtract(amount)
	} else {
		nwa, err = netWorth.Add(amount)
	}

	if err != nil {
		log.Debug("updating net worth", "error", err)
		return netWorth
	}

	return nwa
}

type Summary struct {
	totalIncomeEarned money.Money
	totalSpent        money.Money
	netIncome         money.Money
	savingsRate       float64
}

type TransactionMetrics struct {
	total      int
	pending    int
	unreviewed int
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
	m.updateTransactionMetrics()
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

func (m *Model) SetUser(user *lm.User) {
	m.user = user
	m.UpdateViewport()
}

func New(opts ...Option) Model {
	m := Model{
		Styles:      defaultStyles(),
		Viewport:    viewport.New(0, 20),
		summary:     Summary{},
		accountTree: tree.New(),
		titleCaser:  cases.Title(language.English),
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
					lipgloss.NewStyle().
						MarginTop(1).Render(fmt.Sprintf("Estimated Net Worth: %s", m.Styles.IncomeStyle.Render(netWorth.Display()))),
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
			m.userInfoView(),
			m.transactionMetricsView(),
			m.summaryView(),
			spendingBreakdown,
		),
	)

	m.Viewport.SetContent(mainContent)
}

func (m *Model) userInfoView() string {
	if m.user == nil {
		return lipgloss.JoinVertical(lipgloss.Top,
			lipgloss.NewStyle().Bold(true).Render("User Info"),
			m.Styles.SummaryStyle.Render("Loading user information..."),
		)
	}

	var b strings.Builder

	if m.user.BudgetName != "" {
		b.WriteString(fmt.Sprintf("Budget: %s\n", m.user.BudgetName))
	}
	if m.user.UserName != "" {
		b.WriteString(fmt.Sprintf("User: %s\n", m.user.UserName))
	}
	if m.user.PrimaryCurrency != "" {
		b.WriteString(fmt.Sprintf("Currency: %s\n", m.user.PrimaryCurrency))
	}
	if m.user.APIKeyLabel != "" {
		b.WriteString(fmt.Sprintf("API Key: %s", m.user.APIKeyLabel))
	}

	return lipgloss.JoinVertical(lipgloss.Top,
		lipgloss.NewStyle().Bold(true).Render("User Info"),
		m.Styles.SummaryStyle.Render(b.String()),
	)
}

func (m *Model) transactionMetricsView() string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("Total: %d\n", m.transactionMetrics.total))
	b.WriteString(fmt.Sprintf("Pending: %s\n", m.Styles.SpentStyle.Render(strconv.Itoa(m.transactionMetrics.pending))))
	b.WriteString(fmt.Sprintf("Unreviewed: %s", m.Styles.SpentStyle.Render(strconv.Itoa(m.transactionMetrics.unreviewed))))

	return lipgloss.JoinVertical(lipgloss.Top,
		lipgloss.NewStyle().Bold(true).Render("Transaction Metrics"),
		m.Styles.SummaryStyle.Render(b.String()),
	)
}

func (m *Model) summaryView() string {
	if m.summary.totalIncomeEarned.Currency() == nil ||
		m.summary.totalSpent.Currency() == nil ||
		m.summary.netIncome.Currency() == nil {
		return lipgloss.JoinVertical(lipgloss.Top,
			lipgloss.NewStyle().Bold(true).Render("Period Summary"),
			m.Styles.SummaryStyle.Render("No data available"),
		)
	}

	var b strings.Builder

	b.WriteString(fmt.Sprintf("Income: %s\n", m.Styles.IncomeStyle.Render(m.summary.totalIncomeEarned.Display())))
	b.WriteString(fmt.Sprintf("Spent: %s\n", m.Styles.SpentStyle.Render(m.summary.totalSpent.Display())))
	if m.summary.netIncome.IsNegative() {
		b.WriteString(fmt.Sprintf("Net Income: %s\n", m.Styles.SpentStyle.Render(m.summary.netIncome.Display())))
	} else {
		b.WriteString(fmt.Sprintf("Net Income: %s\n", m.Styles.IncomeStyle.Render(m.summary.netIncome.Display())))
	}

	// Display savings rate
	if m.summary.savingsRate >= 0 {
		b.WriteString(fmt.Sprintf(
			"Savings Rate: %s", m.Styles.IncomeStyle.Render(fmt.Sprintf("%.1f%%", m.summary.savingsRate)),
		))
	} else {
		b.WriteString(fmt.Sprintf(
			"Savings Rate: %s", m.Styles.SpentStyle.Render(fmt.Sprintf("%.1f%%", m.summary.savingsRate)),
		))
	}

	return lipgloss.JoinVertical(lipgloss.Top,
		lipgloss.NewStyle().Bold(true).Render("Period Summary"),
		m.Styles.SummaryStyle.Render(b.String()),
	)
}

func (m *Model) updateTransactionMetrics() {
	var metrics TransactionMetrics
	metrics.total = len(m.transactions)

	for _, t := range m.transactions {
		switch t.Status {
		case "pending":
			metrics.pending++
		case "uncleared":
			metrics.unreviewed++
		}
	}

	m.transactionMetrics = metrics
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
		if category == nil || category.ExcludeFromTotals {
			continue
		}

		amount, err := t.ParsedAmount()
		if err != nil {
			log.Debug("parsing amount", "error", err)
			continue
		}

		if category.IsIncome {
			tie, additionError := totalIncomeEarned.Add(amount)
			if additionError != nil {
				log.Debug("adding amount to total income earned", "error", additionError)
				continue
			}

			totalIncomeEarned = tie
		} else {
			tsa, additionError := totalSpent.Add(amount)
			if additionError != nil {
				log.Debug("adding amount to total spent", "error", additionError)
				continue
			}

			totalSpent = tsa
		}
	}

	netIncome, _ := totalIncomeEarned.Add(totalSpent)

	// Calculate savings rate as (net income / total income) * 100
	var savingsRate float64
	if totalIncomeEarned.Amount() > 0 {
		savingsRate = (float64(netIncome.Amount()) / float64(totalIncomeEarned.Amount())) * 100
	}

	m.summary = Summary{
		totalIncomeEarned: *totalIncomeEarned,
		totalSpent:        *totalSpent,
		netIncome:         *netIncome,
		savingsRate:       savingsRate,
	}
}

func (m *Model) updateAccountTree() {
	log.Debug("updating account tree")
	m.accountTree = tree.New()
	m.accountTree.Root(m.Styles.TreeRootStyle.Render("Accounts"))

	// organize the assets by the type into a map
	assets := make(map[string][]lm.Asset)
	for _, a := range m.assets {
		assets[a.TypeName] = append(assets[a.TypeName], *a)
	}

	// add a child for each asset
	for typeName, assets := range assets {
		assetTree := tree.New().Root(m.titleCaser.String(m.Styles.AssetTypeStyle.Render(typeName)))
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
		accountTree := tree.New().Root(m.titleCaser.String(m.Styles.AssetTypeStyle.Render(typeName)))
		for _, a := range accounts {
			pa := money.NewFromFloat(a.ToBase, m.currency)
			text := fmt.Sprintf("%s (%s)", a.Name, pa.Display())
			accountTree.Child(m.Styles.AccountStyle.Render(text))
		}

		m.accountTree.Child(accountTree)
	}
}
