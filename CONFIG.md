# Configuration

Lunchtui supports configuration files in TOML format. This allows you to set default values for commonly used options without having to specify them on the command line every time.

## Configuration File Locations

Lunchtui looks for configuration files in the following locations, in order of precedence:

1. **Current directory**: `lunchtui.toml`
2. **User config directory**: `~/.config/lunchtui/config.toml` (or `$XDG_CONFIG_HOME/lunchtui/config.toml`)
3. **User home directory**: `~/.lunchtui.toml`
4. **User .config directory**: `~/.config/lunchtui/config.toml` 
5. **System-wide config**: `/etc/lunchtui/config.toml`

The first configuration file found will be used. You can also specify a custom configuration file path using the `--config` or `-c` flag.

## Configuration Options

All global command-line flags can be configured in the configuration file:

| Configuration Key | Type | Description | Default |
|------------------|------|-------------|--------|
| `debug` | boolean | Enable debug logging | `false` |
| `token` | string | Lunch Money API token | "" |
| `debits_as_negative` | boolean | Show debits as negative numbers | `false` |
| `hide_pending_transactions` | boolean | Hide pending transactions from all transaction lists | `false` |

## Example Configuration File

```toml
# Lunchtui Configuration File
# This file should be saved as lunchtui.toml

# Enable debug logging
debug = true

# Lunch Money API token
# You can also use the LUNCHMONEY_API_TOKEN environment variable
token = "your-api-token-here"

# Show debits as negative numbers
debits_as_negative = false
```

## Precedence Order

Configuration values are applied in the following order (later values override earlier ones):

1. Configuration file values
2. Environment variables
3. Command-line flags

This means that command-line flags will always override configuration file settings.

## Usage Examples

### Using a configuration file in the current directory

```bash
# Create a config file
echo 'debug = true' > lunchtui.toml
echo 'token = "your-token"' >> lunchtui.toml

# Run lunchtui - it will automatically find and use the config
lunchtui categories list
```

### Using a custom configuration file

```bash
# Specify a custom config file location
lunchtui --config /path/to/my-config.toml categories list
```

### Overriding configuration with command-line flags

```bash
# Even if debug=false in config file, this will enable debug
lunchtui --debug categories list
```

## No Configuration File

If no configuration file is found, lunchtui will work normally using default values and any environment variables or command-line flags you provide. A missing configuration file is not an error.
