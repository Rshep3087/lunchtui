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
	"github.com/icco/lunchmoney"
)

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// always check for quit key first
	if msg, ok := msg.(tea.KeyMsg); ok {
		if model, cmd := handleKeyPress(msg, &m); cmd != nil {
			log.Debug("key press handled, cmd returned")
			return model, cmd
		}
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.handleWindowSize(msg)

	case spinner.TickMsg:
		return m.handleSpinnerTick(msg)

	case getCategoriesMsg:
		return m.handleGetCategories(msg)

	case getAccountsMsg:
		return m.handleGetAccounts(msg)

	case getsTransactionsMsg:
		return m.handleGetTransactions(msg)

	case getUserMsg:
		return m.handleGetUser(msg)

	case getRecurringExpensesMsg:
		return m.handleGetRecurringExpenses(msg)

	case getTagsMsg:
		return m.handleGetTags(msg)

	case getBudgetsMsg:
		return m.handleGetBudgets(msg)

	case authErrorMsg:
		m.sessionState = errorState
		m.errorMsg = fmt.Sprintf("Check your API token: %s", msg.err.Error())
		return m, nil

	case insertTransactionMsg:
		if msg.err != nil {
			return m, m.transactions.NewStatusMessage(
				fmt.Sprintf("Error inserting transaction: %s", msg.err.Error()),
			)
		}

		return m, tea.Batch(m.getTransactions,
			m.transactions.NewStatusMessage("Transaction inserted successfully!"),
		)
	}

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
		form, formCmd := m.insertTransactionForm.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			m.insertTransactionForm = f
		} else {
			log.Debug("insertTransactionForm did not return a form, returning nil")
			return m, nil
		}

		if m.insertTransactionForm.State == huh.StateCompleted {
			m.previousSessionState = m.sessionState
			m.sessionState = transactions
			return m, func() tea.Msg {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				cid, ok := m.insertTransactionForm.Get("category").(int64)
				if !ok {
					log.Debug("category ID not found in form")
					return insertTransactionMsg{err: errors.New("category ID not found in form")}
				}

				account, ok := m.insertTransactionForm.Get("account").(accountOpt)
				if !ok {
					log.Debug("account not found in form")
					return insertTransactionMsg{err: errors.New("account not found in form")}
				}

				transaction := lunchmoney.InsertTransaction{
					Date:       m.insertTransactionForm.GetString("date"),
					Payee:      m.insertTransactionForm.GetString("payee"),
					Amount:     m.insertTransactionForm.GetString("amount"),
					Currency:   m.user.PrimaryCurrency,
					CategoryID: ptr(cid),
					Notes:      m.insertTransactionForm.GetString("notes"),
					Status:     m.insertTransactionForm.GetString("status"),
				}

				if account.ID != 0 {
					switch account.Type {
					case "plaid":
						transaction.PlaidAccountID = ptr(account.ID)
					case "asset":
						transaction.AssetID = ptr(account.ID)
					}
				}

				req := lunchmoney.InsertTransactionsRequest{
					ApplyRules:        true,
					SkipDuplicates:    true,
					CheckForRecurring: true,
					DebitAsNegative:   false,
					Transactions:      []lunchmoney.InsertTransaction{transaction},
				}

				log.Debug("inserting transaction", "request", req, "category_id", cid)

				resp, err := m.lmc.InsertTransactions(ctx, req)
				if err != nil {
					log.Debug("error inserting transaction", "error", err)
					return insertTransactionMsg{err: fmt.Errorf("error inserting transaction: %w", err)}
				}

				log.Debug("transaction inserted successfully", "ids", resp.IDs)

				return insertTransactionMsg{}
			}
		}

		return m, formCmd

	case budgets:
		return updateBudgets(msg, m)

	case configView:
		m.configView, cmd = m.configView.Update(msg)
		return m, cmd

	case loading:
		m.loadingSpinner, cmd = m.loadingSpinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

type insertTransactionMsg struct {
	err error
}

func ptr[T any](v T) *T { return &v }
