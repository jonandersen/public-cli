# pub

A CLI for trading stocks, ETFs, options, and crypto via Public.com's API.

## Installation

### Homebrew (macOS/Linux)

```bash
brew install jonandersen/tap/pub
```

### Download Binary

Download the latest release for your platform from [GitHub Releases](https://github.com/jonandersen/public-cli/releases), or use the GitHub CLI:

```bash
# macOS (Apple Silicon)
gh release download --repo jonandersen/public-cli --pattern '*darwin_arm64.tar.gz'
tar xzf pub_*_darwin_arm64.tar.gz
sudo mv pub /usr/local/bin/

# macOS (Intel)
gh release download --repo jonandersen/public-cli --pattern '*darwin_amd64.tar.gz'
tar xzf pub_*_darwin_amd64.tar.gz
sudo mv pub /usr/local/bin/

# Linux (x86_64)
gh release download --repo jonandersen/public-cli --pattern '*linux_amd64.tar.gz'
tar xzf pub_*_linux_amd64.tar.gz
sudo mv pub /usr/local/bin/

# Linux (ARM64)
gh release download --repo jonandersen/public-cli --pattern '*linux_arm64.tar.gz'
tar xzf pub_*_linux_arm64.tar.gz
sudo mv pub /usr/local/bin/
```

### Go Install

```bash
go install github.com/jonandersen/public-cli@latest
```

### Build from Source

```bash
git clone https://github.com/jonandersen/public-cli.git
cd public-cli
make build
```

## Setup

1. Generate a secret key at https://public.com/settings/security/api
2. Configure the CLI:

```bash
pub configure
```

Your secret key is stored securely in your system keyring (macOS Keychain, Linux Secret Service, or Windows Credential Manager).

## Usage

### Get quotes

```bash
pub quote AAPL                  # Single stock
pub quote AAPL GOOGL MSFT       # Multiple stocks
```

### View accounts and portfolio

```bash
pub account                     # List all accounts
pub account portfolio           # View portfolio positions and balances
```

### Place orders

```bash
pub order buy AAPL 10           # Buy 10 shares of AAPL at market price
pub order sell AAPL 5           # Sell 5 shares
pub order buy AAPL 10 --limit 150.00   # Limit order at $150
pub order list                  # View open orders
pub order cancel <order-id>     # Cancel an order
```

### Options trading

```bash
pub options chain AAPL          # View options chain
pub options buy AAPL 2025-01-17 150 call 1   # Buy 1 call contract
pub options sell AAPL 2025-01-17 150 put 1   # Sell 1 put contract
```

### Transaction history

```bash
pub history                     # View recent transactions
pub history --limit 50          # Limit number of results
```

### Instruments

```bash
pub instrument AAPL             # Get details for a symbol
pub instruments --type stock    # List available instruments
```

### Output formats

```bash
pub quote AAPL --json           # JSON output for scripting
pub account portfolio --json    # Works with any command
```

## Terminal UI

Launch an interactive terminal interface with real-time portfolio monitoring:

```bash
pub ui
```

**Features:**
- **Portfolio view** - See all positions with live P&L
- **Watchlist** - Track symbols you're interested in
- **Keyboard navigation** - Switch views with number keys (1-4)

**Key bindings:**
- `1-4` - Switch between views
- `a` - Add symbol to watchlist (in watchlist view)
- `d` - Delete symbol from watchlist
- `q` - Quit

## Configuration

Config file: `~/.config/pub/config.yaml`

```yaml
account_uuid: "your-default-account"
api_base_url: "https://api.public.com"
```

## Development

```bash
make test    # Run tests
make lint    # Run linter
make all     # Format, lint, test, and build
```

## License

MIT
