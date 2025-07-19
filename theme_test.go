package main

import (
	"testing"

	"github.com/Rshep3087/lunchtui/config"
	"github.com/charmbracelet/lipgloss"
)

func TestNewTheme(t *testing.T) {
	colors := config.Colors{
		Primary:       "#ff0000",
		Error:         "21",
		Success:       "#00ff00",
		Warning:       "33",
		Muted:         "#888888",
		Income:        "#00cc00",
		Expense:       "#cc0000",
		Border:        "#0000ff",
		Background:    "#333333",
		Text:          "#ffffff",
		SecondaryText: "245",
	}

	theme := newTheme(colors)

	// Test that colors are properly parsed
	if theme.Primary != lipgloss.Color("#ff0000") {
		t.Errorf("Theme.Primary = %v, want %v", theme.Primary, lipgloss.Color("#ff0000"))
	}

	if theme.Error != lipgloss.Color("21") {
		t.Errorf("Theme.Error = %v, want %v", theme.Error, lipgloss.Color("21"))
	}

	if theme.SecondaryText != lipgloss.Color("245") {
		t.Errorf("Theme.SecondaryText = %v, want %v", theme.SecondaryText, lipgloss.Color("245"))
	}
}

func TestParseColor(t *testing.T) {
	tests := []struct {
		name         string
		colorStr     string
		defaultColor string
		expected     lipgloss.Color
	}{
		{"hex color", "#ff0000", "#000000", lipgloss.Color("#ff0000")},
		{"ansi color", "21", "#000000", lipgloss.Color("21")},
		{"empty string", "", "#000000", lipgloss.Color("#000000")},
		{"invalid color still accepted", "invalid", "#000000", lipgloss.Color("invalid")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseColor(tt.colorStr, tt.defaultColor)
			if result != tt.expected {
				t.Errorf("parseColor(%q, %q) = %v, want %v", tt.colorStr, tt.defaultColor, result, tt.expected)
			}
		})
	}
}
