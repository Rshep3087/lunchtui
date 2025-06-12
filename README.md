<div align="center">

# 🍽️ lunchtui

**A beautiful command-line interface for your [Lunch Money](https://lunchmoney.app/) account**

[![Go Report Card](https://goreportcard.com/badge/github.com/Rshep3087/lunchtui)](https://goreportcard.com/report/github.com/Rshep3087/lunchtui)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/github/go-mod/go-version/Rshep3087/lunchtui)](https://golang.org/)
[![GitHub release](https://img.shields.io/github/release/Rshep3087/lunchtui.svg)](https://github.com/Rshep3087/lunchtui/releases/)
[![GitHub stars](https://img.shields.io/github/stars/Rshep3087/lunchtui.svg)](https://github.com/Rshep3087/lunchtui/stargazers)

<img width="1312" alt="lunchtui overview" src="https://github.com/user-attachments/assets/17424945-7767-4743-ab82-fb894f0e6563" />

*Manage your finances directly from your terminal with style and efficiency*

</div>

---

## 📋 Table of Contents

- [✨ Features](#-features)
- [🚀 Installation](#-installation)
- [📖 Usage](#-usage)
- [🔧 Configuration](#-configuration)
- [📸 Screenshots](#-screenshots)
- [🤝 Contributing](#-contributing)
- [📄 License](#-license)
- [🙏 Acknowledgments](#-acknowledgments)

## ✨ Features

- 💰 **Account Balances** - View all your account balances at a glance
- 🔄 **Recurring Expenses** - Monitor your subscription and recurring payments
- 📊 **Transaction Management** - Browse and search through your transactions
- 🏷️ **Smart Categorization** - Easily categorize transactions with intuitive interface
- ✅ **Transaction Status** - Mark transactions as cleared or uncleared
- 🎨 **Beautiful UI** - Enjoy a clean, modern terminal interface built with Bubble Tea
- ⚡ **Fast & Lightweight** - Built in Go for optimal performance

## 🚀 Installation

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

## 📖 Usage

### Quick Start

```bash
# Set your API token
export LUNCHMONEY_API_TOKEN="your-api-token-here"

# Launch lunchtui
lunchtui
```

### Command Line Options

| Flag | Description | Default |
|------|-------------|----------|
| `--token` | Lunch Money API token | Uses `LUNCHMONEY_API_TOKEN` env var |
| `--debits-as-negative` | Show debits as negative numbers | `false` |
| `--help` | Show help message | - |

### Examples

```bash
# Use a specific API token
lunchtui --token="your-token-here"

# Show debits as negative values
lunchtui --debits-as-negative

# Combine options
lunchtui --token="your-token" --debits-as-negative
```

## 🔧 Configuration

### API Token Setup

1. Log into your [Lunch Money account](https://my.lunchmoney.app/)
2. Navigate to **Settings** → **Developers**
3. Create a new API token
4. Set the token as an environment variable:

```bash
# Add to your shell profile (.bashrc, .zshrc, etc.)
export LUNCHMONEY_API_TOKEN="your-api-token-here"
```

## 📸 Screenshots

### 🔄 Recurring Expenses View
<img width="1312" alt="Recurring expenses management" src="https://github.com/user-attachments/assets/81cba900-d185-41be-88d0-148a05a0f4f0" />

*Easily track and manage your recurring expenses and subscriptions*

### 📊 Transaction Browser
<img width="1312" alt="Transaction browser interface" src="https://github.com/user-attachments/assets/8c1379f2-0c7c-4f09-80eb-b89ea31c3f10" />

*Browse through your transactions with powerful filtering and search capabilities*

### 🏷️ Transaction Categorization
<img width="1312" alt="Transaction categorization" src="https://github.com/user-attachments/assets/0a36b35b-f913-4fe7-b29f-612c132842dc" />

*Quickly categorize transactions with an intuitive interface*

## 🤝 Contributing

Contributions are welcome! Here's how you can help:

1. **Fork** the repository
2. **Create** a feature branch (`git checkout -b feature/amazing-feature`)
3. **Commit** your changes (`git commit -m 'Add some amazing feature'`)
4. **Push** to the branch (`git push origin feature/amazing-feature`)
5. **Open** a Pull Request

### Development Setup

```bash
# Clone the repository
git clone https://github.com/Rshep3087/lunchtui.git
cd lunchtui

# Install dependencies
go mod download

# Run tests
go test ./...

# Build the project
go build -o lunchtui
```

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [Lunch Money](https://lunchmoney.app/) - For providing an excellent personal finance platform
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - For the amazing TUI framework
- [Charm](https://charm.sh/) - For the beautiful terminal UI components

---

<div align="center">

**Enjoyed using lunchtui?** Give it a ⭐ to show your support!

[Report Bug](https://github.com/Rshep3087/lunchtui/issues) • [Request Feature](https://github.com/Rshep3087/lunchtui/issues) • [Discussions](https://github.com/Rshep3087/lunchtui/discussions)

</div>
