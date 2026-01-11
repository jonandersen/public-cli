package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/jonandersen/pub/internal/api"
	"github.com/jonandersen/pub/internal/auth"
	"github.com/jonandersen/pub/internal/config"
	"github.com/jonandersen/pub/internal/keyring"
)

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

// Model is the main bubbletea model for the TUI.
type Model struct {
	currentView view
	width       int
	height      int
	ready       bool

	// Config and auth
	cfg       *config.Config
	authToken string
	store     keyring.Store

	// Portfolio view state
	portfolio       portfolioData
	portfolioTable  table.Model
	refreshInterval time.Duration
}

// Message types for async operations
type portfolioLoadedMsg struct {
	portfolio Portfolio
}

type portfolioErrorMsg struct {
	err error
}

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
func newModel(cfg *config.Config, store keyring.Store) Model {
	columns := []table.Column{
		{Title: "Symbol", Width: 10},
		{Title: "Qty", Width: 8},
		{Title: "Price", Width: 10},
		{Title: "Value", Width: 12},
		{Title: "Day G/L", Width: 12},
		{Title: "Day %", Width: 8},
		{Title: "Total G/L", Width: 12},
		{Title: "Total %", Width: 8},
	}

	t := table.New(
		table.WithColumns(columns),
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
	t.SetStyles(s)

	return Model{
		currentView:     viewPortfolio,
		cfg:             cfg,
		store:           store,
		portfolioTable:  t,
		refreshInterval: 30 * time.Second,
		portfolio: portfolioData{
			state: portfolioStateLoading,
		},
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.fetchPortfolio(),
		m.tickCmd(),
	)
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
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		case "1":
			m.currentView = viewPortfolio
		case "2":
			m.currentView = viewWatchlist
		case "3":
			m.currentView = viewOrders
		case "4":
			m.currentView = viewTrade
		case "r":
			// Manual refresh
			if m.currentView == viewPortfolio {
				m.portfolio.state = portfolioStateLoading
				cmds = append(cmds, m.fetchPortfolio())
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		// Resize table to fit content area
		headerHeight := 1
		footerHeight := 1
		summaryHeight := 5 // Account summary section
		tableHeight := m.height - headerHeight - footerHeight - summaryHeight - 4
		if tableHeight < 3 {
			tableHeight = 3
		}
		m.portfolioTable.SetHeight(tableHeight)

	case portfolioLoadedMsg:
		m.portfolio.state = portfolioStateLoaded
		m.portfolio.portfolio = msg.portfolio
		m.portfolio.lastUpdated = time.Now()
		m.portfolio.err = nil
		m.updatePortfolioTable()

	case portfolioErrorMsg:
		m.portfolio.state = portfolioStateError
		m.portfolio.err = msg.err

	case tickMsg:
		// Auto-refresh portfolio
		if m.currentView == viewPortfolio && m.portfolio.state != portfolioStateLoading {
			cmds = append(cmds, m.fetchPortfolio())
		}
		cmds = append(cmds, m.tickCmd())
	}

	// Update table if we're on the portfolio view
	if m.currentView == viewPortfolio {
		m.portfolioTable, cmd = m.portfolioTable.Update(msg)
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
		content = "Watchlist view - Coming soon"
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
			// Load config
			cfg, err := config.Load(config.ConfigPath())
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			// Create keyring store
			store := keyring.NewEnvStore(keyring.NewSystemStore())

			p := tea.NewProgram(newModel(cfg, store), tea.WithAltScreen())
			_, err = p.Run()
			return err
		},
	}

	uiCmd.SilenceUsage = true
	rootCmd.AddCommand(uiCmd)
}
