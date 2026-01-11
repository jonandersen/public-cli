package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/jonandersen/pub/internal/config"
	"github.com/jonandersen/pub/internal/keyring"
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
	view := om.View()
	assert.Contains(t, view, "Orders")
	assert.Contains(t, view, "Coming soon")
}

func TestTradeModel(t *testing.T) {
	tm := NewTradeModel()
	assert.NotNil(t, tm)
	view := tm.View()
	assert.Contains(t, view, "Trade")
	assert.Contains(t, view, "Coming soon")
}

func TestTradeModelWithSymbol(t *testing.T) {
	tm := NewTradeModel()
	tm.SetSymbol("AAPL")
	view := tm.View()
	assert.Contains(t, view, "AAPL")
}
