---
name: public
description: Use the `pub` CLI to interact with Public.com trading accounts. Use this skill when the user asks about their portfolio, stock quotes, placing orders, options trading, or transaction history. Always use --json flag for programmatic access.
---

# Public.com CLI (`pub`) - Complete Command Reference

A CLI for trading stocks, ETFs, options, and crypto via Public.com's API.

## Important Notes for AI Agents

- **Always use `--json` flag** for programmatic access to data
- Commands require authentication via `pub configure` (one-time setup)
- Most commands accept `--account` flag but use default account if configured
- Use `--yes` flag to skip confirmation prompts for automated trading

---

## Global Flags

| Flag | Description |
|------|-------------|
| `--json, -j` | Output in JSON format (for programmatic access) |
| `--version` | Show version information |

---

## Configuration

### `pub configure`

Configure CLI credentials for Public.com API.

**Flags:**
- `--account` - Default account UUID (optional)

**Interactive Menu Options (when already configured):**
1. Select different default account
2. Configure new secret key
3. View current configuration
4. Toggle trading (enable/disable)
5. Clear secret key

**Examples:**
```bash
pub configure                              # Interactive setup
pub configure --account YOUR_ACCOUNT_UUID  # Set default account
```

**Setup Instructions:**
1. Get secret key from: https://public.com/settings/security/api
2. Run `pub configure`
3. Enter secret key when prompted
4. Select default account

---

## Account & Portfolio

### `pub account`

List all accounts.

**Example:**
```bash
pub account --json
```

### `pub account portfolio`

View portfolio positions and balances.

**Flags:**
| Flag | Description |
|------|-------------|
| `--account, -a` | Account ID (uses default if configured) |
| `--only` | Filter JSON output: `buying-power`, `positions`, `equity` |

**Examples:**
```bash
pub account portfolio --json
pub account portfolio --json --only buying-power
pub account portfolio --json --only positions
pub account portfolio --json --only equity
```

**JSON Output Structure:**
```json
{
  "buyingPower": {
    "cashOnlyBuyingPower": "45.03",
    "buyingPower": "45.03",
    "optionsBuyingPower": "45.03"
  },
  "equity": [
    {
      "type": "STOCK|CRYPTO|CASH",
      "value": "12471.98",
      "percentageOfPortfolio": "99.10"
    }
  ],
  "positions": [
    {
      "instrument": {
        "symbol": "AAPL",
        "name": "Apple Inc",
        "type": "EQUITY"
      },
      "quantity": "10.5",
      "currentValue": "2500.00",
      "percentOfPortfolio": "20.0",
      "lastPrice": {
        "lastPrice": "238.09",
        "timestamp": "2026-01-30T00:59:49Z"
      },
      "positionDailyGain": {
        "gainValue": "26.60",
        "gainPercentage": "3.70"
      },
      "costBasis": {
        "totalCost": "2000.00",
        "unitCost": "190.48",
        "gainValue": "500.00",
        "gainPercentage": "25.0"
      }
    }
  ]
}
```

---

## Quotes

### `pub quote SYMBOL [SYMBOL...]`

Get real-time quotes for one or more symbols.

**Flags:**
| Flag | Description |
|------|-------------|
| `--account, -a` | Account ID (uses default if not specified) |

**Examples:**
```bash
pub quote AAPL --json
pub quote AAPL GOOGL MSFT --json
```

**JSON Output:**
```json
[
  {
    "Symbol": "AAPL",
    "Last": "238.09",
    "Bid": "238.05",
    "Ask": "238.15",
    "Volume": "39,996,967"
  }
]
```

---

## Order Management

### `pub order buy SYMBOL`

Place a buy order for shares.

**Flags:**
| Flag | Description |
|------|-------------|
| `--quantity, -q` | Number of shares (required) |
| `--limit, -l` | Limit price (creates LIMIT order) |
| `--stop, -s` | Stop price (creates STOP order) |
| `--expiration, -e` | Order duration: `DAY` (default) or `GTC` |
| `--yes, -y` | Skip confirmation prompt |
| `--account, -a` | Account ID |

**Order Types (determined by flags):**
- No price flags → MARKET order
- `--limit` → LIMIT order
- `--stop` → STOP order
- `--limit` + `--stop` → STOP_LIMIT order

**Examples:**
```bash
pub order buy AAPL --quantity 10 --yes --json                       # Market
pub order buy AAPL --quantity 10 --limit 175 --yes --json           # Limit
pub order buy AAPL --quantity 10 --stop 180 --yes --json            # Stop
pub order buy AAPL --quantity 10 --limit 175 --stop 174 --yes --json # Stop-limit
pub order buy AAPL --quantity 10 --limit 175 --expiration GTC --yes --json
```

### `pub order sell SYMBOL`

Place a sell order for shares. Same flags as `buy`.

**Examples:**
```bash
pub order sell AAPL --quantity 5 --yes --json                # Market
pub order sell AAPL --quantity 5 --limit 180 --yes --json    # Limit
pub order sell AAPL --quantity 5 --stop 145 --yes --json     # Stop loss
```

### `pub order list`

List all open orders.

**Flags:**
| Flag | Description |
|------|-------------|
| `--account, -a` | Account ID |

**Example:**
```bash
pub order list --json
```

**JSON Output:**
```json
[
  {
    "orderId": "912710f1-1a45-4ef0-88a7-cd513781933d",
    "symbol": "AAPL",
    "side": "BUY",
    "quantity": "10",
    "orderType": "LIMIT",
    "limitPrice": "175.00",
    "status": "NEW",
    "filledQuantity": "0",
    "createdAt": "2026-01-30T15:30:00Z"
  }
]
```

### `pub order status ORDER_ID`

Check status of a specific order.

**Order Statuses:**
- `NEW` - Awaiting execution
- `PARTIALLY_FILLED` - Some shares filled
- `FILLED` - Fully executed
- `CANCELLED` - User cancelled
- `REJECTED` - Order rejected
- `EXPIRED` - DAY order expired

**Example:**
```bash
pub order status 912710f1-1a45-4ef0-88a7-cd513781933d --json
```

### `pub order cancel ORDER_ID`

Cancel an open order.

**Flags:**
| Flag | Description |
|------|-------------|
| `--yes, -y` | Skip confirmation prompt |
| `--account, -a` | Account ID |

**Examples:**
```bash
pub order cancel 912710f1-1a45-4ef0-88a7-cd513781933d --yes --json
```

---

## Transaction History

### `pub history`

View account transaction history.

**Flags:**
| Flag | Description |
|------|-------------|
| `--account, -a` | Account ID |
| `--start` | Start timestamp (ISO 8601: `2026-01-01T00:00:00Z`) |
| `--end` | End timestamp (ISO 8601) |
| `--limit, -l` | Maximum number of transactions |

**Examples:**
```bash
pub history --json
pub history --limit 10 --json
pub history --start 2026-01-01T00:00:00Z --json
pub history --start 2026-01-01T00:00:00Z --end 2026-01-31T23:59:59Z --json
```

**JSON Output:**
```json
{
  "nextToken": "...",
  "pageSize": 10,
  "transactions": [
    {
      "id": "e3afc2cd-...",
      "timestamp": "2026-01-15T04:00:00Z",
      "type": "TRADE|MONEY_MOVEMENT|DIVIDEND",
      "subType": "TRADE|MISC|...",
      "symbol": "AAPL",
      "securityType": "EQUITY|CRYPTO|OPTION",
      "side": "BUY|SELL",
      "description": "BUY 10 AAPL at 175.00",
      "netAmount": "-1750.00",
      "quantity": "10",
      "direction": "INCOMING|OUTGOING",
      "fees": "0.00"
    }
  ]
}
```

---

## Instrument Information

### `pub instrument SYMBOL`

Get trading capabilities for a single symbol.

**Flags:**
| Flag | Description |
|------|-------------|
| `--type, -t` | Instrument type (default: `EQUITY`) |

**Valid Types:** `EQUITY`, `OPTION`, `CRYPTO`, `ALT`, `TREASURY`, `BOND`, `INDEX`

**Examples:**
```bash
pub instrument AAPL --json
pub instrument BTC --type CRYPTO --json
```

**JSON Output:**
```json
{
  "instrument": {
    "symbol": "AAPL",
    "type": "EQUITY"
  },
  "trading": "BUY_AND_SELL",
  "fractionalTrading": "BUY_AND_SELL",
  "optionTrading": "BUY_AND_SELL",
  "optionSpreadTrading": "BUY_AND_SELL"
}
```

**Trading Status Values:**
- `BUY_AND_SELL` - Full trading enabled
- `LIQUIDATION_ONLY` - Can only sell existing positions
- `DISABLED` - No trading allowed

### `pub instruments`

List all available instruments.

**Flags:**
| Flag | Description |
|------|-------------|
| `--type, -t` | Filter by type: `EQUITY`, `CRYPTO`, etc. |
| `--trading` | Filter by status: `BUY_AND_SELL`, `LIQUIDATION_ONLY`, `DISABLED` |

**Examples:**
```bash
pub instruments --json
pub instruments --type EQUITY --json
pub instruments --type CRYPTO --json
pub instruments --trading BUY_AND_SELL --json
```

---

## Options Trading

### `pub options expirations SYMBOL`

List available expiration dates for options.

**Example:**
```bash
pub options expirations AAPL --json
```

### `pub options chain SYMBOL`

Display option chain for a symbol.

**Flags:**
| Flag | Description |
|------|-------------|
| `--expiration, -e` | Expiration date YYYY-MM-DD (required) |
| `--strikes` | Show N strikes centered around ATM |
| `--min-strike` | Minimum strike price |
| `--max-strike` | Maximum strike price |
| `--min-oi` | Minimum open interest |
| `--min-volume` | Minimum daily volume |
| `--calls-only` | Show only calls |
| `--puts-only` | Show only puts |
| `--account, -a` | Account ID |

**Examples:**
```bash
pub options chain AAPL --expiration 2026-02-21 --json
pub options chain AAPL -e 2026-02-21 --strikes 10 --json
pub options chain AAPL -e 2026-02-21 --calls-only --min-oi 100 --json
pub options chain AAPL -e 2026-02-21 --min-strike 170 --max-strike 190 --json
```

### `pub options greeks SYMBOL [SYMBOL...]`

Get greeks for option symbols.

**Option Symbol Format (OSI):** `AAPL260221C00175000`
- `AAPL` - Underlying symbol
- `260221` - Expiration (YYMMDD)
- `C` or `P` - Call or Put
- `00175000` - Strike price * 1000 (175.00)

**Examples:**
```bash
pub options greeks AAPL260221C00175000 --json
pub options greeks AAPL260221C00175000 AAPL260221P00175000 --json
```

**JSON Output:**
```json
[
  {
    "symbol": "AAPL260221C00175000",
    "delta": "0.65",
    "gamma": "0.02",
    "theta": "-0.15",
    "vega": "0.25",
    "rho": "0.10",
    "iv": "0.32"
  }
]
```

**Greeks Explained:**
- `delta` - Price sensitivity to underlying movement
- `gamma` - Rate of delta change
- `theta` - Time decay (daily value loss)
- `vega` - Sensitivity to implied volatility
- `rho` - Sensitivity to interest rates
- `iv` - Implied volatility

### `pub options buy SYMBOL`

Buy options contracts (single-leg).

**Flags:**
| Flag | Description |
|------|-------------|
| `--quantity, -q` | Number of contracts (required) |
| `--limit, -l` | Limit price (required) |
| `--expiration, -e` | Order duration: `DAY` (default) or `GTC` |
| `--open` | Buy to open a new position |
| `--close` | Buy to close an existing short position |
| `--yes, -y` | Skip confirmation prompt |
| `--account, -a` | Account ID |

**Must specify either `--open` or `--close`**

**Examples:**
```bash
pub options buy AAPL260221C00175000 --quantity 1 --limit 2.50 --open --yes --json
pub options buy AAPL260221P00170000 --quantity 1 --limit 1.25 --close --yes --json
```

### `pub options sell SYMBOL`

Sell options contracts (single-leg). Same flags as `buy`.

**Examples:**
```bash
pub options sell AAPL260221C00175000 --quantity 1 --limit 2.50 --close --yes --json
pub options sell AAPL260221P00170000 --quantity 1 --limit 1.25 --open --yes --json
```

### `pub options multileg preflight`

Preview a multi-leg options order before placing.

**Flags:**
| Flag | Description |
|------|-------------|
| `--leg, -L` | Leg format: `"SIDE SYMBOL OPEN\|CLOSE [RATIO]"` (repeat for each leg) |
| `--limit, -l` | Limit price (required) |
| `--quantity, -q` | Number of spreads (default: 1) |
| `--expiration, -e` | Order duration: `DAY` (default) or `GTC` |
| `--account, -a` | Account ID |

**Leg Format:**
- `SIDE` - BUY or SELL
- `SYMBOL` - Option symbol (OSI format)
- `OPEN|CLOSE` - Opening or closing the position
- `RATIO` - Quantity multiplier (optional, default 1)

**Requires 2-6 legs**

**Examples:**
```bash
# Vertical call spread
pub options multileg preflight \
  --leg "BUY AAPL260221C00175000 OPEN" \
  --leg "SELL AAPL260221C00180000 OPEN" \
  --limit 2.50 --quantity 1 --json

# Iron condor
pub options multileg preflight \
  --leg "SELL AAPL260221P00165000 OPEN" \
  --leg "BUY AAPL260221P00160000 OPEN" \
  --leg "SELL AAPL260221C00185000 OPEN" \
  --leg "BUY AAPL260221C00190000 OPEN" \
  --limit 1.20 --quantity 1 --json
```

### `pub options multileg order`

Place a multi-leg options order. Same flags as `preflight`, plus `--yes`.

**Examples:**
```bash
# Vertical call spread
pub options multileg order \
  --leg "BUY AAPL260221C00175000 OPEN" \
  --leg "SELL AAPL260221C00180000 OPEN" \
  --limit 2.50 --quantity 1 --yes --json

# Iron condor
pub options multileg order \
  --leg "SELL AAPL260221P00165000 OPEN" \
  --leg "BUY AAPL260221P00160000 OPEN" \
  --leg "SELL AAPL260221C00185000 OPEN" \
  --leg "BUY AAPL260221C00190000 OPEN" \
  --limit 1.20 --quantity 1 --yes --json
```

---

## Common Workflows

### Check Portfolio Health
```bash
pub account portfolio --json
```

### Get Quote Before Trading
```bash
pub quote AAPL --json
```

### Place and Monitor Stock Order
```bash
pub order buy AAPL --quantity 10 --limit 175 --yes --json
pub order list --json
pub order status ORDER_ID --json
```

### Options Analysis Flow
```bash
pub options expirations AAPL --json
pub options chain AAPL --expiration 2026-02-21 --json
pub options greeks AAPL260221C00175000 --json
```

### Place Options Trade
```bash
pub options buy AAPL260221C00175000 --quantity 1 --limit 2.50 --open --yes --json
```

### Place Multi-Leg Options Trade
```bash
pub options multileg preflight \
  --leg "BUY AAPL260221C00175000 OPEN" \
  --leg "SELL AAPL260221C00180000 OPEN" \
  --limit 2.50 --quantity 1 --json

pub options multileg order \
  --leg "BUY AAPL260221C00175000 OPEN" \
  --leg "SELL AAPL260221C00180000 OPEN" \
  --limit 2.50 --quantity 1 --yes --json
```

---

## Error Handling

Commands return non-zero exit codes on failure. Common errors:

| Error | Solution |
|-------|----------|
| Authentication expired | Re-run `pub configure` |
| Insufficient buying power | Check `buyingPower` in portfolio |
| Invalid symbol | Verify with `pub instrument SYMBOL` |
| Market closed | Orders queued for next open |
| Trading disabled | Run `pub configure` and enable trading |
