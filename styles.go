package main

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/lipgloss"
)

type styles struct {
	docStyle   lipgloss.Style
	titleStyle lipgloss.Style
}

func createStyles() styles {
	return styles{
		docStyle:   lipgloss.NewStyle().Margin(1, 2),
		titleStyle: lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#000000", Dark: "#ffd644"}).Bold(true),
	}
}

func createHelpModel() help.Model {
	helpModel := help.New()
	helpModel.ShortSeparator = " + "
	helpModel.Styles = help.Styles{
		Ellipsis:       lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")),
		ShortKey:       lipgloss.NewStyle().Foreground(lipgloss.Color("#ffd644")).Bold(true),
		ShortDesc:      lipgloss.NewStyle().Foreground(lipgloss.Color("#ffffff")),
		ShortSeparator: lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")),
		FullKey:        lipgloss.NewStyle().Foreground(lipgloss.Color("#ffd644")).Bold(true),
		FullDesc:       lipgloss.NewStyle().Foreground(lipgloss.Color("#ffffff")),
		FullSeparator:  lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")),
	}
	return helpModel
}
