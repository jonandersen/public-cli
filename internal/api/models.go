package api

import "github.com/jonandersen/public-cli/pkg/publicapi"

// =============================================================================
// Account Types (aliased from pkg/publicapi)
// =============================================================================

type (
	Account          = publicapi.Account
	AccountsResponse = publicapi.AccountsResponse
)

// =============================================================================
// Portfolio Types (aliased from pkg/publicapi)
// =============================================================================

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

// =============================================================================
// Quote Types (aliased from pkg/publicapi)
// =============================================================================

type (
	QuoteRequest    = publicapi.QuoteRequest
	QuoteInstrument = publicapi.QuoteInstrument
	QuotesResponse  = publicapi.QuotesResponse
	Quote           = publicapi.Quote
)

// =============================================================================
// Order Types (aliased from pkg/publicapi)
// =============================================================================

type (
	Order               = publicapi.Order
	OrdersResponse      = publicapi.OrdersResponse
	OrderRequest        = publicapi.OrderRequest
	OrderInstrument     = publicapi.OrderInstrument
	OrderExpiration     = publicapi.OrderExpiration
	OrderResponse       = publicapi.OrderResponse
	OrderStatusResponse = publicapi.OrderStatusResponse
	OrderListResponse   = publicapi.OrderListResponse
	PreflightRequest    = publicapi.PreflightRequest
	RegulatoryFees      = publicapi.RegulatoryFees
	PreflightResponse   = publicapi.PreflightResponse
)

// =============================================================================
// Single-Leg Options Order Types (aliased from pkg/publicapi)
// =============================================================================

type (
	OptionsOrderRequest      = publicapi.OptionsOrderRequest
	OptionsPreflightRequest  = publicapi.OptionsPreflightRequest
	OptionsPreflightResponse = publicapi.OptionsPreflightResponse
	OptionsRegulatoryFees    = publicapi.OptionsRegulatoryFees
)

// =============================================================================
// Multi-Leg Options Order Types (aliased from pkg/publicapi)
// =============================================================================

type (
	MultilegPreflightRequest  = publicapi.MultilegPreflightRequest
	MultilegExpiration        = publicapi.MultilegExpiration
	MultilegLeg               = publicapi.MultilegLeg
	MultilegInstrument        = publicapi.MultilegInstrument
	MultilegPreflightResponse = publicapi.MultilegPreflightResponse
	MultilegPreflightLeg      = publicapi.MultilegPreflightLeg
	MultilegRegulatoryFees    = publicapi.MultilegRegulatoryFees
	MultilegPriceIncrement    = publicapi.MultilegPriceIncrement
	MultilegOrderRequest      = publicapi.MultilegOrderRequest
	MultilegOrderResponse     = publicapi.MultilegOrderResponse
)

// =============================================================================
// Instrument Types (aliased from pkg/publicapi)
// =============================================================================

type InstrumentsResponse = publicapi.InstrumentsResponse

// =============================================================================
// History Types (aliased from pkg/publicapi)
// =============================================================================

type (
	Transaction     = publicapi.Transaction
	HistoryResponse = publicapi.HistoryResponse
)
