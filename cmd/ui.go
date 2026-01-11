package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/jonandersen/pub/internal/api"
	"github.com/jonandersen/pub/internal/auth"
	"github.com/jonandersen/pub/internal/config"
	"github.com/jonandersen/pub/internal/keyring"
)

// UIConfig holds TUI-specific configuration separate from CLI config.
type UIConfig struct {
	Watchlist []string `yaml:"watchlist,omitempty"`
}

// uiConfigPath returns the path to the TUI config file.
func uiConfigPath() string {
	return filepath.Join(config.ConfigDir(), "ui.yaml")
}

// loadUIConfig loads the TUI config from disk.
func loadUIConfig() (*UIConfig, error) {
	cfg := &UIConfig{}
	data, err := os.ReadFile(uiConfigPath())
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// saveUIConfig saves the TUI config to disk.
func saveUIConfig(cfg *UIConfig) error {
	path := uiConfigPath()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// View represents the current active view in the TUI.
type view int

const (
	viewPortfolio view = iota
	viewWatchlist
	viewOrders
	viewTrade
)

// portfolioState represents the current state of portfolio data.
type portfolioState int

const (
	portfolioStateLoading portfolioState = iota
	portfolioStateLoaded
	portfolioStateError
)

// portfolioData holds the portfolio information for the TUI.
type portfolioData struct {
	state       portfolioState
	portfolio   Portfolio
	err         error
	lastUpdated time.Time
}

// watchlistState represents the current state of watchlist data.
type watchlistState int

const (
	watchlistStateLoading watchlistState = iota
	watchlistStateLoaded
	watchlistStateError
)

// watchlistMode represents the current input mode of the watchlist view.
type watchlistMode int

const (
	watchlistModeNormal watchlistMode = iota
	watchlistModeAdding
	watchlistModeDeleting
)

// watchlistData holds the watchlist information for the TUI.
type watchlistData struct {
	state       watchlistState
	symbols     []string
	quotes      map[string]Quote
	err         error
	lastUpdated time.Time
}

// Model is the main bubbletea model for the TUI.
type Model struct {
	currentView view
	width       int
	height      int
	ready       bool

	// Config and auth
	cfg       *config.Config
	uiCfg     *UIConfig
	authToken string
	store     keyring.Store

	// Portfolio view state
	portfolio       portfolioData
	portfolioTable  table.Model
	refreshInterval time.Duration

	// Watchlist view state
	watchlist      watchlistData
	watchlistTable table.Model
	watchlistMode  watchlistMode
	addInput       textinput.Model
	deleteSymbol   string
}

// Message types for async operations
type portfolioLoadedMsg struct {
	portfolio Portfolio
}

type portfolioErrorMsg struct {
	err error
}

type watchlistQuotesMsg struct {
	quotes map[string]Quote
}

type watchlistErrorMsg struct {
	err error
}

type watchlistSavedMsg struct{}

type tickMsg time.Time

// Styles for the TUI
var (
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39")).
			Background(lipgloss.Color("236")).
			Padding(0, 1)

	contentStyle = lipgloss.NewStyle().
			Padding(1, 2)

	keyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")).
			Bold(true)

	descStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))
)

// newModel creates a new TUI model.
func newModel(cfg *config.Config, uiCfg *UIConfig, store keyring.Store) Model {
	// Portfolio table
	portfolioCols := []table.Column{
		{Title: "Symbol", Width: 10},
		{Title: "Qty", Width: 8},
		{Title: "Price", Width: 10},
		{Title: "Value", Width: 12},
		{Title: "Day G/L", Width: 12},
		{Title: "Day %", Width: 8},
		{Title: "Total G/L", Width: 12},
		{Title: "Total %", Width: 8},
	}

	pt := table.New(
		table.WithColumns(portfolioCols),
		table.WithFocused(true),
		table.WithHeight(10),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(true)
	pt.SetStyles(s)

	// Watchlist table
	watchlistCols := []table.Column{
		{Title: "Symbol", Width: 10},
		{Title: "Last", Width: 12},
		{Title: "Bid", Width: 10},
		{Title: "Ask", Width: 10},
		{Title: "Volume", Width: 14},
	}

	wt := table.New(
		table.WithColumns(watchlistCols),
		table.WithFocused(true),
		table.WithHeight(10),
	)
	wt.SetStyles(s)

	// Add symbol text input
	ti := textinput.New()
	ti.Placeholder = "Enter symbol (e.g., AAPL)"
	ti.CharLimit = 10
	ti.Width = 20

	// Load watchlist from UI config
	watchlistSymbols := uiCfg.Watchlist
	if watchlistSymbols == nil {
		watchlistSymbols = []string{}
	}

	return Model{
		currentView:     viewPortfolio,
		cfg:             cfg,
		uiCfg:           uiCfg,
		store:           store,
		portfolioTable:  pt,
		refreshInterval: 30 * time.Second,
		portfolio: portfolioData{
			state: portfolioStateLoading,
		},
		watchlistTable: wt,
		watchlistMode:  watchlistModeNormal,
		addInput:       ti,
		watchlist: watchlistData{
			state:   watchlistStateLoading,
			symbols: watchlistSymbols,
			quotes:  make(map[string]Quote),
		},
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{
		m.fetchPortfolio(),
		m.tickCmd(),
	}
	if len(m.watchlist.symbols) > 0 {
		cmds = append(cmds, m.fetchWatchlistQuotes())
	}
	return tea.Batch(cmds...)
}

// tickCmd returns a command that sends a tick message after the refresh interval.
func (m Model) tickCmd() tea.Cmd {
	return tea.Tick(m.refreshInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// fetchPortfolio returns a command that fetches portfolio data from the API.
func (m Model) fetchPortfolio() tea.Cmd {
	return func() tea.Msg {
		if m.cfg.AccountUUID == "" {
			return portfolioErrorMsg{err: fmt.Errorf("no account configured")}
		}

		// Get auth token
		token, err := m.getAuthToken()
		if err != nil {
			return portfolioErrorMsg{err: err}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		client := api.NewClient(m.cfg.APIBaseURL, token)
		path := fmt.Sprintf("/userapigateway/trading/%s/portfolio/v2", m.cfg.AccountUUID)
		resp, err := client.Get(ctx, path)
		if err != nil {
			return portfolioErrorMsg{err: fmt.Errorf("failed to fetch portfolio: %w", err)}
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != 200 {
			body, _ := io.ReadAll(resp.Body)
			return portfolioErrorMsg{err: fmt.Errorf("API error: %d - %s", resp.StatusCode, string(body))}
		}

		var portfolio Portfolio
		if err := json.NewDecoder(resp.Body).Decode(&portfolio); err != nil {
			return portfolioErrorMsg{err: fmt.Errorf("failed to decode response: %w", err)}
		}

		return portfolioLoadedMsg{portfolio: portfolio}
	}
}

// fetchWatchlistQuotes returns a command that fetches quotes for watchlist symbols.
func (m Model) fetchWatchlistQuotes() tea.Cmd {
	return func() tea.Msg {
		if len(m.watchlist.symbols) == 0 {
			return watchlistQuotesMsg{quotes: make(map[string]Quote)}
		}

		if m.cfg.AccountUUID == "" {
			return watchlistErrorMsg{err: fmt.Errorf("no account configured")}
		}

		// Get auth token
		token, err := m.getAuthToken()
		if err != nil {
			return watchlistErrorMsg{err: err}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Build request
		instruments := make([]QuoteInstrument, 0, len(m.watchlist.symbols))
		for _, sym := range m.watchlist.symbols {
			instruments = append(instruments, QuoteInstrument{
				Symbol: strings.ToUpper(sym),
				Type:   "EQUITY",
			})
		}

		reqBody := QuoteRequest{Instruments: instruments}
		body, err := json.Marshal(reqBody)
		if err != nil {
			return watchlistErrorMsg{err: fmt.Errorf("failed to encode request: %w", err)}
		}

		client := api.NewClient(m.cfg.APIBaseURL, token)
		path := fmt.Sprintf("/userapigateway/marketdata/%s/quotes", m.cfg.AccountUUID)
		resp, err := client.Post(ctx, path, bytes.NewReader(body))
		if err != nil {
			return watchlistErrorMsg{err: fmt.Errorf("failed to fetch quotes: %w", err)}
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != 200 {
			respBody, _ := io.ReadAll(resp.Body)
			return watchlistErrorMsg{err: fmt.Errorf("API error: %d - %s", resp.StatusCode, string(respBody))}
		}

		var quotesResp QuotesResponse
		if err := json.NewDecoder(resp.Body).Decode(&quotesResp); err != nil {
			return watchlistErrorMsg{err: fmt.Errorf("failed to decode response: %w", err)}
		}

		quotes := make(map[string]Quote)
		for _, q := range quotesResp.Quotes {
			quotes[q.Instrument.Symbol] = q
		}

		return watchlistQuotesMsg{quotes: quotes}
	}
}

// saveWatchlist saves the watchlist to the UI config file.
func (m Model) saveWatchlist() tea.Cmd {
	return func() tea.Msg {
		m.uiCfg.Watchlist = m.watchlist.symbols
		if err := saveUIConfig(m.uiCfg); err != nil {
			return watchlistErrorMsg{err: fmt.Errorf("failed to save watchlist: %w", err)}
		}
		return watchlistSavedMsg{}
	}
}

// getAuthToken retrieves or caches the auth token.
func (m Model) getAuthToken() (string, error) {
	if m.authToken != "" {
		return m.authToken, nil
	}

	secret, err := m.store.Get(keyring.ServiceName, keyring.KeySecretKey)
	if err != nil {
		if err == keyring.ErrNotFound {
			return "", fmt.Errorf("CLI not configured. Run: pub configure")
		}
		return "", fmt.Errorf("failed to retrieve secret: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	token, err := auth.ExchangeToken(ctx, m.cfg.APIBaseURL, secret)
	if err != nil {
		return "", fmt.Errorf("failed to authenticate: %w", err)
	}

	return token.AccessToken, nil
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle watchlist input modes first
		if m.currentView == viewWatchlist {
			switch m.watchlistMode {
			case watchlistModeAdding:
				switch msg.String() {
				case "enter":
					symbol := strings.ToUpper(strings.TrimSpace(m.addInput.Value()))
					if symbol != "" {
						// Check if symbol already exists
						exists := false
						for _, s := range m.watchlist.symbols {
							if s == symbol {
								exists = true
								break
							}
						}
						if !exists {
							m.watchlist.symbols = append(m.watchlist.symbols, symbol)
							cmds = append(cmds, m.saveWatchlist(), m.fetchWatchlistQuotes())
						}
					}
					m.watchlistMode = watchlistModeNormal
					m.addInput.Reset()
					return m, tea.Batch(cmds...)
				case "esc":
					m.watchlistMode = watchlistModeNormal
					m.addInput.Reset()
					return m, nil
				default:
					m.addInput, cmd = m.addInput.Update(msg)
					return m, cmd
				}
			case watchlistModeDeleting:
				switch msg.String() {
				case "y", "Y":
					// Remove the symbol
					newSymbols := make([]string, 0, len(m.watchlist.symbols))
					for _, s := range m.watchlist.symbols {
						if s != m.deleteSymbol {
							newSymbols = append(newSymbols, s)
						}
					}
					m.watchlist.symbols = newSymbols
					delete(m.watchlist.quotes, m.deleteSymbol)
					m.updateWatchlistTable()
					cmds = append(cmds, m.saveWatchlist())
					m.watchlistMode = watchlistModeNormal
					m.deleteSymbol = ""
					return m, tea.Batch(cmds...)
				case "n", "N", "esc":
					m.watchlistMode = watchlistModeNormal
					m.deleteSymbol = ""
					return m, nil
				}
				return m, nil
			}
		}

		// Handle normal key presses
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "esc":
			// Esc doesn't quit in normal mode, just reset state
			return m, nil
		case "1":
			m.currentView = viewPortfolio
		case "2":
			m.currentView = viewWatchlist
			if m.watchlist.state == watchlistStateLoading && len(m.watchlist.symbols) > 0 {
				cmds = append(cmds, m.fetchWatchlistQuotes())
			}
		case "3":
			m.currentView = viewOrders
		case "4":
			m.currentView = viewTrade
		case "r":
			// Manual refresh
			switch m.currentView {
			case viewPortfolio:
				m.portfolio.state = portfolioStateLoading
				cmds = append(cmds, m.fetchPortfolio())
			case viewWatchlist:
				m.watchlist.state = watchlistStateLoading
				cmds = append(cmds, m.fetchWatchlistQuotes())
			}
		case "a":
			// Add symbol (watchlist only)
			if m.currentView == viewWatchlist && m.watchlistMode == watchlistModeNormal {
				m.watchlistMode = watchlistModeAdding
				m.addInput.Focus()
				return m, textinput.Blink
			}
		case "d", "x":
			// Delete symbol (watchlist only)
			if m.currentView == viewWatchlist && m.watchlistMode == watchlistModeNormal {
				if len(m.watchlist.symbols) > 0 {
					selectedRow := m.watchlistTable.SelectedRow()
					if len(selectedRow) > 0 {
						m.deleteSymbol = selectedRow[0]
						m.watchlistMode = watchlistModeDeleting
					}
				}
			}
		case "enter":
			// Jump to trade from selected symbol
			if m.currentView == viewWatchlist && m.watchlistMode == watchlistModeNormal {
				selectedRow := m.watchlistTable.SelectedRow()
				if len(selectedRow) > 0 {
					// TODO: Pre-populate trade view with selected symbol
					m.currentView = viewTrade
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
		m.portfolioTable.SetHeight(tableHeight)
		m.watchlistTable.SetHeight(tableHeight)

	case portfolioLoadedMsg:
		m.portfolio.state = portfolioStateLoaded
		m.portfolio.portfolio = msg.portfolio
		m.portfolio.lastUpdated = time.Now()
		m.portfolio.err = nil
		m.updatePortfolioTable()

	case portfolioErrorMsg:
		m.portfolio.state = portfolioStateError
		m.portfolio.err = msg.err

	case watchlistQuotesMsg:
		m.watchlist.state = watchlistStateLoaded
		m.watchlist.quotes = msg.quotes
		m.watchlist.lastUpdated = time.Now()
		m.watchlist.err = nil
		m.updateWatchlistTable()

	case watchlistErrorMsg:
		m.watchlist.state = watchlistStateError
		m.watchlist.err = msg.err

	case watchlistSavedMsg:
		// Config saved successfully, nothing to do

	case tickMsg:
		// Auto-refresh based on current view
		if m.currentView == viewPortfolio && m.portfolio.state != portfolioStateLoading {
			cmds = append(cmds, m.fetchPortfolio())
		} else if m.currentView == viewWatchlist && m.watchlist.state != watchlistStateLoading && len(m.watchlist.symbols) > 0 {
			cmds = append(cmds, m.fetchWatchlistQuotes())
		}
		cmds = append(cmds, m.tickCmd())
	}

	// Update active table
	if m.currentView == viewPortfolio {
		m.portfolioTable, cmd = m.portfolioTable.Update(msg)
		cmds = append(cmds, cmd)
	} else if m.currentView == viewWatchlist && m.watchlistMode == watchlistModeNormal {
		m.watchlistTable, cmd = m.watchlistTable.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// updatePortfolioTable updates the table rows with current portfolio data.
func (m *Model) updatePortfolioTable() {
	rows := make([]table.Row, 0, len(m.portfolio.portfolio.Positions))
	for _, pos := range m.portfolio.portfolio.Positions {
		totalGainValue := pos.CostBasis.GainValue
		totalGainPct := pos.CostBasis.GainPercentage
		if totalGainValue == "" {
			totalGainValue = "0"
			totalGainPct = "0"
		}
		rows = append(rows, table.Row{
			pos.Instrument.Symbol,
			pos.Quantity,
			"$" + pos.LastPrice.LastPrice,
			"$" + pos.CurrentValue,
			formatGainLoss(pos.PositionDailyGain.GainValue),
			pos.PositionDailyGain.GainPercentage + "%",
			formatGainLoss(totalGainValue),
			totalGainPct + "%",
		})
	}
	m.portfolioTable.SetRows(rows)
}

// updateWatchlistTable updates the table rows with current watchlist data.
func (m *Model) updateWatchlistTable() {
	rows := make([]table.Row, 0, len(m.watchlist.symbols))
	for _, sym := range m.watchlist.symbols {
		quote, hasQuote := m.watchlist.quotes[sym]
		if hasQuote && quote.Outcome == "SUCCESS" {
			rows = append(rows, table.Row{
				sym,
				"$" + quote.Last,
				"$" + quote.Bid,
				"$" + quote.Ask,
				formatVolume(quote.Volume),
			})
		} else {
			rows = append(rows, table.Row{
				sym,
				"-",
				"-",
				"-",
				"-",
			})
		}
	}
	m.watchlistTable.SetRows(rows)
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
	title := headerStyle.Render("pub")

	tabs := []struct {
		name   string
		key    string
		active bool
	}{
		{"Portfolio", "1", m.currentView == viewPortfolio},
		{"Watchlist", "2", m.currentView == viewWatchlist},
		{"Orders", "3", m.currentView == viewOrders},
		{"Trade", "4", m.currentView == viewTrade},
	}

	var tabStrs []string
	for _, tab := range tabs {
		style := lipgloss.NewStyle().Padding(0, 1)
		if tab.active {
			style = style.Bold(true).Foreground(lipgloss.Color("39"))
		} else {
			style = style.Foreground(lipgloss.Color("241"))
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
		Background(lipgloss.Color("236")).
		Width(m.width).
		Render(headerContent)
}

// renderContent renders the main content area.
func (m Model) renderContent() string {
	var content string
	switch m.currentView {
	case viewPortfolio:
		content = m.renderPortfolioView()
	case viewWatchlist:
		content = m.renderWatchlistView()
	case viewOrders:
		content = "Orders view - Coming soon"
	case viewTrade:
		content = "Trade view - Coming soon"
	}
	return contentStyle.Render(content)
}

// renderPortfolioView renders the portfolio view with account summary and positions table.
func (m Model) renderPortfolioView() string {
	var b strings.Builder

	switch m.portfolio.state {
	case portfolioStateLoading:
		b.WriteString("Loading portfolio...")
		return b.String()

	case portfolioStateError:
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
		b.WriteString(errStyle.Render(fmt.Sprintf("Error: %v", m.portfolio.err)))
		b.WriteString("\n\nPress 'r' to retry")
		return b.String()

	case portfolioStateLoaded:
		// Account Summary Section
		summaryStyle := lipgloss.NewStyle().Bold(true)
		labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
		valueStyle := lipgloss.NewStyle().Bold(true)
		greenStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
		redStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))

		b.WriteString(summaryStyle.Render("Account Summary"))
		b.WriteString("\n")

		p := m.portfolio.portfolio

		// Calculate total value from equity
		var totalValue, cashValue, dayChange string
		for _, eq := range p.Equity {
			switch eq.Type {
			case "TOTAL":
				totalValue = eq.Value
			case "CASH":
				cashValue = eq.Value
			}
		}

		// Calculate day P/L from positions
		var totalDayGain float64
		for _, pos := range p.Positions {
			if val, err := strconv.ParseFloat(pos.PositionDailyGain.GainValue, 64); err == nil {
				totalDayGain += val
			}
		}
		dayChange = fmt.Sprintf("%.2f", totalDayGain)

		b.WriteString(labelStyle.Render("Total Value: "))
		b.WriteString(valueStyle.Render("$" + totalValue))
		b.WriteString("  ")
		b.WriteString(labelStyle.Render("Cash: "))
		b.WriteString(valueStyle.Render("$" + cashValue))
		b.WriteString("  ")
		b.WriteString(labelStyle.Render("Day P/L: "))
		if totalDayGain >= 0 {
			b.WriteString(greenStyle.Render("+$" + dayChange))
		} else {
			b.WriteString(redStyle.Render("-$" + dayChange[1:]))
		}
		b.WriteString("\n")

		b.WriteString(labelStyle.Render("Buying Power: "))
		b.WriteString(valueStyle.Render("$" + p.BuyingPower.BuyingPower))
		b.WriteString("  ")
		b.WriteString(labelStyle.Render("Options BP: "))
		b.WriteString(valueStyle.Render("$" + p.BuyingPower.OptionsBuyingPower))
		b.WriteString("\n\n")

		// Positions Table
		if len(p.Positions) == 0 {
			b.WriteString(labelStyle.Render("No positions"))
		} else {
			b.WriteString(summaryStyle.Render("Positions"))
			b.WriteString(labelStyle.Render(fmt.Sprintf(" (%d)", len(p.Positions))))
			b.WriteString("\n")
			b.WriteString(m.portfolioTable.View())
		}

		// Last updated
		b.WriteString("\n")
		b.WriteString(labelStyle.Render(fmt.Sprintf("Updated: %s", m.portfolio.lastUpdated.Format("3:04:05 PM"))))
	}

	return b.String()
}

// renderWatchlistView renders the watchlist view with symbols and quotes.
func (m Model) renderWatchlistView() string {
	var b strings.Builder

	summaryStyle := lipgloss.NewStyle().Bold(true)
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	inputStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("39")).
		Padding(0, 1)
	confirmStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("220")).
		Bold(true)

	// Handle input modes first
	switch m.watchlistMode {
	case watchlistModeAdding:
		b.WriteString(summaryStyle.Render("Add Symbol"))
		b.WriteString("\n\n")
		b.WriteString(inputStyle.Render(m.addInput.View()))
		b.WriteString("\n\n")
		b.WriteString(labelStyle.Render("Press Enter to add, Esc to cancel"))
		return b.String()

	case watchlistModeDeleting:
		b.WriteString(confirmStyle.Render(fmt.Sprintf("Delete %s from watchlist?", m.deleteSymbol)))
		b.WriteString("\n\n")
		b.WriteString(labelStyle.Render("Press Y to confirm, N to cancel"))
		return b.String()
	}

	// Normal mode - show watchlist
	switch m.watchlist.state {
	case watchlistStateLoading:
		b.WriteString("Loading quotes...")
		return b.String()

	case watchlistStateError:
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
		b.WriteString(errStyle.Render(fmt.Sprintf("Error: %v", m.watchlist.err)))
		b.WriteString("\n\nPress 'r' to retry")
		return b.String()

	case watchlistStateLoaded:
		b.WriteString(summaryStyle.Render("Watchlist"))
		b.WriteString(labelStyle.Render(fmt.Sprintf(" (%d symbols)", len(m.watchlist.symbols))))
		b.WriteString("\n\n")

		if len(m.watchlist.symbols) == 0 {
			b.WriteString(labelStyle.Render("No symbols in watchlist"))
			b.WriteString("\n\n")
			b.WriteString(labelStyle.Render("Press 'a' to add a symbol"))
		} else {
			b.WriteString(m.watchlistTable.View())
			b.WriteString("\n")
			b.WriteString(labelStyle.Render(fmt.Sprintf("Updated: %s", m.watchlist.lastUpdated.Format("3:04:05 PM"))))
		}
	}

	return b.String()
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
	case viewPortfolio:
		keys = append(keys, struct{ key, desc string }{"↑/↓", "navigate"})
		keys = append(keys, struct{ key, desc string }{"r", "refresh"})
	case viewWatchlist:
		switch m.watchlistMode {
		case watchlistModeNormal:
			keys = append(keys, struct{ key, desc string }{"↑/↓", "navigate"})
			keys = append(keys, struct{ key, desc string }{"a", "add"})
			keys = append(keys, struct{ key, desc string }{"d", "delete"})
			keys = append(keys, struct{ key, desc string }{"enter", "trade"})
			keys = append(keys, struct{ key, desc string }{"r", "refresh"})
		case watchlistModeAdding:
			keys = []struct{ key, desc string }{
				{"enter", "add"},
				{"esc", "cancel"},
			}
		case watchlistModeDeleting:
			keys = []struct{ key, desc string }{
				{"y", "confirm"},
				{"n", "cancel"},
			}
		}
	}

	keys = append(keys, struct{ key, desc string }{"q", "quit"})

	var parts []string
	for _, k := range keys {
		parts = append(parts, keyStyle.Render(k.key)+" "+descStyle.Render(k.desc))
	}

	footerContent := strings.Join(parts, "  •  ")

	// Pad to full width
	padding := m.width - lipgloss.Width(footerContent)
	if padding > 0 {
		footerContent += strings.Repeat(" ", padding)
	}

	return lipgloss.NewStyle().
		Background(lipgloss.Color("236")).
		Width(m.width).
		Render(footerContent)
}

func init() {
	uiCmd := &cobra.Command{
		Use:   "ui",
		Short: "Interactive terminal UI",
		Long: `Launch an interactive terminal UI for trading.

The UI provides a full-screen experience with keyboard navigation,
live data updates, and views for:
  - Portfolio: View your positions and account summary
  - Watchlist: Track symbols with live quotes
  - Orders: View and manage open orders
  - Trade: Buy and sell securities

Keyboard shortcuts:
  1-4     Switch between views
  ↑/↓     Navigate positions
  r       Refresh data
  q/esc   Quit the application`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Load CLI config
			cfg, err := config.Load(config.ConfigPath())
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			// Load TUI config
			uiCfg, err := loadUIConfig()
			if err != nil {
				return fmt.Errorf("failed to load UI config: %w", err)
			}

			// Create keyring store
			store := keyring.NewEnvStore(keyring.NewSystemStore())

			p := tea.NewProgram(newModel(cfg, uiCfg, store), tea.WithAltScreen())
			_, err = p.Run()
			return err
		},
	}

	uiCmd.SilenceUsage = true
	rootCmd.AddCommand(uiCmd)
}
