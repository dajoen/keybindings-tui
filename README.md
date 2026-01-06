# Keybindings TUI

A beautiful terminal user interface for viewing all your keyboard shortcuts across multiple applications.

![Keybindings TUI](https://img.shields.io/badge/Made%20with-Charm-F25D94?logo=data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAA4AAAAOCAYAAAAfSC3RAAAACXBIWXMAAAsTAAALEwEAmpwYAAAAAXNSR0IArs4c6QAAAARnQU1BAACxjwv8YQUAAADFSURBVHgBlZLRDcIwDETf)

## Features

- 🎨 Beautiful TUI built with [Charm](https://charm.sh)
- ⌨️ Parses keybindings from multiple applications with sidebar selection:
  - **Hyprland** (Window Manager)
  - **Kitty** (Terminal)
  - **Neovim** (Editor)
  - **VSCode** (coming soon)
- 🔍 Interactive search and filtering
- 📱 Organized by application and category
- 🎯 Easy to navigate with keyboard shortcuts

## Installation

```bash
go install github.com/dajoen/keybindings-tui@latest
```

Or build from source:

```bash
git clone https://github.com/dajoen/keybindings-tui.git
cd keybindings-tui
go build -o keybindings-tui
```

### Make targets

- `make install` – build and copy to `~/.local/bin`
- `make install-go` – use `go install` to place the binary in your Go bin (typically `~/go/bin` or `$GOBIN`)

Ensure your PATH includes either `~/.local/bin` or your Go bin (e.g. add `export PATH="$HOME/.local/bin:$PATH"` or `export PATH="$HOME/go/bin:$PATH"`).

## Usage

Simply run the command:

```bash
keybindings-tui
```

### Keybindings

- `↑/↓` or `j/k` - Navigate through bindings
- `/` - Filter/search bindings
- `q` or `Ctrl+C` - Quit

## Supported Configurations

The TUI automatically reads from:
- `~/.config/hypr/hyprland.conf`
- `~/.config/kitty/kitty.conf`
- `~/.config/nvim/init.lua` or `init.vim`

## Development

Built with:
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - Style definitions
- [Bubbles](https://github.com/charmbracelet/bubbles) - TUI components

## License

MIT

## Author

[Jeroen Verhoeven](https://github.com/dajoen)
