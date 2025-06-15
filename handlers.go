package main

import (
	"context"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/log"
	lm "github.com/icco/lunchmoney"
	"golang.org/x/sync/errgroup"
)

// Message types for different API responses.
type (
	getRecurringExpensesMsg struct {
		recurringExpenses []*lm.RecurringExpense
	}

	getAccountsMsg struct {
		plaidAccounts []*lm.PlaidAccount
		assets        []*lm.Asset
	}

	getCategoriesMsg struct {
		categories []*lm.Category
	}

	getsTransactionsMsg struct {
		ts     []*lm.Transaction
		period Period
	}

	getUserMsg struct {
		user *lm.User
	}

	getTagsMsg struct {
		tags []*lm.Tag
	}

	updateTransactionMsg struct {
		t            *lm.Transaction
		fieldUpdated string
	}

	getBudgetsMsg struct {
		budgets []*lm.Budget
		period  Period
	}
)

// Message handlers.
func (m model) handleWindowSize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	h, v := m.styles.docStyle.GetFrameSize()

	takenHeight := 5
	m.overview.SetSize(msg.Width-h, msg.Height-v-takenHeight)
	m.overview.Viewport.Width = msg.Width
	m.overview.Viewport.Height = msg.Height - takenHeight

	m.transactions.SetSize(msg.Width-h, msg.Height-v-takenHeight)
	m.budgets.SetSize(msg.Width-h, msg.Height-v-takenHeight)
	m.recurringExpenses.SetSize(msg.Width-h, msg.Height-v-3)

	m.help.Width = msg.Width

	if m.categoryForm != nil {
		m.categoryForm = m.categoryForm.WithHeight(msg.Height - 5).WithWidth(msg.Width)
	}

	return m, nil
}

func (m model) handleSpinnerTick(msg spinner.TickMsg) (tea.Model, tea.Cmd) {
	if m.sessionState != loading {
		return m, nil
	}

	var cmd tea.Cmd
	m.loadingSpinner, cmd = m.loadingSpinner.Update(msg)
	return m, cmd
}

func (m model) handleGetCategories(msg getCategoriesMsg) (tea.Model, tea.Cmd) {
	m.idToCategory = make(map[int64]*lm.Category, len(msg.categories)+1)
	// set the uncategorized category which does not come from the API
	m.idToCategory[0] = &lm.Category{
		ID:          0,
		Name:        "Uncategorized",
		Description: "Transactions without a category",
	}

	for _, c := range msg.categories {
		m.idToCategory[c.ID] = c
	}

	m.categories = msg.categories
	m.overview.SetCategories(m.idToCategory)
	m.loadingState.set("categories")
	m.sessionState = m.checkIfLoading()

	return m, tea.Batch(m.getTransactions, tea.WindowSize())
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

func (m model) handleGetTransactions(msg getsTransactionsMsg) (tea.Model, tea.Cmd) {
	items := make([]list.Item, len(msg.ts))
	for i, t := range msg.ts {
		items[i] = transactionItem{
			t:            t,
			category:     m.idToCategory[t.CategoryID],
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

func (m model) handleGetUser(msg getUserMsg) (tea.Model, tea.Cmd) {
	m.user = msg.user
	m.loadingState.set("user")
	m.sessionState = m.checkIfLoading()
	m.overview.SetCurrency(m.user.PrimaryCurrency)
	m.overview.SetUser(m.user)
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

func (m model) handleGetBudgets(msg getBudgetsMsg) (tea.Model, tea.Cmd) {
	items := make([]list.Item, len(msg.budgets))
	for i, b := range msg.budgets {
		items[i] = budgetItem{
			b:        b,
			category: m.idToCategory[int64(b.CategoryID)],
		}
	}

	cmd := m.budgets.SetItems(items)
	m.period = msg.period

	m.loadingState.set("budgets")
	m.sessionState = m.checkIfLoading()

	return m, cmd
}

// API call functions.
func (m model) getRecurringExpenses() tea.Msg {
	ctx := context.Background()

	recurringExpenses, err := m.lmc.GetRecurringExpenses(ctx, nil)
	if err != nil {
		return nil
	}
	log.Debug("got recurring expenses")

	return getRecurringExpensesMsg{recurringExpenses: recurringExpenses}
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

func (m model) getCategories() tea.Msg {
	ctx := context.Background()

	cs, err := m.lmc.GetCategories(ctx)
	if err != nil {
		return nil
	}

	return getCategoriesMsg{categories: cs}
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
	for i, j := 0, len(ts)-1; i < j; i, j = i+1, j-1 {
		ts[i], ts[j] = ts[j], ts[i]
	}

	return getsTransactionsMsg{ts: ts, period: m.period}
}

func (m model) getUser() tea.Msg {
	u, err := m.lmc.GetUser(context.Background())
	if err != nil {
		return nil
	}

	return getUserMsg{user: u}
}

func (m model) getTags() tea.Msg {
	ctx := context.Background()

	tags, err := m.lmc.GetTags(ctx)
	if err != nil {
		return nil
	}

	return getTagsMsg{tags: tags}
}

func (m model) getBudgets() tea.Msg {
	ctx := context.Background()

	m.period.setPeriod(m.currentPeriod, m.periodType)

	sd := m.period.startDate()
	ed := m.period.endDate()

	budgets, err := m.lmc.GetBudgets(ctx, &lm.BudgetFilters{
		StartDate: sd,
		EndDate:   ed,
	})
	if err != nil {
		return nil
	}

	return getBudgetsMsg{budgets: budgets, period: m.period}
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
