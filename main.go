package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"slices"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	lm "github.com/rshep3087/lunchmoney"
	"github.com/urfave/cli/v2"
)

var docStyle = lipgloss.NewStyle().Margin(1, 2)

type model struct {
	ts list.Model

	lmc *lm.Client
}

type transactionsResp struct {
	ts []*lm.Transaction
}

type transactionItem struct {
	t *lm.Transaction
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

	return fmt.Sprintf("%s %s %s", t.t.Date, amount.Display(), t.t.Status)
}

func (t transactionItem) FilterValue() string {
	return t.t.Payee
}

func (m model) Init() tea.Cmd {
	return m.getTransactions
}

func (m model) getTransactions() tea.Msg {
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

func (m model) updateTransactionStatus(listModel *list.Model, t *lm.Transaction) tea.Cmd {
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

		return listModel.SetItem(listModel.Index(), transactionItem{t: t})
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.ts.SetSize(msg.Width-h, msg.Height-v)

	case transactionsResp:
		log.Printf("got %d transactions", len(msg.ts))

		var items = make([]list.Item, len(msg.ts))
		for i, t := range msg.ts {
			items[i] = transactionItem{t: t}
		}
		cmd := m.ts.SetItems(items)
		return m, cmd
	}

	var cmd tea.Cmd
	m.ts, cmd = m.ts.Update(msg)

	return m, cmd
}

func (m model) View() string {
	return docStyle.Render(m.ts.View())
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

			model := model{lmc: lmc}

			delegate := model.newItemDelegate(newDeleteKeyMap())
			transactionList := list.New([]list.Item{}, delegate, 0, 0)
			transactionList.Title = "Transactions"

			model.ts = transactionList
			p := tea.NewProgram(model, tea.WithAltScreen())
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
