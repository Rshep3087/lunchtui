package main

import (
	"context"
	"os"
	"time"

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

type model struct {
	// loadingSpinner is a spinner model for the initial loading state
	loadingSpinner spinner.Model

	keys keyMap
	help help.Model

	overview overview.Model
	// transactionsListKeys is the keybindings for the transactions list
	transactionsListKeys *transactionListKeyMap
	// sessionState is the current state of the session
	sessionState sessionState
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
		m.recurringExpenses.Init(),
		m.getTags,
		m.getBudgets,
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// always check for quit key first
	if msg, ok := msg.(tea.KeyMsg); ok {
		if model, cmd := handleKeyPress(msg, &m); cmd != nil {
			log.Debug("key press handled, cmd returned")
			return model, cmd
		}
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.handleWindowSize(msg)

	case spinner.TickMsg:
		return m.handleSpinnerTick(msg)

	// set the categories on the model,
	// send a cmd to get transactions
	case getCategoriesMsg:
		return m.handleGetCategories(msg)

	case getAccountsMsg:
		return m.handleGetAccounts(msg)

	case getsTransactionsMsg:
		return m.handleGetTransactions(msg)

	case getUserMsg:
		return m.handleGetUser(msg)

	case getRecurringExpensesMsg:
		m.recurringExpenses.SetRecurringExpenses(msg.recurringExpenses)
		return m, nil

	case getTagsMsg:
		return m.handleGetTags(msg)

	case getBudgetsMsg:
		return m.handleGetBudgets(msg)
	}

	var cmd tea.Cmd
	switch m.sessionState {
	case overviewState:
		m.overview, cmd = m.overview.Update(msg)
		return m, cmd

	case categorizeTransaction:
		return updateCategorizeTransaction(msg, &m)

	case detailedTransaction:
		return updateDetailedTransaction(msg, m)

	case transactions:
		return updateTransactions(msg, m)

	case recurringExpenses:
		m.recurringExpenses, cmd = m.recurringExpenses.Update(msg)
		return m, cmd

	case budgets:
		return updateBudgets(msg, m)

	case loading:
		m.loadingSpinner, cmd = m.loadingSpinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

func rootAction(ctx context.Context, c *cli.Command) error {
	var config *Config
	if cfg := ctx.Value(configContextKey); cfg != nil {
		var ok bool
		config, ok = cfg.(*Config)
		if !ok {
			return cli.Exit("failed to assert config context value to *Config", 1)
		}

	}

	debugEnabled := c.Bool("debug") || (config != nil && config.Debug)
	if debugEnabled {
		f, err := tea.LogToFileWith("lunchtui.log", "lunchtui", log.Default())
		if err != nil {
			return err
		}
		defer f.Close()

		log.SetLevel(log.DebugLevel)
		if config != nil {
			log.Debug("Debug logging enabled", "configPath", config.configPathUsed)
		}
	}

	lmc, err := getClientFromContext(ctx)
	if err != nil {
		return err
	}

	// Get debits-as-negative setting from command line or config
	debitsAsNegative := c.Bool("debits-as-negative") || (config != nil && config.DebitsAsNegative)

	tlKeyMap := newTransactionListKeyMap()
	m := model{
		keys:                 initializeKeyMap(),
		styles:               createStyles(),
		help:                 createHelpModel(),
		sessionState:         loading,
		previousSessionState: overviewState,
		lmc:                  lmc,
		transactionsListKeys: tlKeyMap,
		debitsAsNegative:     debitsAsNegative,
		currentPeriod:        time.Now(),
		period:               Period{},
		periodType:           "month",
		loadingSpinner:       spinner.New(spinner.WithSpinner(spinner.Dot)),
		overview:             overview.New(),
		recurringExpenses:    recurring.New(),
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
