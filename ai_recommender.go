package main

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/log"
	lm "github.com/icco/lunchmoney"
)

// AIProvider defines the interface for AI-powered category recommendations.
type AIProvider interface {
	// RecommendCategory returns a recommended category ID for the given transaction
	// Returns the category ID, confidence score (0-100), and any error
	RecommendCategory(
		ctx context.Context,
		transaction *lm.Transaction,
		categories []*lm.Category,
	) (*CategoryRecommendation, error)
}

// CategoryRecommendation represents an AI recommendation for a transaction category.
type CategoryRecommendation struct {
	CategoryID   int64   `json:"category_id"`
	CategoryName string  `json:"category_name"`
	Confidence   float64 `json:"confidence"` // 0-100 confidence score
	Reasoning    string  `json:"reasoning"`  // Why this category was recommended
}

// AIRecommendationMsg is sent when AI recommendation is completed.
type AIRecommendationMsg struct {
	Recommendation *CategoryRecommendation
	Error          error
	TransactionID  int
}

// AIRecommendationLoadingMsg indicates AI recommendation is in progress.
type AIRecommendationLoadingMsg struct {
	TransactionID int
}

// AIRecommender manages AI-powered category recommendations.
type AIRecommender struct {
	provider AIProvider
	enabled  bool
}

// NewAIRecommender creates a new AI recommender with the given provider.
func NewAIRecommender(provider AIProvider) *AIRecommender {
	return &AIRecommender{
		provider: provider,
		enabled:  provider != nil,
	}
}

// IsEnabled returns true if AI recommendations are available.
func (r *AIRecommender) IsEnabled() bool {
	return r.enabled
}

// RecommendCategory creates a tea.Cmd to get AI recommendation for a transaction.
func (r *AIRecommender) RecommendCategory(transaction *lm.Transaction, categories []*lm.Category) tea.Cmd {
	if !r.enabled {
		log.Debug("AIRecommender.RecommendCategory: not enabled")
		return nil
	}

	log.Debug("AIRecommender.RecommendCategory: creating recommendation command", "transaction_id", transaction.ID)

	return func() tea.Msg {
		log.Debug("AIRecommender recommendation command executing", "transaction_id", transaction.ID)

		ctx, cancel := context.WithTimeout(context.Background(), aiRecommendationTimeout)
		defer cancel()

		recommendation, err := r.provider.RecommendCategory(ctx, transaction, categories)

		if err != nil {
			log.Error("AIRecommender recommendation failed", "error", err, "transaction_id", transaction.ID)
		} else {
			log.Debug("AIRecommender recommendation succeeded",
				"transaction_id", transaction.ID,
				"category", recommendation.CategoryName,
				"confidence", recommendation.Confidence)
		}

		return AIRecommendationMsg{
			Recommendation: recommendation,
			Error:          err,
			TransactionID:  int(transaction.ID),
		}
	}
}

// RecommendCategoryCmd is a helper function to create the recommendation command.
func (m model) RecommendCategoryCmd(transaction *lm.Transaction) tea.Cmd {
	log.Debug("RecommendCategoryCmd called", "transaction_id", transaction.ID, "payee", transaction.Payee)

	if m.aiRecommender == nil {
		log.Debug("AI recommender is nil, skipping recommendation")
		return nil
	}

	if !m.aiRecommender.IsEnabled() {
		log.Debug("AI recommender is disabled, skipping recommendation")
		return nil
	}

	log.Debug("AI recommender is enabled, starting recommendation process", "categories_count", len(m.categories))

	// Send loading message immediately
	loadingCmd := func() tea.Msg {
		log.Debug("Sending AI recommendation loading message", "transaction_id", transaction.ID)
		return AIRecommendationLoadingMsg{
			TransactionID: int(transaction.ID),
		}
	}

	// Get recommendation
	recommendCmd := m.aiRecommender.RecommendCategory(transaction, m.categories)

	return tea.Batch(loadingCmd, recommendCmd)
}

// formatTransactionForAI formats transaction data for AI analysis.
func formatTransactionForAI(transaction *lm.Transaction) string {
	return fmt.Sprintf(`Transaction Details:
- Payee: %s
- Amount: %s
- Date: %s
- Notes: %s`,
		transaction.Payee,
		transaction.Amount,
		transaction.Date,
		transaction.Notes,
	)
}

// formatCategoriesForAI formats available categories for AI analysis.
func formatCategoriesForAI(categories []*lm.Category) string {
	var sb strings.Builder
	sb.WriteString("Available Categories:\n")
	for _, cat := range categories {
		fmt.Fprintf(&sb, "- ID: %d, Name: %s\n", cat.ID, cat.Name)
	}
	return sb.String()
}
