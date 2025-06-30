package main

import (
	"fmt"
	"strings"
)

func (m model) View() string {
	var b strings.Builder

	b.WriteString(m.renderTitle())
	b.WriteString("\n\n")

	switch m.sessionState {
	case overviewState:
		b.WriteString(m.overview.View())
	case transactions:
		b.WriteString(transactionsView(m))
	case detailedTransaction:
		b.WriteString(detailedTransactionView(m))
	case categorizeTransaction:
		b.WriteString(categorizeTransactionView(m))
	case recurringExpenses:
		b.WriteString(m.recurringExpenses.View())
	case budgets:
		b.WriteString(budgetsView(m))
	case configView:
		b.WriteString(m.configView.View())
	case loading:
		b.WriteString(fmt.Sprintf("%s Loading data...", m.loadingSpinner.View()))
	case errorState:
		b.WriteString(m.styles.errorStyle.Render(fmt.Sprintf("%s - 'q' to quit", m.errorMsg)))
		return m.styles.docStyle.Render(b.String())
	}

	b.WriteString("\n\n")
	b.WriteString(m.help.View(m.keys))

	return m.styles.docStyle.Render(b.String())
}

func (m model) renderTitle() string {
	var b strings.Builder

	if m.period.String() == "" {
		b.WriteString(m.styles.titleStyle.Render(fmt.Sprintf("lunchtui | %s", m.sessionState.String())))
		return b.String()
	}

	b.WriteString(m.styles.titleStyle.Render(
		fmt.Sprintf("lunchtui | %s | %s | %s",
			m.sessionState.String(),
			m.period.String(),
			m.periodType,
		),
	))

	return b.String()
}
