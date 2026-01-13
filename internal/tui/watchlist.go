package tui

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/jonandersen/public-cli/internal/api"
	"github.com/jonandersen/public-cli/internal/config"
	"github.com/jonandersen/public-cli/internal/keyring"
)

// WatchlistState represents the loading state of watchlist data.
type WatchlistState int

const (
	WatchlistStateLoading WatchlistState = iota
	WatchlistStateLoaded
	WatchlistStateError
)

// WatchlistMode represents the input mode of the watchlist view.
type WatchlistMode int

const (
	WatchlistModeNormal WatchlistMode = iota
	WatchlistModeAdding
	WatchlistModeDeleting
)

// WatchlistModel holds the state for the watchlist view.
type WatchlistModel struct {
	State        WatchlistState
	Symbols      []string
	Quotes       map[string]Quote
	Err          error
	LastUpdated  time.Time
	Table        table.Model
	Mode         WatchlistMode
	AddInput     textinput.Model
	DeleteSymbol string
}

// NewWatchlistModel creates a new watchlist model.
func NewWatchlistModel(symbols []string) *WatchlistModel {
	cols := []table.Column{
		{Title: "Symbol", Width: 10},
		{Title: "Last", Width: 12},
		{Title: "Bid", Width: 10},
		{Title: "Ask", Width: 10},
		{Title: "Volume", Width: 14},
	}

	t := table.New(
		table.WithColumns(cols),
		table.WithFocused(true),
		table.WithHeight(10),
	)
	t.SetStyles(TableStyles())

	ti := textinput.New()
	ti.Placeholder = "Enter symbol (e.g., AAPL)"
	ti.CharLimit = 10
	ti.Width = 20

	if symbols == nil {
		symbols = []string{}
	}

	// If no symbols, start in loaded state (empty watchlist)
	// Otherwise start loading and wait for quotes to be fetched
	initialState := WatchlistStateLoading
	if len(symbols) == 0 {
		initialState = WatchlistStateLoaded
	}

	return &WatchlistModel{
		State:    initialState,
		Symbols:  symbols,
		Quotes:   make(map[string]Quote),
		Table:    t,
		Mode:     WatchlistModeNormal,
		AddInput: ti,
	}
}

// SetHeight sets the table height.
func (m *WatchlistModel) SetHeight(height int) {
	m.Table.SetHeight(height)
}

// Update handles messages for the watchlist view.
// Returns the model, command, and whether the event was handled.
func (m *WatchlistModel) Update(msg tea.Msg, uiCfg *UIConfig) (*WatchlistModel, tea.Cmd, bool) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case WatchlistQuotesMsg:
		m.State = WatchlistStateLoaded
		m.Quotes = msg.Quotes
		m.LastUpdated = time.Now()
		m.Err = nil
		m.updateTable()
		return m, nil, true

	case WatchlistErrorMsg:
		m.State = WatchlistStateError
		m.Err = msg.Err
		return m, nil, true

	case WatchlistSavedMsg:
		// Config saved successfully
		return m, nil, true

	case tea.KeyMsg:
		switch m.Mode {
		case WatchlistModeAdding:
			switch msg.String() {
			case "enter":
				symbol := strings.ToUpper(strings.TrimSpace(m.AddInput.Value()))
				if symbol != "" {
					// Check if symbol already exists
					exists := false
					for _, s := range m.Symbols {
						if s == symbol {
							exists = true
							break
						}
					}
					if !exists {
						m.Symbols = append(m.Symbols, symbol)
						cmds = append(cmds, m.saveWatchlist(uiCfg))
					}
				}
				m.Mode = WatchlistModeNormal
				m.AddInput.Reset()
				return m, tea.Batch(cmds...), true
			case "esc":
				m.Mode = WatchlistModeNormal
				m.AddInput.Reset()
				return m, nil, true
			default:
				m.AddInput, cmd = m.AddInput.Update(msg)
				return m, cmd, true
			}

		case WatchlistModeDeleting:
			switch msg.String() {
			case "y", "Y":
				// Remove the symbol
				newSymbols := make([]string, 0, len(m.Symbols))
				for _, s := range m.Symbols {
					if s != m.DeleteSymbol {
						newSymbols = append(newSymbols, s)
					}
				}
				m.Symbols = newSymbols
				delete(m.Quotes, m.DeleteSymbol)
				m.updateTable()
				cmds = append(cmds, m.saveWatchlist(uiCfg))
				m.Mode = WatchlistModeNormal
				m.DeleteSymbol = ""
				return m, tea.Batch(cmds...), true
			case "n", "N", "esc":
				m.Mode = WatchlistModeNormal
				m.DeleteSymbol = ""
				return m, nil, true
			}
			return m, nil, true

		case WatchlistModeNormal:
			switch msg.String() {
			case "a":
				m.Mode = WatchlistModeAdding
				m.AddInput.Focus()
				return m, textinput.Blink, true
			case "d", "x":
				if len(m.Symbols) > 0 {
					selectedRow := m.Table.SelectedRow()
					if len(selectedRow) > 0 {
						m.DeleteSymbol = selectedRow[0]
						m.Mode = WatchlistModeDeleting
					}
				}
				return m, nil, true
			}
		}
	}

	// Pass to table in normal mode
	if m.Mode == WatchlistModeNormal {
		m.Table, cmd = m.Table.Update(msg)
		return m, cmd, false
	}

	return m, nil, false
}

// updateTable updates the table rows from watchlist data.
func (m *WatchlistModel) updateTable() {
	rows := make([]table.Row, 0, len(m.Symbols))
	for _, sym := range m.Symbols {
		quote, hasQuote := m.Quotes[sym]
		if hasQuote && quote.Outcome == "SUCCESS" {
			rows = append(rows, table.Row{
				sym,
				"$" + quote.Last,
				"$" + quote.Bid,
				"$" + quote.Ask,
				formatVolume(strconv.FormatInt(quote.Volume, 10)),
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
	m.Table.SetRows(rows)
}

// View renders the watchlist view.
func (m *WatchlistModel) View() string {
	var b strings.Builder

	// Handle input modes first
	switch m.Mode {
	case WatchlistModeAdding:
		b.WriteString(SummaryStyle.Render("Add Symbol"))
		b.WriteString("\n\n")
		b.WriteString(InputStyle.Render(m.AddInput.View()))
		b.WriteString("\n\n")
		b.WriteString(LabelStyle.Render("Press Enter to add, Esc to cancel"))
		return b.String()

	case WatchlistModeDeleting:
		b.WriteString(WarningStyle.Render(fmt.Sprintf("Delete %s from watchlist?", m.DeleteSymbol)))
		b.WriteString("\n\n")
		b.WriteString(LabelStyle.Render("Press Y to confirm, N to cancel"))
		return b.String()
	}

	// Normal mode - show watchlist
	switch m.State {
	case WatchlistStateLoading:
		b.WriteString("Loading quotes...")
		return b.String()

	case WatchlistStateError:
		b.WriteString(ErrorStyle.Render(fmt.Sprintf("Error: %v", m.Err)))
		b.WriteString("\n\nPress 'r' to retry")
		return b.String()

	case WatchlistStateLoaded:
		b.WriteString(SummaryStyle.Render("Watchlist"))
		b.WriteString(LabelStyle.Render(fmt.Sprintf(" (%d symbols)", len(m.Symbols))))
		b.WriteString("\n\n")

		if len(m.Symbols) == 0 {
			b.WriteString(LabelStyle.Render("No symbols in watchlist"))
			b.WriteString("\n\n")
			b.WriteString(LabelStyle.Render("Press 'a' to add a symbol"))
		} else {
			b.WriteString(m.Table.View())
			b.WriteString("\n")
			b.WriteString(LabelStyle.Render(fmt.Sprintf("Updated: %s", m.LastUpdated.Format("3:04:05 PM"))))
		}
	}

	return b.String()
}

// SelectedSymbol returns the currently selected symbol, if any.
func (m *WatchlistModel) SelectedSymbol() string {
	if m.Mode != WatchlistModeNormal {
		return ""
	}
	selectedRow := m.Table.SelectedRow()
	if len(selectedRow) > 0 {
		return selectedRow[0]
	}
	return ""
}

// saveWatchlist returns a command to save the watchlist config.
func (m *WatchlistModel) saveWatchlist(uiCfg *UIConfig) tea.Cmd {
	return func() tea.Msg {
		uiCfg.Watchlist = m.Symbols
		if err := SaveConfig(uiCfg); err != nil {
			return WatchlistErrorMsg{Err: fmt.Errorf("failed to save watchlist: %w", err)}
		}
		return WatchlistSavedMsg{}
	}
}

// FetchWatchlistQuotes returns a command that fetches quotes for watchlist symbols.
func FetchWatchlistQuotes(symbols []string, cfg *config.Config, store keyring.Store) tea.Cmd {
	return func() tea.Msg {
		if len(symbols) == 0 {
			return WatchlistQuotesMsg{Quotes: make(map[string]Quote)}
		}

		if cfg.AccountUUID == "" {
			return WatchlistErrorMsg{Err: fmt.Errorf("no account configured")}
		}

		token, err := getAuthToken(store, cfg.APIBaseURL, false)
		if err != nil {
			return WatchlistErrorMsg{Err: err}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Build request
		instruments := make([]QuoteInstrument, 0, len(symbols))
		for _, sym := range symbols {
			instruments = append(instruments, QuoteInstrument{
				Symbol: strings.ToUpper(sym),
				Type:   "EQUITY",
			})
		}

		reqBody := QuoteRequest{Instruments: instruments}
		body, err := json.Marshal(reqBody)
		if err != nil {
			return WatchlistErrorMsg{Err: fmt.Errorf("failed to encode request: %w", err)}
		}

		client := api.NewClient(cfg.APIBaseURL, token)
		path := fmt.Sprintf("/userapigateway/marketdata/%s/quotes", cfg.AccountUUID)
		resp, err := client.Post(ctx, path, bytes.NewReader(body))
		if err != nil {
			return WatchlistErrorMsg{Err: fmt.Errorf("failed to fetch quotes: %w", err)}
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != 200 {
			respBody, _ := io.ReadAll(resp.Body)
			return WatchlistErrorMsg{Err: fmt.Errorf("API error: %d - %s", resp.StatusCode, string(respBody))}
		}

		var quotesResp QuotesResponse
		if err := json.NewDecoder(resp.Body).Decode(&quotesResp); err != nil {
			return WatchlistErrorMsg{Err: fmt.Errorf("failed to decode response: %w", err)}
		}

		quotes := make(map[string]Quote)
		for _, q := range quotesResp.Quotes {
			quotes[q.Instrument.Symbol] = q
		}

		return WatchlistQuotesMsg{Quotes: quotes}
	}
}
