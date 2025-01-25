package overview

import (
	"fmt"
	"log"

	"github.com/Rhymond/go-money"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
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
	KeyMap        KeyMap
	Help          help.Model
	Styles        Styles
	viewport      viewport.Model
	summary       Summary
	transactions  []*lm.Transaction
	categories    map[int]*lm.Category
	assets        map[int64]*lm.Asset
	plaidAccounts map[int64]*lm.PlaidAccount
	accountTree   *tree.Tree
	user          *lm.User
}

type Summary struct {
	totalIncomeEarned money.Money
	totalSpent        money.Money
	netIncome         money.Money
}

type KeyMap struct {
	Quit key.Binding
}

type Styles struct {
	IncomeStyle lipgloss.Style
	SpentStyle  lipgloss.Style
}

func defaultStyles() Styles {
	return Styles{
		IncomeStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("#00ff00")),
		SpentStyle:  lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000")),
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
		KeyMap:   defaultKeyMap(),
		Help:     help.New(),
		Styles:   defaultStyles(),
		viewport: viewport.New(0, 20),
		summary: Summary{
			// setting them to 0 so that the currency is set,
			// otherwise it's nil and blows up
			totalIncomeEarned: *money.New(0, "USD"),
			totalSpent:        *money.New(0, "USD"),
			netIncome:         *money.New(0, "USD"),
		},
		accountTree: tree.New().Root("Accounts"),
	}

	for _, opt := range opts {
		opt(&m)
	}

	m.UpdateViewport()

	return m
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	return m, nil
}

func (m Model) View() string {
	var (
		sections []string
	)

	sections = append(sections, m.viewport.View())
	sections = append(sections, m.helpView())
	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (m Model) helpView() string {
	return m.Help.View(m.KeyMap)

}

func (m *Model) SetSize(width, height int) {
	m.setSize(width, height)
}

func (m *Model) setSize(width, height int) {
	m.viewport.Width = width
	m.viewport.Height = height
	m.Help.Width = width
}

func (m *Model) UpdateViewport() {
	m.viewport.SetContent(
		lipgloss.JoinVertical(lipgloss.Top,
			m.headerView(),
			m.summaryView(),
			m.accountTree.String(),
		),
	)
}

func (m *Model) headerView() string {
	if m.user == nil {
		return "Overview"
	}

	return fmt.Sprintf("Welcome - %s!", m.user.UserName)
}

func defaultKeyMap() KeyMap {
	return KeyMap{
		Quit: key.NewBinding(
			key.WithKeys("q", "esc"),
			key.WithHelp("q", "quit"),
		),
	}
}

func (km KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		km.Quit,
	}
}

func (km KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{km.Quit},
	}
}

func (m Model) summaryView() string {
	var msg string

	msg += fmt.Sprintf("Income: %s\n", m.Styles.IncomeStyle.Render(m.summary.totalIncomeEarned.Display()))
	msg += fmt.Sprintf("Spent: %s\n", m.Styles.SpentStyle.Render(m.summary.totalSpent.Display()))
	if m.summary.netIncome.IsNegative() {
		msg += fmt.Sprintf("Net Income: %s\n", m.Styles.SpentStyle.Render(m.summary.netIncome.Display()))
	} else {
		msg += fmt.Sprintf("Net Income: %s\n", m.Styles.IncomeStyle.Render(m.summary.netIncome.Display()))
	}

	return msg
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
		assetTree := tree.New().Root(titleCaser.String(typeName))
		for _, a := range assets {
			m, err := a.ParsedAmount()
			if err != nil {
				log.Printf("error parsing amount: %v", err)
				continue
			}

			assetTree.Child(fmt.Sprintf("%s (%s)", a.Name, m.Display()))
		}

		m.accountTree.Child(assetTree)
	}

	// // organize the plaid accounts by the type into a map
	plaidAccounts := make(map[string][]lm.PlaidAccount)
	for _, a := range m.plaidAccounts {
		plaidAccounts[a.Type] = append(plaidAccounts[a.Type], *a)
	}

	for typeName, accounts := range plaidAccounts {
		accountTree := tree.New().Root(titleCaser.String(typeName))
		for _, a := range accounts {
			m, err := a.ParsedAmount()
			if err != nil {
				log.Printf("error parsing amount: %v", err)
				continue
			}

			accountTree.Child(fmt.Sprintf("%s (%s)", a.Name, m.Display()))
		}

		m.accountTree.Child(accountTree)
	}
}
