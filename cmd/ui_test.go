package cmd

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"

	"github.com/jonandersen/pub/internal/config"
	"github.com/jonandersen/pub/internal/keyring"
)

func TestUICommandExists(t *testing.T) {
	// Verify the ui command is registered
	cmd := rootCmd.Commands()
	var found bool
	for _, c := range cmd {
		if c.Name() == "ui" {
			found = true
			break
		}
	}
	assert.True(t, found, "ui command should be registered")
}

func TestUICommandDescription(t *testing.T) {
	// Find the ui command
	var uiCmd *cobra.Command
	for _, c := range rootCmd.Commands() {
		if c.Name() == "ui" {
			uiCmd = c
			break
		}
	}
	assert.NotNil(t, uiCmd)
	assert.Equal(t, "ui", uiCmd.Use)
	assert.Contains(t, uiCmd.Short, "Interactive")
}

func testConfig() *config.Config {
	return &config.Config{
		APIBaseURL:  "https://api.public.com",
		AccountUUID: "test-account-123",
	}
}

func testUIConfig() *UIConfig {
	return &UIConfig{}
}

func testStore() keyring.Store {
	store := keyring.NewMockStore()
	_ = store.Set(keyring.ServiceName, keyring.KeySecretKey, "test-secret")
	return store
}

func TestNewModel(t *testing.T) {
	m := newModel(testConfig(), testUIConfig(), testStore())
	assert.NotNil(t, m)
	assert.Equal(t, viewPortfolio, m.currentView)
	assert.Equal(t, portfolioStateLoading, m.portfolio.state)
}

func TestModelInit(t *testing.T) {
	m := newModel(testConfig(), testUIConfig(), testStore())
	cmd := m.Init()
	// Init should return a batch command (fetch + tick)
	assert.NotNil(t, cmd)
}

func TestModelView(t *testing.T) {
	m := newModel(testConfig(), testUIConfig(), testStore())
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
	m := newModel(testConfig(), testUIConfig(), testStore())
	m.width = 80
	m.height = 24
	m.ready = true
	m.portfolio.state = portfolioStateLoading

	view := m.View()
	assert.Contains(t, view, "Loading")
}

func TestModelViewError(t *testing.T) {
	m := newModel(testConfig(), testUIConfig(), testStore())
	m.width = 80
	m.height = 24
	m.ready = true
	m.portfolio.state = portfolioStateError
	m.portfolio.err = assert.AnError

	view := m.View()
	assert.Contains(t, view, "Error")
	assert.Contains(t, view, "retry")
}

func TestModelViewWithPositions(t *testing.T) {
	m := newModel(testConfig(), testUIConfig(), testStore())
	m.width = 120
	m.height = 30
	m.ready = true
	m.portfolio.state = portfolioStateLoaded
	m.portfolio.portfolio = Portfolio{
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
	m.updatePortfolioTable()

	view := m.View()
	assert.Contains(t, view, "Account Summary")
	assert.Contains(t, view, "AAPL")
	assert.Contains(t, view, "Positions")
}

func TestWatchlistViewEmpty(t *testing.T) {
	m := newModel(testConfig(), testUIConfig(), testStore())
	m.width = 80
	m.height = 24
	m.ready = true
	m.currentView = viewWatchlist
	m.watchlist.state = watchlistStateLoaded
	m.watchlist.symbols = []string{}

	view := m.View()
	assert.Contains(t, view, "Watchlist")
	assert.Contains(t, view, "No symbols")
}

func TestWatchlistViewWithSymbols(t *testing.T) {
	m := newModel(testConfig(), testUIConfig(), testStore())
	m.width = 80
	m.height = 24
	m.ready = true
	m.currentView = viewWatchlist
	m.watchlist.state = watchlistStateLoaded
	m.watchlist.symbols = []string{"AAPL", "GOOGL"}
	m.watchlist.quotes = map[string]Quote{
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
	m.updateWatchlistTable()

	view := m.View()
	assert.Contains(t, view, "Watchlist")
	assert.Contains(t, view, "AAPL")
	assert.Contains(t, view, "GOOGL")
}

func TestWatchlistAddMode(t *testing.T) {
	m := newModel(testConfig(), testUIConfig(), testStore())
	m.width = 80
	m.height = 24
	m.ready = true
	m.currentView = viewWatchlist
	m.watchlistMode = watchlistModeAdding

	view := m.View()
	assert.Contains(t, view, "Add Symbol")
	assert.Contains(t, view, "Enter to add")
}

func TestWatchlistDeleteMode(t *testing.T) {
	m := newModel(testConfig(), testUIConfig(), testStore())
	m.width = 80
	m.height = 24
	m.ready = true
	m.currentView = viewWatchlist
	m.watchlistMode = watchlistModeDeleting
	m.deleteSymbol = "AAPL"

	view := m.View()
	assert.Contains(t, view, "Delete AAPL")
	assert.Contains(t, view, "confirm")
}

func TestWatchlistLoading(t *testing.T) {
	m := newModel(testConfig(), testUIConfig(), testStore())
	m.width = 80
	m.height = 24
	m.ready = true
	m.currentView = viewWatchlist
	m.watchlist.state = watchlistStateLoading

	view := m.View()
	assert.Contains(t, view, "Loading")
}

func TestWatchlistError(t *testing.T) {
	m := newModel(testConfig(), testUIConfig(), testStore())
	m.width = 80
	m.height = 24
	m.ready = true
	m.currentView = viewWatchlist
	m.watchlist.state = watchlistStateError
	m.watchlist.err = assert.AnError

	view := m.View()
	assert.Contains(t, view, "Error")
	assert.Contains(t, view, "retry")
}

func TestWatchlistFooterKeys(t *testing.T) {
	m := newModel(testConfig(), testUIConfig(), testStore())
	m.width = 80
	m.height = 24
	m.ready = true
	m.currentView = viewWatchlist
	m.watchlistMode = watchlistModeNormal
	m.watchlist.state = watchlistStateLoaded

	view := m.View()
	// Footer should contain watchlist-specific keys
	assert.Contains(t, view, "add")
	assert.Contains(t, view, "delete")
}

func testUIConfigWithWatchlist() *UIConfig {
	return &UIConfig{
		Watchlist: []string{"AAPL", "GOOGL"},
	}
}

func TestNewModelLoadsWatchlist(t *testing.T) {
	uiCfg := testUIConfigWithWatchlist()
	m := newModel(testConfig(), uiCfg, testStore())

	assert.Equal(t, []string{"AAPL", "GOOGL"}, m.watchlist.symbols)
}

func TestUpdateWatchlistTable(t *testing.T) {
	m := newModel(testConfig(), testUIConfig(), testStore())
	m.watchlist.symbols = []string{"AAPL", "MSFT"}
	m.watchlist.quotes = map[string]Quote{
		"AAPL": {
			Instrument: QuoteInstrument{Symbol: "AAPL"},
			Outcome:    "SUCCESS",
			Last:       "150.00",
			Bid:        "149.95",
			Ask:        "150.05",
			Volume:     1000000,
		},
	}
	m.updateWatchlistTable()

	rows := m.watchlistTable.Rows()
	assert.Len(t, rows, 2)
	// AAPL should have quote data
	assert.Equal(t, "AAPL", rows[0][0])
	assert.Equal(t, "$150.00", rows[0][1])
	// MSFT should have placeholders (no quote)
	assert.Equal(t, "MSFT", rows[1][0])
	assert.Equal(t, "-", rows[1][1])
}
