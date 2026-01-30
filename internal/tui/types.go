package tui

import "github.com/jonandersen/public-cli/pkg/publicapi"

// Type aliases for API models - allows TUI code to use short names
// while the actual types are defined in pkg/publicapi/types.go

// Portfolio types
type (
	Portfolio   = publicapi.Portfolio
	BuyingPower = publicapi.BuyingPower
	Equity      = publicapi.Equity
	Position    = publicapi.Position
	Instrument  = publicapi.Instrument
	Price       = publicapi.Price
	Gain        = publicapi.Gain
	CostBasis   = publicapi.CostBasis
)

// Quote types
type (
	QuoteRequest    = publicapi.QuoteRequest
	QuoteInstrument = publicapi.QuoteInstrument
	QuotesResponse  = publicapi.QuotesResponse
	Quote           = publicapi.Quote
)

// Order types
type (
	Order          = publicapi.Order
	OrdersResponse = publicapi.OrdersResponse
)

// Account types
type (
	Account          = publicapi.Account
	AccountsResponse = publicapi.AccountsResponse
)

// History types
type (
	Transaction     = publicapi.Transaction
	HistoryResponse = publicapi.HistoryResponse
)
