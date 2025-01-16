package main

import (
	"context"
	"fmt"
	"os"
	"slices"
	"time"

	"github.com/Rhymond/go-money"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	lm "github.com/rshep3087/lunchmoney"
	"github.com/urfave/cli/v2"
	"golang.org/x/sync/errgroup"
)

var docStyle = lipgloss.NewStyle().Margin(1, 2)
var uncategorized *lm.Category = &lm.Category{ID: 0, Name: "Uncategorized", Description: "Transactions without a category"}

type sessionState int

const (
	overview sessionState = iota
	transactions
	categorizeTransaction
)

type summary struct {
	totalIncomeEarned *money.Money
	totalSpent        *money.Money
	netIncome         *money.Money
}

func (m model) newSummary() *summary {
	var totalIncomeEarned, totalSpent = money.New(0, "USD"), money.New(0, "USD")

	for _, t := range m.transactions.Items() {
		t := t.(transactionItem).t
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

	return &summary{
		totalIncomeEarned: totalIncomeEarned,
		totalSpent:        totalSpent,
		netIncome:         netIncome,
	}
}

type model struct {
	summary *summary
	// transactionsListKeys is the keybindings for the transactions list
	transactionsListKeys *transactionListKeyMap
	// sessionState is the current state of the session
	sessionState sessionState
	// transactions is a bubbletea list model of financial transactions
	transactions list.Model
	// debitsAsNegative is a flag to show debits as negative numbers
	debitsAsNegative bool
	// categiorizeTransactions is a bubbletea list model of categories
	categorizeTransactions list.Model
	// categories is a map of category ID to category
	categories map[int]*lm.Category
	// plaidAccounts are individual bank accounts that you have linked to Lunch Money via Plaid.
	// You may link one bank but one bank might contain 4 accounts.
	// Each of these accounts is a Plaid Account.
	plaidAccounts []*lm.PlaidAccount
	// assets are manually managed assets
	assets []*lm.Asset
	// user is the current user
	user *lm.User
	// lmc is the Lunch Money client
	lmc *lm.Client
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.getCategories, m.getUser, m.getAccounts)
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
		return nil
	}

	return getUserMsg{user: u}
}

type updateTransactionStatusMsg struct {
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

		return updateTransactionStatusMsg{t: t, fieldUpdated: "status"}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// always check for quit key first
	if msg, ok := msg.(tea.KeyMsg); ok {
		k := msg.String()
		if (k == "q" || k == "ctrl+c") && m.sessionState == overview {
			return m, tea.Quit
		}
	}

	switch msg := msg.(type) {
	// set the categories on the model,
	// send a cmd to get transactions
	case getCategoriesMsg:
		m.categories = make(map[int]*lm.Category, len(msg.categories)+1)
		// set the uncategorized category which does not come from the API
		m.categories[uncategorized.ID] = uncategorized

		var categoryItems = make([]list.Item, 0, len(msg.categories))
		for _, c := range msg.categories {
			m.categories[c.ID] = c
			categoryItems = append(categoryItems, categoryItem{c: c})
		}

		setItemsCmd := m.categorizeTransactions.SetItems(categoryItems)

		return m, tea.Batch(setItemsCmd, m.getTransactions)

	case getAccountsMsg:
		m.plaidAccounts = msg.plaidAccounts
		m.assets = msg.assets

		return m, nil

	case transactionsResp:
		var items = make([]list.Item, len(msg.ts))
		for i, t := range msg.ts {
			items[i] = transactionItem{
				t:        t,
				category: m.categories[int(t.CategoryID)],
			}
		}

		cmd := m.transactions.SetItems(items)

		m.summary = m.newSummary()

		return m, cmd

	case getUserMsg:
		m.user = msg.user
	}

	if m.sessionState == overview {
		return updateOverview(msg, m)
	} else if m.sessionState == categorizeTransaction {
		return updateCategorizeTransaction(msg, &m)
	} else if m.sessionState == transactions {
		return updateTransactions(msg, m)
	} else {
		return m, nil
	}
}

func (m model) View() string {
	var s string

	if m.sessionState == overview {
		s = overviewView(m)
	} else if m.sessionState == transactions {
		s = transactionsView(m)
	} else if m.sessionState == categorizeTransaction {
		s = categorizeTransactionView(m)
	}

	return docStyle.Render(s)
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

			tlKeyMap := newTransactionListKeyMap()
			m := model{
				sessionState:         overview,
				lmc:                  lmc,
				transactionsListKeys: tlKeyMap,
				debitsAsNegative:     c.Bool("debits-as-negative"),
			}

			delegate := m.newItemDelegate(newDeleteKeyMap())

			transactionList := list.New([]list.Item{}, delegate, 0, 0)
			transactionList.Title = "Transactions"
			transactionList.StatusMessageLifetime = 3 * time.Second
			transactionList.AdditionalFullHelpKeys = func() []key.Binding {
				return []key.Binding{
					tlKeyMap.overview,
				}
			}
			m.transactions = transactionList

			m.categorizeTransactions = newCategorizeTransactionModel()

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
