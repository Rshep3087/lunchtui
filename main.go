package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/rshep3087/lunchtui/overview"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	lm "github.com/rshep3087/lunchmoney"
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
)

type keyMap struct {
	transactions key.Binding
	overview     key.Binding
	quit         key.Binding
}

func (km keyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		km.transactions,
		km.overview,
		km.quit,
	}
}

func (km keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{
			km.transactions,
			km.overview,
			km.quit,
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
	transactions list.Model

	transactionsStats *transactionsStats
	// debitsAsNegative is a flag to show debits as negative numbers
	debitsAsNegative bool

	categoryForm *huh.Form
	// categories is a map of category ID to category
	categories map[int]*lm.Category
	// plaidAccounts are individual bank accounts that you have linked to Lunch Money via Plaid.
	// You may link one bank but one bank might contain 4 accounts.
	// Each of these accounts is a Plaid Account.
	plaidAccounts map[int64]*lm.PlaidAccount
	// assets are manually managed assets
	assets map[int64]*lm.Asset
	// user is the current user determined by the API token
	user *lm.User
	// lmc is the Lunch Money client
	lmc *lm.Client
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		m.getCategories,
		m.getUser,
		m.getAccounts,
		m.loadingSpinner.Tick,
	)
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
		log.Println(err)
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
		log.Println(err)
		return nil
	}

	return getCategoriesMsg{categories: cs}
}

type transactionsResp struct {
	ts []*lm.Transaction
}

func (m model) getTransactions() tea.Msg {
	ctx := context.Background()

	now := time.Now()
	nowFormatted := now.Format("2006-01-02")
	firstOfTheMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02")

	ts, err := m.lmc.GetTransactions(ctx, &lm.TransactionFilters{
		DebitAsNegative: &m.debitsAsNegative,
		StartDate:       &firstOfTheMonth,
		EndDate:         &nowFormatted,
	})
	if err != nil {
		log.Println(err)
		return err
	}

	// reverse the slice so the most recent transactions are at the top
	slices.Reverse(ts)

	return transactionsResp{ts: ts}
}

type getUserMsg struct {
	user *lm.User
}

func (m model) getUser() tea.Msg {
	u, err := m.lmc.GetUser(context.Background())
	if err != nil {
		log.Println(err)
		return nil
	}

	return getUserMsg{user: u}
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
		if (k == "q" || k == "ctrl+c") && m.sessionState == overviewState {
			return m, tea.Quit
		}

		if k == "t" {
			m.sessionState = transactions
			return m, nil
		}
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.overview.SetSize(msg.Width-h, msg.Height-v-5)
		m.transactions.SetSize(msg.Width-h, msg.Height-v-5)
		m.help.Width = msg.Width
		m.overview.Viewport.Width = msg.Width
		m.overview.Viewport.Height = msg.Height - 5

		if m.categoryForm != nil {
			m.categoryForm = m.categoryForm.WithHeight(msg.Height - 5).WithWidth(msg.Width)
		}

	case spinner.TickMsg:
		if m.sessionState != loading {
			return m, nil
		}

		var cmd tea.Cmd
		m.loadingSpinner, cmd = m.loadingSpinner.Update(msg)
		return m, cmd

	// set the categories on the model,
	// send a cmd to get transactions
	case getCategoriesMsg:
		m.categories = make(map[int]*lm.Category, len(msg.categories)+1)
		// set the uncategorized category which does not come from the API
		m.categories[uncategorized.ID] = uncategorized

		for _, c := range msg.categories {
			m.categories[c.ID] = c
		}

		m.categoryForm = newCategorizeTransactionForm(msg.categories)
		m.overview.SetCategories(m.categories)

		m.sessionState = m.checkIfLoading()

		return m, tea.Batch(m.getTransactions, m.categoryForm.Init(), tea.WindowSize())

	case getAccountsMsg:
		m.plaidAccounts = make(map[int64]*lm.PlaidAccount, len(msg.plaidAccounts))
		for _, pa := range msg.plaidAccounts {
			m.plaidAccounts[pa.ID] = pa
		}

		m.assets = make(map[int64]*lm.Asset, len(msg.assets))
		for _, a := range msg.assets {
			m.assets[a.ID] = a
		}

		m.overview.SetAccounts(m.assets, m.plaidAccounts)

		m.sessionState = m.checkIfLoading()

		return m, nil

	case transactionsResp:
		var items = make([]list.Item, len(msg.ts))
		for i, t := range msg.ts {
			items[i] = transactionItem{
				t:            t,
				category:     m.categories[int(t.CategoryID)],
				plaidAccount: m.plaidAccounts[t.PlaidAccountID],
				asset:        m.assets[t.AssetID],
			}
		}

		cmd := m.transactions.SetItems(items)

		m.transactionsStats = newTransactionStats(items)
		m.overview.SetTransactions(msg.ts)

		m.sessionState = m.checkIfLoading()

		return m, cmd

	case getUserMsg:
		m.user = msg.user
		m.sessionState = m.checkIfLoading()
		return m, nil
	}

	if m.sessionState == overviewState {
		var cmd tea.Cmd
		m.overview, cmd = m.overview.Update(msg)
		return m, cmd
	} else if m.sessionState == categorizeTransaction {
		return updateCategorizeTransaction(msg, &m)
	} else if m.sessionState == transactions {
		return updateTransactions(msg, m)
	} else {
		return m, nil
	}
}

func (m model) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("lunchtui"))
	b.WriteString("\n\n")

	switch m.sessionState {
	case overviewState:
		b.WriteString(m.overview.View())
	case transactions:
		b.WriteString(transactionsView(m))
	case categorizeTransaction:
		b.WriteString(categorizeTransactionView(m))
	case loading:
		b.WriteString(fmt.Sprintf("%s Loading data...", m.loadingSpinner.View()))
		return docStyle.Render(b.String())
	}

	b.WriteString("\n\n")
	b.WriteString(m.help.View(m.keys))

	return docStyle.Render(b.String())
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
				f, err := tea.LogToFile("lunchtui.log", "lunchtui")
				if err != nil {
					return err
				}
				defer f.Close()
			}

			lmc, err := lm.NewClient(c.String("token"))
			if err != nil {
				return err
			}

			helpModel := help.New()
			helpModel.ShortSeparator = " + "
			helpModel.Styles = help.Styles{
				Ellipsis:       lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")),
				ShortKey:       lipgloss.NewStyle().Foreground(lipgloss.Color("#00ff00")).Bold(true),
				ShortDesc:      lipgloss.NewStyle().Foreground(lipgloss.Color("#ffffff")),
				ShortSeparator: lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")),
				FullKey:        lipgloss.NewStyle().Foreground(lipgloss.Color("#00ff00")).Bold(true),
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
				loadingSpinner: spinner.New(
					spinner.WithSpinner(spinner.Dot),
				),
				overview: overview.New(),
			}

			delegate := m.newItemDelegate(newDeleteKeyMap())

			transactionList := list.New([]list.Item{}, delegate, 0, 0)
			transactionList.SetShowTitle(false)
			transactionList.StatusMessageLifetime = 3 * time.Second
			transactionList.AdditionalFullHelpKeys = func() []key.Binding {
				return []key.Binding{
					tlKeyMap.overview,
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
	if m.user == nil || m.categories == nil || m.plaidAccounts == nil || m.assets == nil {
		return loading
	}

	if len(m.transactions.Items()) == 0 {
		return loading
	}

	return overviewState
}
