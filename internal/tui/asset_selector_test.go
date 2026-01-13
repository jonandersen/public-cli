package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAssetSelectorModel(t *testing.T) {
	tests := []struct {
		name string
		mode AssetSelectorMode
	}{
		{"search mode", AssetSelectorModeSearch},
		{"watchlist mode", AssetSelectorModeWatchlist},
		{"portfolio mode", AssetSelectorModePortfolio},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewAssetSelectorModel(tt.mode)

			assert.Equal(t, tt.mode, m.Mode)
			assert.Equal(t, 0, m.Cursor)
			assert.NotNil(t, m.WatchQuotes)
		})
	}
}

func TestAssetSelectorSetWatchlistData(t *testing.T) {
	m := NewAssetSelectorModel(AssetSelectorModeWatchlist)

	symbols := []string{"AAPL", "GOOGL", "MSFT"}
	quotes := map[string]Quote{
		"AAPL": {Instrument: QuoteInstrument{Symbol: "AAPL"}, Outcome: "SUCCESS", Last: "150.00"},
	}

	m.SetWatchlistData(symbols, quotes)

	assert.Equal(t, symbols, m.WatchSymbols)
	assert.Equal(t, quotes, m.WatchQuotes)
}

func TestAssetSelectorSetPortfolioData(t *testing.T) {
	m := NewAssetSelectorModel(AssetSelectorModePortfolio)

	positions := []Position{
		{Instrument: Instrument{Symbol: "AAPL", Name: "Apple Inc."}, Quantity: "10", CurrentValue: "1500.00"},
		{Instrument: Instrument{Symbol: "GOOGL", Name: "Alphabet Inc."}, Quantity: "5", CurrentValue: "750.00"},
	}

	m.SetPortfolioData(positions)

	assert.Equal(t, positions, m.Positions)
}

func TestAssetSelectorSearchModeNavigation(t *testing.T) {
	m := NewAssetSelectorModel(AssetSelectorModeSearch)
	cfg := testConfig()
	store := testStore()

	// Test escape cancels
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc}, cfg, store)
	require.NotNil(t, cmd)
	msg := cmd()
	_, ok := msg.(AssetSelectorCancelledMsg)
	assert.True(t, ok, "Expected AssetSelectorCancelledMsg")
}

func TestAssetSelectorSearchModeEnterWithInput(t *testing.T) {
	m := NewAssetSelectorModel(AssetSelectorModeSearch)
	cfg := testConfig()
	store := testStore()

	// Set input value
	m.SearchInput.SetValue("AAPL")

	// Press enter
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter}, cfg, store)
	require.NotNil(t, cmd)
	msg := cmd()
	selected, ok := msg.(AssetSelectedMsg)
	assert.True(t, ok, "Expected AssetSelectedMsg")
	assert.Equal(t, "AAPL", selected.Symbol)
	assert.Equal(t, "EQUITY", selected.Type)
}

func TestAssetSelectorSearchModeEnterEmpty(t *testing.T) {
	m := NewAssetSelectorModel(AssetSelectorModeSearch)
	cfg := testConfig()
	store := testStore()

	// Press enter with empty input
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter}, cfg, store)
	assert.Nil(t, cmd, "Should not return command for empty input")
}

func TestAssetSelectorSearchModeSwitchToWatchlist(t *testing.T) {
	m := NewAssetSelectorModel(AssetSelectorModeSearch)
	cfg := testConfig()
	store := testStore()

	// With empty input, w should switch to watchlist mode
	m.SearchInput.SetValue("")
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}}, cfg, store)

	assert.Equal(t, AssetSelectorModeWatchlist, m.Mode)
}

func TestAssetSelectorSearchModeSwitchToPortfolio(t *testing.T) {
	m := NewAssetSelectorModel(AssetSelectorModeSearch)
	cfg := testConfig()
	store := testStore()

	// With empty input, p should switch to portfolio mode
	m.SearchInput.SetValue("")
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}}, cfg, store)

	assert.Equal(t, AssetSelectorModePortfolio, m.Mode)
}

func TestAssetSelectorWatchlistModeNavigation(t *testing.T) {
	m := NewAssetSelectorModel(AssetSelectorModeWatchlist)
	m.SetWatchlistData([]string{"AAPL", "GOOGL", "MSFT"}, nil)
	cfg := testConfig()
	store := testStore()

	// Initial cursor at 0
	assert.Equal(t, 0, m.Cursor)

	// Move down
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown}, cfg, store)
	assert.Equal(t, 1, m.Cursor)

	// Move down again
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown}, cfg, store)
	assert.Equal(t, 2, m.Cursor)

	// Move down at end (should stay at 2)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown}, cfg, store)
	assert.Equal(t, 2, m.Cursor)

	// Move up
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp}, cfg, store)
	assert.Equal(t, 1, m.Cursor)
}

func TestAssetSelectorWatchlistModeVimNavigation(t *testing.T) {
	m := NewAssetSelectorModel(AssetSelectorModeWatchlist)
	m.SetWatchlistData([]string{"AAPL", "GOOGL", "MSFT"}, nil)
	cfg := testConfig()
	store := testStore()

	// j moves down
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}, cfg, store)
	assert.Equal(t, 1, m.Cursor)

	// k moves up
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}, cfg, store)
	assert.Equal(t, 0, m.Cursor)
}

func TestAssetSelectorWatchlistModeSelect(t *testing.T) {
	m := NewAssetSelectorModel(AssetSelectorModeWatchlist)
	m.SetWatchlistData([]string{"AAPL", "GOOGL"}, map[string]Quote{
		"AAPL": {Instrument: QuoteInstrument{Symbol: "AAPL"}, Outcome: "SUCCESS", Last: "150.00"},
	})
	cfg := testConfig()
	store := testStore()

	// Press enter to select
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter}, cfg, store)
	require.NotNil(t, cmd)
	msg := cmd()
	selected, ok := msg.(AssetSelectedMsg)
	assert.True(t, ok, "Expected AssetSelectedMsg")
	assert.Equal(t, "AAPL", selected.Symbol)
	assert.NotNil(t, selected.Quote)
	assert.Equal(t, "150.00", selected.Quote.Last)
}

func TestAssetSelectorWatchlistModeSwitchToSearch(t *testing.T) {
	m := NewAssetSelectorModel(AssetSelectorModeWatchlist)
	cfg := testConfig()
	store := testStore()

	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}}, cfg, store)

	assert.Equal(t, AssetSelectorModeSearch, m.Mode)
	assert.NotNil(t, cmd) // Should return textinput.Blink
}

func TestAssetSelectorPortfolioModeNavigation(t *testing.T) {
	m := NewAssetSelectorModel(AssetSelectorModePortfolio)
	m.SetPortfolioData([]Position{
		{Instrument: Instrument{Symbol: "AAPL"}, Quantity: "10", CurrentValue: "1500.00"},
		{Instrument: Instrument{Symbol: "GOOGL"}, Quantity: "5", CurrentValue: "750.00"},
	})
	cfg := testConfig()
	store := testStore()

	// Move down
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown}, cfg, store)
	assert.Equal(t, 1, m.Cursor)

	// Move down at end (should stay at 1)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown}, cfg, store)
	assert.Equal(t, 1, m.Cursor)

	// Move up
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp}, cfg, store)
	assert.Equal(t, 0, m.Cursor)
}

func TestAssetSelectorPortfolioModeSelect(t *testing.T) {
	m := NewAssetSelectorModel(AssetSelectorModePortfolio)
	m.SetPortfolioData([]Position{
		{Instrument: Instrument{Symbol: "AAPL", Name: "Apple Inc.", Type: "EQUITY"}, Quantity: "10", CurrentValue: "1500.00"},
	})
	cfg := testConfig()
	store := testStore()

	// Press enter to select
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter}, cfg, store)
	require.NotNil(t, cmd)
	msg := cmd()
	selected, ok := msg.(AssetSelectedMsg)
	assert.True(t, ok, "Expected AssetSelectedMsg")
	assert.Equal(t, "AAPL", selected.Symbol)
	assert.Equal(t, "Apple Inc.", selected.Name)
	assert.Equal(t, "EQUITY", selected.Type)
}

func TestAssetSelectorViewSearchMode(t *testing.T) {
	m := NewAssetSelectorModel(AssetSelectorModeSearch)

	view := m.View()

	assert.Contains(t, view, "Select Asset")
	assert.Contains(t, view, "[s]")
	assert.Contains(t, view, "Symbol:")
	assert.Contains(t, view, "Enter")
	assert.Contains(t, view, "Esc")
}

func TestAssetSelectorViewWatchlistMode(t *testing.T) {
	m := NewAssetSelectorModel(AssetSelectorModeWatchlist)
	m.SetWatchlistData([]string{"AAPL", "GOOGL"}, map[string]Quote{
		"AAPL": {Instrument: QuoteInstrument{Symbol: "AAPL"}, Outcome: "SUCCESS", Last: "150.00"},
	})

	view := m.View()

	assert.Contains(t, view, "Select Asset")
	assert.Contains(t, view, "[w]")
	assert.Contains(t, view, "From Watchlist:")
	assert.Contains(t, view, "AAPL")
	assert.Contains(t, view, "GOOGL")
}

func TestAssetSelectorViewWatchlistModeEmpty(t *testing.T) {
	m := NewAssetSelectorModel(AssetSelectorModeWatchlist)
	m.SetWatchlistData([]string{}, nil)

	view := m.View()

	assert.Contains(t, view, "No symbols in watchlist")
	assert.Contains(t, view, "Press 's' to search")
}

func TestAssetSelectorViewPortfolioMode(t *testing.T) {
	m := NewAssetSelectorModel(AssetSelectorModePortfolio)
	m.SetPortfolioData([]Position{
		{Instrument: Instrument{Symbol: "AAPL"}, Quantity: "10", CurrentValue: "1500.00"},
	})

	view := m.View()

	assert.Contains(t, view, "Select Asset")
	assert.Contains(t, view, "[p]")
	assert.Contains(t, view, "From Portfolio:")
	assert.Contains(t, view, "AAPL")
	assert.Contains(t, view, "10 shares")
}

func TestAssetSelectorViewPortfolioModeEmpty(t *testing.T) {
	m := NewAssetSelectorModel(AssetSelectorModePortfolio)
	m.SetPortfolioData([]Position{})

	view := m.View()

	assert.Contains(t, view, "No positions")
	assert.Contains(t, view, "Press 's' to search")
}

func TestAssetSelectorInstrumentLoaded(t *testing.T) {
	m := NewAssetSelectorModel(AssetSelectorModeSearch)
	m.SearchSymbol = "AAPL"
	m.SearchLoading = true
	cfg := testConfig()
	store := testStore()

	msg := AssetInstrumentLoadedMsg{
		Symbol:        "AAPL",
		Type:          "EQUITY",
		Name:          "Apple Inc.",
		OptionTrading: "ENABLED",
	}

	m, _ = m.Update(msg, cfg, store)

	assert.False(t, m.SearchLoading)
	assert.NotNil(t, m.SearchResult)
	assert.Equal(t, "AAPL", m.SearchResult.Symbol)
	assert.Equal(t, "ENABLED", m.SearchResult.OptionTrading)
}

func TestAssetSelectorInstrumentError(t *testing.T) {
	m := NewAssetSelectorModel(AssetSelectorModeSearch)
	m.SearchSymbol = "INVALID"
	m.SearchLoading = true
	cfg := testConfig()
	store := testStore()

	msg := AssetInstrumentErrorMsg{
		Symbol: "INVALID",
		Err:    assert.AnError,
	}

	m, _ = m.Update(msg, cfg, store)

	assert.False(t, m.SearchLoading)
	assert.Nil(t, m.SearchResult)
	assert.NotNil(t, m.SearchErr)
}

func TestAssetSelectorViewSearchModeWithResult(t *testing.T) {
	m := NewAssetSelectorModel(AssetSelectorModeSearch)
	m.SearchSymbol = "AAPL"
	m.SearchResult = &AssetInstrumentLoadedMsg{
		Symbol:        "AAPL",
		Type:          "EQUITY",
		Name:          "Apple Inc.",
		OptionTrading: "ENABLED",
	}

	view := m.View()

	assert.Contains(t, view, "AAPL")
	assert.Contains(t, view, "EQUITY")
	assert.Contains(t, view, "[Options]")
}

func TestAssetSelectorViewSearchModeLoading(t *testing.T) {
	m := NewAssetSelectorModel(AssetSelectorModeSearch)
	m.SearchLoading = true

	view := m.View()

	assert.Contains(t, view, "Looking up symbol")
}

func TestAssetSelectorViewSearchModeError(t *testing.T) {
	m := NewAssetSelectorModel(AssetSelectorModeSearch)
	m.SearchSymbol = "INVALID"
	m.SearchErr = assert.AnError

	view := m.View()

	assert.Contains(t, view, "Not found")
	assert.Contains(t, view, "INVALID")
}

func TestAssetSelectorSearchWithExistingResult(t *testing.T) {
	m := NewAssetSelectorModel(AssetSelectorModeSearch)
	m.SearchSymbol = "AAPL"
	m.SearchResult = &AssetInstrumentLoadedMsg{
		Symbol:        "AAPL",
		Type:          "EQUITY",
		OptionTrading: "ENABLED",
	}
	m.SearchInput.SetValue("AAPL")
	cfg := testConfig()
	store := testStore()

	// Press enter with existing result
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter}, cfg, store)
	require.NotNil(t, cmd)
	msg := cmd()
	selected, ok := msg.(AssetSelectedMsg)
	assert.True(t, ok, "Expected AssetSelectedMsg")
	assert.Equal(t, "AAPL", selected.Symbol)
	assert.True(t, selected.Optionable)
}

func TestAssetSelectorTabTriggersLookup(t *testing.T) {
	m := NewAssetSelectorModel(AssetSelectorModeSearch)
	m.SearchInput.SetValue("AAPL")
	cfg := testConfig()
	store := testStore()

	// Press tab
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyTab}, cfg, store)

	assert.True(t, m.SearchLoading)
	assert.Equal(t, "AAPL", m.SearchSymbol)
	assert.NotNil(t, cmd) // Should return FetchInstrumentInfo command
}

func TestAssetSelectorTextInputUpdate(t *testing.T) {
	m := NewAssetSelectorModel(AssetSelectorModeSearch)
	cfg := testConfig()
	store := testStore()

	// Type a character
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'A'}}, cfg, store)

	// The text input should have been updated
	assert.Contains(t, m.SearchInput.Value(), "A")
}
