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
