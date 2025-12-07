package main

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/lipgloss"
)

type styles struct {
	docStyle   lipgloss.Style
	titleStyle lipgloss.Style
	errorStyle lipgloss.Style
}

func createStyles(theme Theme) styles {
	return styles{
		docStyle: lipgloss.NewStyle().Margin(1, standardMargin),
		titleStyle: lipgloss.NewStyle().Foreground(
			lipgloss.AdaptiveColor{Light: "#000000", Dark: string(theme.Primary)},
		).Bold(true),
		errorStyle: lipgloss.NewStyle().Foreground(theme.Error).Bold(true),
	}
}

func createHelpModel(theme Theme) help.Model {
	helpModel := help.New()
	helpModel.ShortSeparator = " + "
	helpModel.Styles = help.Styles{
		Ellipsis:       lipgloss.NewStyle().Foreground(theme.SecondaryText),
		ShortKey:       lipgloss.NewStyle().Foreground(theme.Primary).Bold(true),
		ShortDesc:      lipgloss.NewStyle().Foreground(theme.Text),
		ShortSeparator: lipgloss.NewStyle().Foreground(theme.SecondaryText),
		FullKey:        lipgloss.NewStyle().Foreground(theme.Primary).Bold(true),
		FullDesc:       lipgloss.NewStyle().Foreground(theme.Text),
		FullSeparator:  lipgloss.NewStyle().Foreground(theme.SecondaryText),
	}
	return helpModel
}
