package main

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

func (m model) newItemDelegate(keys *delegateKeyMap) list.DefaultDelegate {
	d := list.NewDefaultDelegate()

	d.UpdateFunc = func(msg tea.Msg, listModel *list.Model) tea.Cmd {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch {
			case key.Matches(msg, keys.review), key.Matches(msg, keys.unreview):
				action := "cleared"
				if key.Matches(msg, keys.unreview) {
					action = "uncleared"
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
			key.WithKeys("r"),
			key.WithHelp("r", "review"),
		),
		unreview: key.NewBinding(
			key.WithKeys("u"),
			key.WithHelp("u", "unreview"),
		),
	}
}
