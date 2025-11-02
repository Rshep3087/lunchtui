package main

import (
	"context"
	"time"

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

func (cs *CategoryService) GetCategories() tea.Msg {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ca, err := cs.categoryGetter.GetCategories(ctx)
	if err != nil {
		if is401Error(err) {
			return handleAuthError(err)
		}
		return nil
	}

	return getCategoriesMsg{categories: ca}
}
