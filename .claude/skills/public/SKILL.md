---
name: public-cli
description: Use the `pub` CLI to interact with Public.com trading accounts. Use this skill when the user asks about their portfolio, stock quotes, placing orders, options trading, or transaction history. Always use --json flag for programmatic access.
---

# Public.com CLI (`pub`) - AI Agent Skill

A CLI for trading stocks, ETFs, options, and crypto via Public.com's API.

## Important Notes for AI Agents

- **Always use `--json` flag** for programmatic access to data
- Commands require authentication via `pub configure` (one-time setup)
- Most commands accept `--account` flag but use default account if configured

---

## Commands Reference

### Account & Portfolio

#### `pub account portfolio --json`
View portfolio including buying power, positions, and gains.

**Purpose:** Get complete overview of user's holdings, cash balance, and performance.

**JSON Output Structure:**
```json
{
  "buyingPower": {
    "cashOnlyBuyingPower": "45.03",      // Cash available for non-margin trades
    "buyingPower": "45.03",               // Total buying power
    "optionsBuyingPower": "45.03"         // Cash available for options
  },
  "equity": [
    {
      "type": "STOCK|CRYPTO|CASH",        // Asset type
      "value": "12471.98",                // Current value in USD
      "percentageOfPortfolio": "99.10"    // Allocation percentage
    }
  ],
  "positions": [
    {
      "instrument": {
        "symbol": "AVGO",
        "name": "Broadcom",
        "type": "EQUITY"
      },
      "quantity": "2.1608",               // Shares held (can be fractional)
      "currentValue": "745.02",           // Current market value
      "percentOfPortfolio": "4.57",       // Portfolio allocation %
      "lastPrice": {
        "lastPrice": "344.79",
        "timestamp": "2026-01-10T00:59:49Z"
      },
      "instrumentGain": {
        "gainValue": "12.31",             // Dollar gain/loss today
        "gainPercentage": "3.70"          // Percent gain/loss today
      },
      "positionDailyGain": {
        "gainValue": "26.60",             // Position's daily dollar change
        "gainPercentage": "3.70"          // Position's daily percent change
      },
      "costBasis": {
        "totalCost": "225.08",            // Total amount invested
        "unitCost": "104.17",             // Average cost per share
        "gainValue": "519.94",            // Total dollar gain/loss
        "gainPercentage": "231.00",       // Total percent gain/loss
        "lastUpdate": "2024-05-01T20:00:00Z"
      }
    }
  ]
}
```

**Key Metrics:**
- `buyingPower.buyingPower` - Available cash for new purchases
- `positions[].costBasis.gainPercentage` - Total return on position
- `positions[].positionDailyGain` - Today's performance

---

### Quotes

#### `pub quote SYMBOL [SYMBOL...] --json`
Get real-time quotes for one or more symbols.

**Purpose:** Check current prices, bid/ask spread, and volume.

**Example:** `pub quote AAPL GOOGL MSFT --json`

**JSON Output:**
```json
[
  {
    "Symbol": "AAPL",
    "Last": "259.3692",      // Last traded price
    "Bid": "259.35",         // Current bid price
    "Ask": "259.45",         // Current ask price
    "Volume": "39,996,967"   // Daily volume
  }
]
```

**Key Metrics:**
- `Last` - Most recent trade price
- `Bid/Ask` spread - Liquidity indicator (tight = good liquidity)
- `Volume` - Trading activity level

---

### Transaction History

#### `pub history --json`
View account transaction history.

**Purpose:** Review past trades, deposits, withdrawals, dividends.

**Flags:**
- `--limit N` - Limit number of results
- `--start YYYY-MM-DDTHH:MM:SSZ` - Filter by start date
- `--end YYYY-MM-DDTHH:MM:SSZ` - Filter by end date

**Example:** `pub history --limit 10 --json`

**JSON Output:**
```json
{
  "nextToken": "...",           // For pagination
  "pageSize": 3,
  "transactions": [
    {
      "id": "e3afc2cd-...",
      "timestamp": "2019-06-10T04:00:00Z",
      "type": "TRADE|MONEY_MOVEMENT|DIVIDEND",
      "subType": "TRADE|MISC|...",
      "accountNumber": "5RE24223",
      "symbol": "SPOT",
      "securityType": "EQUITY|CRYPTO|OPTION",
      "side": "BUY|SELL",
      "description": "BUY 1 SPOT at 139.9699",
      "netAmount": "-139.97",       // Negative = outflow, Positive = inflow
      "principalAmount": "-139.9699",
      "quantity": "1",
      "direction": "INCOMING|OUTGOING",
      "fees": "0.00"
    }
  ]
}
```

**Transaction Types:**
- `TRADE` - Buy/sell execution
- `MONEY_MOVEMENT` - Deposits/withdrawals
- `DIVIDEND` - Dividend payments

---

### Order Management

#### `pub order buy SYMBOL --quantity N --json`
Place a buy order.

**Order Types (determined by flags):**
- No price flags = MARKET order
- `--limit PRICE` = LIMIT order
- `--stop PRICE` = STOP order
- `--limit PRICE --stop PRICE` = STOP_LIMIT order

**Flags:**
- `--quantity N` - Number of shares (required)
- `--limit PRICE` - Limit price
- `--stop PRICE` - Stop trigger price
- `--expiration DAY|GTC` - Order duration (default: DAY)
- `--yes` - Skip confirmation (required for automation)

**Examples:**
```bash
pub order buy AAPL --quantity 10 --yes --json              # Market order
pub order buy AAPL --quantity 10 --limit 175 --yes --json  # Limit order
pub order buy AAPL --quantity 10 --stop 180 --yes --json   # Stop order
```

#### `pub order sell SYMBOL --quantity N --json`
Place a sell order. Same flags as buy.

**Examples:**
```bash
pub order sell AAPL --quantity 5 --yes --json              # Market order
pub order sell AAPL --quantity 5 --limit 180 --yes --json  # Limit order
pub order sell AAPL --quantity 5 --stop 145 --yes --json   # Stop loss
```

#### `pub order list --json`
List all open orders.

**Purpose:** Check pending orders awaiting execution.

**JSON Output:**
```json
[
  {
    "orderId": "912710f1-...",
    "symbol": "AAPL",
    "side": "BUY",
    "quantity": "10",
    "orderType": "LIMIT",
    "limitPrice": "175.00",
    "status": "NEW|PARTIALLY_FILLED",
    "filledQuantity": "0",
    "createdAt": "2026-01-10T15:30:00Z"
  }
]
```

#### `pub order status ORDER_ID --json`
Check status of a specific order.

**Order Statuses:**
- `NEW` - Order submitted, awaiting execution
- `PARTIALLY_FILLED` - Some shares filled
- `FILLED` - Fully executed
- `CANCELLED` - User cancelled
- `REJECTED` - Order rejected (insufficient funds, etc.)
- `EXPIRED` - DAY order expired at market close

#### `pub order cancel ORDER_ID --yes --json`
Cancel an open order.

---

### Instrument Information

#### `pub instrument SYMBOL --json`
Get trading capabilities for a symbol.

**Purpose:** Check if instrument supports options, fractional trading, etc.

**Flags:**
- `--type EQUITY|CRYPTO|OPTION|ALT|TREASURY|BOND|INDEX`

**Example:** `pub instrument AAPL --json`

**JSON Output:**
```json
{
  "instrument": {
    "symbol": "AAPL",
    "type": "EQUITY"
  },
  "trading": "BUY_AND_SELL|LIQUIDATION_ONLY|DISABLED",
  "fractionalTrading": "BUY_AND_SELL|LIQUIDATION_ONLY|DISABLED",
  "optionTrading": "BUY_AND_SELL|LIQUIDATION_ONLY|DISABLED",
  "optionSpreadTrading": "BUY_AND_SELL|LIQUIDATION_ONLY|DISABLED"
}
```

**Trading Status Values:**
- `BUY_AND_SELL` - Full trading enabled
- `LIQUIDATION_ONLY` - Can only sell existing positions
- `DISABLED` - No trading allowed

#### `pub instruments --json`
List all available instruments.

**Flags:**
- `--type EQUITY|CRYPTO|...` - Filter by type
- `--trading BUY_AND_SELL|LIQUIDATION_ONLY|DISABLED` - Filter by status

---

### Options Trading

#### `pub options expirations SYMBOL --json`
List available expiration dates for options.

**Purpose:** Find valid expiration dates before viewing chain.

**Example:** `pub options expirations AAPL --json`

#### `pub options chain SYMBOL --expiration YYYY-MM-DD --json`
Display option chain for a symbol and expiration.

**Purpose:** View all available strikes, calls, and puts.

**Example:** `pub options chain AAPL --expiration 2026-01-17 --json`

#### `pub options greeks SYMBOL [SYMBOL...] --json`
Get option greeks for specific option symbols.

**Purpose:** Analyze option risk metrics.

**Option Symbol Format (OSI):** `AAPL250117C00175000`
- `AAPL` - Underlying
- `250117` - Expiration (YYMMDD)
- `C` or `P` - Call or Put
- `00175000` - Strike price * 1000

**Example:** `pub options greeks AAPL250117C00175000 --json`

**Greeks Explained:**
- `delta` - Price sensitivity to underlying movement
- `gamma` - Rate of delta change
- `theta` - Time decay (daily value loss)
- `vega` - Sensitivity to implied volatility
- `rho` - Sensitivity to interest rates
- `iv` - Implied volatility

#### `pub options multileg preflight --json`
Preview a multi-leg options order before placing.

**Purpose:** Estimate costs and validate strategy.

**Leg Format:** `"SIDE SYMBOL OPEN|CLOSE [RATIO]"`
- `SIDE` - BUY or SELL
- `SYMBOL` - Option symbol (OSI format)
- `OPEN|CLOSE` - Opening or closing position
- `RATIO` - Quantity multiplier (optional, default 1)

**Example - Vertical Call Spread:**
```bash
pub options multileg preflight \
  --leg "BUY AAPL250117C00175000 OPEN" \
  --leg "SELL AAPL250117C00180000 OPEN" \
  --limit 2.50 --quantity 1 --json
```

#### `pub options multileg order --json`
Place a multi-leg options order.

**Requires:** `--yes` flag for confirmation

**Example - Iron Condor:**
```bash
pub options multileg order \
  --leg "SELL AAPL250117P00165000 OPEN" \
  --leg "BUY AAPL250117P00160000 OPEN" \
  --leg "SELL AAPL250117C00185000 OPEN" \
  --leg "BUY AAPL250117C00190000 OPEN" \
  --limit 1.20 --quantity 1 --yes --json
```

---

## Common Workflows

### Check Portfolio Health
```bash
pub account portfolio --json
```
Look at: `buyingPower`, position `gainPercentage`, portfolio allocation

### Get Quote Before Trading
```bash
pub quote AAPL --json
```
Check bid/ask spread and recent price

### Place and Monitor Order
```bash
pub order buy AAPL --quantity 10 --limit 175 --yes --json
pub order list --json
pub order status ORDER_ID --json
```

### Options Analysis
```bash
pub options expirations AAPL --json
pub options chain AAPL --expiration 2026-02-21 --json
pub options greeks AAPL260221C00175000 --json
```

---

## Error Handling

Commands return non-zero exit codes on failure. Common errors:
- Authentication expired - Re-run `pub configure`
- Insufficient buying power - Check `buyingPower` in portfolio
- Invalid symbol - Verify with `pub instrument SYMBOL`
- Market closed - Orders may be queued for next open
