package main

import (
	"context"
	"os"
	"time"

	configview "github.com/rshep3087/lunchtui/config"
	"github.com/rshep3087/lunchtui/overview"
	"github.com/rshep3087/lunchtui/recurring"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/log"
	lm "github.com/icco/lunchmoney"
	"github.com/urfave/cli/v3"
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
}

type model struct {
	// config holds the application configuration
	config Config
	// loadingSpinner is a spinner model for the initial loading state
	loadingSpinner spinner.Model

	keys keyMap
	help help.Model

	overview overview.Model
	// transactionsListKeys is the keybindings for the transactions list
	transactionsListKeys *transactionListKeyMap
	// sessionState is the current state of the session
	sessionState sessionState
	// errorMsg is the error message to display in the error state
	errorMsg string
	// previousSessionState is the state before the current session state
	previousSessionState sessionState
	// transactions is a bubbletea list model of financial transactions
	transactions  list.Model
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

	categoryForm          *huh.Form
	insertTransactionForm *huh.Form
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

func rootAction(ctx context.Context, c *cli.Command) error {
	var config Config
	if cfg := ctx.Value(configContextKey); cfg != nil {
		var ok bool
		config, ok = cfg.(Config)
		if !ok {
			return cli.Exit("failed to assert config context value to Config", 1)
		}
	}

	debugEnabled := c.Bool("debug") || config.Debug
	if debugEnabled {
		f, err := tea.LogToFileWith("lunchtui.log", "lunchtui", log.Default())
		if err != nil {
			return err
		}
		defer f.Close()

		log.SetLevel(log.DebugLevel)
	}

	lmc, err := getClientFromContext(ctx)
	if err != nil {
		return err
	}

	// Get debits-as-negative setting from command line or config
	debitsAsNegative := c.Bool("debits-as-negative") || config.DebitsAsNegative
	// Get hide-pending-transactions setting from command line or config
	hidePendingTransactions := c.Bool("hide-pending-transactions") || config.HidePendingTransactions

	tlKeyMap := newTransactionListKeyMap()
	m := model{
		config:                  config,
		keys:                    initializeKeyMap(),
		styles:                  createStyles(),
		help:                    createHelpModel(),
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
		overview:                overview.New(),
		recurringExpenses:       recurring.New(),
		configView:              configview.New(),
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
	}
	m.configView.SetConfig(configData)

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err = p.Run(); err != nil {
		return err
	}

	return nil
}

func main() {
	app := createRootCommand()

	if err := app.Run(context.TODO(), os.Args); err != nil {
		log.Error("lunchtui ran into an error", "error", err)
		os.Exit(1)
	}
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
