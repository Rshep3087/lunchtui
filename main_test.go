package main

import (
	"testing"
	"time"

	"github.com/carlmjohnson/be"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/huh"
	lm "github.com/icco/lunchmoney"
	"github.com/rshep3087/lunchtui/overview"
)

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
