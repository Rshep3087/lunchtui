package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/log"
	lm "github.com/icco/lunchmoney"
)

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		if model, cmd := handleKeyPress(msg, &m); cmd != nil {
			log.Debug("key press handled, cmd returned")
			return model, cmd
		}
	}

	if model, cmd, handled := m.handleMessages(msg); handled {
		return model, cmd
	}

	return m.handleSessionState(msg)
}

func (m model) handleMessages(msg tea.Msg) (tea.Model, tea.Cmd, bool) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		model, cmd := m.handleWindowSize(msg)
		return model, cmd, true
	case spinner.TickMsg:
		model, cmd := m.handleSpinnerTick(msg)
		return model, cmd, true
	case getCategoriesMsg:
		model, cmd := m.handleGetCategories(msg)
		return model, cmd, true
	case getAccountsMsg:
		model, cmd := m.handleGetAccounts(msg)
		return model, cmd, true
	case getsTransactionsMsg:
		model, cmd := m.handleGetTransactions(msg)
		return model, cmd, true
	case getUserMsg:
		model, cmd := m.handleGetUser(msg)
		return model, cmd, true
	case getRecurringExpensesMsg:
		model, cmd := m.handleGetRecurringExpenses(msg)
		return model, cmd, true
	case getTagsMsg:
		model, cmd := m.handleGetTags(msg)
		return model, cmd, true
	case getBudgetsMsg:
		model, cmd := m.handleGetBudgets(msg)
		return model, cmd, true
	case authErrorMsg:
		m.sessionState = errorState
		m.errorMsg = fmt.Sprintf("Check your API token: %s", msg.err.Error())
		return m, nil, true
	case insertTransactionMsg:
		model, cmd := m.handleInsertTransactionMsg(msg)
		return model, cmd, true
	}
	return m, nil, false
}

func (m model) handleInsertTransactionMsg(msg insertTransactionMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		return m, m.transactions.NewStatusMessage(
			fmt.Sprintf("Error inserting transaction: %s", msg.err.Error()),
		)
	}
	return m, tea.Batch(m.getTransactions,
		m.transactions.NewStatusMessage("Transaction inserted successfully!"),
	)
}

func (m model) handleSessionState(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch m.sessionState {
	case overviewState:
		m.overview, cmd = m.overview.Update(msg)
		return m, cmd
	case categorizeTransaction:
		return updateCategorizeTransaction(msg, &m)
	case detailedTransaction:
		return updateDetailedTransaction(msg, m)
	case transactions:
		return updateTransactions(msg, m)
	case recurringExpenses:
		m.recurringExpenses, cmd = m.recurringExpenses.Update(msg)
		return m, cmd
	case insertTransaction:
		return m.handleInsertTransactionState(msg)
	case budgets:
		return updateBudgets(msg, m)
	case configView:
		m.configView, cmd = m.configView.Update(msg)
		return m, cmd
	case loading:
		m.loadingSpinner, cmd = m.loadingSpinner.Update(msg)
		return m, cmd
	case errorState:
		// Error state is static - quit is handled by global key handler
		return m, nil
	}
	return m, nil
}

func (m model) handleInsertTransactionState(msg tea.Msg) (tea.Model, tea.Cmd) {
	form, formCmd := m.insertTransactionForm.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.insertTransactionForm = f
	} else {
		log.Debug("insertTransactionForm did not return a form, returning nil")
		return m, nil
	}

	if m.insertTransactionForm.State == huh.StateCompleted {
		return m.handleCompletedTransactionForm()
	}

	return m, formCmd
}

func (m model) handleCompletedTransactionForm() (tea.Model, tea.Cmd) {
	m.previousSessionState = m.sessionState
	m.sessionState = transactions
	return m, m.createInsertTransactionCmd()
}

func (m model) createInsertTransactionCmd() tea.Cmd {
	return func() tea.Msg {
		if !m.insertTransactionForm.GetBool("submit") {
			log.Debug("not submitting form")
			return m.transactions.NewStatusMessage("Transaction not submitted")
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		transaction, err := m.buildTransactionFromForm()
		if err != nil {
			return insertTransactionMsg{err: err}
		}

		req := lm.InsertTransactionsRequest{
			ApplyRules:        true,
			SkipDuplicates:    true,
			CheckForRecurring: true,
			DebitAsNegative:   false,
			Transactions:      []lm.InsertTransaction{transaction},
		}

		log.Debug("inserting transaction", "request", req)

		resp, err := m.lmc.InsertTransactions(ctx, req)
		if err != nil {
			log.Debug("error inserting transaction", "error", err)
			return insertTransactionMsg{err: fmt.Errorf("error inserting transaction: %w", err)}
		}

		log.Debug("transaction inserted successfully", "ids", resp.IDs)
		return insertTransactionMsg{}
	}
}

func (m model) buildTransactionFromForm() (lm.InsertTransaction, error) {
	cid, ok := m.insertTransactionForm.Get("category").(int64)
	if !ok {
		log.Debug("category ID not found in form")
		return lm.InsertTransaction{}, errors.New("category ID not found in form")
	}

	account, ok := m.insertTransactionForm.Get("account").(accountOpt)
	if !ok {
		log.Debug("account not found in form")
		return lm.InsertTransaction{}, errors.New("account not found in form")
	}

	transaction := lm.InsertTransaction{
		Date:       m.insertTransactionForm.GetString("date"),
		Payee:      m.insertTransactionForm.GetString("payee"),
		Amount:     m.insertTransactionForm.GetString("amount"),
		Currency:   m.user.PrimaryCurrency,
		CategoryID: ptr(cid),
		Notes:      m.insertTransactionForm.GetString("notes"),
		Status:     m.insertTransactionForm.GetString("status"),
	}

	tags, ok := m.insertTransactionForm.Get("tags").([]int)
	if ok && len(tags) > 0 {
		transaction.TagsIDs = tags
	}

	if account.ID != 0 {
		switch account.Type {
		case "plaid":
			transaction.PlaidAccountID = ptr(account.ID)
		case "asset":
			transaction.AssetID = ptr(account.ID)
		}
	}

	return transaction, nil
}

type insertTransactionMsg struct {
	err error
}

func ptr[T any](v T) *T { return &v }
