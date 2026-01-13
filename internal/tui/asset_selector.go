package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/jonandersen/public-cli/internal/api"
	"github.com/jonandersen/public-cli/internal/config"
	"github.com/jonandersen/public-cli/internal/keyring"
)

// AssetSelectorMode represents the current mode of the asset selector.
type AssetSelectorMode int

const (
	AssetSelectorModeSearch AssetSelectorMode = iota
	AssetSelectorModeWatchlist
	AssetSelectorModePortfolio
)

// AssetSelectedMsg is sent when an asset is selected.
type AssetSelectedMsg struct {
	Symbol     string
	Type       string
	Name       string
	Optionable bool
	Quote      *Quote // Optional quote data if available
}

// AssetSelectorCancelledMsg is sent when the selector is cancelled.
type AssetSelectorCancelledMsg struct{}

// AssetInstrumentLoadedMsg is sent when instrument info is loaded.
type AssetInstrumentLoadedMsg struct {
	Symbol        string
	Type          string
	Name          string
	OptionTrading string
}

// AssetInstrumentErrorMsg is sent when instrument lookup fails.
type AssetInstrumentErrorMsg struct {
	Symbol string
	Err    error
}

// AssetSelectorModel holds the state for the asset selector component.
type AssetSelectorModel struct {
	Mode        AssetSelectorMode
	SearchInput textinput.Model
	Cursor      int

	// Data sources
	WatchSymbols []string
	WatchQuotes  map[string]Quote
	Positions    []Position

	// Search results
	SearchSymbol  string
	SearchResult  *AssetInstrumentLoadedMsg
	SearchLoading bool
	SearchErr     error

	// Dimensions
	Width  int
	Height int
}

// NewAssetSelectorModel creates a new asset selector in the specified mode.
func NewAssetSelectorModel(mode AssetSelectorMode) *AssetSelectorModel {
	ti := textinput.New()
	ti.Placeholder = "Enter symbol (e.g., AAPL)"
	ti.CharLimit = 10
	ti.Width = 20
	ti.Focus()

	return &AssetSelectorModel{
		Mode:        mode,
		SearchInput: ti,
		Cursor:      0,
		WatchQuotes: make(map[string]Quote),
	}
}

// SetWatchlistData sets the watchlist data for the selector.
func (m *AssetSelectorModel) SetWatchlistData(symbols []string, quotes map[string]Quote) {
	m.WatchSymbols = symbols
	m.WatchQuotes = quotes
}

// SetPortfolioData sets the portfolio data for the selector.
func (m *AssetSelectorModel) SetPortfolioData(positions []Position) {
	m.Positions = positions
}

// Update handles messages for the asset selector.
func (m *AssetSelectorModel) Update(msg tea.Msg, cfg *config.Config, store keyring.Store) (*AssetSelectorModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case AssetInstrumentLoadedMsg:
		if msg.Symbol == m.SearchSymbol {
			m.SearchLoading = false
			m.SearchResult = &msg
			m.SearchErr = nil
		}
		return m, nil

	case AssetInstrumentErrorMsg:
		if msg.Symbol == m.SearchSymbol {
			m.SearchLoading = false
			m.SearchResult = nil
			m.SearchErr = msg.Err
		}
		return m, nil

	case tea.KeyMsg:
		switch m.Mode {
		case AssetSelectorModeSearch:
			return m.updateSearchMode(msg, cfg, store)
		case AssetSelectorModeWatchlist:
			return m.updateWatchlistMode(msg)
		case AssetSelectorModePortfolio:
			return m.updatePortfolioMode(msg)
		}
	}

	return m, cmd
}

func (m *AssetSelectorModel) updateSearchMode(msg tea.KeyMsg, cfg *config.Config, store keyring.Store) (*AssetSelectorModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg.String() {
	case "enter":
		symbol := strings.ToUpper(strings.TrimSpace(m.SearchInput.Value()))
		if symbol != "" {
			// If we have a search result, return it
			if m.SearchResult != nil && m.SearchResult.Symbol == symbol {
				return m, func() tea.Msg {
					return AssetSelectedMsg{
						Symbol:     m.SearchResult.Symbol,
						Type:       m.SearchResult.Type,
						Name:       m.SearchResult.Name,
						Optionable: m.SearchResult.OptionTrading == "ENABLED",
					}
				}
			}
			// Otherwise just return the symbol
			return m, func() tea.Msg {
				return AssetSelectedMsg{
					Symbol: symbol,
					Type:   "EQUITY",
				}
			}
		}
		return m, nil

	case "esc":
		return m, func() tea.Msg {
			return AssetSelectorCancelledMsg{}
		}

	case "tab":
		// Trigger lookup when tab is pressed
		symbol := strings.ToUpper(strings.TrimSpace(m.SearchInput.Value()))
		if symbol != "" && symbol != m.SearchSymbol {
			m.SearchSymbol = symbol
			m.SearchLoading = true
			m.SearchResult = nil
			m.SearchErr = nil
			return m, FetchInstrumentInfo(symbol, cfg, store)
		}
		return m, nil

	case "w":
		// Check if input is empty to switch modes
		if m.SearchInput.Value() == "" {
			m.Mode = AssetSelectorModeWatchlist
			m.Cursor = 0
			return m, nil
		}

	case "p":
		// Check if input is empty to switch modes
		if m.SearchInput.Value() == "" {
			m.Mode = AssetSelectorModePortfolio
			m.Cursor = 0
			return m, nil
		}
	}

	// Update text input
	m.SearchInput, cmd = m.SearchInput.Update(msg)

	// Clear search result if input changed
	newSymbol := strings.ToUpper(strings.TrimSpace(m.SearchInput.Value()))
	if newSymbol != m.SearchSymbol {
		m.SearchResult = nil
		m.SearchErr = nil
	}

	return m, cmd
}

func (m *AssetSelectorModel) updateWatchlistMode(msg tea.KeyMsg) (*AssetSelectorModel, tea.Cmd) {
	switch msg.String() {
	case "enter":
		if len(m.WatchSymbols) > 0 && m.Cursor < len(m.WatchSymbols) {
			symbol := m.WatchSymbols[m.Cursor]
			quote, hasQuote := m.WatchQuotes[symbol]
			result := AssetSelectedMsg{
				Symbol: symbol,
				Type:   "EQUITY",
			}
			if hasQuote {
				result.Quote = &quote
			}
			return m, func() tea.Msg {
				return result
			}
		}
		return m, nil

	case "esc":
		return m, func() tea.Msg {
			return AssetSelectorCancelledMsg{}
		}

	case "up", "k":
		if m.Cursor > 0 {
			m.Cursor--
		}
		return m, nil

	case "down", "j":
		if m.Cursor < len(m.WatchSymbols)-1 {
			m.Cursor++
		}
		return m, nil

	case "s":
		m.Mode = AssetSelectorModeSearch
		m.SearchInput.Focus()
		return m, textinput.Blink

	case "p":
		m.Mode = AssetSelectorModePortfolio
		m.Cursor = 0
		return m, nil
	}

	return m, nil
}

func (m *AssetSelectorModel) updatePortfolioMode(msg tea.KeyMsg) (*AssetSelectorModel, tea.Cmd) {
	switch msg.String() {
	case "enter":
		if len(m.Positions) > 0 && m.Cursor < len(m.Positions) {
			pos := m.Positions[m.Cursor]
			return m, func() tea.Msg {
				return AssetSelectedMsg{
					Symbol: pos.Instrument.Symbol,
					Type:   pos.Instrument.Type,
					Name:   pos.Instrument.Name,
				}
			}
		}
		return m, nil

	case "esc":
		return m, func() tea.Msg {
			return AssetSelectorCancelledMsg{}
		}

	case "up", "k":
		if m.Cursor > 0 {
			m.Cursor--
		}
		return m, nil

	case "down", "j":
		if m.Cursor < len(m.Positions)-1 {
			m.Cursor++
		}
		return m, nil

	case "s":
		m.Mode = AssetSelectorModeSearch
		m.SearchInput.Focus()
		return m, textinput.Blink

	case "w":
		m.Mode = AssetSelectorModeWatchlist
		m.Cursor = 0
		return m, nil
	}

	return m, nil
}

// View renders the asset selector.
func (m *AssetSelectorModel) View() string {
	var b strings.Builder

	// Title with mode indicator
	modeIndicator := ""
	switch m.Mode {
	case AssetSelectorModeSearch:
		modeIndicator = "[s]"
	case AssetSelectorModeWatchlist:
		modeIndicator = "[w]"
	case AssetSelectorModePortfolio:
		modeIndicator = "[p]"
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary)
	b.WriteString(titleStyle.Render("Select Asset"))
	b.WriteString("  ")
	b.WriteString(LabelStyle.Render(modeIndicator))
	b.WriteString("\n\n")

	switch m.Mode {
	case AssetSelectorModeSearch:
		b.WriteString(m.renderSearchMode())
	case AssetSelectorModeWatchlist:
		b.WriteString(m.renderWatchlistMode())
	case AssetSelectorModePortfolio:
		b.WriteString(m.renderPortfolioMode())
	}

	b.WriteString("\n\n")
	b.WriteString(m.renderFooter())

	return b.String()
}

func (m *AssetSelectorModel) renderSearchMode() string {
	var b strings.Builder

	b.WriteString(LabelStyle.Render("Symbol:"))
	b.WriteString("\n")
	b.WriteString(InputStyle.Render(m.SearchInput.View()))
	b.WriteString("\n\n")

	if m.SearchLoading {
		b.WriteString(LabelStyle.Render("Looking up symbol..."))
	} else if m.SearchErr != nil {
		b.WriteString(ErrorStyle.Render(fmt.Sprintf("Not found: %s", m.SearchSymbol)))
	} else if m.SearchResult != nil {
		// Show instrument details
		rowStyle := lipgloss.NewStyle().Foreground(ColorPrimary)
		name := m.SearchResult.Name
		if name == "" {
			name = m.SearchResult.Symbol
		}
		optionStatus := ""
		if m.SearchResult.OptionTrading == "ENABLED" {
			optionStatus = GreenStyle.Render(" [Options]")
		}
		b.WriteString(rowStyle.Render(fmt.Sprintf("  %s  %s  %s%s",
			m.SearchResult.Symbol,
			name,
			m.SearchResult.Type,
			optionStatus)))
	} else if m.SearchInput.Value() != "" {
		b.WriteString(LabelStyle.Render("Press Tab to look up symbol"))
	}

	return b.String()
}

func (m *AssetSelectorModel) renderWatchlistMode() string {
	var b strings.Builder

	b.WriteString(LabelStyle.Render("From Watchlist:"))
	b.WriteString("\n")

	if len(m.WatchSymbols) == 0 {
		b.WriteString(LabelStyle.Render("  No symbols in watchlist"))
		b.WriteString("\n")
		b.WriteString(LabelStyle.Render("  Press 's' to search"))
		return b.String()
	}

	// Show up to 8 items
	maxItems := 8
	if len(m.WatchSymbols) < maxItems {
		maxItems = len(m.WatchSymbols)
	}

	for i := 0; i < maxItems; i++ {
		symbol := m.WatchSymbols[i]
		quote, hasQuote := m.WatchQuotes[symbol]

		prefix := "  "
		style := LabelStyle
		if i == m.Cursor {
			prefix = "> "
			style = lipgloss.NewStyle().Foreground(ColorSelectedFg).Background(ColorSelected).Bold(true)
		}

		row := symbol
		if hasQuote && quote.Outcome == "SUCCESS" {
			row = fmt.Sprintf("%-8s  $%-10s", symbol, quote.Last)
		}

		b.WriteString(style.Render(prefix + row))
		b.WriteString("\n")
	}

	if len(m.WatchSymbols) > maxItems {
		b.WriteString(LabelStyle.Render(fmt.Sprintf("  ... and %d more", len(m.WatchSymbols)-maxItems)))
	}

	return b.String()
}

func (m *AssetSelectorModel) renderPortfolioMode() string {
	var b strings.Builder

	b.WriteString(LabelStyle.Render("From Portfolio:"))
	b.WriteString("\n")

	if len(m.Positions) == 0 {
		b.WriteString(LabelStyle.Render("  No positions"))
		b.WriteString("\n")
		b.WriteString(LabelStyle.Render("  Press 's' to search"))
		return b.String()
	}

	// Show up to 8 items
	maxItems := 8
	if len(m.Positions) < maxItems {
		maxItems = len(m.Positions)
	}

	for i := 0; i < maxItems; i++ {
		pos := m.Positions[i]

		prefix := "  "
		style := LabelStyle
		if i == m.Cursor {
			prefix = "> "
			style = lipgloss.NewStyle().Foreground(ColorSelectedFg).Background(ColorSelected).Bold(true)
		}

		row := fmt.Sprintf("%-8s  %s shares  $%s",
			pos.Instrument.Symbol,
			pos.Quantity,
			pos.CurrentValue)

		b.WriteString(style.Render(prefix + row))
		b.WriteString("\n")
	}

	if len(m.Positions) > maxItems {
		b.WriteString(LabelStyle.Render(fmt.Sprintf("  ... and %d more", len(m.Positions)-maxItems)))
	}

	return b.String()
}

func (m *AssetSelectorModel) renderFooter() string {
	var parts []string

	parts = append(parts, KeyStyle.Render("Enter")+" "+DescStyle.Render("select"))

	switch m.Mode {
	case AssetSelectorModeSearch:
		parts = append(parts, KeyStyle.Render("Tab")+" "+DescStyle.Render("lookup"))
		parts = append(parts, KeyStyle.Render("w")+" "+DescStyle.Render("watchlist"))
		parts = append(parts, KeyStyle.Render("p")+" "+DescStyle.Render("portfolio"))
	case AssetSelectorModeWatchlist:
		parts = append(parts, KeyStyle.Render("s")+" "+DescStyle.Render("search"))
		parts = append(parts, KeyStyle.Render("p")+" "+DescStyle.Render("portfolio"))
	case AssetSelectorModePortfolio:
		parts = append(parts, KeyStyle.Render("s")+" "+DescStyle.Render("search"))
		parts = append(parts, KeyStyle.Render("w")+" "+DescStyle.Render("watchlist"))
	}

	parts = append(parts, KeyStyle.Render("Esc")+" "+DescStyle.Render("cancel"))

	return strings.Join(parts, "  ")
}

// FetchInstrumentInfo returns a command that fetches instrument information.
func FetchInstrumentInfo(symbol string, cfg *config.Config, store keyring.Store) tea.Cmd {
	return func() tea.Msg {
		token, err := getAuthToken(store, cfg.APIBaseURL, false)
		if err != nil {
			return AssetInstrumentErrorMsg{Symbol: symbol, Err: err}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		client := api.NewClient(cfg.APIBaseURL, token)
		inst, err := client.GetInstrument(ctx, symbol, "EQUITY")
		if err != nil {
			return AssetInstrumentErrorMsg{Symbol: symbol, Err: err}
		}

		return AssetInstrumentLoadedMsg{
			Symbol:        inst.Instrument.Symbol,
			Type:          inst.Instrument.Type,
			Name:          "", // Name not available in instrument response
			OptionTrading: inst.OptionTrading,
		}
	}
}
