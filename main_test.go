package main

import (
	"strings"
	"testing"
	"time"

	"github.com/carlmjohnson/be"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	lm "github.com/icco/lunchmoney"
	"github.com/rshep3087/lunchtui/overview"
)

func TestBudgetsNavigation(t *testing.T) {
	m := model{
		sessionState:         overviewState,
		previousSessionState: overviewState,
		loadingState:         newLoadingState("budgets"),
	}

	// Test navigating to budgets
	resultModel, cmd := handleKeyPress(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}}, &m)
	result := resultModel.(*model)

	if result.sessionState != loading {
		t.Errorf("Expected session state to be loading, got %v", result.sessionState)
	}

	if result.previousSessionState != overviewState {
		t.Errorf("Expected previous session state to be overviewState, got %v", result.previousSessionState)
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

			resultModel, _ := handleEscape(m)
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

func TestHandleGetUser(t *testing.T) {
	ov := overview.New()
	m := &model{
		loadingState: loadingState{"user": false},
		overview:     ov,
	}
	returnedModel, cmd := m.handleGetUser(getUserMsg{
		user: &lm.User{PrimaryCurrency: "USD"},
	})

	gotModel, ok := returnedModel.(model)
	be.True(t, ok)
	be.Equal(t, lm.User{PrimaryCurrency: "USD"}, *gotModel.user)
	be.Zero(t, cmd)
}

func TestOverviewUserDisplay(t *testing.T) {
	// Create an overview model
	overview := overview.New()

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
		name              string
		transactions      []*lm.Transaction
		expectedItems     int
		expectedPeriod    Period
		initialState      sessionState
		expectedState     sessionState
		expectedPrevState sessionState
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
			initialState:      transactions,
			expectedState:     transactions,
			expectedPrevState: transactions,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idc := map[int64]*lm.Category{0: {ID: 1, Name: "Category 1"}}

			// Set up test model
			m := &model{
				sessionState:         tt.initialState,
				loadingState:         newLoadingState("transactions"),
				previousSessionState: tt.initialState,
				idToCategory:         idc,
				plaidAccounts:        map[int64]*lm.PlaidAccount{},
				assets:               map[int64]*lm.Asset{},
				transactions:         list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0),
				overview:             overview.New(),
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
