package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"slices"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	lm "github.com/rshep3087/lunchmoney"
	"github.com/urfave/cli/v2"
)

var docStyle = lipgloss.NewStyle().Margin(1, 2)

type sessionState int

const (
	overview sessionState = iota
	transactions
)

type model struct {
	// transactionsListKeys is the keybindings for the transactions list
	transactionsListKeys *transactionListKeyMap
	// sessionState is the current state of the session
	sessionState sessionState
	// transactions is a bubbletea list model of financial transactions
	transactions list.Model
	// categories is a map of category ID to category
	categories map[int64]*lm.Category
	// user is the current user
	user *lm.User
	// lmc is the Lunch Money client
	lmc *lm.Client
}

type transactionsResp struct {
	ts []*lm.Transaction
}

type transactionItem struct {
	t        *lm.Transaction
	category *lm.Category
}

func (t transactionItem) Title() string {
	return t.t.Payee
}

func (t transactionItem) Description() string {
	amount, err := t.t.ParsedAmount()
	if err != nil {
		log.Printf("error parsing amount: %v", err)
		return fmt.Sprintf("error parsing amount: %v", err)
	}

	return fmt.Sprintf("%s %s %s %s", t.t.Date, t.category.Name, amount.Display(), t.t.Status)
}

func (t transactionItem) FilterValue() string {
	return t.t.Payee
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.getCategories, m.getUser)
}

type getCategoriesMsg struct {
	categories []*lm.Category
}

func (m model) getCategories() tea.Msg {
	log.Println("getting categories")
	ctx := context.Background()

	cs, err := m.lmc.GetCategories(ctx)
	if err != nil {
		log.Printf("error getting categories: %v", err)
		return nil
	}
	log.Printf("got %d categories", len(cs))

	return getCategoriesMsg{categories: cs}
}

func (m model) getTransactions() tea.Msg {
	log.Println("getting transactions")
	ctx := context.Background()

	ts, err := m.lmc.GetTransactions(ctx, nil)
	if err != nil {
		log.Printf("error getting transactions: %v", err)
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
	log.Println("getting user")
	ctx := context.Background()

	u, err := m.lmc.GetUser(ctx)
	if err != nil {
		log.Printf("error getting user: %v", err)
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
		log.Printf("clearing transaction for id: %d", t.ID)
		ctx := context.Background()

		resp, err := m.lmc.UpdateTransaction(ctx, t.ID, &lm.UpdateTransaction{Status: &t.Status})
		if err != nil {
			log.Printf("error clearing transaction: %v", err)
			return err
		}

		if !resp.Updated {
			log.Printf("transaction not updated")
			return nil
		}

		return updateTransactionStatusMsg{t: t, fieldUpdated: "status"}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// always check for quit key first
	if msg, ok := msg.(tea.KeyMsg); ok {
		k := msg.String()
		if k == "q" || k == "ctrl+c" {
			return m, tea.Quit
		}
	}

	switch msg := msg.(type) {
	// set the categories on the model,
	// send a cmd to get transactions
	case getCategoriesMsg:
		m.categories = make(map[int64]*lm.Category, len(msg.categories))
		for _, c := range msg.categories {
			m.categories[c.ID] = c
		}

		return m, m.getTransactions

	case transactionsResp:
		log.Printf("got %d transactions", len(msg.ts))

		var items = make([]list.Item, len(msg.ts))
		for i, t := range msg.ts {
			items[i] = transactionItem{
				t:        t,
				category: m.categories[t.CategoryID],
			}
		}
		cmd := m.transactions.SetItems(items)
		return m, cmd

	case getUserMsg:
		m.user = msg.user
	}

	if m.sessionState == overview {
		return updateOverview(msg, m)
	}

	return updateTransactions(msg, m)
}

func updateOverview(msg tea.Msg, m model) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		k := msg.String()
		if k == "t" {
			m.sessionState = transactions
			return m, tea.WindowSize()
		}
	}

	return m, nil
}

func updateTransactions(msg tea.Msg, m model) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.transactions.SetSize(msg.Width-h, msg.Height-v)
		return m, nil

	case updateTransactionStatusMsg:
		setItemCmd := m.transactions.SetItem(m.transactions.Index(), transactionItem{t: msg.t, category: m.categories[msg.t.CategoryID]})
		statusCmd := m.transactions.NewStatusMessage(fmt.Sprintf("Updated %s for transaction: %s", msg.fieldUpdated, msg.t.Payee))
		return m, tea.Batch(setItemCmd, statusCmd)

	case tea.KeyMsg:
		// if the list is filtering, don't process key events
		if m.transactions.FilterState() == list.Filtering {
			break
		}

		if key.Matches(msg, m.transactionsListKeys.overview) {
			m.sessionState = overview
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.transactions, cmd = m.transactions.Update(msg)

	return m, cmd
}

func (m model) View() string {
	var s string
	if m.sessionState == overview {
		s = overviewView(m)
	} else {
		s = transactionsView(m)
	}

	return docStyle.Render("\n" + s + "\n\n")
}

func overviewView(m model) string {
	if m.user == nil {
		return "Loading..."
	}

	msg := fmt.Sprintf("Welcome %s!", m.user.UserName)
	return lipgloss.NewStyle().Width(80).Render(msg)
}

func transactionsView(m model) string {
	return m.transactions.View()
}

type transactionListKeyMap struct {
	overview key.Binding
}

func newTransactionListKeyMap() *transactionListKeyMap {
	return &transactionListKeyMap{
		overview: key.NewBinding(
			key.WithKeys("o"),
			key.WithHelp("o", "overview"),
		),
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
			}

			delegate := m.newItemDelegate(newDeleteKeyMap())

			transactionList := list.New([]list.Item{}, delegate, 0, 0)
			transactionList.Title = "Transactions"
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
