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
// History Types
// =============================================================================

// Transaction represents a single transaction in account history.
type Transaction struct {
	ID              string `json:"id"`
	Timestamp       string `json:"timestamp"`
	Type            string `json:"type"`
	SubType         string `json:"subType"`
	AccountNumber   string `json:"accountNumber"`
	Symbol          string `json:"symbol"`
	SecurityType    string `json:"securityType"`
	Side            string `json:"side"`
	Description     string `json:"description"`
	NetAmount       string `json:"netAmount"`
	PrincipalAmount string `json:"principalAmount"`
	Quantity        string `json:"quantity"`
	Direction       string `json:"direction"`
	Fees            string `json:"fees"`
}

// HistoryResponse represents the API response for account history.
type HistoryResponse struct {
	Transactions []Transaction `json:"transactions"`
	NextToken    string        `json:"nextToken"`
	Start        string        `json:"start"`
	End          string        `json:"end"`
	PageSize     int           `json:"pageSize"`
}
