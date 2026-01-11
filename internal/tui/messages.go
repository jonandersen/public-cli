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

// OrdersLoadedMsg is sent when orders are loaded successfully.
type OrdersLoadedMsg struct {
	Orders []Order
}

// OrdersErrorMsg is sent when orders loading fails.
type OrdersErrorMsg struct {
	Err error
}

// OrderCancelledMsg is sent when an order is cancelled.
type OrderCancelledMsg struct {
	OrderID string
}

// OrderCancelErrorMsg is sent when order cancellation fails.
type OrderCancelErrorMsg struct {
	Err error
}

// TickMsg is sent periodically for auto-refresh.
type TickMsg time.Time
