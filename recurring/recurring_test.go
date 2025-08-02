package recurring

import (
	"strings"
	"testing"

	"github.com/carlmjohnson/be"
	"github.com/icco/lunchmoney"
)

func TestNew(t *testing.T) {
	colors := Colors{
		Primary: "#ff0000",
	}

	model := New(colors)

	// Test that model is initialized properly
	be.Nonzero(t, model.recurringExpenses)
	// Test that the table has the expected columns
	columns := model.recurringExpenses.Columns()
	be.Equal(t, 5, len(columns))
	be.Equal(t, "Merchant", columns[0].Title)
	be.Equal(t, "Description", columns[1].Title)
	be.Equal(t, "Repeats", columns[2].Title)
	be.Equal(t, "Billing Day", columns[3].Title)
	be.Equal(t, "Amount", columns[4].Title)
}

func TestSetFocus(t *testing.T) {
	model := New(Colors{Primary: "#ff0000"})

	// Test setting focus to true
	model.SetFocus(true)
	// We can't directly test the internal focus state, but we can ensure no panic
	be.Nonzero(t, model)

	// Test setting focus to false
	model.SetFocus(false)
	be.Nonzero(t, model)
}

func TestSetSize(t *testing.T) {
	model := New(Colors{Primary: "#ff0000"})

	// Test setting size
	model.SetSize(100, 50)
	// We can't directly test the internal size, but we can ensure no panic
	be.Nonzero(t, model)
}

func TestSetRecurringExpenses(t *testing.T) {
	model := New(Colors{Primary: "#ff0000"})

	tests := []struct {
		name     string
		expenses []*lunchmoney.RecurringExpense
	}{
		{
			name:     "empty expenses",
			expenses: []*lunchmoney.RecurringExpense{},
		},
		{
			name: "single expense",
			expenses: []*lunchmoney.RecurringExpense{
				{
					ID:          1,
					Payee:       "Netflix",
					Description: "Streaming service",
					Amount:      "15.99",
					BillingDate: "1",
					Type:        "monthly",
				},
			},
		},
		{
			name: "multiple expenses",
			expenses: []*lunchmoney.RecurringExpense{
				{
					ID:          1,
					Payee:       "Netflix",
					Description: "Streaming service",
					Amount:      "15.99",
					BillingDate: "1",
					Type:        "monthly",
				},
				{
					ID:          2,
					Payee:       "Spotify",
					Description: "Music streaming",
					Amount:      "9.99",
					BillingDate: "15",
					Type:        "monthly",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model.SetRecurringExpenses(tt.expenses)
			// Test that no panic occurs and model is still valid
			be.Nonzero(t, model)

			// Test that rows match the number of expenses
			rows := model.recurringExpenses.Rows()
			be.Equal(t, len(tt.expenses), len(rows))
		})
	}
}

func TestInit(t *testing.T) {
	model := New(Colors{Primary: "#ff0000"})

	cmd := model.Init()
	// Init should return nil command for this model
	if cmd != nil {
		t.Errorf("Expected nil command, got %v", cmd)
	}
}

func TestView(t *testing.T) {
	model := New(Colors{Primary: "#ff0000"})

	// Test view with no expenses
	view := model.View()
	be.Nonzero(t, view)

	// Test view with expenses
	model.SetRecurringExpenses([]*lunchmoney.RecurringExpense{
		{
			ID:          1,
			Payee:       "Netflix",
			Description: "Streaming service",
			Amount:      "15.99",
			BillingDate: "1",
			Type:        "monthly",
		},
	})

	view = model.View()
	be.Nonzero(t, view)
	// View should contain the payee name
	if !strings.Contains(view, "Netflix") {
		t.Errorf("Expected view to contain 'Netflix', got: %s", view)
	}
}
