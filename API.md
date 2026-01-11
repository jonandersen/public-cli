# Public.com API Documentation

This document provides comprehensive documentation for the Public.com Trading API.

**Base URL:** `https://api.public.com`

## Table of Contents

- [Authentication](#authentication)
- [Accounts](#accounts)
- [Portfolio & History](#portfolio--history)
- [Instruments](#instruments)
- [Market Data](#market-data)
- [Order Placement](#order-placement)
- [Options](#options)
- [Error Handling](#error-handling)
- [Changelog](#changelog)

---

## Authentication

The API uses a two-step authentication flow:

1. Generate a secret key from [public.com/settings/security/api](https://public.com/settings/security/api)
2. Exchange the secret key for a short-lived access token
3. Use the access token in the `Authorization: Bearer` header for all requests

> **Security:** Keep your secret keys secure and never expose them in client-side code.

### Create Personal Access Token

Generates a new personal access token (JWT) with a specified validity period.

**Endpoint:** `POST /userapiauthservice/personal/access-tokens`

**Request Body:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `secret` | string | Yes | Your personal secret token |
| `validityInMinutes` | integer | No | Token expiration (5-1440 minutes, default: 15) |

**Example Request:**

```bash
curl --request POST \
  --url https://api.public.com/userapiauthservice/personal/access-tokens \
  --header 'Content-Type: application/json' \
  --data '{
    "validityInMinutes": 60,
    "secret": "YOUR_SECRET_KEY"
  }'
```

**Response (200):**

```json
{
  "accessToken": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**Error Responses:**

| Status | Description |
|--------|-------------|
| 401 | Invalid or expired secret provided |
| 429 | Rate limit exceeded; retry after waiting |

---

## Accounts

### Get Accounts

Retrieves the list of financial accounts associated with the authenticated user.

**Endpoint:** `GET /userapigateway/trading/account`

**Headers:**
- `Authorization: Bearer YOUR_ACCESS_TOKEN`
- `Content-Type: application/json`

**Example Request:**

```bash
curl --request GET \
  --url https://api.public.com/userapigateway/trading/account \
  --header 'Authorization: Bearer YOUR_ACCESS_TOKEN' \
  --header 'Content-Type: application/json'
```

**Response (200):**

```json
{
  "accounts": [
    {
      "accountId": "abc123-def456-ghi789",
      "accountType": "BROKERAGE",
      "optionsLevel": "LEVEL_2",
      "brokerageAccountType": "CASH",
      "tradePermissions": "BUY_AND_SELL"
    }
  ]
}
```

**Account Fields:**

| Field | Type | Description |
|-------|------|-------------|
| `accountId` | string | Stable, persistent account identifier |
| `accountType` | enum | BROKERAGE, HIGH_YIELD, BOND_ACCOUNT, RIA_ASSET, TREASURY, TRADITIONAL_IRA, ROTH_IRA |
| `optionsLevel` | enum | NONE, LEVEL_1, LEVEL_2, LEVEL_3, LEVEL_4 |
| `brokerageAccountType` | enum | CASH, MARGIN |
| `tradePermissions` | enum | BUY_AND_SELL, RESTRICTED_SETTLED_FUNDS_ONLY, RESTRICTED_CLOSE_ONLY, RESTRICTED_NO_TRADING |

**Error Responses:**

| Status | Description |
|--------|-------------|
| 401 | Unauthorized access |
| 404 | No accounts found for user |

---

## Portfolio & History

### Get Account Portfolio

Retrieves a snapshot of the account's portfolio including positions, equity breakdown, buying power, and open orders.

**Endpoint:** `GET /userapigateway/trading/{accountId}/portfolio/v2`

**Path Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `accountId` | string | Yes | Account identifier |

**Response (200):**

```json
{
  "accountId": "abc123-def456-ghi789",
  "accountType": "BROKERAGE",
  "buyingPower": {
    "cashOnlyBuyingPower": "10000.00",
    "buyingPower": "10000.00",
    "optionsBuyingPower": "5000.00"
  },
  "equity": [
    {
      "type": "CASH",
      "value": "5000.00",
      "portfolioPercentage": "50.00"
    },
    {
      "type": "EQUITY",
      "value": "5000.00",
      "portfolioPercentage": "50.00"
    }
  ],
  "positions": [
    {
      "instrument": {
        "symbol": "AAPL",
        "type": "EQUITY"
      },
      "quantity": "10",
      "currentValue": "1750.00",
      "lastPrice": "175.00",
      "lastPriceTimestamp": "2025-01-10T15:30:00Z",
      "unrealizedGain": "250.00",
      "unrealizedGainPercent": "16.67",
      "dailyGain": "50.00",
      "dailyGainPercent": "2.94",
      "costBasis": "1500.00"
    }
  ],
  "orders": []
}
```

### Get Account History

Retrieves historical transaction records for an account.

**Endpoint:** `GET /userapigateway/trading/{accountId}/history`

**Path Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `accountId` | string | Yes | Target account ID |

**Query Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `start` | string (date-time) | No | Start timestamp (ISO 8601 with timezone) |
| `end` | string (date-time) | No | End timestamp (ISO 8601 with timezone) |
| `pageSize` | integer | No | Maximum number of records to return |
| `nextToken` | string | No | Pagination token for next result set |

**Response (200):**

```json
{
  "transactions": [
    {
      "timestamp": "2025-01-10T10:30:00Z",
      "id": "txn-12345",
      "type": "TRADE",
      "subType": "BUY",
      "accountNumber": "12345678",
      "symbol": "AAPL",
      "securityType": "EQUITY",
      "side": "BUY",
      "description": "Bought 10 shares of AAPL",
      "netAmount": "-1750.00",
      "principalAmount": "-1750.00",
      "quantity": "10",
      "direction": "OUTGOING",
      "fees": "0.00"
    }
  ],
  "nextToken": "abc123",
  "start": "2025-01-01T00:00:00Z",
  "end": "2025-01-10T23:59:59Z",
  "pageSize": 50
}
```

**Transaction Types:**

| Type | Description |
|------|-------------|
| `TRADE` | Buy/sell transactions |
| `MONEY_MOVEMENT` | Deposits, withdrawals |
| `POSITION_ADJUSTMENT` | Corporate actions, transfers |

**Transaction SubTypes:** DEPOSIT, WITHDRAWAL, DIVIDEND, FEE, BUY, SELL, etc.

---

## Instruments

### Get All Instruments

Retrieves all available trading instruments with optional filtering.

**Endpoint:** `GET /userapigateway/trading/instruments`

**Query Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| `typeFilter` | array | Filter by security types |
| `tradingFilter` | array | Filter by trading statuses |
| `fractionalTradingFilter` | array | Filter by fractional trading statuses |
| `optionTradingFilter` | array | Filter by option trading statuses |
| `optionSpreadTradingFilter` | array | Filter by option spread trading statuses |

**Security Types:** EQUITY, OPTION, MULTI_LEG_INSTRUMENT, CRYPTO, ALT, TREASURY, BOND, INDEX

**Trading Statuses:** BUY_AND_SELL, LIQUIDATION_ONLY, DISABLED

**Response (200):**

```json
{
  "instruments": [
    {
      "instrument": {
        "symbol": "AAPL",
        "type": "EQUITY"
      },
      "trading": "BUY_AND_SELL",
      "fractionalTrading": "BUY_AND_SELL",
      "optionTrading": "BUY_AND_SELL",
      "optionSpreadTrading": "BUY_AND_SELL",
      "instrumentDetails": null
    }
  ]
}
```

### Get Instrument

Retrieves details for a specific instrument.

**Endpoint:** `GET /userapigateway/trading/instruments/{symbol}/{type}`

**Path Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `symbol` | string | Yes | Trading symbol |
| `type` | string | Yes | Instrument type (EQUITY, OPTION, CRYPTO, etc.) |

**Response (200):**

```json
{
  "instrument": {
    "symbol": "AAPL",
    "type": "EQUITY"
  },
  "trading": "BUY_AND_SELL",
  "fractionalTrading": "BUY_AND_SELL",
  "optionTrading": "BUY_AND_SELL",
  "optionSpreadTrading": "BUY_AND_SELL",
  "instrumentDetails": null
}
```

---

## Market Data

### Get Quotes

Retrieves real-time quotes for specified instruments.

**Endpoint:** `POST /userapigateway/marketdata/{accountId}/quotes`

**Path Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `accountId` | string | Yes | Account identifier |

**Request Body:**

```json
{
  "instruments": [
    {
      "symbol": "AAPL",
      "type": "EQUITY"
    },
    {
      "symbol": "AAPL240119C00175000",
      "type": "OPTION"
    }
  ]
}
```

**Instrument Types:** EQUITY, OPTION, INDEX

**Response (200):**

```json
{
  "quotes": [
    {
      "instrument": {
        "symbol": "AAPL",
        "type": "EQUITY"
      },
      "outcome": "SUCCESS",
      "last": "175.50",
      "lastTimestamp": "2025-01-10T15:30:00Z",
      "bid": "175.45",
      "bidSize": 100,
      "bidTimestamp": "2025-01-10T15:30:00Z",
      "ask": "175.55",
      "askSize": 200,
      "askTimestamp": "2025-01-10T15:30:00Z",
      "volume": 50000000,
      "openInterest": null
    }
  ]
}
```

**Response Fields:**

| Field | Description |
|-------|-------------|
| `last` | Last traded price |
| `bid`/`ask` | Current market bid/ask prices |
| `bidSize`/`askSize` | Available quantity at bid/ask |
| `volume` | Total traded volume |
| `openInterest` | Open option contracts (options only) |

### Get Option Expirations

Retrieves available expiration dates for options on an underlying.

**Endpoint:** `POST /userapigateway/marketdata/{accountId}/option-expirations`

**Request Body:**

```json
{
  "instrument": {
    "symbol": "AAPL",
    "type": "EQUITY"
  }
}
```

**Supported Types:** EQUITY, UNDERLYING_SECURITY_FOR_INDEX_OPTION

**Response (200):**

```json
{
  "baseSymbol": "AAPL",
  "expirations": [
    "2025-01-17",
    "2025-01-24",
    "2025-01-31",
    "2025-02-21"
  ]
}
```

### Get Option Chain

Retrieves the option chain for a specific expiration date.

**Endpoint:** `POST /userapigateway/marketdata/{accountId}/option-chain`

**Request Body:**

```json
{
  "instrument": {
    "symbol": "AAPL",
    "type": "EQUITY"
  },
  "expirationDate": "2025-01-17"
}
```

**Response (200):**

```json
{
  "baseSymbol": "AAPL",
  "calls": [
    {
      "instrument": {
        "symbol": "AAPL250117C00175000",
        "type": "OPTION"
      },
      "outcome": "SUCCESS",
      "last": "5.50",
      "lastTimestamp": "2025-01-10T15:30:00Z",
      "bid": "5.45",
      "bidSize": 50,
      "ask": "5.55",
      "askSize": 100,
      "volume": 1000,
      "openInterest": 5000
    }
  ],
  "puts": [
    {
      "instrument": {
        "symbol": "AAPL250117P00175000",
        "type": "OPTION"
      },
      "outcome": "SUCCESS",
      "last": "4.50",
      "bid": "4.45",
      "ask": "4.55",
      "volume": 800,
      "openInterest": 3000
    }
  ]
}
```

---

## Order Placement

### Preflight Single Leg

Calculates the estimated financial impact of a potential trade before execution.

**Endpoint:** `POST /userapigateway/trading/{accountId}/preflight/single-leg`

**Request Body:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `instrument` | object | Yes | Trading instrument (symbol + type) |
| `orderSide` | enum | Yes | BUY or SELL |
| `orderType` | enum | Yes | MARKET, LIMIT, STOP, STOP_LIMIT |
| `expiration` | object | Yes | Time-in-force configuration |
| `quantity` | string | No | Share count (mutually exclusive with amount) |
| `amount` | string | No | Dollar value (mutually exclusive with quantity) |
| `limitPrice` | string | Conditional | Required for LIMIT/STOP_LIMIT orders |
| `stopPrice` | string | Conditional | Required for STOP/STOP_LIMIT orders |
| `equityMarketSession` | enum | No | CORE or EXTENDED |
| `openCloseIndicator` | enum | No | OPEN or CLOSE (options only) |

**Response (200):**

```json
{
  "instrument": {
    "symbol": "AAPL",
    "type": "EQUITY"
  },
  "estimatedCommission": "0.00",
  "regulatoryFees": {
    "secFee": "0.01",
    "tafFee": "0.00",
    "orfFee": "0.00"
  },
  "estimatedCost": "1755.01",
  "buyingPowerRequirement": "1755.01",
  "orderValue": "1755.00",
  "estimatedQuantity": "10"
}
```

### Preflight Multi Leg

Calculates the estimated financial impact of a complex multi-leg trade.

**Endpoint:** `POST /userapigateway/trading/{accountId}/preflight/multi-leg`

**Request Body:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `orderType` | enum | Yes | MARKET, LIMIT, STOP, STOP_LIMIT |
| `expiration` | object | Yes | Time-in-force configuration |
| `quantity` | string | No | Number of strategies to execute |
| `limitPrice` | string | Yes | Limit price for the order |
| `legs` | array | Yes | 1-6 order legs |

**Leg Structure:**

```json
{
  "instrument": {
    "symbol": "AAPL250117C00175000",
    "type": "OPTION"
  },
  "side": "BUY",
  "openCloseIndicator": "OPEN",
  "ratioQuantity": 1
}
```

### Place Order

Submits a new order asynchronously.

**Endpoint:** `POST /userapigateway/trading/{accountId}/order`

**Request Body:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `orderId` | string (UUID) | Yes | Globally unique deduplication key |
| `instrument` | object | Yes | Trading instrument |
| `orderSide` | enum | Yes | BUY or SELL |
| `orderType` | enum | Yes | MARKET, LIMIT, STOP, STOP_LIMIT |
| `expiration` | object | Yes | Time-in-force configuration |
| `quantity` | string | Conditional | Order quantity (mutually exclusive with amount) |
| `amount` | string | Conditional | Dollar value (mutually exclusive with quantity) |
| `limitPrice` | string | Conditional | Required for LIMIT/STOP_LIMIT |
| `stopPrice` | string | Conditional | Required for STOP/STOP_LIMIT |
| `equityMarketSession` | enum | No | CORE or EXTENDED (4:00 AM-8:00 PM ET) |
| `openCloseIndicator` | enum | No | OPEN or CLOSE (options only) |

**Example Request:**

```json
{
  "orderId": "912710f1-1a45-4ef0-88a7-cd513781933d",
  "instrument": {
    "symbol": "AAPL",
    "type": "EQUITY"
  },
  "orderSide": "BUY",
  "orderType": "LIMIT",
  "expiration": {
    "timeInForce": "DAY"
  },
  "quantity": "10",
  "limitPrice": "175.00"
}
```

**Response (200):**

```json
{
  "orderId": "912710f1-1a45-4ef0-88a7-cd513781933d"
}
```

> **Note:** Order placement is asynchronous. Use GET /order/{orderId} to check execution status.

### Place Multileg Order

Submits a multi-leg options order.

**Endpoint:** `POST /userapigateway/trading/{accountId}/order/multileg`

**Request Body:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `orderId` | string (UUID) | Yes | Globally unique deduplication key |
| `quantity` | integer | Yes | Number of strategies to execute |
| `type` | enum | Yes | Only LIMIT orders permitted |
| `limitPrice` | string | Yes | Positive for debit, negative for credit spreads |
| `expiration` | object | Yes | DAY or GTD |
| `legs` | array | Yes | 2-6 legs with at most 1 equity leg |

**Response (200):**

```json
{
  "orderId": "cb930ef3-14d8-460f-bbcc-d8f635f9fd74"
}
```

### Get Order

Fetches the status and details of a specific order.

**Endpoint:** `GET /userapigateway/trading/{accountId}/order/{orderId}`

**Path Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `accountId` | string | Yes | Account identifier |
| `orderId` | string (UUID) | Yes | Order identifier |

**Response (200):**

```json
{
  "orderId": "912710f1-1a45-4ef0-88a7-cd513781933d",
  "instrument": {
    "symbol": "AAPL",
    "type": "EQUITY"
  },
  "createdAt": "2025-01-10T10:30:00Z",
  "type": "LIMIT",
  "side": "BUY",
  "status": "FILLED",
  "quantity": "10",
  "limitPrice": "175.00",
  "filledQuantity": "10",
  "averagePrice": "174.95",
  "closedAt": "2025-01-10T10:30:05Z"
}
```

**Order Status Values:**

| Status | Description |
|--------|-------------|
| `NEW` | Order submitted, not yet executed |
| `PARTIALLY_FILLED` | Partial execution |
| `FILLED` | Fully executed |
| `CANCELLED` | Cancelled by user |
| `REJECTED` | Rejected by exchange |
| `EXPIRED` | Expired without execution |

> **Note:** May return 404 if indexing hasn't completed yet.

### Cancel Order

Submits an asynchronous request to cancel an order.

**Endpoint:** `DELETE /userapigateway/trading/{accountId}/order/{orderId}`

**Path Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `accountId` | string | Yes | Account identifier |
| `orderId` | string (UUID) | Yes | Order identifier |

**Response (200):** No content (cancellation request submitted)

> **Note:** Verify cancellation status using GET /order/{orderId}.

---

## Options

### Get Option Greeks

Retrieves option Greeks for a list of OSI-normalized symbols (max 250 per request).

**Endpoint:** `GET /userapigateway/option-details/{accountId}/greeks`

**Query Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `osiSymbols` | array | Yes | Array of OSI-normalized option symbols |

**Response (200):**

```json
{
  "greeks": [
    {
      "symbol": "AAPL250117C00175000",
      "greeks": {
        "delta": "0.65",
        "gamma": "0.05",
        "theta": "-0.15",
        "vega": "0.25",
        "rho": "0.10",
        "impliedVolatility": "0.30"
      }
    }
  ]
}
```

**Greeks Definitions:**

| Greek | Description |
|-------|-------------|
| Delta | How much option value changes with $1 move in underlying |
| Gamma | Rate of change of delta |
| Theta | Time decay per day |
| Vega | Sensitivity to 1% change in implied volatility |
| Rho | Sensitivity to 1% change in interest rate |
| Implied Volatility | Market's forecast of underlying volatility |

---

## Error Handling

The API returns standard HTTP status codes:

| Status | Description |
|--------|-------------|
| 200 | Success |
| 400 | Bad Request - Invalid parameters |
| 401 | Unauthorized - Invalid or missing access token |
| 404 | Not Found - Resource doesn't exist |
| 429 | Too Many Requests - Rate limit exceeded |

Error responses include descriptive messages:

```json
{
  "error": "invalid_request",
  "message": "The instrument symbol is not valid"
}
```

---

## Changelog

| Date | Change |
|------|--------|
| **Nov 24, 2025** | Crypto trading fees reduced from 1.2% to 0.6%. Added batch option Greeks endpoint. |
| **Nov 12, 2025** | Added crypto precision details to instrument endpoint. |
| **Nov 6, 2025** | Added cryptocurrency trading support via Zerohash integration. |
| **Jun 24, 2025** | Added preflight endpoints for order estimation. |
| **Jun 17, 2025** | Initial API release. |

---

## Additional Resources

- **Secret Key Generation:** [public.com/settings/security/api](https://public.com/settings/security/api)
- **Support:** api-concierge@public.com
- **Updates:** [@public](https://twitter.com/public) on X/Twitter
