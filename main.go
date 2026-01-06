package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Keybinding represents a single keyboard shortcut
type Keybinding struct {
	Shortcut string
	Action   string // underlying command/target
	Desc     string // human-friendly description
	Category string
	App      string
}

// Sidebar item representing an application bucket
type appItem struct {
	Name  string
	Count int
}

func (a appItem) Title() string       { return a.Name }
func (a appItem) Description() string { return fmt.Sprintf("%d", a.Count) }
func (a appItem) FilterValue() string { return a.Name }

// Item implements list.Item interface
func (k Keybinding) FilterValue() string { return k.Shortcut + " " + k.Desc + " " + k.Action }
func (k Keybinding) Title() string       { return k.Shortcut }
func (k Keybinding) Description() string { return k.Desc }

// Model represents the application state
type model struct {
	sidebar      list.Model
	main         list.Model
	keybindings  []Keybinding
	filteredKeys []Keybinding
	apps         []string
	selectedApp  string
	focusSidebar bool
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

	descStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A6E3A1"))

	cmdStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8BE9FD"))

	rowBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(0, 1)

	docStyle = lipgloss.NewStyle().Margin(1, 2)
)

// keyMap defines keybindings for the TUI
type keyMap struct {
	Quit  key.Binding
	Focus key.Binding
}

var keys = keyMap{
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Focus: key.NewBinding(
		key.WithKeys("tab", "shift+tab"),
		key.WithHelp("tab", "toggle focus"),
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
		sidebarWidth := 24
		if msg.Width < 70 {
			sidebarWidth = 18
		} else if msg.Width > 110 {
			sidebarWidth = 28
		}
		contentWidth := msg.Width - sidebarWidth - 4
		if contentWidth < 24 {
			// squeeze sidebar if terminal is very narrow
			sidebarWidth = 14
			contentWidth = msg.Width - sidebarWidth - 4
			if contentWidth < 20 {
				contentWidth = 20
			}
		}

		listHeight := msg.Height - 4
		if listHeight < 5 {
			listHeight = 5
		}

		m.sidebar.SetWidth(sidebarWidth)
		m.sidebar.SetHeight(listHeight)
		m.main.SetWidth(contentWidth)
		m.main.SetHeight(listHeight)
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Quit):
			m.quitting = true
			return m, tea.Quit
		case key.Matches(msg, keys.Focus):
			m.focusSidebar = !m.focusSidebar
			if m.focusSidebar {
				m.sidebar.Select(0)
			}
		}
	}

	var cmd tea.Cmd
	if m.focusSidebar {
		m.sidebar, cmd = m.sidebar.Update(msg)
		if sel, ok := m.sidebar.SelectedItem().(appItem); ok && sel.Name != m.selectedApp {
			m.selectedApp = sel.Name
			m.applyFilter()
		}
	} else {
		m.main, cmd = m.main.Update(msg)
	}
	return m, cmd
}

func (m model) View() string {
	if m.quitting {
		return ""
	}

	title := titleStyle.Render("⌨️  Keyboard Shortcuts")
	sidebar := m.sidebar.View()
	content := m.main.View()

	layout := lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(m.sidebar.Width()+2).Render(sidebar),
		lipgloss.NewStyle().PaddingLeft(2).Render(content),
	)

	return docStyle.Render(title + "\n\n" + layout)
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

	apps, sidebarItems := buildSidebar(keybindings)

	// Sidebar list
	sideDelegate := list.NewDefaultDelegate()
	sideDelegate.ShowDescription = true
	sideDelegate.SetSpacing(0)
	sidebar := list.New(sidebarItems, sideDelegate, 0, 0)
	sidebar.Title = "Apps"
	sidebar.SetShowStatusBar(false)
	sidebar.DisableQuitKeybindings()

	// Main list
	items := toListItems(keybindings)
	mainDelegate := newRowDelegate()
	mainList := list.New(items, mainDelegate, 0, 0)
	mainList.Title = "Keybindings"
	mainList.SetShowStatusBar(true)
	mainList.SetFilteringEnabled(true)
	mainList.Styles.Title = titleStyle
	mainList.DisableQuitKeybindings()

	m := model{
		sidebar:      sidebar,
		main:         mainList,
		keybindings:  keybindings,
		filteredKeys: keybindings,
		apps:         apps,
		selectedApp:  "All",
		focusSidebar: true,
	}
	m.applyFilter()

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

		rawAction := action
		// Clean up exec prefix
		if strings.HasPrefix(rawAction, "exec, ") {
			rawAction = strings.TrimSpace(rawAction[6:])
		}

		desc := humanizeAction(rawAction)

		// Format shortcut
		shortcut := key
		if modifiers != "" {
			shortcut = modifiers + " + " + key
		}

		category := categorizeBinding(key, action)

		bindings = append(bindings, Keybinding{
			Shortcut: shortcut,
			Action:   rawAction,
			Desc:     desc,
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
		desc := humanizeAction(action)

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
			Desc:     desc,
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

			desc := humanizeAction(action)
			bindings = append(bindings, Keybinding{
				Shortcut: fmt.Sprintf("%s (%s)", key, mode),
				Action:   action,
				Desc:     desc,
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

			desc := humanizeAction(action)
			bindings = append(bindings, Keybinding{
				Shortcut: fmt.Sprintf("%s (%s)", key, mode),
				Action:   action,
				Desc:     desc,
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

// Custom main list delegate rendering as bordered row with columns
type rowDelegate struct{}

func newRowDelegate() list.ItemDelegate {
	return rowDelegate{}
}

func (d rowDelegate) Height() int                               { return 3 }
func (d rowDelegate) Spacing() int                              { return 0 }
func (d rowDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }

func (d rowDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	kb, ok := listItem.(Keybinding)
	if !ok {
		return
	}

	width := m.Width()
	if width <= 0 {
		width = 80
	}
	// Account for two separators (3 chars each)
	sepWidth := 6
	avail := width - sepWidth
	if avail < 30 {
		avail = width
	}

	colShortcut := int(float64(avail) * 0.22)
	if colShortcut < 12 {
		colShortcut = 12
	}

	colDesc := int(float64(avail) * 0.45)
	if colDesc < 24 {
		colDesc = 24
	}

	colCmd := avail - colShortcut - colDesc
	if colCmd < 14 {
		deficit := 14 - colCmd
		colCmd = 14
		// Steal from desc first
		if colDesc-deficit > 18 {
			colDesc -= deficit
		} else {
			colShortcut -= deficit
			if colShortcut < 10 {
				colShortcut = 10
			}
		}
	}

	sep := lipgloss.NewStyle().Foreground(lipgloss.Color("#44475A")).Render(" │ ")

	shortcutCell := lipgloss.NewStyle().Width(colShortcut).Render(shortcutStyle.Render(kb.Shortcut))
	descCell := lipgloss.NewStyle().Width(colDesc).Render(descStyle.Render(kb.Desc))
	cmdCell := lipgloss.NewStyle().Width(colCmd).Render(cmdStyle.Render(kb.Action))

	row := lipgloss.JoinHorizontal(lipgloss.Top, shortcutCell, sep, descCell, sep, cmdCell)

	box := rowBoxStyle
	if index == m.Index() {
		box = box.BorderForeground(lipgloss.Color("#BD93F9"))
		row = lipgloss.NewStyle().Bold(true).Render(row)
	}

	fmt.Fprint(w, box.Width(width).Render(row))
}

// Helpers for UI data

func buildSidebar(bindings []Keybinding) ([]string, []list.Item) {
	counts := map[string]int{}
	for _, kb := range bindings {
		counts[kb.App]++
	}

	apps := make([]string, 0, len(counts)+1)
	apps = append(apps, "All")
	for app := range counts {
		apps = append(apps, app)
	}
	// Sort but keep All first
	sortedApps := []string{"All"}
	if len(apps) > 1 {
		rest := apps[1:]
		sort.Strings(rest)
		sortedApps = append(sortedApps, rest...)
	}

	items := make([]list.Item, 0, len(sortedApps))
	for _, app := range sortedApps {
		count := 0
		if app == "All" {
			for _, c := range counts {
				count += c
			}
		} else {
			count = counts[app]
		}
		items = append(items, appItem{Name: app, Count: count})
	}

	return sortedApps, items
}

func toListItems(bindings []Keybinding) []list.Item {
	items := make([]list.Item, len(bindings))
	for i, kb := range bindings {
		items[i] = kb
	}
	return items
}

// applyFilter refreshes the main list based on selected app
func (m *model) applyFilter() {
	filtered := m.keybindings
	if m.selectedApp != "All" {
		tmp := []Keybinding{}
		for _, kb := range m.keybindings {
			if kb.App == m.selectedApp {
				tmp = append(tmp, kb)
			}
		}
		filtered = tmp
	}
	items := toListItems(filtered)
	m.main.SetItems(items)
	m.filteredKeys = filtered
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
