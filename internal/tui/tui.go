package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/jonandersen/pub/internal/config"
	"github.com/jonandersen/pub/internal/keyring"
)

// View represents the current active view in the TUI.
type View int

const (
	ViewPortfolio View = iota
	ViewWatchlist
	ViewOrders
	ViewTrade
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

	// Refresh settings
	refreshInterval time.Duration
}

// New creates a new TUI model.
func New(cfg *config.Config, uiCfg *UIConfig, store keyring.Store) Model {
	return Model{
		currentView:     ViewPortfolio,
		cfg:             cfg,
		uiCfg:           uiCfg,
		store:           store,
		portfolio:       NewPortfolioModel(),
		watchlist:       NewWatchlistModel(uiCfg.Watchlist),
		orders:          NewOrdersModel(),
		trade:           NewTradeModel(),
		refreshInterval: 30 * time.Second,
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{
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

		// Handle global keys
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "esc":
			// Esc doesn't quit in normal mode
			return m, nil
		case "1":
			m.currentView = ViewPortfolio
		case "2":
			m.currentView = ViewWatchlist
			if m.watchlist.State == WatchlistStateLoading && len(m.watchlist.Symbols) > 0 {
				cmds = append(cmds, FetchWatchlistQuotes(m.watchlist.Symbols, m.cfg, m.store))
			}
		case "3":
			m.currentView = ViewOrders
		case "4":
			m.currentView = ViewTrade
		case "r":
			// Manual refresh
			switch m.currentView {
			case ViewPortfolio:
				m.portfolio.State = PortfolioStateLoading
				cmds = append(cmds, FetchPortfolio(m.cfg, m.store))
			case ViewWatchlist:
				m.watchlist.State = WatchlistStateLoading
				cmds = append(cmds, FetchWatchlistQuotes(m.watchlist.Symbols, m.cfg, m.store))
			}
		case "enter":
			// Jump to trade from watchlist
			if m.currentView == ViewWatchlist && m.watchlist.Mode == WatchlistModeNormal {
				symbol := m.watchlist.SelectedSymbol()
				if symbol != "" {
					m.trade.SetSymbol(symbol)
					m.currentView = ViewTrade
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

	case PortfolioLoadedMsg, PortfolioErrorMsg:
		m.portfolio, cmd = m.portfolio.Update(msg)
		cmds = append(cmds, cmd)

	case WatchlistQuotesMsg, WatchlistErrorMsg, WatchlistSavedMsg:
		m.watchlist, cmd, _ = m.watchlist.Update(msg, m.uiCfg)
		cmds = append(cmds, cmd)

	case TickMsg:
		// Auto-refresh based on current view
		if m.currentView == ViewPortfolio && m.portfolio.State != PortfolioStateLoading {
			cmds = append(cmds, FetchPortfolio(m.cfg, m.store))
		} else if m.currentView == ViewWatchlist && m.watchlist.State != WatchlistStateLoading && len(m.watchlist.Symbols) > 0 {
			cmds = append(cmds, FetchWatchlistQuotes(m.watchlist.Symbols, m.cfg, m.store))
		}
		cmds = append(cmds, m.tickCmd())
	}

	// Update active table for navigation keys
	if m.currentView == ViewPortfolio {
		m.portfolio, cmd = m.portfolio.Update(msg)
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
	content := m.renderContent()

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
	}

	var tabStrs []string
	for _, tab := range tabs {
		style := lipgloss.NewStyle().Padding(0, 1)
		if tab.active {
			style = style.Bold(true).Foreground(ColorPrimary)
		} else {
			style = style.Foreground(ColorMuted)
		}
		tabStrs = append(tabStrs, style.Render(fmt.Sprintf("[%s] %s", tab.key, tab.name)))
	}

	tabBar := strings.Join(tabStrs, " ")
	headerContent := title + "  " + tabBar

	// Pad to full width
	padding := m.width - lipgloss.Width(headerContent)
	if padding > 0 {
		headerContent += strings.Repeat(" ", padding)
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
	}
	return ContentStyle.Render(content)
}

// renderFooter renders the footer bar with key hints.
func (m Model) renderFooter() string {
	keys := []struct {
		key  string
		desc string
	}{
		{"1-4", "switch view"},
	}

	// Add view-specific keys
	switch m.currentView {
	case ViewPortfolio:
		keys = append(keys, struct{ key, desc string }{"↑/↓", "navigate"})
		keys = append(keys, struct{ key, desc string }{"r", "refresh"})
	case ViewWatchlist:
		switch m.watchlist.Mode {
		case WatchlistModeNormal:
			keys = append(keys, struct{ key, desc string }{"↑/↓", "navigate"})
			keys = append(keys, struct{ key, desc string }{"a", "add"})
			keys = append(keys, struct{ key, desc string }{"d", "delete"})
			keys = append(keys, struct{ key, desc string }{"enter", "trade"})
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
