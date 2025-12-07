package main

import (
	"strings"
	"testing"
	"time"

	"github.com/Rshep3087/lunchtui/overview"
	"github.com/carlmjohnson/be"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	lm "github.com/icco/lunchmoney"
)

func TestBudgetsNavigation(t *testing.T) {
	m := model{
		sessionState:         overviewState,
		previousSessionState: overviewState,
		loadingState:         newLoadingState("categories", "transactions", "user", "accounts", "tags"),
		keys:                 initializeKeyMap(),
	}

	// Test navigating to budgets
	resultModel, cmd := handleKeyPress(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}}, &m)
	result := resultModel.(*model)

	// Budget loading doesn't block UI, so we go directly to budgets state
	if result.sessionState != budgets {
		t.Errorf("Expected session state to be budgets, got %v", result.sessionState)
	}

	if result.previousSessionState != budgets {
		t.Errorf("Expected previous session state to be budgets, got %v", result.previousSessionState)
	}

	if cmd == nil {
		t.Error("Expected command to fetch budgets, got nil")
	}
}

func TestHandleEscape(t *testing.T) {
	tests := []struct {
		name          string
		initialState  sessionState
		expectedState sessionState
		categoryForm  *huh.Form
		expectedForm  huh.FormState
		previousState sessionState
	}{
		{
			name:          "from categorize transaction state",
			initialState:  categorizeTransaction,
			expectedState: transactions,
			categoryForm:  &huh.Form{State: huh.StateNormal},
			expectedForm:  huh.StateAborted,
			previousState: overviewState,
		},
		{
			name:          "from transactions state",
			initialState:  transactions,
			expectedState: overviewState,
			previousState: transactions,
		},
		{
			name:          "from overview state",
			initialState:  overviewState,
			expectedState: overviewState,
			previousState: overviewState,
		},
		{
			name:          "from recurring expenses state",
			initialState:  recurringExpenses,
			expectedState: overviewState,
			previousState: recurringExpenses,
		},
		{
			name:          "from budgets state",
			initialState:  budgets,
			expectedState: overviewState,
			previousState: budgets,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &model{
				sessionState:         tt.initialState,
				previousSessionState: tt.previousState,
				categoryForm:         tt.categoryForm,
			}

			resultModel, _ := handleEscape(tea.KeyMsg{}, m)
			result := resultModel.(*model)

			be.Equal(t, tt.expectedState, result.sessionState)
			if tt.categoryForm != nil {
				be.Equal(t, tt.expectedForm, result.categoryForm.State)
			}
		})
	}
}

func TestAdvancePeriod(t *testing.T) {
	tests := []struct {
		name              string
		periodType        string
		initialDate       time.Time
		expectedDate      time.Time
		initialState      sessionState
		expectedState     sessionState
		expectedPrevState sessionState
	}{
		{
			name:              "advance monthly period",
			periodType:        monthlyPeriodType,
			initialDate:       time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
			expectedDate:      time.Date(2024, 2, 15, 0, 0, 0, 0, time.UTC),
			initialState:      transactions,
			expectedState:     loading,
			expectedPrevState: transactions,
		},
		{
			name:              "advance annual period",
			periodType:        annualPeriodType,
			initialDate:       time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC),
			expectedDate:      time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC),
			initialState:      transactions,
			expectedState:     loading,
			expectedPrevState: transactions,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up test model
			m := &model{
				periodType:           tt.periodType,
				currentPeriod:        tt.initialDate,
				sessionState:         tt.initialState,
				loadingState:         newLoadingState("transactions"),
				previousSessionState: tt.initialState,
			}

			// Execute function under test
			resultModel, cmd := advancePeriod(m)
			result := resultModel.(*model)

			// Verify date advanced correctly
			be.Equal(t, tt.expectedDate, result.currentPeriod)

			// Verify session state changes
			be.Equal(t, tt.expectedState, result.sessionState)
			be.Equal(t, tt.expectedPrevState, result.previousSessionState)

			// Verify command was returned
			be.Nonzero(t, cmd)
		})
	}
}

func TestRetrievePreviousPeriod(t *testing.T) {
	tests := []struct {
		name              string
		periodType        string
		initialDate       time.Time
		expectedDate      time.Time
		initialState      sessionState
		expectedState     sessionState
		expectedPrevState sessionState
	}{
		{
			name:              "retrieve previous monthly period",
			periodType:        monthlyPeriodType,
			initialDate:       time.Date(2024, 2, 15, 0, 0, 0, 0, time.UTC),
			expectedDate:      time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
			initialState:      transactions,
			expectedState:     loading,
			expectedPrevState: transactions,
		},
		{
			name:              "retrieve previous annual period",
			periodType:        annualPeriodType,
			initialDate:       time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC),
			expectedDate:      time.Date(2023, 6, 15, 0, 0, 0, 0, time.UTC),
			initialState:      transactions,
			expectedState:     loading,
			expectedPrevState: transactions,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up test model
			m := &model{
				periodType:           tt.periodType,
				currentPeriod:        tt.initialDate,
				sessionState:         tt.initialState,
				loadingState:         newLoadingState("transactions"),
				previousSessionState: tt.initialState,
			}

			// Execute function under test
			resultModel, cmd := retrievePreviousPeriod(m)
			result := resultModel.(*model)

			// Verify date was set back correctly
			be.Equal(t, tt.expectedDate, result.currentPeriod)

			// Verify session state changes
			be.Equal(t, tt.expectedState, result.sessionState)
			be.Equal(t, tt.expectedPrevState, result.previousSessionState)

			// Verify command was returned
			be.Nonzero(t, cmd)
		})
	}
}

func TestSwitchPeriodType(t *testing.T) {
	tests := []struct {
		name              string
		initialPeriodType string
		expectedPeriod    string
		initialState      sessionState
		expectedState     sessionState
		expectedPrevState sessionState
	}{
		{
			name:              "switch from monthly to annual",
			initialPeriodType: monthlyPeriodType,
			expectedPeriod:    annualPeriodType,
			initialState:      transactions,
			expectedState:     loading,
			expectedPrevState: transactions,
		},
		{
			name:              "switch from annual to monthly",
			initialPeriodType: annualPeriodType,
			expectedPeriod:    monthlyPeriodType,
			initialState:      transactions,
			expectedState:     loading,
			expectedPrevState: transactions,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up test model
			m := &model{
				periodType:           tt.initialPeriodType,
				sessionState:         tt.initialState,
				loadingState:         newLoadingState("transactions"),
				previousSessionState: tt.initialState,
			}

			// Execute function under test
			resultModel, cmd := switchPeriodType(m)
			result := resultModel.(*model)

			// Verify period type changed correctly
			be.Equal(t, tt.expectedPeriod, result.periodType)

			// Verify session state changes
			be.Equal(t, tt.expectedState, result.sessionState)
			be.Equal(t, tt.expectedPrevState, result.previousSessionState)

			// Verify command was returned
			be.Nonzero(t, cmd)
		})
	}
}

func TestOverviewUserDisplay(t *testing.T) {
	// Create an overview model
	overview := overview.New(overview.Config{
		ShowUserInfo: true,
	})

	// Create mock user data
	mockUser := &lm.User{
		UserName:        "John Doe",
		UserEmail:       "john@example.com",
		UserID:          123,
		AccountID:       456,
		BudgetName:      "My Budget",
		PrimaryCurrency: "USD",
		APIKeyLabel:     "Personal API Key",
	}

	// Set the user
	overview.SetUser(mockUser)

	// Get the view content
	view := overview.View()

	// Check that user information is displayed
	be.True(t, strings.Contains(view, "User Info"))
	be.True(t, strings.Contains(view, "Budget: My Budget"))
	be.True(t, strings.Contains(view, "User: John Doe"))
	be.True(t, strings.Contains(view, "Currency: USD"))
	be.True(t, strings.Contains(view, "API Key: Personal API Key"))
}

func TestHandleGetTransactions(t *testing.T) {
	tests := []struct {
		name                    string
		transactions            []*lm.Transaction
		expectedItems           int
		expectedPeriod          Period
		initialState            sessionState
		expectedState           sessionState
		expectedPrevState       sessionState
		hidePendingTransactions bool
	}{
		{
			name: "handle get transactions",
			transactions: []*lm.Transaction{
				{ID: 1, Date: "2024-01-01"},
				{ID: 2, Date: "2024-01-02"},
			},
			expectedItems: 2,
			expectedPeriod: Period{
				start: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				end:   time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC),
			},
			initialState:            transactions,
			expectedState:           transactions,
			expectedPrevState:       transactions,
			hidePendingTransactions: false,
		},
		{
			name: "hide pending transactions",
			transactions: []*lm.Transaction{
				{ID: 1, Date: "2024-01-01", Status: "cleared"},
				{ID: 2, Date: "2024-01-02", Status: "pending"},
				{ID: 3, Date: "2024-01-03", Status: "uncleared"},
			},
			expectedItems: 2, // Should exclude the pending transaction
			expectedPeriod: Period{
				start: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				end:   time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC),
			},
			initialState:            transactions,
			expectedState:           transactions,
			expectedPrevState:       transactions,
			hidePendingTransactions: true,
		},
		{
			name: "show all transactions when flag is false",
			transactions: []*lm.Transaction{
				{ID: 1, Date: "2024-01-01", Status: "cleared"},
				{ID: 2, Date: "2024-01-02", Status: "pending"},
				{ID: 3, Date: "2024-01-03", Status: "uncleared"},
			},
			expectedItems: 3, // Should include all transactions
			expectedPeriod: Period{
				start: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				end:   time.Date(2024, 1, 31, 23, 59, 59, 0, time.UTC),
			},
			initialState:            transactions,
			expectedState:           transactions,
			expectedPrevState:       transactions,
			hidePendingTransactions: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idc := map[int64]*lm.Category{0: {ID: 1, Name: "Category 1"}}

			// Set up test model
			m := &model{
				sessionState:            tt.initialState,
				loadingState:            newLoadingState("transactions"),
				previousSessionState:    tt.initialState,
				idToCategory:            idc,
				plaidAccounts:           map[int64]*lm.PlaidAccount{},
				assets:                  map[int64]*lm.Asset{},
				transactions:            list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0),
				overview:                overview.New(overview.Config{ShowUserInfo: true}),
				hidePendingTransactions: tt.hidePendingTransactions,
			}

			m.overview.SetCategories(idc)

			// Execute function under test
			resultModel, cmd := m.handleGetTransactions(getsTransactionsMsg{
				ts:     tt.transactions,
				period: tt.expectedPeriod,
			})
			result, ok := resultModel.(model)
			be.True(t, ok)

			// Verify transactions were set correctly
			be.Equal(t, tt.expectedItems, len(result.transactions.Items()))

			// Verify period was set correctly
			be.Equal(t, tt.expectedPeriod, result.period)

			// Verify session state changes
			be.Equal(t, tt.expectedState, result.sessionState)
			be.Equal(t, tt.expectedPrevState, result.previousSessionState)

			// Verify command was returned
			be.Zero(t, cmd)
		})
	}
}

func TestFilterUncategorizedTransactions(t *testing.T) {
	// Create test transactions with different CategoryIDs
	transactions := []list.Item{
		transactionItem{
			t:        &lm.Transaction{ID: 1, CategoryID: 0, Payee: "Uncategorized 1"},
			category: &lm.Category{ID: 0, Name: "Uncategorized"},
		},
		transactionItem{
			t:        &lm.Transaction{ID: 2, CategoryID: 1, Payee: "Categorized 1"},
			category: &lm.Category{ID: 1, Name: "Food"},
		},
		transactionItem{
			t:        &lm.Transaction{ID: 3, CategoryID: 0, Payee: "Uncategorized 2"},
			category: &lm.Category{ID: 0, Name: "Uncategorized"},
		},
		transactionItem{
			t:        &lm.Transaction{ID: 4, CategoryID: 2, Payee: "Categorized 2"},
			category: &lm.Category{ID: 2, Name: "Transport"},
		},
	}

	// Create a model with transactions
	m := model{}
	m.transactions = list.New(transactions, list.NewDefaultDelegate(), 80, 20)
	m.transactions.SetItems(transactions)
	// Initialize originalTransactions for the filter to work properly
	m.originalTransactions = transactions

	// Apply the uncategorized filter
	result, _ := filterUncategorizedTransactions(m)

	// Check that only uncategorized transactions remain
	resultModel := result.(model)
	filteredItems := resultModel.transactions.Items()
	expectedCount := 2

	if len(filteredItems) != expectedCount {
		t.Errorf("Expected %d uncategorized transactions, got %d", expectedCount, len(filteredItems))
	}

	// Verify that all remaining transactions have CategoryID of 0
	for i, item := range filteredItems {
		if trans, ok := item.(transactionItem); ok {
			if trans.t.CategoryID != 0 {
				t.Errorf("Transaction %d should have CategoryID 0, got %d", i, trans.t.CategoryID)
			}
		} else {
			t.Errorf("Item %d is not a transactionItem", i)
		}
	}
}

// TestNotesEditing tests the notes editing functionality.
func TestNotesEditing(t *testing.T) {
	// Create a test model with a current transaction
	m := model{
		sessionState: detailedTransaction,
		currentTransaction: &transactionItem{
			t: &lm.Transaction{
				ID:    123,
				Payee: "Test Transaction",
				Notes: "Original notes",
			},
		},
		notesInput: textinput.New(),
	}
	m.notesInput.Placeholder = "Enter notes..."

	t.Run("enter edit mode with n key", func(t *testing.T) {
		// Simulate pressing 'n' key
		updatedModel, _ := updateDetailedTransaction(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}, m)
		updatedM := updatedModel.(model)

		if !updatedM.isEditingNotes {
			t.Error("Expected isEditingNotes to be true after pressing 'n'")
		}

		if updatedM.notesInput.Value() != "Original notes" {
			t.Errorf(
				"Expected notes input to be pre-filled with 'Original notes', got '%s'",
				updatedM.notesInput.Value(),
			)
		}
	})

	t.Run("cancel edit mode with escape", func(t *testing.T) {
		// Start in edit mode
		m.isEditingNotes = true
		m.notesInput.Focus()

		// Simulate pressing escape
		updatedModel, _ := updateDetailedTransaction(tea.KeyMsg{Type: tea.KeyEsc}, m)
		updatedM := updatedModel.(model)

		if updatedM.isEditingNotes {
			t.Error("Expected isEditingNotes to be false after pressing escape")
		}
	})

	t.Run("escape handling in keybindings", func(t *testing.T) {
		// Test detailed transaction with notes editing
		m.isEditingNotes = true
		updatedModel, _ := handleEscape(tea.KeyMsg{}, &m)
		updatedM := updatedModel.(*model)

		if updatedM.isEditingNotes {
			t.Error("Expected isEditingNotes to be false after handleEscape")
		}

		if updatedM.sessionState != detailedTransaction {
			t.Error("Expected to remain in detailedTransaction state when canceling notes editing")
		}

		// Test normal escape (should exit detailed view)
		m.isEditingNotes = false
		updatedModel, _ = handleEscape(tea.KeyMsg{}, &m)
		updatedM = updatedModel.(*model)

		if updatedM.sessionState == detailedTransaction {
			t.Error("Expected to exit detailedTransaction state on normal escape")
		}
	})

	t.Run("input blocking during notes editing", func(t *testing.T) {
		m.isEditingNotes = true
		if !isInputBlocked(&m) {
			t.Error("Expected input to be blocked while editing notes")
		}

		m.isEditingNotes = false
		if isInputBlocked(&m) {
			t.Error("Expected input to not be blocked when not editing notes")
		}
	})

	t.Run("save notes with enter key", func(t *testing.T) {
		// Start in edit mode with modified notes
		m.isEditingNotes = true
		m.notesInput.Focus()
		m.notesInput.SetValue("Updated notes")

		// Simulate pressing enter - this would normally trigger updateTransactionNotes
		// Since we can't easily test the API call, we just verify the state change
		updatedModel, cmd := updateDetailedTransaction(tea.KeyMsg{Type: tea.KeyEnter}, m)
		updatedM := updatedModel.(model)

		if updatedM.isEditingNotes {
			t.Error("Expected isEditingNotes to be false after pressing enter")
		}

		// Verify that a command was returned (the update function)
		if cmd == nil {
			t.Error("Expected a command to be returned for updating transaction notes")
		}
	})
}

func TestFilterUnclearedTransactionsToggle(t *testing.T) {
	// Create test transactions with different statuses
	transactions := []list.Item{
		transactionItem{
			t:        &lm.Transaction{ID: 1, Status: unclearedStatus, Payee: "Uncleared 1"},
			category: &lm.Category{ID: 1, Name: "Food"},
		},
		transactionItem{
			t:        &lm.Transaction{ID: 2, Status: "cleared", Payee: "Cleared 1"},
			category: &lm.Category{ID: 1, Name: "Food"},
		},
		transactionItem{
			t:        &lm.Transaction{ID: 3, Status: unclearedStatus, Payee: "Uncleared 2"},
			category: &lm.Category{ID: 2, Name: "Transport"},
		},
		transactionItem{
			t:        &lm.Transaction{ID: 4, Status: "pending", Payee: "Pending 1"},
			category: &lm.Category{ID: 2, Name: "Transport"},
		},
	}

	// Create a model with transactions
	m := model{}
	m.transactions = list.New(transactions, list.NewDefaultDelegate(), 80, 20)
	m.transactions.SetItems(transactions)
	m.originalTransactions = transactions
	m.isFilteredUncleared = false

	// Test initial state - should show all transactions
	if len(m.transactions.Items()) != 4 {
		t.Errorf("Expected 4 total transactions initially, got %d", len(m.transactions.Items()))
	}

	// Apply uncleared filter for the first time
	result, _ := filterUnclearedTransactions(m)
	resultModel := result.(model)

	// Should now show only uncleared transactions
	filteredItems := resultModel.transactions.Items()
	if len(filteredItems) != 2 {
		t.Errorf("Expected 2 uncleared transactions after first filter, got %d", len(filteredItems))
	}

	// Check that filter state is updated
	if !resultModel.isFilteredUncleared {
		t.Error("Expected isFilteredUncleared to be true after applying filter")
	}

	// Verify that all remaining transactions are uncleared
	for i, item := range filteredItems {
		if trans, ok := item.(transactionItem); ok {
			if trans.t.Status != unclearedStatus {
				t.Errorf("Transaction %d should be uncleared, got status: %s", i, trans.t.Status)
			}
		} else {
			t.Errorf("Item %d is not a transactionItem", i)
		}
	}

	// Apply uncleared filter again - should toggle back to show all transactions
	result2, _ := filterUnclearedTransactions(resultModel)
	resultModel2 := result2.(model)

	// Should now show all transactions again
	allItems := resultModel2.transactions.Items()
	if len(allItems) != 4 {
		t.Errorf("Expected 4 total transactions after toggle, got %d", len(allItems))
	}

	// Check that filter state is reset
	if resultModel2.isFilteredUncleared {
		t.Error("Expected isFilteredUncleared to be false after toggle")
	}
}
