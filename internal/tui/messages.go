package tui

import "time"

// Message types for async operations

// PortfolioLoadedMsg is sent when portfolio data is loaded successfully.
type PortfolioLoadedMsg struct {
	Portfolio Portfolio
}

// PortfolioErrorMsg is sent when portfolio loading fails.
type PortfolioErrorMsg struct {
	Err error
}

// WatchlistQuotesMsg is sent when watchlist quotes are loaded.
type WatchlistQuotesMsg struct {
	Quotes map[string]Quote
}

// WatchlistErrorMsg is sent when watchlist loading fails.
type WatchlistErrorMsg struct {
	Err error
}

// WatchlistSavedMsg is sent when watchlist config is saved.
type WatchlistSavedMsg struct{}

// TickMsg is sent periodically for auto-refresh.
type TickMsg time.Time
