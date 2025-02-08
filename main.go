package main

import (
	"context"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/rshep3087/lunchtui/overview"
	"github.com/rshep3087/lunchtui/recurring"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	lm "github.com/icco/lunchmoney"
	"github.com/urfave/cli/v2"
	"golang.org/x/sync/errgroup"
)

var (
	// styles
	// docStyle is the style for the document
	docStyle = lipgloss.NewStyle().Margin(1, 2)
	// titleStyle is the style for the main title
	titleStyle = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#000000", Dark: "#ffd644"}).Bold(true)

	uncategorized *lm.Category = &lm.Category{ID: 0, Name: "Uncategorized", Description: "Transactions without a category"}
	keys                       = keyMap{
		transactions: key.NewBinding(
			key.WithKeys("t"),
			key.WithHelp("t", "transactions"),
		),
		overview: key.NewBinding(
			key.WithKeys("o"),
			key.WithHelp("o", "overview"),
		),
		recurring: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "recurring expenses"),
		),
		nextPeriod: key.NewBinding(
			key.WithKeys("!"),
			key.WithHelp("shift+1", "next month"),
		),
		previousPeriod: key.NewBinding(
			key.WithKeys("@"),
			key.WithHelp("shift+2", "previous month"),
		),
		switchPeriod: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "switch range"),
		),
		fullHelp: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}
)

type sessionState int

const (
	overviewState sessionState = iota
	transactions
	categorizeTransaction
	loading
	recurringExpenses
)

const (
	monthlyPeriodType = "month"
	annualPeriodType  = "year"
)

type keyMap struct {
	transactions   key.Binding
	overview       key.Binding
	recurring      key.Binding
	nextPeriod     key.Binding
	previousPeriod key.Binding
	switchPeriod   key.Binding
	fullHelp       key.Binding
	quit           key.Binding
}

func (km keyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		km.overview,
		km.transactions,
		km.recurring,
		km.switchPeriod,
		km.quit,
		km.fullHelp,
	}
}

func (km keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{
			km.overview,
			km.transactions,
			km.recurring,
			km.quit,
			km.fullHelp,
		},
		{
			km.nextPeriod,
			km.previousPeriod,
			km.switchPeriod,
		},
	}
}

type model struct {
	// loadingSpinner is a spinner model for the initial loading state
	loadingSpinner spinner.Model

	keys keyMap
	help help.Model

	overview overview.Model
	// transactionsListKeys is the keybindings for the transactions list
	transactionsListKeys *transactionListKeyMap
	// sessionState is the current state of the session
	sessionState sessionState
	// transactions is a bubbletea list model of financial transactions
	transactions  list.Model
	period        Period
	currentPeriod time.Time
	// periodType is the type of range for the transactions
	// ex. month, year
	periodType string

	transactionsStats *transactionsStats
	// debitsAsNegative is a flag to show debits as negative numbers
	debitsAsNegative bool

	categoryForm *huh.Form
	// categories is a map of category ID to category
	categories map[int64]*lm.Category
	// plaidAccounts are individual bank accounts that you have linked to Lunch Money via Plaid.
	// You may link one bank but one bank might contain 4 accounts.
	// Each of these accounts is a Plaid Account.
	plaidAccounts map[int64]*lm.PlaidAccount
	// assets are manually managed assets
	assets map[int64]*lm.Asset
	// user is the current user determined by the API token
	user *lm.User

	tags map[int]*lm.Tag

	// recurringExpenses is a model for the recurring expenses widget
	recurringExpenses recurring.Model
	// lmc is the Lunch Money client
	lmc *lm.Client

	loadingState loadingState
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		m.getCategories,
		m.getUser,
		m.getAccounts,
		m.loadingSpinner.Tick,
		m.getRecurringExpenses,
		m.recurringExpenses.Init(),
		m.getTags,
	)
}

type getRecurringExpensesMsg struct {
	recurringExpenses []*lm.RecurringExpense
}

func (m model) getRecurringExpenses() tea.Msg {
	ctx := context.Background()

	recurringExpenses, err := m.lmc.GetRecurringExpenses(ctx, nil)
	if err != nil {
		return nil
	}
	log.Debug("got recurring expenses")

	return getRecurringExpensesMsg{recurringExpenses: recurringExpenses}
}

type getAccountsMsg struct {
	plaidAccounts []*lm.PlaidAccount
	assets        []*lm.Asset
}

func (m model) getAccounts() tea.Msg {
	ctx := context.Background()

	var errGroup errgroup.Group
	var plaidAccounts []*lm.PlaidAccount
	var assets []*lm.Asset

	errGroup.Go(func() error {
		pas, err := m.lmc.GetPlaidAccounts(ctx)
		if err != nil {
			return err
		}
		plaidAccounts = pas
		return nil
	})

	errGroup.Go(func() error {
		as, err := m.lmc.GetAssets(ctx)
		if err != nil {
			return err
		}
		assets = as
		return nil
	})

	if err := errGroup.Wait(); err != nil {
		return err
	}

	return getAccountsMsg{plaidAccounts: plaidAccounts, assets: assets}
}

type getCategoriesMsg struct {
	categories []*lm.Category
}

func (m model) getCategories() tea.Msg {
	ctx := context.Background()

	cs, err := m.lmc.GetCategories(ctx)
	if err != nil {
		return nil
	}

	return getCategoriesMsg{categories: cs}
}

type getsTransactionsMsg struct {
	ts     []*lm.Transaction
	period Period
}

func (m model) getTransactions() tea.Msg {
	ctx := context.Background()

	m.period.setPeriod(m.currentPeriod, m.periodType)

	sd := m.period.startDate()
	ed := m.period.endDate()

	ts, err := m.lmc.GetTransactions(ctx, &lm.TransactionFilters{
		DebitAsNegative: &m.debitsAsNegative,
		StartDate:       &sd,
		EndDate:         &ed,
	})
	if err != nil {
		return nil
	}

	// reverse the slice so the most recent transactions are at the top
	slices.Reverse(ts)

	return getsTransactionsMsg{ts: ts, period: m.period}
}

type getUserMsg struct {
	user *lm.User
}

func (m model) getUser() tea.Msg {
	u, err := m.lmc.GetUser(context.Background())
	if err != nil {
		return nil
	}

	return getUserMsg{user: u}
}

type getTagsMsg struct {
	tags []*lm.Tag
}

func (m model) getTags() tea.Msg {
	ctx := context.Background()

	tags, err := m.lmc.GetTags(ctx)
	if err != nil {
		return nil
	}

	return getTagsMsg{tags: tags}
}

type updateTransactionMsg struct {
	t            *lm.Transaction
	fieldUpdated string
}

func (m model) updateTransactionStatus(t *lm.Transaction) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		resp, err := m.lmc.UpdateTransaction(ctx, t.ID, &lm.UpdateTransaction{Status: &t.Status})
		if err != nil {
			return err
		}

		if !resp.Updated {
			return nil
		}

		return updateTransactionMsg{t: t, fieldUpdated: "status"}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// always check for quit key first
	if msg, ok := msg.(tea.KeyMsg); ok {
		k := msg.String()
		log.Debug("key pressed", "key", k)

		if k == "q" || k == "ctrl+c" {
			return m, tea.Quit
		}

		if k == "esc" {
			m.sessionState = overviewState
			return m, nil
		}

		if k == "!" {
			m.currentPeriod = m.currentPeriod.AddDate(0, 1, 0)
			return m, m.getTransactions
		}

		if k == "@" {
			m.currentPeriod = m.currentPeriod.AddDate(0, -1, 0)
			return m, m.getTransactions
		}

		if k == "s" {
			if m.periodType == monthlyPeriodType {
				m.periodType = annualPeriodType
			} else {
				m.periodType = monthlyPeriodType
			}
			return m, m.getTransactions
		}

		if m.sessionState == loading {
			return m, nil
		}

		if k == "t" && m.sessionState != transactions {
			m.sessionState = transactions
			return m, nil
		}

		if k == "r" && m.sessionState != recurringExpenses {
			m.recurringExpenses.SetFocus(true)
			m.sessionState = recurringExpenses
			return m, nil
		}

		if k == "o" && m.sessionState != overviewState {
			m.sessionState = overviewState
			return m, nil
		}

		if k == "?" && m.sessionState != transactions {
			m.help.ShowAll = !m.help.ShowAll
			return m, nil
		}
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.handleWindowSize(msg)

	case spinner.TickMsg:
		return m.handleSpinnerTick(msg)

	// set the categories on the model,
	// send a cmd to get transactions
	case getCategoriesMsg:
		return m.handleGetCategories(msg)

	case getAccountsMsg:
		return m.handleGetAccounts(msg)

	case getsTransactionsMsg:
		return m.handleGetTransactions(msg)

	case getUserMsg:
		return m.handleGetUser(msg)

	case getRecurringExpensesMsg:
		m.recurringExpenses.SetRecurringExpenses(msg.recurringExpenses)
		return m, nil

	case getTagsMsg:
		return m.handleGetTags(msg)

	}

	var cmd tea.Cmd
	switch m.sessionState {
	case overviewState:
		m.overview, cmd = m.overview.Update(msg)
		return m, cmd

	case categorizeTransaction:
		return updateCategorizeTransaction(msg, &m)

	case transactions:
		return updateTransactions(msg, m)

	case recurringExpenses:
		m.recurringExpenses, cmd = m.recurringExpenses.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m model) handleGetTags(msg getTagsMsg) (tea.Model, tea.Cmd) {
	tags := make(map[int]*lm.Tag, len(msg.tags))
	for _, t := range msg.tags {
		tags[t.ID] = t
	}
	m.tags = tags
	m.loadingState.set("tags")
	m.sessionState = m.checkIfLoading()
	return m, nil
}

func (m model) handleWindowSize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	h, v := docStyle.GetFrameSize()

	m.overview.SetSize(msg.Width-h, msg.Height-v-5)
	m.overview.Viewport.Width = msg.Width
	m.overview.Viewport.Height = msg.Height - 5

	m.transactions.SetSize(msg.Width-h, msg.Height-v-5)
	m.recurringExpenses.SetSize(msg.Width-h, msg.Height-v-3)

	m.help.Width = msg.Width

	if m.categoryForm != nil {
		m.categoryForm = m.categoryForm.WithHeight(msg.Height - 5).WithWidth(msg.Width)
	}

	return m, nil
}

func (m model) handleGetUser(msg getUserMsg) (tea.Model, tea.Cmd) {
	m.user = msg.user
	m.loadingState.set("user")
	m.sessionState = m.checkIfLoading()
	m.overview.SetCurrency(m.user.PrimaryCurrency)
	return m, nil
}

func (m model) handleGetTransactions(msg getsTransactionsMsg) (tea.Model, tea.Cmd) {
	var items = make([]list.Item, len(msg.ts))
	for i, t := range msg.ts {
		items[i] = transactionItem{
			t:            t,
			category:     m.categories[t.CategoryID],
			plaidAccount: m.plaidAccounts[t.PlaidAccountID],
			asset:        m.assets[t.AssetID],
		}
	}

	cmd := m.transactions.SetItems(items)

	m.transactionsStats = newTransactionStats(items)
	m.overview.SetTransactions(msg.ts)
	m.period = msg.period

	m.loadingState.set("transactions")
	m.sessionState = m.checkIfLoading()

	return m, cmd
}

func (m model) handleGetAccounts(msg getAccountsMsg) (tea.Model, tea.Cmd) {
	m.plaidAccounts = make(map[int64]*lm.PlaidAccount, len(msg.plaidAccounts))
	for _, pa := range msg.plaidAccounts {
		m.plaidAccounts[pa.ID] = pa
	}

	m.assets = make(map[int64]*lm.Asset, len(msg.assets))
	for _, a := range msg.assets {
		m.assets[a.ID] = a
	}

	m.overview.SetAccounts(m.assets, m.plaidAccounts)

	m.loadingState.set("accounts")
	m.sessionState = m.checkIfLoading()

	return m, nil
}

func (m model) handleGetCategories(msg getCategoriesMsg) (tea.Model, tea.Cmd) {
	m.categories = make(map[int64]*lm.Category, len(msg.categories)+1)
	// set the uncategorized category which does not come from the API
	m.categories[uncategorized.ID] = uncategorized

	for _, c := range msg.categories {
		m.categories[c.ID] = c
	}

	m.categoryForm = newCategorizeTransactionForm(msg.categories)
	m.overview.SetCategories(m.categories)

	m.loadingState.set("categories")
	m.sessionState = m.checkIfLoading()

	return m, tea.Batch(m.getTransactions, m.categoryForm.Init(), tea.WindowSize())
}

func (m model) handleSpinnerTick(msg spinner.TickMsg) (tea.Model, tea.Cmd) {
	if m.sessionState != loading {
		return m, nil
	}

	var cmd tea.Cmd
	m.loadingSpinner, cmd = m.loadingSpinner.Update(msg)
	return m, cmd
}

func (m model) View() string {
	var b strings.Builder

	b.WriteString(m.renderTitle())

	b.WriteString("\n\n")

	switch m.sessionState {
	case overviewState:
		b.WriteString(m.overview.View())
	case transactions:
		b.WriteString(transactionsView(m))
	case categorizeTransaction:
		b.WriteString(categorizeTransactionView(m))
	case recurringExpenses:
		b.WriteString(m.recurringExpenses.View())
	case loading:
		b.WriteString(fmt.Sprintf("%s Loading data...", m.loadingSpinner.View()))
		return docStyle.Render(b.String())
	}

	b.WriteString("\n\n")
	b.WriteString(m.help.View(m.keys))

	return docStyle.Render(b.String())
}

func (m model) renderTitle() string {
	var b strings.Builder

	var currentPage string
	switch m.sessionState {
	case overviewState:
		currentPage = "overview"
	case transactions:
		currentPage = "transactions"
	case categorizeTransaction:
		currentPage = "categorize transaction"
	case recurringExpenses:
		currentPage = "recurring expenses"
	case loading:
		currentPage = "loading"
	}

	if m.period.String() == "" {
		b.WriteString(titleStyle.Render(fmt.Sprintf("lunchtui | %s", currentPage)))
	} else {
		b.WriteString(titleStyle.Render(fmt.Sprintf("lunchtui | %s | %s | %s", currentPage, m.period.String(), m.periodType)))
	}

	return b.String()
}

type Period struct {
	start time.Time
	end   time.Time
}

func (p Period) String() string {
	return fmt.Sprintf("%s - %s", p.start.Format("2006-01-02"), p.end.Format("2006-01-02"))
}

func (p Period) startDate() string {
	return p.start.Format("2006-01-02")
}

func (p Period) endDate() string {
	return p.end.Format("2006-01-02")
}

func (p *Period) setPeriod(current time.Time, periodType string) {
	switch periodType {
	case monthlyPeriodType:
		p.start = time.Date(current.Year(), current.Month(), 1, 0, 0, 0, 0, current.Location())
		p.end = time.Date(current.Year(), current.Month()+1, 1, 0, 0, 0, 0, current.Location()).Add(-time.Second)
	case annualPeriodType:
		p.start = time.Date(current.Year(), 1, 1, 0, 0, 0, 0, current.Location())
		p.end = time.Date(current.Year()+1, 1, 1, 0, 0, 0, 0, current.Location()).Add(-time.Second)
	default:
		// default to month
		p.start = time.Date(current.Year(), current.Month(), 1, 0, 0, 0, 0, current.Location())
		p.end = time.Date(current.Year(), current.Month()+1, 1, 0, 0, 0, 0, current.Location()).Add(-time.Second)
	}
}

func main() {
	app := &cli.App{
		Name:  "lunchtui",
		Usage: "A terminal UI for Lunch Money",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "token",
				Usage:    "The API token for Lunch Money",
				EnvVars:  []string{"LUNCHMONEY_API_TOKEN"},
				Required: true,
			},
			// debits-as-negative flag
			&cli.BoolFlag{
				Name:  "debits-as-negative",
				Usage: "Show debits as negative numbers",
			},
			&cli.BoolFlag{
				Name:  "debug",
				Usage: "Enable debug logging",
				Value: false,
			},
		},
		Action: func(c *cli.Context) error {
			if c.Bool("debug") {
				f, err := tea.LogToFileWith("lunchtui.log", "lunchtui", log.Default())
				if err != nil {
					return err
				}
				defer f.Close()

				log.SetLevel(log.DebugLevel)
			}

			lmc, err := lm.NewClient(c.String("token"))
			if err != nil {
				return err
			}

			helpModel := help.New()
			helpModel.ShortSeparator = " + "
			helpModel.Styles = help.Styles{
				Ellipsis:       lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")),
				ShortKey:       lipgloss.NewStyle().Foreground(lipgloss.Color("#ffd644")).Bold(true),
				ShortDesc:      lipgloss.NewStyle().Foreground(lipgloss.Color("#ffffff")),
				ShortSeparator: lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")),
				FullKey:        lipgloss.NewStyle().Foreground(lipgloss.Color("#ffd644")).Bold(true),
				FullDesc:       lipgloss.NewStyle().Foreground(lipgloss.Color("#ffffff")),
				FullSeparator:  lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")),
			}

			tlKeyMap := newTransactionListKeyMap()
			m := model{
				keys:                 keys,
				help:                 helpModel,
				sessionState:         loading,
				lmc:                  lmc,
				transactionsListKeys: tlKeyMap,
				debitsAsNegative:     c.Bool("debits-as-negative"),
				currentPeriod:        time.Now(),
				period:               Period{},
				periodType:           "month",
				loadingSpinner: spinner.New(
					spinner.WithSpinner(spinner.Dot),
				),
				overview:          overview.New(),
				recurringExpenses: recurring.New(),
				loadingState: newLoadingState(
					"categories",
					"transactions",
					"user",
					"accounts",
					"tags",
				),
			}

			delegate := m.newItemDelegate(newDeleteKeyMap())

			transactionList := list.New([]list.Item{}, delegate, 0, 0)
			transactionList.SetShowTitle(false)
			transactionList.StatusMessageLifetime = 3 * time.Second
			transactionList.AdditionalFullHelpKeys = func() []key.Binding {
				return []key.Binding{
					tlKeyMap.categorizeTransaction,
				}
			}
			m.transactions = transactionList

			p := tea.NewProgram(m, tea.WithAltScreen())
			if _, err := p.Run(); err != nil {
				return err
			}

			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Printf("lunchtui ran into an error: %v", err)
		os.Exit(1)
	}
}

func (m model) checkIfLoading() sessionState {
	if m.sessionState != loading {
		return m.sessionState
	}

	if loaded, notLoaded := m.loadingState.allLoaded(); !loaded {
		log.Debugf("not loaded: %s", notLoaded)
		return loading
	}

	log.Debug("everything is loaded, showing overview")
	return overviewState
}
