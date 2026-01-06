package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Keybinding represents a single keyboard shortcut
type Keybinding struct {
	Shortcut string
	Action   string
	Category string
	App      string
}

// Item implements list.Item interface
func (k Keybinding) FilterValue() string { return k.Shortcut + " " + k.Action }
func (k Keybinding) Title() string       { return k.Shortcut }
func (k Keybinding) Description() string { return k.Action }

// Model represents the application state
type model struct {
	list         list.Model
	keybindings  []Keybinding
	filteredKeys []Keybinding
	width        int
	height       int
	quitting     bool
}

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	appStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF79C6"))

	categoryStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#50FA7B"))

	shortcutStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#F1FA8C"))

	docStyle = lipgloss.NewStyle().Margin(1, 2)
)

// keyMap defines keybindings for the TUI
type keyMap struct {
	Quit key.Binding
}

var keys = keyMap{
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetWidth(msg.Width)
		m.list.SetHeight(msg.Height - 4)
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Quit):
			m.quitting = true
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	if m.quitting {
		return ""
	}

	title := titleStyle.Render("⌨️  Keyboard Shortcuts")
	return docStyle.Render(title + "\n\n" + m.list.View())
}

func main() {
	// Parse all keybindings
	keybindings := []Keybinding{}

	homeDir, _ := os.UserHomeDir()

	// Parse Hyprland
	hyprKeys := parseHyprland(filepath.Join(homeDir, ".config/hypr/hyprland.conf"))
	keybindings = append(keybindings, hyprKeys...)

	// Parse Kitty
	kittyKeys := parseKitty(filepath.Join(homeDir, ".config/kitty/kitty.conf"))
	keybindings = append(keybindings, kittyKeys...)

	// Parse Neovim
	nvimKeys := parseNeovim(filepath.Join(homeDir, ".config/nvim/init.lua"))
	keybindings = append(keybindings, nvimKeys...)

	// Convert to list items
	items := make([]list.Item, len(keybindings))
	for i, kb := range keybindings {
		items[i] = kb
	}

	// Create delegate for custom rendering
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = shortcutStyle
	delegate.Styles.SelectedDesc = lipgloss.NewStyle().Foreground(lipgloss.Color("#BD93F9"))

	// Create list
	l := list.New(items, delegate, 0, 0)
	l.Title = "Keybindings"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.Styles.Title = titleStyle

	m := model{
		list:        l,
		keybindings: keybindings,
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}

// parseHyprland extracts keybindings from hyprland.conf
func parseHyprland(configPath string) []Keybinding {
	var bindings []Keybinding

	data, err := os.ReadFile(configPath)
	if err != nil {
		return bindings
	}

	lines := strings.Split(string(data), "\n")
	bindRe := regexp.MustCompile(`bind[lmn]?\s*=\s*([^,]+),\s*([^,]+),\s*(.*)`)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		matches := bindRe.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		modifiers := parseModifiers(matches[1])
		key := strings.TrimSpace(matches[2])
		action := strings.TrimSpace(matches[3])

		// Clean up exec prefix
		if strings.HasPrefix(action, "exec, ") {
			action = action[6:]
		}

		action = humanizeAction(action)

		// Format shortcut
		shortcut := key
		if modifiers != "" {
			shortcut = modifiers + " + " + key
		}

		category := categorizeBinding(key, action)

		bindings = append(bindings, Keybinding{
			Shortcut: shortcut,
			Action:   action,
			Category: category,
			App:      "Hyprland",
		})
	}

	return bindings
}

// parseKitty extracts keybindings from kitty.conf
func parseKitty(configPath string) []Keybinding {
	var bindings []Keybinding

	data, err := os.ReadFile(configPath)
	if err != nil {
		return bindings
	}

	lines := strings.Split(string(data), "\n")
	kittyMod := "ctrl+shift"

	// First pass: find kitty_mod
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "kitty_mod ") {
			kittyMod = strings.TrimSpace(strings.TrimPrefix(line, "kitty_mod"))
			break
		}
	}

	// Second pass: parse map directives
	mapRe := regexp.MustCompile(`map\s+([^\s]+)\s+(.*)`)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		matches := mapRe.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		keyCombo := matches[1]
		action := strings.TrimSpace(matches[2])

		// Replace kitty_mod
		if strings.Contains(keyCombo, "kitty_mod") {
			keyCombo = strings.Replace(keyCombo, "kitty_mod", kittyMod, -1)
		}

		// Format key combo
		keys := strings.Split(keyCombo, "+")
		formatted := []string{}
		for _, k := range keys {
			k = strings.TrimSpace(k)
			if contains([]string{"ctrl", "shift", "alt", "super", "cmd"}, strings.ToLower(k)) {
				formatted = append(formatted, strings.Title(k))
			} else {
				if len(k) == 1 {
					formatted = append(formatted, strings.ToUpper(k))
				} else {
					formatted = append(formatted, strings.Title(k))
				}
			}
		}

		shortcut := strings.Join(formatted, " + ")

		bindings = append(bindings, Keybinding{
			Shortcut: shortcut,
			Action:   action,
			Category: "Terminal",
			App:      "Kitty",
		})
	}

	return bindings
}

// parseNeovim extracts keybindings from init.lua
func parseNeovim(configPath string) []Keybinding {
	var bindings []Keybinding

	data, err := os.ReadFile(configPath)
	if err != nil {
		// Try init.vim
		configPath = strings.Replace(configPath, ".lua", ".vim", 1)
		data, err = os.ReadFile(configPath)
		if err != nil {
			return bindings
		}
	}

	lines := strings.Split(string(data), "\n")

	// Lua format: vim.keymap.set('n', '<leader>ff', ':Telescope find_files<CR>')
	luaRe := regexp.MustCompile(`vim\.keymap\.set\(['"](.)['"],\s*['"]([^'"]+)['"],\s*['"]([^'"]+)['"]`)

	// Vim format: nnoremap <leader>ff :Telescope find_files<CR>
	vimRe := regexp.MustCompile(`([nvicosx]?noremap|[nvicosx]?map)\s+([^\s]+)\s+(.*)`)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "--") || strings.HasPrefix(line, "\"") {
			continue
		}

		// Try Lua format
		if matches := luaRe.FindStringSubmatch(line); matches != nil {
			mode := matches[1]
			key := matches[2]
			action := matches[3]

			key = strings.ReplaceAll(key, "<leader>", "Leader+")
			key = strings.ReplaceAll(key, "<", "")
			key = strings.ReplaceAll(key, ">", "")
			action = strings.ReplaceAll(action, "<CR>", "")
			action = strings.TrimSpace(action)

			bindings = append(bindings, Keybinding{
				Shortcut: fmt.Sprintf("%s (%s)", key, mode),
				Action:   action,
				Category: "Editor",
				App:      "Neovim",
			})
			continue
		}

		// Try Vim format
		if matches := vimRe.FindStringSubmatch(line); matches != nil {
			mapType := matches[1]
			key := matches[2]
			action := matches[3]

			mode := "n"
			modes := []string{"n", "v", "i", "c", "o", "x"}
			if len(mapType) > 0 {
				for _, m := range modes {
					if string(mapType[0]) == m {
						mode = m
						break
					}
				}
			}

			key = strings.ReplaceAll(key, "<leader>", "Leader+")
			key = strings.ReplaceAll(key, "<", "")
			key = strings.ReplaceAll(key, ">", "")
			action = strings.ReplaceAll(action, "<CR>", "")
			action = strings.TrimSpace(action)

			bindings = append(bindings, Keybinding{
				Shortcut: fmt.Sprintf("%s (%s)", key, mode),
				Action:   action,
				Category: "Editor",
				App:      "Neovim",
			})
		}
	}

	return bindings
}

// Helper functions

func parseModifiers(modStr string) string {
	var mods []string

	if strings.Contains(modStr, "$mainMod") {
		mods = append(mods, "Super")
	}
	if strings.Contains(strings.ToUpper(modStr), "SHIFT") {
		mods = append(mods, "Shift")
	}
	if strings.Contains(strings.ToUpper(modStr), "CONTROL") || strings.Contains(strings.ToUpper(modStr), "CTRL") {
		mods = append(mods, "Ctrl")
	}
	if strings.Contains(strings.ToUpper(modStr), "ALT") {
		mods = append(mods, "Alt")
	}

	return strings.Join(mods, " + ")
}

func humanizeAction(action string) string {
	action = strings.TrimSpace(action)
	action = strings.Trim(action, ",")

	humanizations := map[string]string{
		"$terminal":              "Open terminal",
		"$fileManager":           "Open file manager",
		"$menu":                  "Open launcher",
		"$lock":                  "Lock screen",
		"hyprlock":               "Lock screen",
		"killactive":             "Close window",
		"togglefloating":         "Toggle floating/tiling",
		"togglesplit":            "Toggle split layout",
		"togglespecialworkspace": "Toggle special workspace",
		"movetoworkspace":        "Move to workspace",
		"workspace":              "Switch to workspace",
		"movefocus":              "Move focus",
		"movewindow":             "Move window",
		"resizewindow":           "Resize window",
		"pseudo":                 "Pseudo-tiling mode",
		"hyprshot":               "Take screenshot",
		"playerctl":              "Media control",
		"pamixer":                "Volume control",
		"powermenu":              "Power menu",
		"exit":                   "Exit Hyprland",
	}

	for keyword, description := range humanizations {
		if strings.Contains(action, keyword) {
			param := strings.ReplaceAll(action, keyword, "")
			param = strings.Trim(param, ", \"'")
			if param != "" && !strings.HasPrefix(param, "-") {
				return fmt.Sprintf("%s (%s)", description, param)
			}
			return description
		}
	}

	if len(action) > 60 {
		return action[:57] + "..."
	}

	return action
}

func categorizeBinding(key, action string) string {
	combined := strings.ToLower(key + " " + action)

	categories := map[string][]string{
		"Screenshots":  {"PRINT", "screenshot", "hyprshot"},
		"Workspaces":   {"workspace", "movetoworkspace"},
		"Windows":      {"killactive", "togglefloating", "togglesplit", "pseudo"},
		"Focus":        {"movefocus"},
		"Media":        {"XF86Audio", "playerctl", "pamixer"},
		"Applications": {"exec", "terminal", "fileManager", "menu", "lock", "exit", "powermenu"},
		"Mouse":        {"mouse"},
	}

	for category, keywords := range categories {
		for _, keyword := range keywords {
			if strings.Contains(combined, strings.ToLower(keyword)) {
				return category
			}
		}
	}

	return "Other"
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// Custom list item delegate to show app and category
type itemDelegate struct{}

func (d itemDelegate) Height() int                               { return 2 }
func (d itemDelegate) Spacing() int                              { return 1 }
func (d itemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	kb, ok := listItem.(Keybinding)
	if !ok {
		return
	}

	str := fmt.Sprintf("%s %s %s %s",
		appStyle.Render(kb.App),
		categoryStyle.Render(kb.Category),
		shortcutStyle.Render(kb.Shortcut),
		kb.Action,
	)

	fmt.Fprint(w, str)
}
