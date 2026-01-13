package tui

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/jonandersen/public-cli/internal/api"
	"github.com/jonandersen/public-cli/internal/config"
	"github.com/jonandersen/public-cli/internal/keyring"
)

// OptionsState represents the current state of the options view.
type OptionsState int

const (
	OptionsStateIdle OptionsState = iota
	OptionsStateLoadingExpirations
	OptionsStateSelectingExpiration
	OptionsStateLoadingChain
	OptionsStateChainLoaded
	OptionsStateError
)

// OptionsFocus represents what is currently focused.
type OptionsFocus int

const (
	OptionsFocusSymbol OptionsFocus = iota
	OptionsFocusExpiration
	OptionsFocusCalls
	OptionsFocusPuts
)

// OptionsModel holds the state for the options view.
type OptionsModel struct {
	State       OptionsState
	Focus       OptionsFocus
	Err         error
	LastUpdated time.Time

	// Symbol input
	SymbolInput textinput.Model
	Symbol      string
	Quote       *Quote

	// Expirations
	Expirations        []string
	SelectedExpiration int

	// Option chain data
	Chain       *api.OptionChainResponse
	Greeks      map[string]api.GreeksData
	CallsCursor int
	PutsCursor  int
	ShowGreeks  bool
	Height      int
	OptionsBP   string

	// Asset selector (for watchlist/portfolio selection)
	AssetSelector     *AssetSelectorModel
	ShowAssetSelector bool
}

// NewOptionsModel creates a new options model.
func NewOptionsModel() *OptionsModel {
	ti := textinput.New()
	ti.Placeholder = "Enter symbol (e.g., AAPL)"
	ti.CharLimit = 10
	ti.Width = 20
	ti.Focus()

	return &OptionsModel{
		State:       OptionsStateIdle,
		Focus:       OptionsFocusSymbol,
		SymbolInput: ti,
		Greeks:      make(map[string]api.GreeksData),
		Height:      10,
	}
}

// SetHeight sets the available height for the options chain.
func (m *OptionsModel) SetHeight(h int) {
	m.Height = h
}

// Update handles messages for the options view.
func (m *OptionsModel) Update(msg tea.Msg, cfg *config.Config, store keyring.Store) (*OptionsModel, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	// Handle asset selector if open
	if m.ShowAssetSelector {
		switch msg := msg.(type) {
		case AssetSelectedMsg:
			m.ShowAssetSelector = false
			m.Symbol = msg.Symbol
			m.SymbolInput.SetValue(msg.Symbol)
			m.State = OptionsStateLoadingExpirations
			m.Focus = OptionsFocusExpiration
			return m, FetchOptionExpirations(msg.Symbol, cfg, store)

		case AssetSelectorCancelledMsg:
			m.ShowAssetSelector = false
			m.SymbolInput.Focus()
			return m, nil

		case tea.KeyMsg:
			m.AssetSelector, cmd = m.AssetSelector.Update(msg, cfg, store)
			return m, cmd
		}
		return m, nil
	}

	switch msg := msg.(type) {
	case OptionExpirationsLoadedMsg:
		m.State = OptionsStateSelectingExpiration
		m.Expirations = msg.Expirations
		m.OptionsBP = msg.OptionsBP
		m.SelectedExpiration = 0
		if len(m.Expirations) > 0 {
			m.Focus = OptionsFocusExpiration
		}
		return m, nil

	case OptionExpirationsErrorMsg:
		m.State = OptionsStateError
		m.Err = msg.Err
		return m, nil

	case OptionChainLoadedMsg:
		m.State = OptionsStateChainLoaded
		m.Chain = msg.Chain
		m.LastUpdated = time.Now()
		m.CallsCursor = 0
		m.PutsCursor = 0
		m.Focus = OptionsFocusCalls
		// Try to select ATM option
		m.selectATMOption()
		// Fetch greeks for visible options
		return m, m.fetchVisibleGreeks(cfg, store)

	case OptionChainErrorMsg:
		m.State = OptionsStateError
		m.Err = msg.Err
		return m, nil

	case OptionGreeksLoadedMsg:
		for _, g := range msg.Greeks {
			m.Greeks[g.Symbol] = g.Greeks
		}
		return m, nil

	case OptionQuoteLoadedMsg:
		m.Quote = &Quote{
			Last: msg.Last,
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyMsg(msg, cfg, store)
	}

	return m, tea.Batch(cmds...)
}

func (m *OptionsModel) handleKeyMsg(msg tea.KeyMsg, cfg *config.Config, store keyring.Store) (*OptionsModel, tea.Cmd) {
	var cmd tea.Cmd

	switch m.State {
	case OptionsStateIdle, OptionsStateError:
		return m.handleIdleKeys(msg, cfg, store)
	case OptionsStateSelectingExpiration:
		return m.handleExpirationKeys(msg, cfg, store)
	case OptionsStateChainLoaded:
		return m.handleChainKeys(msg, cfg, store)
	}

	// Default: pass to symbol input
	m.SymbolInput, cmd = m.SymbolInput.Update(msg)
	return m, cmd
}

func (m *OptionsModel) handleIdleKeys(msg tea.KeyMsg, cfg *config.Config, store keyring.Store) (*OptionsModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg.String() {
	case "enter":
		symbol := strings.ToUpper(strings.TrimSpace(m.SymbolInput.Value()))
		if symbol != "" {
			m.Symbol = symbol
			m.State = OptionsStateLoadingExpirations
			return m, FetchOptionExpirations(symbol, cfg, store)
		}
		return m, nil

	case "w":
		// Open asset selector in watchlist mode
		m.AssetSelector = NewAssetSelectorModel(AssetSelectorModeWatchlist)
		m.ShowAssetSelector = true
		return m, nil

	case "esc":
		return m, func() tea.Msg { return ToolbarFocusMsg{} }
	}

	m.SymbolInput, cmd = m.SymbolInput.Update(msg)
	return m, cmd
}

func (m *OptionsModel) handleExpirationKeys(msg tea.KeyMsg, cfg *config.Config, store keyring.Store) (*OptionsModel, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.SelectedExpiration > 0 {
			m.SelectedExpiration--
		}
		return m, nil

	case "down", "j":
		if m.SelectedExpiration < len(m.Expirations)-1 {
			m.SelectedExpiration++
		}
		return m, nil

	case "enter":
		if len(m.Expirations) > 0 {
			m.State = OptionsStateLoadingChain
			expiration := m.Expirations[m.SelectedExpiration]
			return m, FetchOptionChain(m.Symbol, expiration, cfg, store)
		}
		return m, nil

	case "esc":
		// Go back to symbol input
		m.State = OptionsStateIdle
		m.Focus = OptionsFocusSymbol
		m.SymbolInput.Focus()
		return m, nil
	}

	return m, nil
}

func (m *OptionsModel) handleChainKeys(msg tea.KeyMsg, cfg *config.Config, store keyring.Store) (*OptionsModel, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		switch m.Focus {
		case OptionsFocusCalls:
			if m.CallsCursor > 0 {
				m.CallsCursor--
			}
		case OptionsFocusPuts:
			if m.PutsCursor > 0 {
				m.PutsCursor--
			}
		}
		return m, nil

	case "down", "j":
		switch m.Focus {
		case OptionsFocusCalls:
			if m.Chain != nil && m.CallsCursor < len(m.Chain.Calls)-1 {
				m.CallsCursor++
			}
		case OptionsFocusPuts:
			if m.Chain != nil && m.PutsCursor < len(m.Chain.Puts)-1 {
				m.PutsCursor++
			}
		}
		return m, nil

	case "c":
		m.Focus = OptionsFocusCalls
		return m, nil

	case "p":
		m.Focus = OptionsFocusPuts
		return m, nil

	case "tab":
		// Toggle between calls and puts
		if m.Focus == OptionsFocusCalls {
			m.Focus = OptionsFocusPuts
		} else {
			m.Focus = OptionsFocusCalls
		}
		return m, nil

	case "e":
		// Go back to expiration selection
		m.State = OptionsStateSelectingExpiration
		m.Focus = OptionsFocusExpiration
		return m, nil

	case "g":
		// Toggle greeks display
		m.ShowGreeks = !m.ShowGreeks
		return m, nil

	case "r":
		// Refresh chain
		if len(m.Expirations) > 0 {
			m.State = OptionsStateLoadingChain
			expiration := m.Expirations[m.SelectedExpiration]
			return m, FetchOptionChain(m.Symbol, expiration, cfg, store)
		}
		return m, nil

	case "esc":
		// Go back to expiration selection
		m.State = OptionsStateSelectingExpiration
		m.Focus = OptionsFocusExpiration
		return m, nil
	}

	return m, nil
}

func (m *OptionsModel) selectATMOption() {
	if m.Chain == nil || m.Quote == nil {
		return
	}

	// Parse the underlying price
	price, err := strconv.ParseFloat(m.Quote.Last, 64)
	if err != nil {
		return
	}

	// Find the closest strike in calls
	minDiff := float64(999999)
	for i, call := range m.Chain.Calls {
		strike := parseStrikeFromOSI(call.Instrument.Symbol)
		diff := abs(strike - price)
		if diff < minDiff {
			minDiff = diff
			m.CallsCursor = i
		}
	}

	// Same for puts
	minDiff = float64(999999)
	for i, put := range m.Chain.Puts {
		strike := parseStrikeFromOSI(put.Instrument.Symbol)
		diff := abs(strike - price)
		if diff < minDiff {
			minDiff = diff
			m.PutsCursor = i
		}
	}
}

func parseStrikeFromOSI(osi string) float64 {
	// OSI format: AAPL250117C00185000
	// Last 8 chars are strike * 1000 (3 decimal places)
	if len(osi) < 8 {
		return 0
	}
	strikeStr := osi[len(osi)-8:]
	strike, err := strconv.ParseFloat(strikeStr, 64)
	if err != nil {
		return 0
	}
	return strike / 1000
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func (m *OptionsModel) fetchVisibleGreeks(cfg *config.Config, store keyring.Store) tea.Cmd {
	if m.Chain == nil {
		return nil
	}

	var symbols []string

	// Add visible calls
	start := m.CallsCursor - 3
	if start < 0 {
		start = 0
	}
	end := m.CallsCursor + 4
	if end > len(m.Chain.Calls) {
		end = len(m.Chain.Calls)
	}
	for i := start; i < end; i++ {
		symbols = append(symbols, m.Chain.Calls[i].Instrument.Symbol)
	}

	// Add visible puts
	start = m.PutsCursor - 3
	if start < 0 {
		start = 0
	}
	end = m.PutsCursor + 4
	if end > len(m.Chain.Puts) {
		end = len(m.Chain.Puts)
	}
	for i := start; i < end; i++ {
		symbols = append(symbols, m.Chain.Puts[i].Instrument.Symbol)
	}

	if len(symbols) == 0 {
		return nil
	}

	return FetchOptionGreeks(symbols, cfg, store)
}

// View renders the options view.
func (m *OptionsModel) View() string {
	// Show asset selector if open
	if m.ShowAssetSelector && m.AssetSelector != nil {
		return m.AssetSelector.View()
	}

	var b strings.Builder

	switch m.State {
	case OptionsStateIdle, OptionsStateLoadingExpirations:
		b.WriteString(m.renderSymbolEntry())
	case OptionsStateSelectingExpiration:
		b.WriteString(m.renderExpirationSelection())
	case OptionsStateLoadingChain:
		b.WriteString(m.renderLoading())
	case OptionsStateChainLoaded:
		b.WriteString(m.renderChain())
	case OptionsStateError:
		b.WriteString(m.renderError())
	}

	return b.String()
}

func (m *OptionsModel) renderSymbolEntry() string {
	var b strings.Builder

	b.WriteString(SummaryStyle.Render("Options Chain"))
	b.WriteString("\n\n")

	b.WriteString(LabelStyle.Render("Underlying:"))
	b.WriteString("\n")

	if m.State == OptionsStateLoadingExpirations {
		b.WriteString(InputStyle.Render(m.SymbolInput.View()))
		b.WriteString("\n\n")
		b.WriteString(LabelStyle.Render("Loading expirations..."))
	} else {
		b.WriteString(InputStyle.Render(m.SymbolInput.View()))
		b.WriteString("\n\n")
		b.WriteString(LabelStyle.Render("Select a symbol to view options chain"))
		b.WriteString("\n\n")
		b.WriteString(LabelStyle.Render("Tip: Press Enter after typing a symbol, or select from your watchlist with 'w'"))
	}

	return b.String()
}

func (m *OptionsModel) renderExpirationSelection() string {
	var b strings.Builder

	b.WriteString(SummaryStyle.Render("Options Chain"))
	b.WriteString("\n\n")

	// Underlying info
	b.WriteString(LabelStyle.Render("Underlying: "))
	b.WriteString(ValueStyle.Render(m.Symbol))
	if m.Quote != nil {
		b.WriteString(LabelStyle.Render("  $"))
		b.WriteString(ValueStyle.Render(m.Quote.Last))
	}
	if m.OptionsBP != "" {
		b.WriteString("    ")
		b.WriteString(LabelStyle.Render("Options BP: "))
		b.WriteString(ValueStyle.Render("$" + m.OptionsBP))
	}
	b.WriteString("\n\n")

	b.WriteString(LabelStyle.Render("Select Expiration:"))
	b.WriteString("\n\n")

	// Show up to 8 expirations
	maxShow := 8
	if len(m.Expirations) < maxShow {
		maxShow = len(m.Expirations)
	}

	for i := 0; i < maxShow; i++ {
		exp := m.Expirations[i]
		dte := calculateDTE(exp)

		prefix := "  "
		style := LabelStyle
		if i == m.SelectedExpiration {
			prefix = "> "
			style = lipgloss.NewStyle().Foreground(ColorSelectedFg).Background(ColorSelected).Bold(true)
		}

		line := fmt.Sprintf("%-12s (%d DTE)", exp, dte)
		b.WriteString(style.Render(prefix + line))
		b.WriteString("\n")
	}

	if len(m.Expirations) > maxShow {
		b.WriteString(LabelStyle.Render(fmt.Sprintf("\n  ... and %d more", len(m.Expirations)-maxShow)))
	}

	b.WriteString("\n\n")
	b.WriteString(LabelStyle.Render("Press Enter to select, Esc to go back"))

	return b.String()
}

func (m *OptionsModel) renderLoading() string {
	var b strings.Builder

	b.WriteString(SummaryStyle.Render("Options Chain"))
	b.WriteString("\n\n")

	b.WriteString(LabelStyle.Render("Underlying: "))
	b.WriteString(ValueStyle.Render(m.Symbol))
	b.WriteString("\n\n")

	b.WriteString(LabelStyle.Render("Loading option chain..."))

	return b.String()
}

func (m *OptionsModel) renderChain() string {
	var b strings.Builder

	// Header
	exp := m.Expirations[m.SelectedExpiration]
	dte := calculateDTE(exp)

	b.WriteString(SummaryStyle.Render(fmt.Sprintf("Options Chain - %s", m.Symbol)))
	if m.Quote != nil {
		b.WriteString(LabelStyle.Render("  $"))
		b.WriteString(ValueStyle.Render(m.Quote.Last))
	}
	b.WriteString("    ")
	b.WriteString(LabelStyle.Render(fmt.Sprintf("Exp: %s (%d DTE)", exp, dte)))
	b.WriteString("\n\n")

	// Render calls table
	b.WriteString(m.renderOptionsTable("CALLS", m.Chain.Calls, m.CallsCursor, m.Focus == OptionsFocusCalls))
	b.WriteString("\n")

	// Render puts table
	b.WriteString(m.renderOptionsTable("PUTS", m.Chain.Puts, m.PutsCursor, m.Focus == OptionsFocusPuts))

	// Updated time
	b.WriteString("\n")
	b.WriteString(LabelStyle.Render(fmt.Sprintf("Updated: %s", m.LastUpdated.Format("3:04:05 PM"))))

	return b.String()
}

func (m *OptionsModel) renderOptionsTable(title string, options []api.OptionQuote, cursor int, focused bool) string {
	var b strings.Builder

	titleStyle := LabelStyle
	if focused {
		titleStyle = ValueStyle.Bold(true)
	}
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n")

	// Header
	if m.ShowGreeks {
		b.WriteString(LabelStyle.Render("  Strike      Bid      Ask     Last    Vol       OI   Delta   Theta      IV"))
	} else {
		b.WriteString(LabelStyle.Render("  Strike      Bid      Ask     Last    Vol       OI"))
	}
	b.WriteString("\n")
	b.WriteString(LabelStyle.Render(strings.Repeat("─", 80)))
	b.WriteString("\n")

	// Calculate visible range
	visibleRows := 5
	start := cursor - visibleRows/2
	if start < 0 {
		start = 0
	}
	end := start + visibleRows
	if end > len(options) {
		end = len(options)
		start = end - visibleRows
		if start < 0 {
			start = 0
		}
	}

	for i := start; i < end; i++ {
		opt := options[i]
		strike := parseStrikeFromOSI(opt.Instrument.Symbol)

		prefix := "  "
		style := LabelStyle
		if i == cursor {
			prefix = "> "
			if focused {
				style = lipgloss.NewStyle().Foreground(ColorSelectedFg).Background(ColorSelected).Bold(true)
			} else {
				style = ValueStyle
			}
		}

		// Check if ATM
		atmMarker := ""
		if m.Quote != nil {
			price, _ := strconv.ParseFloat(m.Quote.Last, 64)
			if abs(strike-price) < 2.5 {
				atmMarker = " ATM"
			}
		}

		var row string
		if m.ShowGreeks {
			greeks := m.Greeks[opt.Instrument.Symbol]
			row = fmt.Sprintf("%-8.2f  %6s   %6s   %6s  %5d  %6d  %6s  %6s  %6s%s",
				strike,
				formatOptPrice(opt.Bid),
				formatOptPrice(opt.Ask),
				formatOptPrice(opt.Last),
				opt.Volume,
				opt.OpenInterest,
				formatGreek(greeks.Delta),
				formatGreek(greeks.Theta),
				formatIV(greeks.ImpliedVolatility),
				atmMarker)
		} else {
			row = fmt.Sprintf("%-8.2f  %6s   %6s   %6s  %5d  %6d%s",
				strike,
				formatOptPrice(opt.Bid),
				formatOptPrice(opt.Ask),
				formatOptPrice(opt.Last),
				opt.Volume,
				opt.OpenInterest,
				atmMarker)
		}

		b.WriteString(style.Render(prefix + row))
		b.WriteString("\n")
	}

	return b.String()
}

func (m *OptionsModel) renderError() string {
	var b strings.Builder

	b.WriteString(SummaryStyle.Render("Options Chain"))
	b.WriteString("\n\n")

	b.WriteString(ErrorStyle.Render(fmt.Sprintf("Error: %v", m.Err)))
	b.WriteString("\n\n")

	b.WriteString(LabelStyle.Render("Press Esc to try again"))

	return b.String()
}

// FooterKeys returns the footer keys for the options view.
func (m *OptionsModel) FooterKeys(keys []struct{ key, desc string }) []struct{ key, desc string } {
	if m.ShowAssetSelector {
		return []struct{ key, desc string }{
			{"Enter", "select"},
			{"↑/↓", "navigate"},
			{"Esc", "cancel"},
		}
	}

	switch m.State {
	case OptionsStateIdle, OptionsStateLoadingExpirations:
		keys = append(keys, struct{ key, desc string }{"Enter", "search"})
		keys = append(keys, struct{ key, desc string }{"w", "watchlist"})
		keys = append(keys, struct{ key, desc string }{"esc", "toolbar"})
	case OptionsStateSelectingExpiration:
		keys = append(keys, struct{ key, desc string }{"↑/↓", "navigate"})
		keys = append(keys, struct{ key, desc string }{"Enter", "select"})
		keys = append(keys, struct{ key, desc string }{"esc", "back"})
	case OptionsStateChainLoaded:
		keys = append(keys, struct{ key, desc string }{"↑/↓", "navigate"})
		keys = append(keys, struct{ key, desc string }{"c/p", "calls/puts"})
		keys = append(keys, struct{ key, desc string }{"e", "expiration"})
		keys = append(keys, struct{ key, desc string }{"g", "greeks"})
		keys = append(keys, struct{ key, desc string }{"r", "refresh"})
	case OptionsStateError:
		keys = append(keys, struct{ key, desc string }{"esc", "back"})
	}

	return keys
}

// Helper functions

func calculateDTE(expiration string) int {
	// Parse expiration date (format: 2025-01-17)
	exp, err := time.Parse("2006-01-02", expiration)
	if err != nil {
		return 0
	}
	return int(time.Until(exp).Hours() / 24)
}

func formatOptPrice(price string) string {
	if price == "" || price == "0" {
		return "-"
	}
	p, err := strconv.ParseFloat(price, 64)
	if err != nil {
		return price
	}
	return fmt.Sprintf("%.2f", p)
}

func formatGreek(value string) string {
	if value == "" {
		return "-"
	}
	v, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return value
	}
	return fmt.Sprintf("%.2f", v)
}

func formatIV(value string) string {
	if value == "" {
		return "-"
	}
	v, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return value
	}
	return fmt.Sprintf("%.1f%%", v*100)
}

// Message types for options operations

// OptionExpirationsLoadedMsg is sent when expirations are loaded.
type OptionExpirationsLoadedMsg struct {
	Expirations []string
	OptionsBP   string
}

// OptionExpirationsErrorMsg is sent when loading expirations fails.
type OptionExpirationsErrorMsg struct {
	Err error
}

// OptionChainLoadedMsg is sent when the option chain is loaded.
type OptionChainLoadedMsg struct {
	Chain *api.OptionChainResponse
}

// OptionChainErrorMsg is sent when loading the chain fails.
type OptionChainErrorMsg struct {
	Err error
}

// OptionGreeksLoadedMsg is sent when greeks are loaded.
type OptionGreeksLoadedMsg struct {
	Greeks []api.OptionGreeks
}

// OptionQuoteLoadedMsg is sent when the underlying quote is loaded.
type OptionQuoteLoadedMsg struct {
	Last string
}

// FetchOptionExpirations returns a command that fetches option expirations.
func FetchOptionExpirations(symbol string, cfg *config.Config, store keyring.Store) tea.Cmd {
	return func() tea.Msg {
		if cfg.AccountUUID == "" {
			return OptionExpirationsErrorMsg{Err: fmt.Errorf("no account configured")}
		}

		token, err := getAuthToken(store, cfg.APIBaseURL, false)
		if err != nil {
			return OptionExpirationsErrorMsg{Err: err}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		client := api.NewClient(cfg.APIBaseURL, token)
		resp, err := client.GetOptionExpirations(ctx, cfg.AccountUUID, symbol)
		if err != nil {
			return OptionExpirationsErrorMsg{Err: err}
		}

		return OptionExpirationsLoadedMsg{
			Expirations: resp.Expirations,
		}
	}
}

// FetchOptionChain returns a command that fetches the option chain.
func FetchOptionChain(symbol, expiration string, cfg *config.Config, store keyring.Store) tea.Cmd {
	return func() tea.Msg {
		if cfg.AccountUUID == "" {
			return OptionChainErrorMsg{Err: fmt.Errorf("no account configured")}
		}

		token, err := getAuthToken(store, cfg.APIBaseURL, false)
		if err != nil {
			return OptionChainErrorMsg{Err: err}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		client := api.NewClient(cfg.APIBaseURL, token)
		resp, err := client.GetOptionChain(ctx, cfg.AccountUUID, symbol, expiration)
		if err != nil {
			return OptionChainErrorMsg{Err: err}
		}

		return OptionChainLoadedMsg{Chain: resp}
	}
}

// FetchOptionGreeks returns a command that fetches greeks for option symbols.
func FetchOptionGreeks(symbols []string, cfg *config.Config, store keyring.Store) tea.Cmd {
	return func() tea.Msg {
		if cfg.AccountUUID == "" || len(symbols) == 0 {
			return nil
		}

		token, err := getAuthToken(store, cfg.APIBaseURL, false)
		if err != nil {
			return nil
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		client := api.NewClient(cfg.APIBaseURL, token)
		resp, err := client.GetOptionGreeks(ctx, cfg.AccountUUID, symbols)
		if err != nil {
			return nil
		}

		return OptionGreeksLoadedMsg{Greeks: resp.Greeks}
	}
}
