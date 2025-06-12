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
	case loading:
		b.WriteString(fmt.Sprintf("%s Loading data...", m.loadingSpinner.View()))
	}

	b.WriteString("\n\n")
	b.WriteString(m.help.View(m.keys))

	return m.styles.docStyle.Render(b.String())
}

func (m model) renderTitle() string {
	var b strings.Builder

	var currentPage string
	switch m.sessionState {
	case overviewState:
		currentPage = "overview"
	case transactions:
		currentPage = "transactions"
	case detailedTransaction:
		currentPage = "transaction details"
	case categorizeTransaction:
		currentPage = "categorize transaction"
	case recurringExpenses:
		currentPage = "recurring expenses"
	case loading:
		currentPage = "loading"
	}

	if m.period.String() == "" {
		b.WriteString(m.styles.titleStyle.Render(fmt.Sprintf("lunchtui | %s", currentPage)))
		return b.String()
	}

	b.WriteString(m.styles.titleStyle.Render(
		fmt.Sprintf("lunchtui | %s | %s | %s",
			currentPage,
			m.period.String(),
			m.periodType,
		),
	))

	return b.String()
}
