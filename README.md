<div align="center">

# 🍽️ lunchtui

**A beautiful command-line interface for your [Lunch Money](https://lunchmoney.app/) account**

[![Go Report Card](https://goreportcard.com/badge/github.com/Rshep3087/lunchtui)](https://goreportcard.com/report/github.com/Rshep3087/lunchtui)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/github/go-mod/go-version/Rshep3087/lunchtui)](https://golang.org/)
[![GitHub release](https://img.shields.io/github/release/Rshep3087/lunchtui.svg)](https://github.com/Rshep3087/lunchtui/releases/)
[![GitHub stars](https://img.shields.io/github/stars/Rshep3087/lunchtui.svg)](https://github.com/Rshep3087/lunchtui/stargazers)

![overview](docs/SCR-20250624-ttcu.png)

*Manage your finances directly from your terminal with style and efficiency*

</div>

---

## Table of Contents

- [Features](#-features)
- [Installation](#-installation)
- [Usage](#-usage)
- [CLI Commands](#-cli-commands)
- [Configuration](#-configuration)
- [Screenshots](#-screenshots)
- [Contributing](#-contributing)
- [License](#-license)
- [Acknowledgments](#-acknowledgments)

## Features

- **Account Balances** - View all your account balances at a glance
- **Transaction Management** - Browse and search through your transactions
- **Recurring Expenses** - Monitor your subscription and recurring payments
- **Budget Tracking** - Monitor your spending against budgets with real-time progress
- **Categorization** - Easily categorize transactions with intuitive interface
- **Transaction Status** - Mark transactions as cleared or uncleared

## Installation

### Option 1: Download Pre-built Binary

Download the latest release from the [releases page](https://github.com/Rshep3087/lunchtui/releases) and extract the binary to a location in your PATH.

### Option 2: Install with Go

```bash
go install github.com/Rshep3087/lunchtui@latest
```

### Option 3: Build from Source

```bash
git clone https://github.com/Rshep3087/lunchtui.git
cd lunchtui
go build -o lunchtui
```

## Usage

### Quick Start

```bash
# Set your API token
export LUNCHMONEY_API_TOKEN="your-api-token-here"

# Launch lunchtui
lunchtui
```

### Navigation

Once lunchtui is running, use these keyboard shortcuts to navigate:

| Key | View | Description |
|-----|------|-------------|
| `o` | Overview | Account balances and financial summary |
| `t` | Transactions | Browse and manage your transactions |
| `b` | Budgets | View budget progress and spending by category |
| `r` | Recurring | Monitor recurring expenses and subscriptions |
| `g` | Configuration | View current configuration settings (sensitive values are masked) |
| `[` / `]` | - | Navigate between previous/next time periods |
| `s` | - | Switch between time period types (month/year) |
| `?` | - | Toggle help menu |
| `q` | - | Quit the application |

### Examples

```bash
# Use a specific API token
lunchtui --token="your-token-here"

# Show debits as negative values
lunchtui --debits-as-negative

# Use a custom configuration file
lunchtui --config /path/to/my-config.toml

# Enable debug logging
lunchtui --debug

# Combine options (flags override config file)
lunchtui --token="your-token" --debits-as-negative --debug
```

## CLI Commands

In addition to the interactive TUI interface, lunchtui provides powerful CLI commands for automation and scripting.

### Available Commands

#### Transaction Management

##### `lunchtui transaction insert`
Insert a new transaction directly from the command line.

**Usage:**
```bash
lunchtui transaction insert --payee "Coffee Shop" --amount "4.50" [options]

# Basic transaction
lunchtui transaction insert --payee "Grocery Store" --amount "45.67"

# Transaction with category and notes
lunchtui transaction insert --payee "Gas Station" --amount "35.00" --category 123 --notes "Weekly fill-up"

# Income transaction (negative amount)
lunchtui transaction insert --payee "Freelance Work" --amount "-500.00" --status cleared

# Transaction with tags
lunchtui transaction insert --payee "Restaurant" --amount "25.50" --tags 1 --tags 2
```

#### Categories Management

##### `lunchtui categories list`
List all categories with their IDs and details.

**Usage:**
```bash
# List categories in table format
lunchtui categories list

# List categories in JSON format
lunchtui categories list --output json
```

#### Accounts Management

##### `lunchtui accounts list`
List all accounts (both assets and Plaid accounts) with their IDs and details.

**Usage:**
```bash
lunchtui accounts list [options]
```

### Global Flags

All commands support these global flags:

- `--token` - Lunch Money API token (can use `LUNCHMONEY_API_TOKEN` env var)
- `--debits-as-negative` - Show debits as negative numbers
- `--debug` - Enable debug logging

### Getting Help

Use the `--help` flag with any command to see detailed usage information:

```bash
# General help
lunchtui --help

# Command-specific help
lunchtui transaction --help
lunchtui categories list --help
```

## Configuration

### Configuration File Support

Lunchtui supports TOML configuration files to set default values for commonly used options. This eliminates the need to specify flags repeatedly.

**Configuration file locations** (in order of precedence):
- Current directory: `lunchtui.toml`
- User config: `~/.config/lunchtui/config.toml`
- User home: `~/.lunchtui.toml`
- System-wide: `/etc/lunchtui/config.toml`

**Example configuration file:**
```toml
# lunchtui.toml
debug = true
token = "your-api-token-here"
debits_as_negative = false
```

You can also specify a custom config file with `--config path/to/config.toml`.

**For complete configuration documentation, see [CONFIG.md](CONFIG.md)**

### API Token Setup

1. Log into your [Lunch Money account](https://my.lunchmoney.app/)
2. Navigate to **Settings** → **Developers**
3. Create a new API token
4. Set the token via configuration file or environment variable:

```bash
# Option 1: Environment variable
export LUNCHMONEY_API_TOKEN="your-api-token-here"

# Option 2: Configuration file
echo 'token = "your-api-token-here"' > lunchtui.toml
```

## Screenshots

### Budget Tracking
![budgets](docs/SCR-20250624-ttgl.png)
*Monitor your spending against budgets with detailed progress tracking*

**Key Features:**
- **Real-time Progress** - See how much you've spent vs. your budget for each category
- **Multi-currency Support** - View budgets in their original currency
- **Transaction Count** - Track the number of transactions per budget category
- **Period Navigation** - Switch between different time periods to view historical budget data
- **Category Integration** - Budgets are automatically linked to your transaction categories

**Budget View Information:**
- Budget amount and currency
- Amount spent to date
- Number of transactions in the category
- Easy navigation between time periods

### Recurring Expenses View
![recurring expenses](docs/SCR-20250624-tthl.png)

*Easily track and manage your recurring expenses and subscriptions*

### Transaction Browser
![alt text](docs/SCR-20250624-ttft.png)

*Browse through your transactions with powerful filtering and search capabilities*

### Transaction Categorization
<img width="1312" alt="Transaction categorization" src="https://github.com/user-attachments/assets/0a36b35b-f913-4fe7-b29f-612c132842dc" />

*Quickly categorize transactions with an intuitive interface*

## Contributing

Contributions are welcome!

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- [Lunch Money](https://lunchmoney.app/) - For providing an excellent personal finance platform
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - For the amazing TUI framework
- [Charm](https://charm.sh/) - For the beautiful terminal UI components

---

<div align="center">

**Enjoyed using lunchtui?** Give it a ⭐ to show your support!

[Report Bug](https://github.com/Rshep3087/lunchtui/issues) • [Request Feature](https://github.com/Rshep3087/lunchtui/issues) • [Discussions](https://github.com/Rshep3087/lunchtui/discussions)

</div>

## Color Customization

You can customize the colors used throughout the TUI application by adding a `[colors]` section to your configuration file. Colors can be specified using either hex values (e.g., `"#ff0000"`) or ANSI color codes (e.g., `"21"`).

### Available Color Options

| Setting | Default | Description |
|---------|---------|-------------|
| `primary` | `"#ffd644"` | Primary accent color (highlights, selected items, key bindings) |
| `error` | `"#ff0000"` | Error messages and failed transactions |
| `success` | `"#22ba46"` | Successful/cleared transactions and positive values |
| `warning` | `"#e05951"` | Uncleared transactions and warnings |
| `muted` | `"#7f7d78"` | Pending transactions and secondary text |
| `income` | `"#00ff00"` | Positive income values |
| `expense` | `"#ff0000"` | Negative expense values |
| `border` | `"#7D56F4"` | Borders and separators |
| `background` | `"#7D56F4"` | Highlighted backgrounds |
| `text` | `"#FAFAFA"` | Primary text |
| `secondary_text` | `"#888888"` | Less important text and separators |

### Example Configuration

```toml
[colors]
primary = "21"              # ANSI blue instead of default yellow
error = "#ff5555"          # Slightly lighter red  
success = "28"             # ANSI green
warning = "#ff8800"        # Orange instead of red
muted = "240"              # ANSI dark gray
income = "#00cc00"         # Darker green
expense = "#cc0000"        # Darker red
border = "27"              # ANSI bright blue
background = "54"          # ANSI purple
text = "#ffffff"           # Pure white
secondary_text = "245"     # ANSI light gray
```

### Color Formats

- **Hex colors**: Use standard hex notation like `"#ff0000"` for red
- **ANSI colors**: Use ANSI color codes like `"21"` for bright blue
  - Standard colors: 0-7 (black, red, green, yellow, blue, magenta, cyan, white)
  - Bright colors: 8-15
  - Extended colors: 16-255

The application will fall back to default colors if any color value is invalid or missing.
