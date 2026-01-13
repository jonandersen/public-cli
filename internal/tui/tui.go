package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/jonandersen/public-cli/internal/api"
	"github.com/jonandersen/public-cli/internal/config"
	"github.com/jonandersen/public-cli/internal/keyring"
)

// View represents the current active view in the TUI.
type View int

const (
	ViewPortfolio View = iota
	ViewWatchlist
	ViewOrders
	ViewTrade
	ViewOptions
	ViewHistory
)

// Model is the main bubbletea model for the TUI.
type Model struct {
	currentView View
	width       int
	height      int
	ready       bool

	// Config and auth
	cfg   *config.Config
	uiCfg *UIConfig
	store keyring.Store

	// Child view models
	portfolio *PortfolioModel
	watchlist *WatchlistModel
	orders    *OrdersModel
	trade     *TradeModel
	options   *OptionsModel
	history   *HistoryModel

	// Refresh settings
	refreshInterval time.Duration

	// Account switcher
	accounts          []Account
	selectedAccountID string
	accountPickerOpen bool
	accountCursor     int

	// Toolbar navigation
	toolbarFocused bool
}

// New creates a new TUI model.
func New(cfg *config.Config, uiCfg *UIConfig, store keyring.Store) Model {
	return Model{
		currentView:       ViewPortfolio,
		cfg:               cfg,
		uiCfg:             uiCfg,
		store:             store,
		portfolio:         NewPortfolioModel(),
		watchlist:         NewWatchlistModel(uiCfg.Watchlist),
		orders:            NewOrdersModel(),
		trade:             NewTradeModel(),
		options:           NewOptionsModel(),
		history:           NewHistoryModel(),
		refreshInterval:   30 * time.Second,
		selectedAccountID: cfg.AccountUUID,
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{
		FetchAccounts(m.cfg, m.store),
		FetchPortfolio(m.cfg, m.store),
		m.tickCmd(),
	}
	if len(m.watchlist.Symbols) > 0 {
		cmds = append(cmds, FetchWatchlistQuotes(m.watchlist.Symbols, m.cfg, m.store))
	}
	return tea.Batch(cmds...)
}

// tickCmd returns a command that sends a tick message after the refresh interval.
func (m Model) tickCmd() tea.Cmd {
	return tea.Tick(m.refreshInterval, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle account picker first - it takes priority
		if m.accountPickerOpen {
			switch msg.String() {
			case "esc", "a":
				m.accountPickerOpen = false
				return m, nil
			case "up", "k":
				if m.accountCursor > 0 {
					m.accountCursor--
				}
				return m, nil
			case "down", "j":
				if m.accountCursor < len(m.accounts)-1 {
					m.accountCursor++
				}
				return m, nil
			case "enter":
				if len(m.accounts) > 0 && m.accountCursor < len(m.accounts) {
					newAccountID := m.accounts[m.accountCursor].AccountID
					if newAccountID != m.selectedAccountID {
						m.selectedAccountID = newAccountID
						m.cfg.AccountUUID = newAccountID
						m.accountPickerOpen = false
						// Refresh data for the new account
						return m, m.refreshCurrentView()
					}
				}
				m.accountPickerOpen = false
				return m, nil
			case "q", "ctrl+c":
				return m, tea.Quit
			}
			return m, nil
		}

		// Handle toolbar navigation - takes priority when toolbar is focused
		if m.toolbarFocused {
			switch msg.String() {
			case "left", "h":
				// Move to previous tab
				if m.currentView > ViewPortfolio {
					m.currentView--
				} else {
					m.currentView = ViewOptions // Wrap around
				}
				return m, nil
			case "right", "l":
				// Move to next tab
				if m.currentView < ViewOptions {
					m.currentView++
				} else {
					m.currentView = ViewPortfolio // Wrap around
				}
				return m, nil
			case "down", "j", "enter":
				// Exit toolbar, focus content
				m.toolbarFocused = false
				return m, nil
			case "esc":
				// Esc in toolbar goes back to content
				m.toolbarFocused = false
				return m, nil
			case "1":
				m.currentView = ViewPortfolio
				m.toolbarFocused = false
				return m, nil
			case "2":
				m.currentView = ViewWatchlist
				m.toolbarFocused = false
				if m.watchlist.State == WatchlistStateLoading && len(m.watchlist.Symbols) > 0 {
					return m, FetchWatchlistQuotes(m.watchlist.Symbols, m.cfg, m.store)
				}
				return m, nil
			case "3":
				m.currentView = ViewOrders
				m.toolbarFocused = false
				if m.orders.State == OrdersStateLoading {
					return m, FetchOrders(m.cfg, m.store)
				}
				return m, nil
			case "4":
				m.currentView = ViewTrade
				m.toolbarFocused = false
				return m, nil
			case "5":
				m.currentView = ViewOptions
				m.toolbarFocused = false
				return m, nil
			case "6":
				m.currentView = ViewHistory
				m.toolbarFocused = false
				if m.history.State == HistoryStateLoading {
					return m, FetchHistory(m.cfg, m.store)
				}
				return m, nil
			case "q", "ctrl+c":
				return m, tea.Quit
			}
			return m, nil
		}

		// Handle watchlist input modes first - they consume all keys
		if m.currentView == ViewWatchlist && m.watchlist.Mode != WatchlistModeNormal {
			m.watchlist, cmd, _ = m.watchlist.Update(msg, m.uiCfg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			// After exiting add mode, fetch quotes if we added something
			if m.watchlist.Mode == WatchlistModeNormal && len(m.watchlist.Symbols) > 0 {
				cmds = append(cmds, FetchWatchlistQuotes(m.watchlist.Symbols, m.cfg, m.store))
			}
			return m, tea.Batch(cmds...)
		}

		// Handle orders cancel mode - consumes all keys
		if m.currentView == ViewOrders && m.orders.Mode != OrdersModeNormal {
			m.orders, cmd, _ = m.orders.Update(msg, m.cfg, m.store)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			return m, tea.Batch(cmds...)
		}

		// Handle trade view - it manages its own input
		if m.currentView == ViewTrade {
			// When not focused on a text field, allow global keys
			if !m.trade.IsTextFieldFocused() && m.trade.Mode == TradeModeForm && m.trade.State != TradeStateSuccess {
				switch msg.String() {
				case "q", "ctrl+c":
					return m, tea.Quit
				case "esc":
					// Esc focuses the toolbar for navigation
					m.toolbarFocused = true
					return m, nil
				case "1":
					m.currentView = ViewPortfolio
					return m, nil
				case "2":
					m.currentView = ViewWatchlist
					if m.watchlist.State == WatchlistStateLoading && len(m.watchlist.Symbols) > 0 {
						cmds = append(cmds, FetchWatchlistQuotes(m.watchlist.Symbols, m.cfg, m.store))
					}
					return m, tea.Batch(cmds...)
				case "3":
					m.currentView = ViewOrders
					if m.orders.State == OrdersStateLoading {
						cmds = append(cmds, FetchOrders(m.cfg, m.store))
					}
					return m, tea.Batch(cmds...)
				case "5":
					m.currentView = ViewOptions
					return m, nil
				}
			}
			// Pass input to trade model
			m.trade, cmd = m.trade.Update(msg, m.cfg, m.store)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			return m, tea.Batch(cmds...)
		}

		// Handle options view - it manages its own input
		if m.currentView == ViewOptions {
			// Pass input to options model
			m.options, cmd = m.options.Update(msg, m.cfg, m.store)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			return m, tea.Batch(cmds...)
		}

		// Handle history view - detail panel consumes all keys
		if m.currentView == ViewHistory && m.history.ShowDetail {
			m.history, cmd, _ = m.history.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			return m, tea.Batch(cmds...)
		}

		// Handle global keys
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "esc":
			// Esc focuses the toolbar for navigation
			m.toolbarFocused = true
			return m, nil
		case "a":
			// On watchlist view, 'a' is for adding symbols, not account picker
			// Let watchlist handle it in the default case below
			if m.currentView == ViewWatchlist && m.watchlist.Mode == WatchlistModeNormal {
				// Fall through to default case
			} else if len(m.accounts) > 0 {
				// Open account picker if we have accounts
				m.accountPickerOpen = true
				// Position cursor on current account
				m.accountCursor = 0
				for i, acc := range m.accounts {
					if acc.AccountID == m.selectedAccountID {
						m.accountCursor = i
						break
					}
				}
				return m, nil
			}
		case "1":
			m.currentView = ViewPortfolio
		case "2":
			m.currentView = ViewWatchlist
			if m.watchlist.State == WatchlistStateLoading && len(m.watchlist.Symbols) > 0 {
				cmds = append(cmds, FetchWatchlistQuotes(m.watchlist.Symbols, m.cfg, m.store))
			}
		case "3":
			m.currentView = ViewOrders
			if m.orders.State == OrdersStateLoading {
				cmds = append(cmds, FetchOrders(m.cfg, m.store))
			}
		case "4":
			m.currentView = ViewTrade
		case "5":
			m.currentView = ViewOptions
		case "6":
			m.currentView = ViewHistory
			if m.history.State == HistoryStateLoading {
				cmds = append(cmds, FetchHistory(m.cfg, m.store))
			}
		case "r":
			// Manual refresh
			switch m.currentView {
			case ViewPortfolio:
				m.portfolio.State = PortfolioStateLoading
				cmds = append(cmds, FetchPortfolio(m.cfg, m.store))
			case ViewWatchlist:
				m.watchlist.State = WatchlistStateLoading
				cmds = append(cmds, FetchWatchlistQuotes(m.watchlist.Symbols, m.cfg, m.store))
			case ViewOrders:
				m.orders.State = OrdersStateLoading
				cmds = append(cmds, FetchOrders(m.cfg, m.store))
			case ViewHistory:
				m.history.State = HistoryStateLoading
				cmds = append(cmds, FetchHistory(m.cfg, m.store))
			}
		case "enter":
			// Jump to trade from watchlist
			if m.currentView == ViewWatchlist && m.watchlist.Mode == WatchlistModeNormal {
				symbol := m.watchlist.SelectedSymbol()
				if symbol != "" {
					m.trade.SetSymbol(symbol)
					m.currentView = ViewTrade
					// Fetch quote for the symbol
					cmds = append(cmds, FetchTradeQuote(symbol, m.cfg, m.store))
				}
			} else if m.currentView == ViewHistory {
				// Show detail for selected transaction
				if len(m.history.Transactions) > 0 {
					m.history.DetailIndex = m.history.Table.Cursor()
					m.history.ShowDetail = true
				}
			}
		default:
			// Pass to active view for view-specific keys
			switch m.currentView {
			case ViewWatchlist:
				m.watchlist, cmd, _ = m.watchlist.Update(msg, m.uiCfg)
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			case ViewOrders:
				m.orders, cmd, _ = m.orders.Update(msg, m.cfg, m.store)
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			case ViewHistory:
				m.history, cmd, _ = m.history.Update(msg)
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		// Resize tables to fit content area
		headerHeight := 1
		footerHeight := 1
		summaryHeight := 5 // Account summary section
		tableHeight := m.height - headerHeight - footerHeight - summaryHeight - 4
		if tableHeight < 3 {
			tableHeight = 3
		}
		m.portfolio.SetHeight(tableHeight)
		m.watchlist.SetHeight(tableHeight)
		m.orders.SetHeight(tableHeight)
		m.options.SetHeight(tableHeight)
		m.history.SetHeight(tableHeight)

	case PortfolioLoadedMsg, PortfolioErrorMsg:
		m.portfolio, cmd = m.portfolio.Update(msg)
		cmds = append(cmds, cmd)

	case WatchlistQuotesMsg, WatchlistErrorMsg, WatchlistSavedMsg:
		m.watchlist, cmd, _ = m.watchlist.Update(msg, m.uiCfg)
		cmds = append(cmds, cmd)

	case OrdersLoadedMsg, OrdersErrorMsg, OrderCancelledMsg, OrderCancelErrorMsg:
		m.orders, cmd, _ = m.orders.Update(msg, m.cfg, m.store)
		cmds = append(cmds, cmd)

	case TradeQuoteMsg, TradeQuoteErrorMsg, TradeOrderPlacedMsg, TradeOrderErrorMsg:
		m.trade, cmd = m.trade.Update(msg, m.cfg, m.store)
		cmds = append(cmds, cmd)

	case OptionExpirationsLoadedMsg, OptionExpirationsErrorMsg, OptionChainLoadedMsg, OptionChainErrorMsg, OptionGreeksLoadedMsg, OptionQuoteLoadedMsg:
		m.options, cmd = m.options.Update(msg, m.cfg, m.store)
		cmds = append(cmds, cmd)

	case HistoryLoadedMsg, HistoryErrorMsg:
		m.history, cmd, _ = m.history.Update(msg)
		cmds = append(cmds, cmd)

	case AccountsLoadedMsg:
		m.accounts = msg.Accounts
		// If no account is selected but we have accounts, select the first one
		if m.selectedAccountID == "" && len(m.accounts) > 0 {
			m.selectedAccountID = m.accounts[0].AccountID
			m.cfg.AccountUUID = m.selectedAccountID
		}

	case AccountsErrorMsg:
		// Silently ignore account loading errors - we still have the default account
		_ = msg.Err

	case ToolbarFocusMsg:
		// Child view requested toolbar focus
		m.toolbarFocused = true

	case TickMsg:
		// Auto-refresh based on current view
		if m.currentView == ViewPortfolio && m.portfolio.State != PortfolioStateLoading {
			cmds = append(cmds, FetchPortfolio(m.cfg, m.store))
		} else if m.currentView == ViewWatchlist && m.watchlist.State != WatchlistStateLoading && len(m.watchlist.Symbols) > 0 {
			cmds = append(cmds, FetchWatchlistQuotes(m.watchlist.Symbols, m.cfg, m.store))
		} else if m.currentView == ViewOrders && m.orders.State != OrdersStateLoading {
			cmds = append(cmds, FetchOrders(m.cfg, m.store))
		}
		cmds = append(cmds, m.tickCmd())
	}

	// Update active table for navigation keys
	switch m.currentView {
	case ViewPortfolio:
		m.portfolio, cmd = m.portfolio.Update(msg)
		cmds = append(cmds, cmd)
	case ViewOrders:
		m.orders, cmd, _ = m.orders.Update(msg, m.cfg, m.store)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View implements tea.Model.
func (m Model) View() string {
	if !m.ready {
		return "Loading..."
	}

	header := m.renderHeader()
	footer := m.renderFooter()

	// Show account picker in content area if open
	var content string
	if m.accountPickerOpen {
		content = m.renderAccountPicker()
	} else {
		content = m.renderContent()
	}

	// Calculate content height
	headerHeight := lipgloss.Height(header)
	footerHeight := lipgloss.Height(footer)
	contentHeight := m.height - headerHeight - footerHeight

	// Pad content to fill available space
	contentLines := strings.Split(content, "\n")
	for len(contentLines) < contentHeight {
		contentLines = append(contentLines, "")
	}
	if len(contentLines) > contentHeight {
		contentLines = contentLines[:contentHeight]
	}
	content = strings.Join(contentLines, "\n")

	return header + "\n" + content + "\n" + footer
}

// renderHeader renders the header bar.
func (m Model) renderHeader() string {
	title := HeaderStyle.Render("pub")

	tabs := []struct {
		name   string
		key    string
		active bool
	}{
		{"Portfolio", "1", m.currentView == ViewPortfolio},
		{"Watchlist", "2", m.currentView == ViewWatchlist},
		{"Orders", "3", m.currentView == ViewOrders},
		{"Trade", "4", m.currentView == ViewTrade},
		{"Options", "5", m.currentView == ViewOptions},
		{"History", "6", m.currentView == ViewHistory},
	}

	var tabStrs []string
	for _, tab := range tabs {
		style := lipgloss.NewStyle().Padding(0, 1)
		if tab.active {
			if m.toolbarFocused {
				// Inverted colors when toolbar is focused
				style = style.Bold(true).
					Foreground(lipgloss.Color("#000000")).
					Background(ColorPrimary)
			} else {
				style = style.Bold(true).Foreground(ColorPrimary)
			}
		} else {
			if m.toolbarFocused {
				// Slightly brighter when toolbar is focused
				style = style.Foreground(lipgloss.Color("#888888"))
			} else {
				style = style.Foreground(ColorMuted)
			}
		}
		tabStrs = append(tabStrs, style.Render(fmt.Sprintf("[%s] %s", tab.key, tab.name)))
	}

	tabBar := strings.Join(tabStrs, " ")

	// Build account indicator
	accountIndicator := ""
	if m.selectedAccountID != "" {
		// Find the account type for display
		accType := ""
		for _, acc := range m.accounts {
			if acc.AccountID == m.selectedAccountID {
				accType = acc.BrokerageAccountType
				if accType == "" {
					accType = acc.AccountType
				}
				break
			}
		}
		// Truncate account ID for display
		displayID := m.selectedAccountID
		if len(displayID) > 8 {
			displayID = displayID[:8] + "..."
		}
		if accType != "" {
			accountIndicator = fmt.Sprintf("[a] %s (%s)", displayID, accType)
		} else {
			accountIndicator = fmt.Sprintf("[a] %s", displayID)
		}
	} else if len(m.accounts) > 0 {
		accountIndicator = "[a] Select account"
	}

	accountStyle := lipgloss.NewStyle().Padding(0, 1).Foreground(ColorMuted)
	if m.accountPickerOpen {
		accountStyle = accountStyle.Foreground(ColorPrimary).Bold(true)
	}
	accountStr := accountStyle.Render(accountIndicator)

	headerContent := title + "  " + tabBar

	// Calculate available space for account indicator on the right
	leftWidth := lipgloss.Width(headerContent)
	rightWidth := lipgloss.Width(accountStr)
	padding := m.width - leftWidth - rightWidth
	if padding > 0 {
		headerContent += strings.Repeat(" ", padding) + accountStr
	} else {
		// Not enough space, just pad to full width
		padding = m.width - leftWidth
		if padding > 0 {
			headerContent += strings.Repeat(" ", padding)
		}
	}

	return lipgloss.NewStyle().
		Background(ColorBackground).
		Width(m.width).
		Render(headerContent)
}

// renderContent renders the main content area.
func (m Model) renderContent() string {
	var content string
	switch m.currentView {
	case ViewPortfolio:
		content = m.portfolio.View()
	case ViewWatchlist:
		content = m.watchlist.View()
	case ViewOrders:
		content = m.orders.View()
	case ViewTrade:
		content = m.trade.View()
	case ViewOptions:
		content = m.options.View()
	case ViewHistory:
		content = m.history.View()
	}
	return ContentStyle.Render(content)
}

// renderFooter renders the footer bar with key hints.
func (m Model) renderFooter() string {
	keys := []struct {
		key  string
		desc string
	}{}

	// Account picker has its own keys
	if m.accountPickerOpen {
		keys = []struct{ key, desc string }{
			{"↑/↓", "navigate"},
			{"enter", "select"},
			{"esc", "close"},
			{"q", "quit"},
		}

		var parts []string
		for _, k := range keys {
			parts = append(parts, KeyStyle.Render(k.key)+" "+DescStyle.Render(k.desc))
		}

		footerContent := strings.Join(parts, "  •  ")

		// Pad to full width
		padding := m.width - lipgloss.Width(footerContent)
		if padding > 0 {
			footerContent += strings.Repeat(" ", padding)
		}

		return lipgloss.NewStyle().
			Background(ColorBackground).
			Width(m.width).
			Render(footerContent)
	}

	// Toolbar navigation has its own keys
	if m.toolbarFocused {
		keys = []struct{ key, desc string }{
			{"←/→", "switch tab"},
			{"↓/enter", "focus content"},
			{"1-5", "jump to tab"},
			{"q", "quit"},
		}

		var parts []string
		for _, k := range keys {
			parts = append(parts, KeyStyle.Render(k.key)+" "+DescStyle.Render(k.desc))
		}

		footerContent := strings.Join(parts, "  •  ")

		// Pad to full width
		padding := m.width - lipgloss.Width(footerContent)
		if padding > 0 {
			footerContent += strings.Repeat(" ", padding)
		}

		return lipgloss.NewStyle().
			Background(ColorBackground).
			Width(m.width).
			Render(footerContent)
	}

	keys = append(keys, struct{ key, desc string }{"1-6", "switch view"})

	// Add view-specific keys
	switch m.currentView {
	case ViewPortfolio:
		keys = append(keys, struct{ key, desc string }{"↑/↓", "navigate"})
		keys = append(keys, struct{ key, desc string }{"esc", "toolbar"})
		keys = append(keys, struct{ key, desc string }{"r", "refresh"})
	case ViewWatchlist:
		switch m.watchlist.Mode {
		case WatchlistModeNormal:
			keys = append(keys, struct{ key, desc string }{"↑/↓", "navigate"})
			keys = append(keys, struct{ key, desc string }{"a", "add"})
			keys = append(keys, struct{ key, desc string }{"d", "delete"})
			keys = append(keys, struct{ key, desc string }{"enter", "trade"})
			keys = append(keys, struct{ key, desc string }{"esc", "toolbar"})
			keys = append(keys, struct{ key, desc string }{"r", "refresh"})
		case WatchlistModeAdding:
			keys = []struct{ key, desc string }{
				{"enter", "add"},
				{"esc", "cancel"},
			}
		case WatchlistModeDeleting:
			keys = []struct{ key, desc string }{
				{"y", "confirm"},
				{"n", "cancel"},
			}
		}
	case ViewOrders:
		switch m.orders.Mode {
		case OrdersModeNormal:
			keys = append(keys, struct{ key, desc string }{"↑/↓", "navigate"})
			keys = append(keys, struct{ key, desc string }{"c", "cancel order"})
			keys = append(keys, struct{ key, desc string }{"esc", "toolbar"})
			keys = append(keys, struct{ key, desc string }{"r", "refresh"})
		case OrdersModeCanceling:
			keys = []struct{ key, desc string }{
				{"y", "confirm"},
				{"n", "cancel"},
			}
		}
	case ViewTrade:
		switch m.trade.Mode {
		case TradeModeForm:
			if m.trade.State == TradeStateSuccess {
				keys = []struct{ key, desc string }{
					{"ctrl+n", "new order"},
				}
			} else if m.trade.IsTextFieldFocused() {
				keys = []struct{ key, desc string }{
					{"tab", "next field"},
					{"enter", "submit"},
					{"esc", "toolbar"},
				}
			} else {
				keys = append(keys, struct{ key, desc string }{"tab", "next field"})
				keys = append(keys, struct{ key, desc string }{"space", "toggle"})
				keys = append(keys, struct{ key, desc string }{"esc", "toolbar"})
				keys = append(keys, struct{ key, desc string }{"enter", "submit"})
			}
		case TradeModeConfirm:
			keys = []struct{ key, desc string }{
				{"y", "confirm"},
				{"n", "cancel"},
			}
		}
	case ViewOptions:
		keys = m.options.FooterKeys(keys)
	case ViewHistory:
		keys = m.history.FooterKeys(keys)
	}

	keys = append(keys, struct{ key, desc string }{"q", "quit"})

	var parts []string
	for _, k := range keys {
		parts = append(parts, KeyStyle.Render(k.key)+" "+DescStyle.Render(k.desc))
	}

	footerContent := strings.Join(parts, "  •  ")

	// Pad to full width
	padding := m.width - lipgloss.Width(footerContent)
	if padding > 0 {
		footerContent += strings.Repeat(" ", padding)
	}

	return lipgloss.NewStyle().
		Background(ColorBackground).
		Width(m.width).
		Render(footerContent)
}

// refreshCurrentView returns a command to refresh data for the current view.
func (m Model) refreshCurrentView() tea.Cmd {
	var cmds []tea.Cmd

	// Always refresh portfolio for account changes
	m.portfolio.State = PortfolioStateLoading
	cmds = append(cmds, FetchPortfolio(m.cfg, m.store))

	// Refresh orders too since they're account-specific
	m.orders.State = OrdersStateLoading
	cmds = append(cmds, FetchOrders(m.cfg, m.store))

	return tea.Batch(cmds...)
}

// renderAccountPicker renders the account picker in the content area.
func (m Model) renderAccountPicker() string {
	var b strings.Builder

	b.WriteString(SummaryStyle.Render("Select Account"))
	b.WriteString("\n\n")

	for i, acc := range m.accounts {
		// Format account display
		accType := acc.AccountType
		if acc.BrokerageAccountType != "" {
			accType = acc.BrokerageAccountType
		}

		line := fmt.Sprintf("%-40s %s", acc.AccountID, accType)

		if i == m.accountCursor {
			// Selected row
			prefix := "  "
			if acc.AccountID == m.selectedAccountID {
				prefix = "✓ "
			}
			b.WriteString(lipgloss.NewStyle().
				Foreground(ColorSelectedFg).
				Background(ColorSelected).
				Bold(true).
				Render(prefix + line))
		} else {
			// Normal row
			if acc.AccountID == m.selectedAccountID {
				b.WriteString(GreenStyle.Render("✓ " + line))
			} else {
				b.WriteString(LabelStyle.Render("  " + line))
			}
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(KeyStyle.Render("↑/↓"))
	b.WriteString(LabelStyle.Render(" navigate  "))
	b.WriteString(KeyStyle.Render("enter"))
	b.WriteString(LabelStyle.Render(" select  "))
	b.WriteString(KeyStyle.Render("esc"))
	b.WriteString(LabelStyle.Render(" close"))

	return ContentStyle.Render(b.String())
}

// FetchAccounts returns a command that fetches the list of accounts.
func FetchAccounts(cfg *config.Config, store keyring.Store) tea.Cmd {
	return func() tea.Msg {
		token, err := getAuthToken(store, cfg.APIBaseURL, false)
		if err != nil {
			return AccountsErrorMsg{Err: err}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		client := api.NewClient(cfg.APIBaseURL, token)
		resp, err := client.Get(ctx, "/userapigateway/trading/account")
		if err != nil {
			return AccountsErrorMsg{Err: fmt.Errorf("failed to fetch accounts: %w", err)}
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != 200 {
			body, _ := io.ReadAll(resp.Body)
			return AccountsErrorMsg{Err: fmt.Errorf("API error: %d - %s", resp.StatusCode, string(body))}
		}

		var accountsResp AccountsResponse
		if err := json.NewDecoder(resp.Body).Decode(&accountsResp); err != nil {
			return AccountsErrorMsg{Err: fmt.Errorf("failed to decode response: %w", err)}
		}

		return AccountsLoadedMsg(accountsResp)
	}
}
