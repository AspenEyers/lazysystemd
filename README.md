# lazysystemd

> A minimal terminal UI for monitoring and controlling systemd services. Think lazydocker, but for systemd.

**⚠️ Experimental** - This is a work-in-progress. Things might break, change, or disappear.

## What is this?

A simple TUI that lets you:
- See your systemd services at a glance
- View logs in real-time
- Start, stop, restart, and reload services
- All from your terminal, no GUI needed

## Installation

### From Source

```bash
# Build it
make build

# Install to /usr/local/bin (requires sudo)
sudo make install

# Or install to a custom location
sudo make install PREFIX=/opt/lazysystemd

# Uninstall
sudo make uninstall
```

### Debian Package

```bash
# Install build deps (if needed)
sudo apt-get install build-essential debhelper golang-go gzip

# You may need newer go: 
wget https://go.dev/dl/go1.22.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.22.0.linux-amd64.tar.gz
sudo ln -s /usr/local/go/bin/go /usr/local/bin/go

# Build the package
dpkg-buildpackage -b -uc -us

# Install it
sudo dpkg -i ../lazysystemd_1.0.0-1_amd64.deb
```

The package installs to:
- `/usr/bin/lazysystemd`
- `/etc/lazysystemd/config.example.yaml`

## Quick Start

```bash
# Run it (config will be created automatically if missing)
lazysystemd
```

That's it. No dependencies beyond Go and systemd.

## Configuration

The app defaults to reading configuration from:
- `$HOME/.config/lazysystemd/config.yaml`

If the file doesn't exist, it will be created automatically as an empty file. If the file is empty, you'll see a message: `reading yaml from ... and is empty`

Create a `config.yaml` file listing the services you want to monitor:

```yaml
services:
  - my-service.service
  - another-service.service
```

You can also specify a custom config path:
```bash
lazysystemd -config /path/to/config.yaml
```

## Controls

| Key | Action |
|-----|--------|
| `↑` / `k` | Move up |
| `↓` / `j` | Move down |
| `g` | Jump to top |
| `G` | Jump to bottom |
| `s` | Start service |
| `t` | Stop service |
| `r` | Restart service |
| `l` | Reload service |
| `f` | Toggle live log following |
| `R` | Force refresh |
| `q` | Quit |

## Status Indicators

- `●` = Active and running
- `○` = Inactive
- `✗` = Failed
- `→` = Activating
- `←` = Deactivating
- `?` = Unknown/Error


## How It Works

- Uses `systemctl show` to get service status (parses key=value output)
- Uses `journalctl -u` for logs
- Follow mode streams with `journalctl -f`
- Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) for the TUI

## Requirements

- Linux with systemd
- Go 1.22+ (for building)

## Known Issues

- If you hit Go module cache issues with `uniseg`, try: `go clean -modcache && go mod tidy`

## Contributing

This is experimental, so:
- Feel free to open issues
- PRs welcome (but no promises on merge speed)

## License

MIT

## Why?

I wanted a simple way to monitor a few systemd services without opening multiple terminal windows or remembering service names. 
