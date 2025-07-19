package overview

import (
	"fmt"
	"strconv"

	"slices"
	"strings"

	"github.com/Rhymond/go-money"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/tree"
	"github.com/charmbracelet/log"
	lm "github.com/icco/lunchmoney"
	"golang.org/x/net/html"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// Config holds the configuration for the overview model.
type Config struct {
	ShowUserInfo bool
	// Colors can be provided to customize the theme
	Colors *Colors
}

// Colors represents theme colors for the overview.
type Colors struct {
	Income        lipgloss.Color
	Expense       lipgloss.Color
	TreeRoot      lipgloss.Color
	AssetType     lipgloss.Color
	Account       lipgloss.Color
	SectionHeader lipgloss.Color
}

// Model deines the state for the overview widget for LunchTUI.
type Model struct {
	cfg                Config
	Styles             Styles
	Viewport           viewport.Model
	summary            Summary
	transactionMetrics TransactionMetrics
	transactions       []*lm.Transaction
	categories         map[int64]*lm.Category
	assets             map[int64]*lm.Asset
	plaidAccounts      map[int64]*lm.PlaidAccount
	accountTree        *tree.Tree
	spendingBreakdown  *tree.Tree
	currency           string
	titleCaser         cases.Caser
	user               *lm.User
}

type spendingData struct {
	categoryTotals      map[int64]*money.Money
	groupTotals         map[int64]*money.Money
	groupCategories     map[int64][]int64 // group_id -> []category_ids
	groupNames          map[int64]string  // group_id -> group_name
	ungroupedCategories []int64           // categories with GroupID == 0
}

func (m *Model) CalculateSpendingBreakdown() *tree.Tree {
	spendingTree := tree.New()
	spendingTree.Root("Categories")

	data := m.collectSpendingData()
	m.addUngroupedCategoriesToTree(spendingTree, data)
	m.addGroupedCategoriesToTree(spendingTree, data)

	return spendingTree
}

func (m *Model) collectSpendingData() *spendingData {
	data := &spendingData{
		categoryTotals:      make(map[int64]*money.Money),
		groupTotals:         make(map[int64]*money.Money),
		groupCategories:     make(map[int64][]int64),
		groupNames:          make(map[int64]string),
		ungroupedCategories: make([]int64, 0),
	}

	// First pass: collect group information
	for _, category := range m.categories {
		if category.IsGroup {
			data.groupNames[category.ID] = category.Name
			data.groupTotals[category.ID] = money.New(0, m.currency)
		}
	}

	// Second pass: calculate totals and organize categories
	for _, t := range m.transactions {
		m.processTransaction(t, data)
	}

	return data
}

func (m *Model) processTransaction(t *lm.Transaction, data *spendingData) {
	category := m.categories[t.CategoryID]
	if category == nil || category.ExcludeFromTotals || category.IsIncome || category.IsGroup {
		return
	}

	amount, err := t.ParsedAmount()
	if err != nil {
		return
	}

	amount = amount.Absolute()

	// Initialize category total if not exists
	if _, exists := data.categoryTotals[category.ID]; !exists {
		data.categoryTotals[category.ID] = money.New(0, amount.Currency().Code)
	}

	// Add to category total
	if data.categoryTotals[category.ID] != nil {
		data.categoryTotals[category.ID], _ = data.categoryTotals[category.ID].Add(amount)
	}

	// Organize by group
	if category.GroupID == 0 {
		if !slices.Contains(data.ungroupedCategories, category.ID) {
			data.ungroupedCategories = append(data.ungroupedCategories, category.ID)
		}
	} else {
		if !slices.Contains(data.groupCategories[category.GroupID], category.ID) {
			data.groupCategories[category.GroupID] = append(data.groupCategories[category.GroupID], category.ID)
		}
		if data.groupTotals[category.GroupID] != nil {
			data.groupTotals[category.GroupID], _ = data.groupTotals[category.GroupID].Add(amount)
		}
	}
}

func (m *Model) sortCategoriesByTotal(categoryIDs []int64, categoryTotals map[int64]*money.Money) {
	slices.SortFunc(categoryIDs, func(a, b int64) int {
		totalA := categoryTotals[a]
		totalB := categoryTotals[b]
		if totalA == nil || totalB == nil {
			return 0
		}
		x, _ := totalA.Compare(totalB)
		return -x // Descending order
	})
}

func (m *Model) addUngroupedCategoriesToTree(spendingTree *tree.Tree, data *spendingData) {
	m.sortCategoriesByTotal(data.ungroupedCategories, data.categoryTotals)
	for _, categoryID := range data.ungroupedCategories {
		category := m.categories[categoryID]
		total := data.categoryTotals[categoryID]
		if category != nil && total != nil && total.Amount() > 0 {
			categoryText := fmt.Sprintf("%s %s", category.Name, total.Display())
			spendingTree.Child(categoryText)
		}
	}
}

func (m *Model) addGroupedCategoriesToTree(spendingTree *tree.Tree, data *spendingData) {
	sortedGroupIDs := m.getSortedGroupIDs(data)

	for _, groupID := range sortedGroupIDs {
		groupName := data.groupNames[groupID]
		groupTotal := data.groupTotals[groupID]
		categoriesInGroup := data.groupCategories[groupID]

		if groupTotal == nil || groupTotal.Amount() <= 0 {
			continue
		}

		// Create group header with total
		groupText := fmt.Sprintf("▼ %s %s", groupName, groupTotal.Display())
		groupTree := tree.New().Root(groupText)

		// Sort and add categories within group
		m.sortCategoriesByTotal(categoriesInGroup, data.categoryTotals)
		for _, categoryID := range categoriesInGroup {
			category := m.categories[categoryID]
			total := data.categoryTotals[categoryID]
			if category != nil && total != nil && total.Amount() > 0 {
				categoryText := fmt.Sprintf("%s %s", category.Name, total.Display())
				groupTree.Child(categoryText)
			}
		}

		spendingTree.Child(groupTree)
	}
}

func (m *Model) getSortedGroupIDs(data *spendingData) []int64 {
	var sortedGroupIDs []int64
	for groupID := range data.groupTotals {
		if len(data.groupCategories[groupID]) > 0 {
			sortedGroupIDs = append(sortedGroupIDs, groupID)
		}
	}
	slices.SortFunc(sortedGroupIDs, func(a, b int64) int {
		totalA := data.groupTotals[a]
		totalB := data.groupTotals[b]
		if totalA == nil || totalB == nil {
			return 0
		}
		x, _ := totalA.Compare(totalB)
		return -x // Descending order
	})
	return sortedGroupIDs
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
	// SectionHeaderStyle is used for section headers in the overview
	SectionHeaderStyle lipgloss.Style
}

func defaultStyles() Styles {
	return Styles{
		IncomeStyle:        lipgloss.NewStyle().Foreground(lipgloss.Color("#00ff00")),
		SpentStyle:         lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000")),
		TreeRootStyle:      lipgloss.NewStyle().Foreground(lipgloss.Color("#828282")),
		AssetTypeStyle:     lipgloss.NewStyle().Foreground(lipgloss.Color("#bbbbbb")),
		AccountStyle:       lipgloss.NewStyle().Foreground(lipgloss.Color("#d29b1d")),
		SummaryStyle:       lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1),
		SectionHeaderStyle: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#ffffff")),
	}
}

func stylesFromColors(colors Colors) Styles {
	return Styles{
		IncomeStyle:        lipgloss.NewStyle().Foreground(colors.Income),
		SpentStyle:         lipgloss.NewStyle().Foreground(colors.Expense),
		TreeRootStyle:      lipgloss.NewStyle().Foreground(colors.TreeRoot),
		AssetTypeStyle:     lipgloss.NewStyle().Foreground(colors.AssetType),
		AccountStyle:       lipgloss.NewStyle().Foreground(colors.Account),
		SummaryStyle:       lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1),
		SectionHeaderStyle: lipgloss.NewStyle().Bold(true).Foreground(colors.SectionHeader),
	}
}

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

func New(cfg Config) Model {
	var styles Styles
	if cfg.Colors != nil {
		styles = stylesFromColors(*cfg.Colors)
	} else {
		styles = defaultStyles()
	}

	m := Model{
		Styles:      styles,
		Viewport:    viewport.New(0, 20),
		summary:     Summary{},
		accountTree: tree.New(),
		titleCaser:  cases.Title(language.English),
		cfg: Config{
			ShowUserInfo: true,
		},
	}

	m.accountTree.Root(m.Styles.TreeRootStyle.Render("Accounts"))

	m.spendingBreakdown = tree.New()
	m.spendingBreakdown.Root("")

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
	m.UpdateViewport()
}

func (m *Model) setSize(width, height int) {
	log.Debug("setting overview viewport size", "width", width, "height", height)
	m.Viewport.Width = width
	m.Viewport.Height = height
}

func (m *Model) UpdateViewport() {
	netWorth := m.calculateNetWorth()
	accountTreeContent := lipgloss.JoinVertical(lipgloss.Top,
		m.Styles.SectionHeaderStyle.Render("Accounts Overview"),
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
	m.spendingBreakdown = m.CalculateSpendingBreakdown()
	// if there are no children in the spending breakdown, display a message
	// otherwise, render the spending breakdown tree
	spendingBreakdownData = m.spendingBreakdown.String()
	if m.spendingBreakdown.Children().Length() == 0 || strings.TrimSpace(spendingBreakdownData) == "" {
		spendingBreakdownData = "No spending data available\nfor this period."
	}

	spendingBreakdown := lipgloss.JoinVertical(lipgloss.Top,
		m.Styles.SectionHeaderStyle.Render("Spending Breakdown"),
		lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(0, 1).
			Render(spendingBreakdownData),
	)

	var leftColumn []string
	if m.cfg.ShowUserInfo {
		log.Debug("showing user info in overview")
		leftColumn = append(leftColumn, m.userInfoView())
	}
	leftColumn = append(leftColumn, m.transactionMetricsView())
	leftColumn = append(leftColumn, m.summaryView())

	mainContent := lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Margin(0, 1, 0, 1).
			Render(lipgloss.JoinVertical(lipgloss.Left, leftColumn...)),
		lipgloss.NewStyle().Margin(0, 1, 0, 1).Render(accountTreeContent),
		lipgloss.NewStyle().Margin(0, 1, 0, 1).Render(spendingBreakdown),
	)

	if m.Viewport.Width <= 122 {
		leftColumn = append(leftColumn, spendingBreakdown)
		log.Debug("narrow viewport detected, adjusting layout")
		mainContent = lipgloss.JoinHorizontal(lipgloss.Top,
			lipgloss.JoinVertical(lipgloss.Left, leftColumn...),
			accountTreeContent,
		)
	}

	m.Viewport.SetContent(mainContent)
}

func (m *Model) userInfoView() string {
	if m.user == nil {
		return lipgloss.JoinVertical(lipgloss.Top,
			m.Styles.SectionHeaderStyle.Render("User Info"),
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

	padded := lipgloss.NewStyle().PaddingBottom(1)

	return padded.Render(lipgloss.JoinVertical(lipgloss.Top,
		m.Styles.SectionHeaderStyle.Render("User Info"),
		m.Styles.SummaryStyle.Render(b.String()),
	))
}

func (m *Model) transactionMetricsView() string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("Total: %d\n", m.transactionMetrics.total))
	b.WriteString(fmt.Sprintf("Pending: %s\n", m.Styles.SpentStyle.Render(strconv.Itoa(m.transactionMetrics.pending))))
	b.WriteString(fmt.Sprintf("Unreviewed: %s", m.Styles.SpentStyle.Render(strconv.Itoa(m.transactionMetrics.unreviewed))))

	return lipgloss.NewStyle().PaddingBottom(1).Render(lipgloss.JoinVertical(lipgloss.Top,
		m.Styles.SectionHeaderStyle.Render("Transaction Metrics"),
		m.Styles.SummaryStyle.Render(b.String()),
	))
}

func (m *Model) summaryView() string {
	if m.summary.totalIncomeEarned.Currency() == nil ||
		m.summary.totalSpent.Currency() == nil ||
		m.summary.netIncome.Currency() == nil {
		return lipgloss.NewStyle().PaddingBottom(1).Render(lipgloss.JoinVertical(lipgloss.Top,
			m.Styles.SectionHeaderStyle.Render("Period Summary"),
			m.Styles.SummaryStyle.Render("No data available"),
		))
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

	return lipgloss.NewStyle().PaddingBottom(1).Render(lipgloss.JoinVertical(lipgloss.Top,
		lipgloss.NewStyle().Bold(true).Render("Period Summary"),
		m.Styles.SummaryStyle.Render(b.String()),
	))
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

// accountItem is a helper struct to hold account information for the tree view.
type accountItem struct {
	name    string
	amount  *money.Money
	isAsset bool
}

func (m *Model) updateAccountTree() {
	log.Debug("updating account tree")
	m.accountTree = tree.New()
	m.accountTree.Root(m.Styles.TreeRootStyle.Render("Accounts"))

	// Combine both plaid accounts and assets into a single map by type
	combinedAccounts := make(map[string][]accountItem)

	// Add plaid accounts
	for _, a := range m.plaidAccounts {
		pa, err := a.ParsedAmount()
		if err != nil {
			log.Debug("parsing plaid account amount", "error", err)
			continue
		}

		// Use the display name if available, otherwise use the account name
		name := a.Name
		if a.DisplayName != "" {
			name = html.UnescapeString(a.DisplayName)
		}

		item := accountItem{
			name:    name,
			amount:  pa,
			isAsset: false,
		}
		combinedAccounts[a.Type] = append(combinedAccounts[a.Type], item)
	}

	// Add assets
	for _, a := range m.assets {
		pa, err := a.ParsedAmount()
		if err != nil {
			log.Debug("parsing asset amount", "error", err)
			continue
		}

		name := a.Name
		if a.DisplayName != "" {
			// url decodes the display name
			name = html.UnescapeString(a.DisplayName)
		}
		item := accountItem{
			name:    name,
			amount:  pa,
			isAsset: true,
		}
		combinedAccounts[a.TypeName] = append(combinedAccounts[a.TypeName], item)
	}

	// Get sorted type names for consistent ordering
	typeNames := make([]string, 0, len(combinedAccounts))
	for typeName := range combinedAccounts {
		typeNames = append(typeNames, typeName)
	}
	slices.Sort(typeNames)

	// Build the tree with combined accounts
	for _, typeName := range typeNames {
		accountList := combinedAccounts[typeName]
		// Sort accounts within each type by name
		slices.SortFunc(accountList, func(a, b accountItem) int {
			return strings.Compare(a.name, b.name)
		})

		accountTree := tree.New().Root(m.titleCaser.String(m.Styles.AssetTypeStyle.Render(typeName)))
		for _, item := range accountList {
			text := fmt.Sprintf("%s (%s)", item.name, item.amount.Display())
			accountTree.Child(m.Styles.AccountStyle.Render(text))
		}

		m.accountTree.Child(accountTree)
	}
}
