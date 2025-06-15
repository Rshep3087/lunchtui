package main

import (
	"fmt"
	"strconv"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	lm "github.com/icco/lunchmoney"
)

type budgetItem struct {
	b        *lm.Budget
	category *lm.Category
}

// Implement list.Item interface for budgetItem.
func (b budgetItem) Title() string {
	return b.b.CategoryName
}

func (b budgetItem) Description() string {
	if len(b.b.Data) == 0 {
		return "No budget data"
	}

	// Get the first available budget data entry
	for _, data := range b.b.Data {
		if data == nil {
			continue
		}
		amount := "0"
		if data.BudgetAmount != "" {
			amount = string(data.BudgetAmount)
		}
		spent := strconv.FormatFloat(data.SpendingToBase, 'f', 2, 64)
		return fmt.Sprintf("Budget: %s %s | Spent: $%s | Transactions: %d",
			amount, data.BudgetCurrency, spent, data.NumTransactions)
	}

	return "No budget data available"
}

func (b budgetItem) FilterValue() string {
	return b.b.CategoryName
}

// createBudgetList creates a new list model for budgets.
func createBudgetList(delegate list.DefaultDelegate) list.Model {
	budgetList := list.New([]list.Item{}, delegate, 0, 0)
	budgetList.SetShowTitle(false)
	budgetList.StatusMessageLifetime = 3 * time.Second
	return budgetList
}

// updateBudgets handles the budgets view updates.
func updateBudgets(msg tea.Msg, m model) (tea.Model, tea.Cmd) {
	// Period navigation and other keys are handled in handleKeyPress
	// so we just need to handle the list updates here
	var cmd tea.Cmd
	m.budgets, cmd = m.budgets.Update(msg)
	return m, cmd
}

// budgetsView renders the budgets view.
func budgetsView(m model) string {
	return m.budgets.View()
}
