package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/jonandersen/public-cli/internal/config"
	"github.com/jonandersen/public-cli/internal/keyring"
)

func testConfig() *config.Config {
	return &config.Config{
		APIBaseURL:  "https://api.public.com",
		AccountUUID: "test-account-123",
	}
}

func testUIConfig() *UIConfig {
	return &UIConfig{}
}

func testUIConfigWithWatchlist() *UIConfig {
	return &UIConfig{
		Watchlist: []string{"AAPL", "GOOGL"},
	}
}

func testStore() keyring.Store {
	store := keyring.NewMockStore()
	_ = store.Set(keyring.ServiceName, keyring.KeySecretKey, "test-secret")
	return store
}

func TestNew(t *testing.T) {
	m := New(testConfig(), testUIConfig(), testStore())
	assert.Equal(t, ViewPortfolio, m.currentView)
	assert.Equal(t, PortfolioStateLoading, m.portfolio.State)
}

func TestModelInit(t *testing.T) {
	m := New(testConfig(), testUIConfig(), testStore())
	cmd := m.Init()
	// Init should return a batch command (fetch + tick)
	assert.NotNil(t, cmd)
}

func TestModelView(t *testing.T) {
	m := New(testConfig(), testUIConfig(), testStore())
	m.width = 80
	m.height = 24
	m.ready = true
	view := m.View()

	// Should contain header
	assert.Contains(t, view, "pub")
	// Should contain footer with key hints
	assert.Contains(t, view, "q")
}

func TestModelViewLoading(t *testing.T) {
	m := New(testConfig(), testUIConfig(), testStore())
	m.width = 80
	m.height = 24
	m.ready = true
	m.portfolio.State = PortfolioStateLoading

	view := m.View()
	assert.Contains(t, view, "Loading")
}

func TestModelViewError(t *testing.T) {
	m := New(testConfig(), testUIConfig(), testStore())
	m.width = 80
	m.height = 24
	m.ready = true
	m.portfolio.State = PortfolioStateError
	m.portfolio.Err = assert.AnError

	view := m.View()
	assert.Contains(t, view, "Error")
	assert.Contains(t, view, "retry")
}

func TestModelViewWithPositions(t *testing.T) {
	m := New(testConfig(), testUIConfig(), testStore())
	m.width = 120
	m.height = 30
	m.ready = true
	m.portfolio.State = PortfolioStateLoaded
	m.portfolio.Data = Portfolio{
		BuyingPower: BuyingPower{
			BuyingPower:        "1000.00",
			OptionsBuyingPower: "500.00",
		},
		Equity: []Equity{
			{Type: "TOTAL", Value: "5000.00"},
			{Type: "CASH", Value: "1000.00"},
		},
		Positions: []Position{
			{
				Instrument:   Instrument{Symbol: "AAPL", Name: "Apple Inc."},
				Quantity:     "10",
				CurrentValue: "1500.00",
				LastPrice:    Price{LastPrice: "150.00"},
				PositionDailyGain: Gain{
					GainValue:      "25.00",
					GainPercentage: "1.7",
				},
				CostBasis: CostBasis{
					GainValue:      "100.00",
					GainPercentage: "7.1",
				},
			},
		},
	}
	m.portfolio.updateTable()

	view := m.View()
	assert.Contains(t, view, "Account Summary")
	assert.Contains(t, view, "AAPL")
	assert.Contains(t, view, "Positions")
}

func TestWatchlistViewEmpty(t *testing.T) {
	m := New(testConfig(), testUIConfig(), testStore())
	m.width = 80
	m.height = 24
	m.ready = true
	m.currentView = ViewWatchlist
	m.watchlist.State = WatchlistStateLoaded
	m.watchlist.Symbols = []string{}

	view := m.View()
	assert.Contains(t, view, "Watchlist")
	assert.Contains(t, view, "No symbols")
}

func TestWatchlistViewWithSymbols(t *testing.T) {
	m := New(testConfig(), testUIConfig(), testStore())
	m.width = 80
	m.height = 24
	m.ready = true
	m.currentView = ViewWatchlist
	m.watchlist.State = WatchlistStateLoaded
	m.watchlist.Symbols = []string{"AAPL", "GOOGL"}
	m.watchlist.Quotes = map[string]Quote{
		"AAPL": {
			Instrument: QuoteInstrument{Symbol: "AAPL", Type: "EQUITY"},
			Outcome:    "SUCCESS",
			Last:       "150.00",
			Bid:        "149.95",
			Ask:        "150.05",
			Volume:     1000000,
		},
		"GOOGL": {
			Instrument: QuoteInstrument{Symbol: "GOOGL", Type: "EQUITY"},
			Outcome:    "SUCCESS",
			Last:       "140.00",
			Bid:        "139.95",
			Ask:        "140.05",
			Volume:     500000,
		},
	}
	m.watchlist.updateTable()

	view := m.View()
	assert.Contains(t, view, "Watchlist")
	assert.Contains(t, view, "AAPL")
	assert.Contains(t, view, "GOOGL")
}

func TestWatchlistAddMode(t *testing.T) {
	m := New(testConfig(), testUIConfig(), testStore())
	m.width = 80
	m.height = 24
	m.ready = true
	m.currentView = ViewWatchlist
	m.watchlist.Mode = WatchlistModeAdding

	view := m.View()
	assert.Contains(t, view, "Add Symbol")
	assert.Contains(t, view, "Enter to add")
}

func TestWatchlistDeleteMode(t *testing.T) {
	m := New(testConfig(), testUIConfig(), testStore())
	m.width = 80
	m.height = 24
	m.ready = true
	m.currentView = ViewWatchlist
	m.watchlist.Mode = WatchlistModeDeleting
	m.watchlist.DeleteSymbol = "AAPL"

	view := m.View()
	assert.Contains(t, view, "Delete AAPL")
	assert.Contains(t, view, "confirm")
}

func TestWatchlistLoading(t *testing.T) {
	m := New(testConfig(), testUIConfig(), testStore())
	m.width = 80
	m.height = 24
	m.ready = true
	m.currentView = ViewWatchlist
	m.watchlist.State = WatchlistStateLoading

	view := m.View()
	assert.Contains(t, view, "Loading")
}

func TestWatchlistError(t *testing.T) {
	m := New(testConfig(), testUIConfig(), testStore())
	m.width = 80
	m.height = 24
	m.ready = true
	m.currentView = ViewWatchlist
	m.watchlist.State = WatchlistStateError
	m.watchlist.Err = assert.AnError

	view := m.View()
	assert.Contains(t, view, "Error")
	assert.Contains(t, view, "retry")
}

func TestWatchlistFooterKeys(t *testing.T) {
	m := New(testConfig(), testUIConfig(), testStore())
	m.width = 80
	m.height = 24
	m.ready = true
	m.currentView = ViewWatchlist
	m.watchlist.Mode = WatchlistModeNormal
	m.watchlist.State = WatchlistStateLoaded

	view := m.View()
	// Footer should contain watchlist-specific keys
	assert.Contains(t, view, "add")
	assert.Contains(t, view, "delete")
}

func TestNewLoadsWatchlist(t *testing.T) {
	uiCfg := testUIConfigWithWatchlist()
	m := New(testConfig(), uiCfg, testStore())

	assert.Equal(t, []string{"AAPL", "GOOGL"}, m.watchlist.Symbols)
}

func TestUpdateWatchlistTable(t *testing.T) {
	m := New(testConfig(), testUIConfig(), testStore())
	m.watchlist.Symbols = []string{"AAPL", "MSFT"}
	m.watchlist.Quotes = map[string]Quote{
		"AAPL": {
			Instrument: QuoteInstrument{Symbol: "AAPL"},
			Outcome:    "SUCCESS",
			Last:       "150.00",
			Bid:        "149.95",
			Ask:        "150.05",
			Volume:     1000000,
		},
	}
	m.watchlist.updateTable()

	rows := m.watchlist.Table.Rows()
	assert.Len(t, rows, 2)
	// AAPL should have quote data
	assert.Equal(t, "AAPL", rows[0][0])
	assert.Equal(t, "$150.00", rows[0][1])
	// MSFT should have placeholders (no quote)
	assert.Equal(t, "MSFT", rows[1][0])
	assert.Equal(t, "-", rows[1][1])
}

func TestPortfolioModel(t *testing.T) {
	pm := NewPortfolioModel()
	assert.Equal(t, PortfolioStateLoading, pm.State)
	assert.NotNil(t, pm.Table)
}

func TestWatchlistModel(t *testing.T) {
	wm := NewWatchlistModel([]string{"AAPL"})
	assert.Equal(t, WatchlistStateLoading, wm.State)
	assert.Equal(t, []string{"AAPL"}, wm.Symbols)
	assert.NotNil(t, wm.Table)
}

func TestOrdersModel(t *testing.T) {
	om := NewOrdersModel()
	assert.NotNil(t, om)
	assert.Equal(t, OrdersStateLoading, om.State)

	// Initial view shows loading
	view := om.View()
	assert.Contains(t, view, "Loading orders...")

	// After loading with empty orders
	om.State = OrdersStateLoaded
	om.Orders = []Order{}
	view = om.View()
	assert.Contains(t, view, "Open Orders")
	assert.Contains(t, view, "No open orders")
}

func TestTradeModel(t *testing.T) {
	tm := NewTradeModel()
	assert.NotNil(t, tm)
	view := tm.View()
	assert.Contains(t, view, "Place Order")
	assert.Contains(t, view, "Symbol")
	assert.Contains(t, view, "Side")
	assert.Contains(t, view, "BUY")
	assert.Contains(t, view, "SELL")
	assert.Contains(t, view, "Type")
	assert.Contains(t, view, "MARKET")
	assert.Contains(t, view, "LIMIT")
	assert.Contains(t, view, "Quantity")
	assert.Contains(t, view, "Est. Cost")
}

func TestTradeModelWithSymbol(t *testing.T) {
	tm := NewTradeModel()
	tm.SetSymbol("AAPL")
	view := tm.View()
	assert.Contains(t, view, "AAPL")
	// When symbol is set, focus moves to quantity
	assert.Equal(t, TradeFieldQuantity, tm.FocusedField)
}

func TestTradeModelSideToggle(t *testing.T) {
	tm := NewTradeModel()
	assert.Equal(t, TradeSideBuy, tm.Side)
	// Toggle to sell
	tm.Side = TradeSideSell
	assert.Equal(t, "SELL", tm.Side.String())
	// Toggle back to buy
	tm.Side = TradeSideBuy
	assert.Equal(t, "BUY", tm.Side.String())
}

func TestTradeModelOrderType(t *testing.T) {
	tm := NewTradeModel()
	assert.Equal(t, TradeOrderTypeMarket, tm.OrderType)
	// Toggle to limit
	tm.OrderType = TradeOrderTypeLimit
	assert.Equal(t, "LIMIT", tm.OrderType.String())
	view := tm.View()
	// Limit price field should appear for limit orders
	assert.Contains(t, view, "Limit Price")
}

func TestTradeModelFormValidation(t *testing.T) {
	tm := NewTradeModel()
	// Empty form should be invalid
	assert.False(t, tm.isFormValid())
	// Set symbol
	tm.SymbolInput.SetValue("AAPL")
	assert.False(t, tm.isFormValid())
	// Set quantity
	tm.QuantityInput.SetValue("10")
	assert.True(t, tm.isFormValid())
	// Limit order requires price
	tm.OrderType = TradeOrderTypeLimit
	assert.False(t, tm.isFormValid())
	// Set limit price
	tm.LimitPriceInput.SetValue("150.00")
	assert.True(t, tm.isFormValid())
}

func TestTradeModelEstimatedCost(t *testing.T) {
	tm := NewTradeModel()
	// No quote, no quantity - should return "-"
	assert.Equal(t, "-", tm.estimatedCost())
	// Set quantity but no price
	tm.QuantityInput.SetValue("10")
	assert.Equal(t, "-", tm.estimatedCost())
	// For limit orders, use limit price
	tm.OrderType = TradeOrderTypeLimit
	tm.LimitPriceInput.SetValue("150.00")
	assert.Equal(t, "$1500.00", tm.estimatedCost())
}

func TestTradeModelIsTextFieldFocused(t *testing.T) {
	tm := NewTradeModel()
	// Symbol field is focused by default
	assert.True(t, tm.IsTextFieldFocused())
	// Side is not a text field
	tm.FocusedField = TradeFieldSide
	assert.False(t, tm.IsTextFieldFocused())
	// OrderType is not a text field
	tm.FocusedField = TradeFieldOrderType
	assert.False(t, tm.IsTextFieldFocused())
	// Quantity is a text field
	tm.FocusedField = TradeFieldQuantity
	assert.True(t, tm.IsTextFieldFocused())
	// LimitPrice is a text field
	tm.FocusedField = TradeFieldLimitPrice
	assert.True(t, tm.IsTextFieldFocused())
}

func TestOptionsModel(t *testing.T) {
	om := NewOptionsModel()
	assert.NotNil(t, om)
	assert.Equal(t, OptionsStateIdle, om.State)
	assert.Equal(t, OptionsFocusSymbol, om.Focus)
}

func TestOptionsModelInitialView(t *testing.T) {
	om := NewOptionsModel()
	view := om.View()
	assert.Contains(t, view, "Options Chain")
	assert.Contains(t, view, "Underlying")
	assert.Contains(t, view, "Select a symbol")
}

func TestOptionsModelExpirationSelection(t *testing.T) {
	om := NewOptionsModel()
	om.State = OptionsStateSelectingExpiration
	om.Symbol = "AAPL"
	om.Expirations = []string{"2026-01-17", "2026-01-24", "2026-01-31"}
	om.SelectedExpiration = 0

	view := om.View()
	assert.Contains(t, view, "Options Chain")
	assert.Contains(t, view, "AAPL")
	assert.Contains(t, view, "Select Expiration")
	assert.Contains(t, view, "2026-01-17")
}

func TestOptionsModelLoadingChain(t *testing.T) {
	om := NewOptionsModel()
	om.State = OptionsStateLoadingChain
	om.Symbol = "AAPL"

	view := om.View()
	assert.Contains(t, view, "Loading option chain")
}

func TestOptionsModelError(t *testing.T) {
	om := NewOptionsModel()
	om.State = OptionsStateError
	om.Err = assert.AnError

	view := om.View()
	assert.Contains(t, view, "Error")
}

func TestOptionsModelFooterKeys(t *testing.T) {
	om := NewOptionsModel()
	keys := om.FooterKeys([]struct{ key, desc string }{})

	// Idle state should have Enter, w, esc keys
	assert.True(t, len(keys) >= 3)

	// Check for expected keys
	hasEnter := false
	hasW := false
	for _, k := range keys {
		if k.key == "Enter" {
			hasEnter = true
		}
		if k.key == "w" {
			hasW = true
		}
	}
	assert.True(t, hasEnter, "should have Enter key")
	assert.True(t, hasW, "should have w key for watchlist")
}

func TestOptionsModelSetHeight(t *testing.T) {
	om := NewOptionsModel()
	om.SetHeight(20)
	assert.Equal(t, 20, om.Height)
}

func TestOptionsModelHelpers(t *testing.T) {
	// Test calculateDTE
	dte := calculateDTE("2026-01-17")
	assert.True(t, dte >= 0)

	// Test formatOptPrice
	assert.Equal(t, "1.50", formatOptPrice("1.5"))
	assert.Equal(t, "-", formatOptPrice(""))
	assert.Equal(t, "-", formatOptPrice("0"))

	// Test formatGreek
	assert.Equal(t, "0.50", formatGreek("0.5"))
	assert.Equal(t, "-", formatGreek(""))

	// Test formatIV
	assert.Equal(t, "50.0%", formatIV("0.5"))
	assert.Equal(t, "-", formatIV(""))

	// Test parseStrikeFromOSI
	strike := parseStrikeFromOSI("AAPL260117C00185000")
	assert.Equal(t, float64(185), strike)
}

func TestOptionsViewSwitch(t *testing.T) {
	m := New(testConfig(), testUIConfig(), testStore())
	m.width = 80
	m.height = 24
	m.ready = true
	m.currentView = ViewOptions

	view := m.View()
	assert.Contains(t, view, "Options")
}
