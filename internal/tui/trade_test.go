package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestNewTradeModel(t *testing.T) {
	m := NewTradeModel()

	assert.Equal(t, TradeStateIdle, m.State)
	assert.Equal(t, TradeModeForm, m.Mode)
	assert.Equal(t, TradeFieldSymbol, m.FocusedField)
	assert.Equal(t, TradeSideBuy, m.Side)
	assert.Equal(t, TradeOrderTypeMarket, m.OrderType)
	assert.False(t, m.ShowAssetSelector)
	assert.Nil(t, m.AssetSelector)
}

func TestTradeModelSetWatchlistData(t *testing.T) {
	m := NewTradeModel()

	symbols := []string{"AAPL", "GOOGL", "MSFT"}
	quotes := map[string]Quote{
		"AAPL": {Instrument: QuoteInstrument{Symbol: "AAPL"}, Outcome: "SUCCESS", Last: "150.00"},
	}

	m.SetWatchlistData(symbols, quotes)

	assert.Equal(t, symbols, m.WatchlistSymbols)
	assert.Equal(t, quotes, m.WatchlistQuotes)
}

func TestTradeModelOpenAssetSelector(t *testing.T) {
	m := NewTradeModel()
	cfg := testConfig()
	store := testStore()

	// Set watchlist data
	symbols := []string{"AAPL", "GOOGL"}
	quotes := map[string]Quote{
		"AAPL": {Instrument: QuoteInstrument{Symbol: "AAPL"}, Outcome: "SUCCESS", Last: "150.00"},
	}
	m.SetWatchlistData(symbols, quotes)

	// Focus should be on symbol field by default
	assert.Equal(t, TradeFieldSymbol, m.FocusedField)

	// Press 'w' to open asset selector
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}}, cfg, store)

	assert.True(t, m.ShowAssetSelector)
	assert.NotNil(t, m.AssetSelector)
	assert.Equal(t, AssetSelectorModeWatchlist, m.AssetSelector.Mode)
	assert.Equal(t, symbols, m.AssetSelector.WatchSymbols)
	assert.Nil(t, cmd) // Opening selector doesn't return a command
}

func TestTradeModelAssetSelectorWKeyOnlyOnSymbolField(t *testing.T) {
	m := NewTradeModel()
	cfg := testConfig()
	store := testStore()

	// Move focus to side field
	m.FocusedField = TradeFieldSide

	// Press 'w' - should not open selector
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}}, cfg, store)

	assert.False(t, m.ShowAssetSelector)
	assert.Nil(t, m.AssetSelector)
}

func TestTradeModelAssetSelectorCancel(t *testing.T) {
	m := NewTradeModel()
	cfg := testConfig()
	store := testStore()

	// Open asset selector
	m.AssetSelector = NewAssetSelectorModel(AssetSelectorModeWatchlist)
	m.ShowAssetSelector = true

	// Receive cancel message
	m, _ = m.Update(AssetSelectorCancelledMsg{}, cfg, store)

	assert.False(t, m.ShowAssetSelector)
}

func TestTradeModelAssetSelectorSelection(t *testing.T) {
	m := NewTradeModel()
	cfg := testConfig()
	store := testStore()

	// Open asset selector
	m.AssetSelector = NewAssetSelectorModel(AssetSelectorModeWatchlist)
	m.ShowAssetSelector = true

	// Receive selection message
	m, cmd := m.Update(AssetSelectedMsg{Symbol: "NVDA", Type: "EQUITY"}, cfg, store)

	assert.False(t, m.ShowAssetSelector)
	assert.Equal(t, "NVDA", m.SymbolInput.Value())
	assert.Equal(t, TradeFieldQuantity, m.FocusedField)
	assert.Equal(t, TradeStateFetchingQuote, m.State)
	assert.NotNil(t, cmd) // Should return command to fetch quote
}

func TestTradeModelKeyRoutingWhenSelectorOpen(t *testing.T) {
	m := NewTradeModel()
	cfg := testConfig()
	store := testStore()

	// Set up asset selector in watchlist mode with data
	m.AssetSelector = NewAssetSelectorModel(AssetSelectorModeWatchlist)
	m.AssetSelector.SetWatchlistData([]string{"AAPL", "GOOGL"}, nil)
	m.ShowAssetSelector = true

	// Press down key - should be routed to selector, not trade form
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown}, cfg, store)

	// Selector should have received the key
	assert.Equal(t, 1, m.AssetSelector.Cursor)
}

func TestTradeViewShowsHintForWatchlist(t *testing.T) {
	m := NewTradeModel()
	m.FocusedField = TradeFieldSymbol

	view := m.View()

	assert.Contains(t, view, "press 'w' for watchlist")
}

func TestTradeViewHidesHintWhenQuoteLoaded(t *testing.T) {
	m := NewTradeModel()
	m.FocusedField = TradeFieldSymbol
	m.QuoteLoaded = true
	m.Quote = &Quote{Last: "150.00"}

	view := m.View()

	assert.NotContains(t, view, "press 'w' for watchlist")
	assert.Contains(t, view, "$150.00")
}

func TestTradeViewShowsAssetSelectorWhenOpen(t *testing.T) {
	m := NewTradeModel()
	m.AssetSelector = NewAssetSelectorModel(AssetSelectorModeWatchlist)
	m.ShowAssetSelector = true

	view := m.View()

	assert.Contains(t, view, "Select Asset")
}
