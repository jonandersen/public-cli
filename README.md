# pub

A CLI for trading stocks, ETFs, options, and crypto via Public.com's API.

## Installation

### Homebrew (macOS/Linux)

```bash
brew install jonandersen/tap/pub
```

### Download Binary

Download the latest release for your platform from [GitHub Releases](https://github.com/jonandersen/pub/releases).

```bash
# macOS (Apple Silicon)
curl -LO https://github.com/jonandersen/pub/releases/latest/download/pub-darwin-arm64.tar.gz
tar xzf pub-darwin-arm64.tar.gz
sudo mv pub /usr/local/bin/

# macOS (Intel)
curl -LO https://github.com/jonandersen/pub/releases/latest/download/pub-darwin-amd64.tar.gz
tar xzf pub-darwin-amd64.tar.gz
sudo mv pub /usr/local/bin/

# Linux (x86_64)
curl -LO https://github.com/jonandersen/pub/releases/latest/download/pub-linux-amd64.tar.gz
tar xzf pub-linux-amd64.tar.gz
sudo mv pub /usr/local/bin/

# Linux (ARM64)
curl -LO https://github.com/jonandersen/pub/releases/latest/download/pub-linux-arm64.tar.gz
tar xzf pub-linux-arm64.tar.gz
sudo mv pub /usr/local/bin/
```

### Go Install

```bash
go install github.com/jonandersen/pub@latest
```

### Build from Source

```bash
git clone https://github.com/jonandersen/pub.git
cd pub
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

### View accounts

```bash
pub account                     # List all accounts
pub account portfolio           # View portfolio positions and balances
```

### Output formats

```bash
pub quote AAPL --json           # JSON output for scripting
```

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
