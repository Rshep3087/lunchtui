package main

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const unclearedStatus = "uncleared"
const clearedStatus string = "cleared"

func (m model) newItemDelegate(keys *delegateKeyMap) list.DefaultDelegate {
	d := list.NewDefaultDelegate()
	d.Styles.SelectedTitle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(lipgloss.AdaptiveColor{Light: "#ffe644", Dark: "#ffb744"}).
		Foreground(lipgloss.AdaptiveColor{Light: "#ffd644", Dark: "#ffd644"}).
		Padding(0, 0, 0, 1)

	d.Styles.SelectedDesc = lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: "#ffe644", Dark: "#ffb744"})

	d.UpdateFunc = func(msg tea.Msg, listModel *list.Model) tea.Cmd {
		if msg, ok := msg.(tea.KeyMsg); ok {
			if key.Matches(msg, keys.review) || key.Matches(msg, keys.unreview) {
				action := clearedStatus
				if key.Matches(msg, keys.unreview) {
					action = unclearedStatus
				}
				if ti, ok := listModel.SelectedItem().(transactionItem); ok {
					ti.t.Status = action
					return m.updateTransactionStatus(ti.t)
				}
			}
		}
		return nil
	}

	help := []key.Binding{keys.review, keys.unreview}

	d.ShortHelpFunc = func() []key.Binding {
		return help
	}

	d.FullHelpFunc = func() [][]key.Binding {
		return [][]key.Binding{help}
	}

	return d
}

type delegateKeyMap struct {
	review   key.Binding
	unreview key.Binding
}

func (d delegateKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		d.review,
		d.unreview,
	}
}

func (d delegateKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{
			d.review,
			d.unreview,
		},
	}
}

func newDeleteKeyMap() *delegateKeyMap {
	return &delegateKeyMap{
		review: key.NewBinding(
			key.WithKeys("R"),
			key.WithHelp("<shift-r>", "review"),
		),
		unreview: key.NewBinding(
			key.WithKeys("U"),
			key.WithHelp("<shift-u>", "unreview"),
		),
	}
}
