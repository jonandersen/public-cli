package publicapi

// =============================================================================
// Account Types
// =============================================================================

// Account represents a Public.com account.
type Account struct {
	AccountID            string `json:"accountId"`
	AccountType          string `json:"accountType"`
	OptionsLevel         string `json:"optionsLevel"`
	BrokerageAccountType string `json:"brokerageAccountType"`
	TradePermissions     string `json:"tradePermissions"`
}

// AccountsResponse represents the API response for listing accounts.
type AccountsResponse struct {
	Accounts []Account `json:"accounts"`
}

// =============================================================================
// Portfolio Types
// =============================================================================

// Portfolio represents a portfolio response from the API.
type Portfolio struct {
	AccountID   string      `json:"accountId"`
	AccountType string      `json:"accountType"`
	BuyingPower BuyingPower `json:"buyingPower"`
	Equity      []Equity    `json:"equity"`
	Positions   []Position  `json:"positions"`
}

// BuyingPower represents buying power information.
type BuyingPower struct {
	CashOnlyBuyingPower string `json:"cashOnlyBuyingPower"`
	BuyingPower         string `json:"buyingPower"`
	OptionsBuyingPower  string `json:"optionsBuyingPower"`
}

// Equity represents an equity breakdown item.
type Equity struct {
	Type                  string `json:"type"`
	Value                 string `json:"value"`
	PercentageOfPortfolio string `json:"percentageOfPortfolio"`
}

// Position represents a portfolio position.
type Position struct {
	Instrument         Instrument `json:"instrument"`
	Quantity           string     `json:"quantity"`
	CurrentValue       string     `json:"currentValue"`
	PercentOfPortfolio string     `json:"percentOfPortfolio"`
	LastPrice          Price      `json:"lastPrice"`
	InstrumentGain     Gain       `json:"instrumentGain"`
	PositionDailyGain  Gain       `json:"positionDailyGain"`
	CostBasis          CostBasis  `json:"costBasis"`
}

// Instrument represents a trading instrument.
type Instrument struct {
	Symbol string `json:"symbol"`
	Name   string `json:"name"`
	Type   string `json:"type"`
}

// Price represents a price with timestamp.
type Price struct {
	LastPrice string `json:"lastPrice"`
	Timestamp string `json:"timestamp"`
}

// Gain represents a gain/loss value with percentage.
type Gain struct {
	GainValue      string `json:"gainValue"`
	GainPercentage string `json:"gainPercentage"`
	Timestamp      string `json:"timestamp"`
}

// CostBasis represents cost basis information.
type CostBasis struct {
	TotalCost      string `json:"totalCost"`
	UnitCost       string `json:"unitCost"`
	GainValue      string `json:"gainValue"`
	GainPercentage string `json:"gainPercentage"`
	LastUpdate     string `json:"lastUpdate"`
}

// =============================================================================
// Quote Types
// =============================================================================

// QuoteRequest represents a request for quotes.
type QuoteRequest struct {
	Instruments []QuoteInstrument `json:"instruments"`
}

// QuoteInstrument represents an instrument to quote.
type QuoteInstrument struct {
	Symbol string `json:"symbol"`
	Type   string `json:"type"`
}

// QuotesResponse represents the API response for quotes.
type QuotesResponse struct {
	Quotes []Quote `json:"quotes"`
}

// Quote represents a single quote.
type Quote struct {
	Instrument    QuoteInstrument `json:"instrument"`
	Outcome       string          `json:"outcome"`
	Last          string          `json:"last"`
	LastTimestamp string          `json:"lastTimestamp"`
	Bid           string          `json:"bid"`
	BidSize       int             `json:"bidSize"`
	BidTimestamp  string          `json:"bidTimestamp"`
	Ask           string          `json:"ask"`
	AskSize       int             `json:"askSize"`
	AskTimestamp  string          `json:"askTimestamp"`
	Volume        int64           `json:"volume"`
	OpenInterest  *int64          `json:"openInterest"`
}

// =============================================================================
// Order Types
// =============================================================================

// Order represents an open order from the API.
type Order struct {
	OrderID        string     `json:"orderId"`
	Instrument     Instrument `json:"instrument"`
	Side           string     `json:"side"`
	Type           string     `json:"type"`
	Status         string     `json:"status"`
	Quantity       string     `json:"quantity"`
	FilledQuantity string     `json:"filledQuantity"`
	LimitPrice     string     `json:"limitPrice,omitempty"`
	StopPrice      string     `json:"stopPrice,omitempty"`
	CreatedAt      string     `json:"createdAt"`
}

// OrdersResponse represents the portfolio API response containing orders.
type OrdersResponse struct {
	Orders []Order `json:"orders"`
}

// OrderRequest represents an order placement request.
type OrderRequest struct {
	OrderID    string          `json:"orderId"`
	Instrument OrderInstrument `json:"instrument"`
	OrderSide  string          `json:"orderSide"`
	OrderType  string          `json:"orderType"`
	Expiration OrderExpiration `json:"expiration"`
	Quantity   string          `json:"quantity,omitempty"`
	Amount     string          `json:"amount,omitempty"`
	LimitPrice string          `json:"limitPrice,omitempty"`
	StopPrice  string          `json:"stopPrice,omitempty"`
}

// OrderInstrument represents the instrument being traded in an order.
type OrderInstrument struct {
	Symbol string `json:"symbol"`
	Type   string `json:"type"`
}

// OrderExpiration represents order time-in-force.
type OrderExpiration struct {
	TimeInForce string `json:"timeInForce"`
}

// OrderResponse represents the API response for order placement.
type OrderResponse struct {
	OrderID string `json:"orderId"`
}

// OrderStatusResponse represents the API response for order status.
type OrderStatusResponse struct {
	OrderID        string          `json:"orderId"`
	Instrument     OrderInstrument `json:"instrument"`
	CreatedAt      string          `json:"createdAt"`
	Type           string          `json:"type"`
	Side           string          `json:"side"`
	Status         string          `json:"status"`
	Quantity       string          `json:"quantity"`
	LimitPrice     string          `json:"limitPrice,omitempty"`
	StopPrice      string          `json:"stopPrice,omitempty"`
	FilledQuantity string          `json:"filledQuantity"`
	AveragePrice   string          `json:"averagePrice,omitempty"`
	ClosedAt       string          `json:"closedAt,omitempty"`
}

// OrderListResponse represents the portfolio API response containing orders.
type OrderListResponse struct {
	AccountID string  `json:"accountId"`
	Orders    []Order `json:"orders"`
}

// PreflightRequest represents a preflight request to estimate order costs.
type PreflightRequest struct {
	Instrument OrderInstrument `json:"instrument"`
	OrderSide  string          `json:"orderSide"`
	OrderType  string          `json:"orderType"`
	Expiration OrderExpiration `json:"expiration"`
	Quantity   string          `json:"quantity,omitempty"`
	LimitPrice string          `json:"limitPrice,omitempty"`
	StopPrice  string          `json:"stopPrice,omitempty"`
}

// RegulatoryFees represents the breakdown of regulatory fees.
type RegulatoryFees struct {
	SECFee string `json:"secFee"`
	TAFFee string `json:"tafFee"`
	ORFFee string `json:"orfFee"`
}

// PreflightResponse represents the API response for preflight estimation.
type PreflightResponse struct {
	Instrument             OrderInstrument `json:"instrument"`
	EstimatedCommission    string          `json:"estimatedCommission"`
	RegulatoryFees         RegulatoryFees  `json:"regulatoryFees"`
	EstimatedCost          string          `json:"estimatedCost"`
	BuyingPowerRequirement string          `json:"buyingPowerRequirement"`
	OrderValue             string          `json:"orderValue"`
	EstimatedQuantity      string          `json:"estimatedQuantity"`
}

// =============================================================================
// Options Types
// =============================================================================

// OptionInstrument represents an instrument for options requests.
type OptionInstrument struct {
	Symbol string `json:"symbol"`
	Type   string `json:"type"`
}

// OptionExpirationsRequest represents a request for option expirations.
type OptionExpirationsRequest struct {
	Instrument OptionInstrument `json:"instrument"`
}

// OptionExpirationsResponse represents the API response for option expirations.
type OptionExpirationsResponse struct {
	BaseSymbol  string   `json:"baseSymbol"`
	Expirations []string `json:"expirations"`
}

// OptionChainRequest represents a request for an option chain.
type OptionChainRequest struct {
	Instrument     OptionInstrument `json:"instrument"`
	ExpirationDate string           `json:"expirationDate"`
}

// OptionChainResponse represents the API response for an option chain.
type OptionChainResponse struct {
	BaseSymbol string        `json:"baseSymbol"`
	Calls      []OptionQuote `json:"calls"`
	Puts       []OptionQuote `json:"puts"`
}

// OptionQuote represents a single option quote in the chain.
type OptionQuote struct {
	Instrument   OptionInstrument `json:"instrument"`
	Outcome      string           `json:"outcome"`
	Last         string           `json:"last"`
	Bid          string           `json:"bid"`
	BidSize      int              `json:"bidSize"`
	Ask          string           `json:"ask"`
	AskSize      int              `json:"askSize"`
	Volume       int              `json:"volume"`
	OpenInterest int              `json:"openInterest"`
}

// GreeksResponse represents the API response for option greeks.
type GreeksResponse struct {
	Greeks []OptionGreeks `json:"greeks"`
}

// OptionGreeks represents greeks for a single option.
type OptionGreeks struct {
	Symbol string     `json:"symbol"`
	Greeks GreeksData `json:"greeks"`
}

// GreeksData contains the actual greek values.
type GreeksData struct {
	Delta             string `json:"delta"`
	Gamma             string `json:"gamma"`
	Theta             string `json:"theta"`
	Vega              string `json:"vega"`
	Rho               string `json:"rho"`
	ImpliedVolatility string `json:"impliedVolatility"`
}

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
// Multi-Leg Options Order Types
// =============================================================================

// MultilegPreflightRequest represents a multi-leg preflight request.
type MultilegPreflightRequest struct {
	OrderType  string             `json:"orderType"`
	Expiration MultilegExpiration `json:"expiration"`
	Quantity   string             `json:"quantity"`
	LimitPrice string             `json:"limitPrice"`
	Legs       []MultilegLeg      `json:"legs"`
}

// MultilegExpiration represents time-in-force for multi-leg orders.
type MultilegExpiration struct {
	TimeInForce string `json:"timeInForce"`
}

// MultilegLeg represents a single leg in a multi-leg order.
type MultilegLeg struct {
	Instrument         MultilegInstrument `json:"instrument"`
	Side               string             `json:"side"`
	OpenCloseIndicator string             `json:"openCloseIndicator"`
	RatioQuantity      int                `json:"ratioQuantity"`
}

// MultilegInstrument represents an instrument in a multi-leg order.
type MultilegInstrument struct {
	Symbol string `json:"symbol"`
	Type   string `json:"type"`
}

// MultilegPreflightResponse represents the API response for multi-leg preflight.
type MultilegPreflightResponse struct {
	BaseSymbol              string                 `json:"baseSymbol"`
	StrategyName            string                 `json:"strategyName"`
	Legs                    []MultilegPreflightLeg `json:"legs"`
	EstimatedCommission     string                 `json:"estimatedCommission"`
	RegulatoryFees          MultilegRegulatoryFees `json:"regulatoryFees"`
	EstimatedIndexOptionFee string                 `json:"estimatedIndexOptionFee"`
	OrderValue              string                 `json:"orderValue"`
	EstimatedQuantity       string                 `json:"estimatedQuantity"`
	EstimatedCost           string                 `json:"estimatedCost"`
	BuyingPowerRequirement  string                 `json:"buyingPowerRequirement"`
	EstimatedProceeds       string                 `json:"estimatedProceeds"`
	PriceIncrement          MultilegPriceIncrement `json:"priceIncrement"`
}

// MultilegPreflightLeg represents a leg in the preflight response.
type MultilegPreflightLeg struct {
	Instrument         MultilegInstrument `json:"instrument"`
	Side               string             `json:"side"`
	OpenCloseIndicator string             `json:"openCloseIndicator"`
	RatioQuantity      int                `json:"ratioQuantity"`
}

// MultilegRegulatoryFees represents regulatory fees for multi-leg orders.
type MultilegRegulatoryFees struct {
	SECFee      string `json:"secFee"`
	TAFFee      string `json:"tafFee"`
	ORFFee      string `json:"orfFee"`
	ExchangeFee string `json:"exchangeFee"`
	OCCFee      string `json:"occFee"`
	CATFee      string `json:"catFee"`
}

// MultilegPriceIncrement represents price increment information.
type MultilegPriceIncrement struct {
	IncrementBelow3  string `json:"incrementBelow3"`
	IncrementAbove3  string `json:"incrementAbove3"`
	CurrentIncrement string `json:"currentIncrement"`
}

// MultilegOrderRequest represents a multi-leg order request.
type MultilegOrderRequest struct {
	OrderID    string             `json:"orderId"`
	OrderType  string             `json:"orderType"`
	Expiration MultilegExpiration `json:"expiration"`
	Quantity   string             `json:"quantity"`
	LimitPrice string             `json:"limitPrice"`
	Legs       []MultilegLeg      `json:"legs"`
}

// MultilegOrderResponse represents the API response for a multi-leg order.
type MultilegOrderResponse struct {
	OrderID string `json:"orderId"`
}

// =============================================================================
// Instrument Types
// =============================================================================

// InstrumentIdentifier represents an instrument identifier in API responses.
type InstrumentIdentifier struct {
	Symbol string `json:"symbol"`
	Type   string `json:"type"`
}

// InstrumentResponse represents the API response for instrument details.
type InstrumentResponse struct {
	Instrument          InstrumentIdentifier `json:"instrument"`
	Trading             string               `json:"trading"`
	FractionalTrading   string               `json:"fractionalTrading"`
	OptionTrading       string               `json:"optionTrading"`
	OptionSpreadTrading string               `json:"optionSpreadTrading"`
	InstrumentDetails   any                  `json:"instrumentDetails,omitempty"`
}
