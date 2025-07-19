package main

import (
	"github.com/Rshep3087/lunchtui/config"
	"github.com/charmbracelet/lipgloss"
)

// Theme contains all the colors used throughout the application.
type Theme struct {
	Primary       lipgloss.Color
	Error         lipgloss.Color
	Success       lipgloss.Color
	Warning       lipgloss.Color
	Muted         lipgloss.Color
	Income        lipgloss.Color
	Expense       lipgloss.Color
	Border        lipgloss.Color
	Background    lipgloss.Color
	Text          lipgloss.Color
	SecondaryText lipgloss.Color
}

// newTheme creates a Theme from config.Colors.
func newTheme(colors config.Colors) Theme {
	return Theme{
		Primary:       parseColor(colors.Primary, "#ffd644"),
		Error:         parseColor(colors.Error, "#ff0000"),
		Success:       parseColor(colors.Success, "#22ba46"),
		Warning:       parseColor(colors.Warning, "#e05951"),
		Muted:         parseColor(colors.Muted, "#7f7d78"),
		Income:        parseColor(colors.Income, "#00ff00"),
		Expense:       parseColor(colors.Expense, "#ff0000"),
		Border:        parseColor(colors.Border, "#7D56F4"),
		Background:    parseColor(colors.Background, "#7D56F4"),
		Text:          parseColor(colors.Text, "#FAFAFA"),
		SecondaryText: parseColor(colors.SecondaryText, "#888888"),
	}
}

// parseColor parses a color string (hex or ANSI) and returns a lipgloss.Color
// Falls back to defaultColor if parsing fails or input is empty.
func parseColor(colorStr, defaultColor string) lipgloss.Color {
	if colorStr == "" {
		return lipgloss.Color(defaultColor)
	}
	// lipgloss.Color accepts both hex colors ("#ff0000") and ANSI codes ("21")
	// No additional parsing is needed - lipgloss handles both formats
	return lipgloss.Color(colorStr)
}
