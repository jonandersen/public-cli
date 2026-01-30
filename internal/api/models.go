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
// Single-Leg Options Order Types
// =============================================================================

// OptionsOrderRequest represents a single-leg options order request.
type OptionsOrderRequest struct {
	OrderID            string          `json:"orderId"`
	Instrument         OrderInstrument `json:"instrument"`
	OrderSide          string          `json:"orderSide"`
	OrderType          string          `json:"orderType"`
	Expiration         OrderExpiration `json:"expiration"`
	Quantity           string          `json:"quantity"`
	LimitPrice         string          `json:"limitPrice,omitempty"`
	OpenCloseIndicator string          `json:"openCloseIndicator"`
}

// OptionsPreflightRequest represents a single-leg options preflight request.
type OptionsPreflightRequest struct {
	Instrument         OrderInstrument `json:"instrument"`
	OrderSide          string          `json:"orderSide"`
	OrderType          string          `json:"orderType"`
	Expiration         OrderExpiration `json:"expiration"`
	Quantity           string          `json:"quantity"`
	LimitPrice         string          `json:"limitPrice,omitempty"`
	OpenCloseIndicator string          `json:"openCloseIndicator"`
}

// OptionsPreflightResponse represents the API response for single-leg options preflight.
type OptionsPreflightResponse struct {
	Instrument             OrderInstrument       `json:"instrument"`
	EstimatedCommission    string                `json:"estimatedCommission"`
	RegulatoryFees         OptionsRegulatoryFees `json:"regulatoryFees"`
	EstimatedCost          string                `json:"estimatedCost"`
	BuyingPowerRequirement string                `json:"buyingPowerRequirement"`
	OrderValue             string                `json:"orderValue"`
	EstimatedQuantity      string                `json:"estimatedQuantity"`
	EstimatedProceeds      string                `json:"estimatedProceeds,omitempty"`
}

// OptionsRegulatoryFees represents the breakdown of regulatory fees for options.
type OptionsRegulatoryFees struct {
	SECFee      string `json:"secFee"`
	TAFFee      string `json:"tafFee"`
	ORFFee      string `json:"orfFee"`
	ExchangeFee string `json:"exchangeFee"`
	OCCFee      string `json:"occFee"`
	CATFee      string `json:"catFee"`
}

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
