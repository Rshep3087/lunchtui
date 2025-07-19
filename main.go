package main

import (
	"context"
	"time"

	configview "github.com/Rshep3087/lunchtui/config"
	"github.com/Rshep3087/lunchtui/overview"
	"github.com/Rshep3087/lunchtui/recurring"
	"github.com/spf13/viper"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/log"
	lm "github.com/icco/lunchmoney"
)

// Config represents the application configuration structure.
type Config struct {
	// Debug enables debug logging
	Debug bool `toml:"debug"`
	// Token is the Lunch Money API token
	Token string `toml:"token"`
	// DebitsAsNegative shows debits as negative numbers
	DebitsAsNegative bool `toml:"debits_as_negative"`
	// HidePendingTransactions hides pending transactions from all transaction lists
	HidePendingTransactions bool `toml:"hide_pending_transactions"`
	// Show UserInfo shows user information in the overview
	ShowUserInfo bool `toml:"show_user_info"`
	// Colors contains customizable color settings
	Colors configview.Colors `toml:"colors"`
}

type model struct {
	// config holds the application configuration
	config Config
	// theme contains the color theme
	theme Theme
	// loadingSpinner is a spinner model for the initial loading state
	loadingSpinner spinner.Model

	keys keyMap
	help help.Model

	overview overview.Model
	// sessionState is the current state of the session
	sessionState sessionState
	// errorMsg is the error message to display in the error state
	errorMsg string
	// previousSessionState is the state before the current session state
	previousSessionState sessionState
	// transactions is a bubbletea list model of financial transactions
	transactions list.Model
	// transactionsListKeys is the keybindings for the transactions list
	transactionsListKeys *transactionListKeyMap
	// inserteTransactionForm is the form for inserting a new transaction
	insertTransactionForm *huh.Form

	// period holds the current period for transactions
	period        Period
	currentPeriod time.Time
	// periodType is the type of range for the transactions
	// ex. month, year
	periodType string

	transactionsStats *transactionsStats
	// debitsAsNegative is a flag to show debits as negative numbers
	debitsAsNegative bool
	// hidePendingTransactions is a flag to hide pending transactions from all transaction lists
	hidePendingTransactions bool
	// originalTransactions stores the full list of transactions before filtering
	originalTransactions []list.Item
	// isFilteredUncleared tracks if the uncleared filter is currently applied
	isFilteredUncleared bool
	// currentTransaction holds the currently selected transaction for detailed view
	currentTransaction *transactionItem
	// notesInput is the text input for editing transaction notes
	notesInput textinput.Model
	// isEditingNotes indicates if the user is currently editing transaction notes
	isEditingNotes bool

	categoryForm *huh.Form
	// idToCategory is a map of category ID to category
	idToCategory map[int64]*lm.Category
	// categories is a list of categories
	categories []*lm.Category
	// plaidAccounts are individual bank accounts that you have linked to Lunch Money via Plaid.
	// You may link one bank but one bank might contain 4 accounts.
	// Each of these accounts is a Plaid Account.
	plaidAccounts map[int64]*lm.PlaidAccount
	// assets are manually managed assets
	assets map[int64]*lm.Asset
	// user is the current user determined by the API token
	user *lm.User

	tags map[int]*lm.Tag

	// recurringExpenses is a model for the recurring expenses widget
	recurringExpenses recurring.Model
	// budgets is a bubbletea list model of budgets
	budgets list.Model
	// configView is a model for the configuration view
	configView configview.Model
	// lmc is the Lunch Money client
	lmc *lm.Client

	loadingState loadingState
	styles       styles
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		m.getCategories,
		m.getUser,
		m.getAccounts,
		m.loadingSpinner.Tick,
		m.getRecurringExpenses,
		m.getTags,
		m.getBudgets,
	)
}

func rootAction(_ context.Context, config Config, lmc *lm.Client) error {
	debugEnabled := config.Debug
	if debugEnabled {
		f, err := tea.LogToFileWith("lunchtui.log", "lunchtui", log.Default())
		if err != nil {
			return err
		}
		defer f.Close()

		log.SetLevel(log.DebugLevel)
	}

	log.Debug("config file used", "config", viper.ConfigFileUsed())

	// Get debits-as-negative setting from config
	debitsAsNegative := config.DebitsAsNegative
	// Get hide-pending-transactions setting from config
	hidePendingTransactions := config.HidePendingTransactions

	tlKeyMap := newTransactionListKeyMap()
	theme := newTheme(config.Colors)

	m := model{
		config:                  config,
		theme:                   theme,
		keys:                    initializeKeyMap(),
		styles:                  createStyles(theme),
		help:                    createHelpModel(theme),
		sessionState:            loading,
		previousSessionState:    overviewState,
		lmc:                     lmc,
		transactionsListKeys:    tlKeyMap,
		debitsAsNegative:        debitsAsNegative,
		hidePendingTransactions: hidePendingTransactions,
		currentPeriod:           time.Now(),
		period:                  Period{},
		periodType:              "month",
		loadingSpinner:          spinner.New(spinner.WithSpinner(spinner.Dot)),
		overview: overview.New(
			overview.Config{
				ShowUserInfo: config.ShowUserInfo,
				Colors: &overview.Colors{
					Income:        theme.Income,
					Expense:       theme.Expense,
					TreeRoot:      theme.SecondaryText,
					AssetType:     theme.Muted,
					Account:       theme.Primary,
					SectionHeader: theme.Text,
				},
			},
		),
		recurringExpenses: recurring.New(recurring.Colors{
			Primary: string(theme.Primary),
		}),
		configView: configview.New(configview.Colors{
			Primary: string(theme.Primary),
		}),
		loadingState: newLoadingState(
			"categories",
			"transactions",
			"user",
			"accounts",
			"tags",
		),
	}

	delegate := m.newItemDelegate(newDeleteKeyMap())
	m.transactions = createTransactionList(delegate, tlKeyMap)
	m.budgets = createBudgetList(delegate)

	// Initialize text input for notes editing
	m.notesInput = textinput.New()
	m.notesInput.Placeholder = "Enter notes..."
	m.notesInput.CharLimit = 500

	// Initialize config view with current configuration
	configData := configview.Config{
		Debug:                   config.Debug,
		Token:                   config.Token,
		DebitsAsNegative:        config.DebitsAsNegative,
		HidePendingTransactions: config.HidePendingTransactions,
		Colors:                  config.Colors,
	}
	m.configView.SetConfig(configData)

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return err
	}

	return nil
}

func main() {
	Execute()
}

func createTransactionList(delegate list.DefaultDelegate, tlKeyMap *transactionListKeyMap) list.Model {
	transactionList := list.New([]list.Item{}, delegate, 0, 0)
	transactionList.SetShowTitle(false)
	transactionList.DisableQuitKeybindings()
	transactionList.StatusMessageLifetime = 3 * time.Second
	transactionList.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			tlKeyMap.categorizeTransaction,
			tlKeyMap.filterUncleared,
			tlKeyMap.refreshTransactions,
			tlKeyMap.insertTransaction,
		}
	}
	return transactionList
}

func (m model) checkIfLoading() sessionState {
	if m.sessionState != loading {
		return m.sessionState
	}

	if loaded, notLoaded := m.loadingState.allLoaded(); !loaded {
		log.Debugf("not loaded: %s", notLoaded)
		return loading
	}

	log.Debugf("all loaded showing %s", m.previousSessionState)
	return m.previousSessionState
}
