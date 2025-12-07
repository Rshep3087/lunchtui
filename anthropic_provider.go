package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/charmbracelet/log"
	lm "github.com/icco/lunchmoney"
)

// AnthropicProvider implements AIProvider for Anthropic's Claude API.
type AnthropicProvider struct {
	client *anthropic.Client
}

// NewAnthropicProvider creates a new Anthropic AI provider.
func NewAnthropicProvider(apiKey string) *AnthropicProvider {
	client := anthropic.NewClient(
		option.WithAPIKey(apiKey),
	)

	return &AnthropicProvider{
		client: &client,
	}
}

// RecommendCategory implements AIProvider interface.
func (p *AnthropicProvider) RecommendCategory(
	ctx context.Context,
	transaction *lm.Transaction,
	categories []*lm.Category,
) (*CategoryRecommendation, error) {
	prompt := p.buildPrompt(transaction, categories)

	log.Debug(
		"sending categorization request to Anthropic",
		"transaction_id",
		transaction.ID,
		"payee",
		transaction.Payee,
	)

	response, err := p.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     "claude-3-haiku-20240307", // Use faster, cheaper model for categorization
		MaxTokens: anthropicMaxTokens,        // Keep response short and focused
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	})
	if err != nil {
		log.Error("failed to call Anthropic API", "error", err)
		return nil, fmt.Errorf("failed to call Anthropic API: %w", err)
	}

	// Extract text from response
	var responseText string
	if len(response.Content) > 0 {
		responseText = response.Content[0].Text
	}

	if responseText == "" {
		return nil, errors.New("empty response from Anthropic API")
	}

	recommendation, err := p.parseResponse(responseText, categories)
	if err != nil {
		log.Error("failed to parse Anthropic response", "error", err, "response", responseText)
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	log.Debug("received categorization recommendation",
		"category_id", recommendation.CategoryID,
		"confidence", recommendation.Confidence)
	return recommendation, nil
}

// buildPrompt constructs the prompt for category recommendation.
func (p *AnthropicProvider) buildPrompt(transaction *lm.Transaction, categories []*lm.Category) string {
	transactionInfo := formatTransactionForAI(transaction)
	categoriesInfo := formatCategoriesForAI(categories)

	return fmt.Sprintf(`You are a financial transaction categorization expert. 
Please analyze the following transaction and recommend the most appropriate category from the available options.

%s

%s

Please respond with ONLY a JSON object in this exact format:
{
  "category_id": <number>,
  "confidence": <number between 0-100>,
  "reasoning": "<brief explanation>"
}

Guidelines:
- Choose the category that best matches the transaction based on the payee, amount, and context
- Confidence should reflect how certain you are (100 = very certain, 50 = moderate, 0 = just guessing)
- Keep reasoning brief (1-2 sentences max)
- If no category seems appropriate, choose the closest match and set confidence low
- Consider common spending patterns and merchant categories`, transactionInfo, categoriesInfo)
}

// parseResponse parses the AI response and extracts the recommendation.
func (p *AnthropicProvider) parseResponse(response string, categories []*lm.Category) (*CategoryRecommendation, error) {
	// Clean up the response - remove any markdown formatting or extra text
	response = strings.TrimSpace(response)

	// Find JSON content between braces
	start := strings.Index(response, "{")
	end := strings.LastIndex(response, "}")

	if start == -1 || end == -1 {
		return nil, fmt.Errorf("no JSON found in response: %s", response)
	}

	jsonStr := response[start : end+1]

	var result struct {
		CategoryID int64   `json:"category_id"`
		Confidence float64 `json:"confidence"`
		Reasoning  string  `json:"reasoning"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		// Try to parse as string category_id in case the AI returned it as string
		var altResult struct {
			CategoryID string  `json:"category_id"`
			Confidence float64 `json:"confidence"`
			Reasoning  string  `json:"reasoning"`
		}
		if err2 := json.Unmarshal([]byte(jsonStr), &altResult); err2 != nil {
			return nil, fmt.Errorf("failed to parse JSON response: %w (original: %s)", err, jsonStr)
		}
		// Convert string to int64
		if id, parseErr := strconv.ParseInt(altResult.CategoryID, 10, 64); parseErr == nil {
			result.CategoryID = id
			result.Confidence = altResult.Confidence
			result.Reasoning = altResult.Reasoning
		} else {
			return nil, fmt.Errorf("invalid category_id format: %s", altResult.CategoryID)
		}
	}

	// Find the category name
	var categoryName string
	for _, cat := range categories {
		if cat.ID == result.CategoryID {
			categoryName = cat.Name
			break
		}
	}

	if categoryName == "" {
		return nil, fmt.Errorf("recommended category ID %d not found in available categories", result.CategoryID)
	}

	// Clamp confidence to 0-100 range
	if result.Confidence < 0 {
		result.Confidence = 0
	} else if result.Confidence > maxConfidenceScore {
		result.Confidence = maxConfidenceScore
	}

	return &CategoryRecommendation{
		CategoryID:   result.CategoryID,
		CategoryName: categoryName,
		Confidence:   result.Confidence,
		Reasoning:    result.Reasoning,
	}, nil
}
