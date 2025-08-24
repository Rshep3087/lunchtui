package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/log"
)

func (m *model) newCategorizeTransactionForm(t transactionItem) *huh.Form {
	opts := make([]huh.Option[int64], len(m.categories))
	for i, c := range m.categories {
		opts[i] = huh.NewOption(c.Name, c.ID)
	}

	// Set default selection to AI recommendation if available
	selectField := huh.NewSelect[int64]().
		Title("New category").
		Description(m.getCategoryDescription()).
		Options(opts...).
		Key("category")

	// Set AI recommendation as default if available
	if m.aiRecommendation != nil {
		selectField = selectField.Value(&m.aiRecommendation.CategoryID)
	}

	form := huh.NewForm(huh.NewGroup(selectField))
	form.SubmitCmd = func() tea.Msg { return submitCategoryForm(*m, t) }

	return form
}

func (m *model) getCategoryDescription() string {
	log.Debug("getCategoryDescription called",
		"is_loading", m.isLoadingRecommendation,
		"has_recommendation", m.aiRecommendation != nil,
		"ai_enabled", m.aiRecommender != nil && m.aiRecommender.IsEnabled())

	if m.isLoadingRecommendation {
		log.Debug("showing loading message")
		return "🤖 Getting AI recommendation..."
	}
	if m.aiRecommendation != nil {
		log.Debug("showing AI recommendation",
			"category", m.aiRecommendation.CategoryName,
			"confidence", m.aiRecommendation.Confidence)
		return fmt.Sprintf("🤖 AI recommends: %s (%.0f%% confidence)\n%s",
			m.aiRecommendation.CategoryName,
			m.aiRecommendation.Confidence,
			m.aiRecommendation.Reasoning)
	}
	if m.aiRecommender != nil && m.aiRecommender.IsEnabled() {
		log.Debug("showing AI available message")
		return "Select a new category for the transaction\nPress 'a' for AI recommendation"
	}
	log.Debug("showing default message")
	return "Select a new category for the transaction"
}

func handleAIRecommendationKey(keyMsg tea.KeyMsg, m *model) []tea.Cmd {
	var cmds []tea.Cmd
	if keyMsg.String() == "a" {
		if m.aiRecommender != nil && m.aiRecommender.IsEnabled() && m.currentTransaction != nil {
			log.Debug("user pressed 'a' key for AI recommendation", "transaction_id", m.currentTransaction.t.ID)
			if aiCmd := m.RecommendCategoryCmd(m.currentTransaction.t); aiCmd != nil {
				cmds = append(cmds, aiCmd)
			}
		}
	}
	return cmds
}

func handleAIRecommendationLoading(msg AIRecommendationLoadingMsg, m *model) []tea.Cmd {
	var cmds []tea.Cmd
	log.Debug("received AIRecommendationLoadingMsg", "transaction_id", msg.TransactionID)
	m.isLoadingRecommendation = true
	if m.categoryForm != nil && m.currentTransaction != nil {
		log.Debug("updating form to show loading state")
		m.categoryForm = m.newCategorizeTransactionForm(*m.currentTransaction)
	}
	return cmds
}

func handleAIRecommendationReceived(msg AIRecommendationMsg, m *model) []tea.Cmd {
	var cmds []tea.Cmd
	log.Debug("received AIRecommendationMsg", "transaction_id", msg.TransactionID, "has_error", msg.Error != nil)
	m.isLoadingRecommendation = false
	if msg.Error != nil {
		log.Error("AI recommendation failed in update handler", "error", msg.Error)
		m.aiRecommendation = nil
	} else {
		log.Debug("AI recommendation received in update handler",
			"category", msg.Recommendation.CategoryName,
			"confidence", msg.Recommendation.Confidence)
		m.aiRecommendation = msg.Recommendation
	}
	if m.categoryForm != nil && m.currentTransaction != nil {
		log.Debug("recreating form with AI recommendation")
		m.categoryForm = m.newCategorizeTransactionForm(*m.currentTransaction)
		cmds = append(cmds, m.categoryForm.Init())
	}
	return cmds
}

func updateCategorizeTransaction(msg tea.Msg, m *model) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	log.Debug("updateCategorizeTransaction called", "msg_type", fmt.Sprintf("%T", msg))

	// Handle key presses before passing to form
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyCmds := handleAIRecommendationKey(keyMsg, m); len(keyCmds) > 0 {
			cmds = append(cmds, keyCmds...)
			return m, tea.Batch(cmds...)
		}
	}

	// Handle AI recommendation messages
	switch msg := msg.(type) {
	case AIRecommendationLoadingMsg:
		cmds = append(cmds, handleAIRecommendationLoading(msg, m)...)
	case AIRecommendationMsg:
		cmds = append(cmds, handleAIRecommendationReceived(msg, m)...)
	}

	// Update the form
	form, cmd := m.categoryForm.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.categoryForm = f
	}
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	if m.categoryForm.State == huh.StateCompleted {
		m.sessionState = m.previousSessionState
		// Clear AI recommendation state
		m.aiRecommendation = nil
		m.isLoadingRecommendation = false
		log.Debug("categorize transaction form completed", "new_state", m.sessionState)
	}

	return m, tea.Batch(cmds...)
}
func categorizeTransactionView(m model) string {
	return m.categoryForm.View()
}
