package main

import (
	"context"

	lm "github.com/icco/lunchmoney"

	tea "github.com/charmbracelet/bubbletea"
)

type CategoryService struct {
	categoryGetter categoriesGetter
}

func NewCategoryService(categoryGetter categoriesGetter) *CategoryService {
	return &CategoryService{
		categoryGetter: categoryGetter,
	}
}

// GetCategories fetches categories with the provided context.
func (cs *CategoryService) GetCategories(ctx context.Context) ([]*lm.Category, error) {
	return cs.categoryGetter.GetCategories(ctx)
}

// GetCategoriesCmd fetches categories asynchronously for TUI use.
func (cs *CategoryService) GetCategoriesCmd() tea.Msg {
	ctx, cancel := context.WithTimeout(context.Background(), categoryServiceTimeout)
	defer cancel()

	ca, err := cs.GetCategories(ctx)
	if err != nil {
		if is401Error(err) {
			return handleAuthError(err)
		}
		return nil
	}

	return getCategoriesMsg{categories: ca}
}
