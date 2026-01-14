package tui

import "github.com/jonandersen/public-cli/internal/api"

// Type aliases for API models - allows TUI code to use short names
// while the actual types are defined in internal/api/models.go

// Portfolio types
type (
	Portfolio   = api.Portfolio
	BuyingPower = api.BuyingPower
	Equity      = api.Equity
	Position    = api.Position
	Instrument  = api.Instrument
	Price       = api.Price
	Gain        = api.Gain
	CostBasis   = api.CostBasis
)

// Quote types
type (
	QuoteRequest    = api.QuoteRequest
	QuoteInstrument = api.QuoteInstrument
	QuotesResponse  = api.QuotesResponse
	Quote           = api.Quote
)

// Order types
type (
	Order          = api.Order
	OrdersResponse = api.OrdersResponse
)

// Account types
type (
	Account          = api.Account
	AccountsResponse = api.AccountsResponse
)

// History types
type (
	Transaction     = api.Transaction
	HistoryResponse = api.HistoryResponse
)
