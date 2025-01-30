# lunchtui

lunchtui is a command line tool for viewing and interacting with your [Lunch Money](https://lunchmoney.app/) account.


## Features

- View account balances
- View recurring expenses
- View transactions
- Categorize transactions
- Mark transactions as cleared/uncleared
  

## Installation

Dowload from the [releases page](https://github.com/Rshep3087/lunchtui/releases) and extract the binary to a location in your PATH.

Or install it with Go:

```bash
go install github.com/Rshep3087/lunchtui@latest
```

## Usage

- `--token` - Set the Lunch Money API token or use the `LUNCHMONEY_API_TOKEN` environment variable
- `--debits-as-negative` - Show debits as negative numbers
- `--help` - Show help message


