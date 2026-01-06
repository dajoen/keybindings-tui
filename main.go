package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
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
	plain := flag.Bool("plain", false, "print keybindings to stdout and exit")
	flag.Parse()

	// Parse all keybindings
	keybindings := []Keybinding{}

	homeDir, _ := os.UserHomeDir()

	// Parse Hyprland
	hyprKeys := parseHyprland(filepath.Join(homeDir, ".config/hypr/hyprland.conf"))
	keybindings = append(keybindings, hyprKeys...)

	// Parse Kitty
	kittyKeys := parseKitty(filepath.Join(homeDir, ".config/kitty/kitty.conf"))
	keybindings = append(keybindings, kittyKeys...)

	// Parse Neovim - scan all plugin files
	nvimKeys := parseNeovimAll(filepath.Join(homeDir, ".config/nvim"))
	keybindings = append(keybindings, nvimKeys...)

	if *plain {
		printPlain(keybindings)
		return
	}

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

// parseNeovimAll scans all Neovim config files recursively
func parseNeovimAll(nvimDir string) []Keybinding {
	var bindings []Keybinding

	// Detect leader and localleader from init.lua
	leader, localLeader := detectNvimLeaders(filepath.Join(nvimDir, "init.lua"))

	// Parse main init.lua
	mainFile := filepath.Join(nvimDir, "init.lua")
	bindings = append(bindings, parseNeovimFile(mainFile, "Core", leader, localLeader)...)

	// Parse all lua files recursively
	luaDir := filepath.Join(nvimDir, "lua")
	filepath.Walk(luaDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() || !strings.HasSuffix(path, ".lua") {
			return nil
		}

		// Extract module name from path
		relPath, _ := filepath.Rel(luaDir, path)
		module := strings.TrimSuffix(relPath, ".lua")
		module = strings.ReplaceAll(module, string(filepath.Separator), ".")

		// Clean up module name for display
		parts := strings.Split(module, ".")
		if len(parts) > 0 {
			// Use the last meaningful part or "plugins.X"
			if len(parts) >= 2 && parts[len(parts)-2] == "plugins" {
				module = "Plugin: " + strings.Title(parts[len(parts)-1])
			} else {
				module = strings.Title(parts[len(parts)-1])
			}
		}

		bindings = append(bindings, parseNeovimFile(path, module, leader, localLeader)...)
		return nil
	})

	return bindings
}

// parseNeovimFile extracts keybindings from a single Neovim lua file
func parseNeovimFile(configPath, module, leader, localLeader string) []Keybinding {
	var bindings []Keybinding

	data, err := os.ReadFile(configPath)
	if err != nil {
		return bindings
	}

	lines := strings.Split(string(data), "\n")

	// Lua format: vim.keymap.set('n', '<leader>ff', ..., { desc = 'Find files' })
	// With description capture
	luaReWithDesc := regexp.MustCompile(`vim\.keymap\.set\(['"](.)['"],\s*['"]([^'"]+)['"],\s*[^,]+,\s*\{[^}]*desc\s*=\s*['"]([^'"]+)['"]`)

	// Without description
	luaRe := regexp.MustCompile(`vim\.keymap\.set\(['"](.)['"],\s*['"]([^'"]+)['"],\s*['"]?([^'"]+)['"]?`)

	// Also look for map() function calls in plugin configs
	mapRe := regexp.MustCompile(`map\(['"](.)['"],\s*['"]([^'"]+)['"],\s*[^,]+,\s*\{[^}]*desc\s*=\s*['"]([^'"]+)['"]`)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "--") {
			continue
		}

		var mode, key, desc, action string
		matched := false

		// Try with description first
		if matches := luaReWithDesc.FindStringSubmatch(line); matches != nil {
			mode = matches[1]
			key = matches[2]
			desc = matches[3]
			matched = true
		} else if matches := mapRe.FindStringSubmatch(line); matches != nil {
			mode = matches[1]
			key = matches[2]
			desc = matches[3]
			matched = true
		} else if matches := luaRe.FindStringSubmatch(line); matches != nil {
			mode = matches[1]
			key = matches[2]
			action = matches[3]
			matched = true
		}

		if !matched {
			continue
		}

		// Format key using detected leader/localleader and friendly names
		friendlyKey := formatNeovimKey(key, leader, localLeader)

		// Clean action and try to enrich description from Neovim help if missing
		if desc == "" {
			action = strings.ReplaceAll(action, "<CR>", "")
			action = strings.ReplaceAll(action, "<cr>", "")
			action = strings.TrimSpace(action)

			if d := lookupNvimHelpDescription(action); d != "" {
				desc = d
			} else {
				desc = humanizeAction(action)
			}
		}

		if action == "" && desc != "" {
			action = "(function)"
		}

		bindings = append(bindings, Keybinding{
			Shortcut: fmt.Sprintf("%s (%s)", friendlyKey, displayMode(mode)),
			Action:   action,
			Desc:     desc,
			Category: module,
			App:      "Neovim",
		})
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

	// Remove common exec prefixes and surrounding quotes
	action = strings.TrimPrefix(action, "exec ")
	action = strings.TrimPrefix(action, "exec,")
	action = strings.TrimSpace(action)
	action = strings.Trim(action, "\"'")

	// Collapse excessive spacing
	fields := strings.Fields(action)
	action = strings.Join(fields, " ")

	if len(action) == 0 {
		return "(command)"
	}

	if len(action) > 80 {
		return action[:77] + "..."
	}

	return action
}

// detectNvimLeaders reads init.lua to find leader and localleader values
func detectNvimLeaders(initPath string) (string, string) {
	data, err := os.ReadFile(initPath)
	if err != nil {
		return "Leader", "LocalLeader"
	}

	leader := "Leader"
	localLeader := "LocalLeader"

	leaderRe := regexp.MustCompile(`vim\.g\.mapleader\s*=\s*['\"]([^'\"]+)['\"]`)
	localLeaderRe := regexp.MustCompile(`vim\.g\.maplocalleader\s*=\s*['\"]([^'\"]+)['\"]`)

	if m := leaderRe.FindStringSubmatch(string(data)); m != nil {
		leader = displayNameForLeaderChar(m[1])
	}
	if m := localLeaderRe.FindStringSubmatch(string(data)); m != nil {
		localLeader = displayNameForLeaderChar(m[1])
	}
	return leader, localLeader
}

// displayNameForLeaderChar converts a single-character leader into a word
func displayNameForLeaderChar(s string) string {
	if s == "" {
		return "Leader"
	}
	switch s {
	case " ":
		return "Space"
	case ",":
		return "Comma"
	case ";":
		return "Semicolon"
	case "/":
		return "Slash"
	case "\\":
		return "Backslash"
	case ".":
		return "Period"
	case "-":
		return "Minus"
	case "_":
		return "Underscore"
	case "=":
		return "Equals"
	default:
		return s
	}
}

// displayMode gives friendly names for Neovim modes
func displayMode(m string) string {
	switch m {
	case "n":
		return "Normal"
	case "v":
		return "Visual"
	case "x":
		return "Visual-Block"
	case "i":
		return "Insert"
	case "o":
		return "Operator"
	case "c":
		return "Command"
	case "t":
		return "Terminal"
	case "s":
		return "Select"
	default:
		return m
	}
}

// formatNeovimKey converts a Neovim key string to a beginner-friendly form
func formatNeovimKey(key, leader, localLeader string) string {
	tokens := tokenizeNeovimKey(key, leader, localLeader)

	// If a single token remains, decide whether to expand or keep as-is
	if len(tokens) == 1 {
		t := prettifyToken(tokens[0])
		if strings.HasPrefix(t, "<") && strings.HasSuffix(t, ">") {
			return t
		}
		if isSpecialWord(t) {
			return t
		}
		if len([]rune(t)) > 1 {
			parts := make([]string, 0, len([]rune(t)))
			for _, r := range t {
				parts = append(parts, string(r))
			}
			return strings.Join(parts, " + ")
		}
		return t
	}

	for i, t := range tokens {
		tokens[i] = prettifyToken(t)
	}
	return strings.Join(tokens, " + ")
}

// tokenizeNeovimKey breaks a key string into meaningful tokens to avoid splitting
// words like Escape/Space while still expanding compact motions like "gd".
func tokenizeNeovimKey(key, leader, localLeader string) []string {
	if leader != "" {
		key = strings.ReplaceAll(key, "<leader>", leaderDisplayToToken(leader))
	}
	if localLeader != "" {
		key = strings.ReplaceAll(key, "<localleader>", leaderDisplayToToken(localLeader))
	}

	// Special replacements remain as angle tokens so we can keep them intact
	specialReplace := map[string]string{
		"<CR>":    "<ENTER>",
		"<cr>":    "<ENTER>",
		"<Esc>":   "<ESC>",
		"<Tab>":   "<TAB>",
		"<BS>":    "<BACKSPACE>",
		"<Space>": "<SPACE>",
		"<space>": "<SPACE>",
	}
	for k, v := range specialReplace {
		key = strings.ReplaceAll(key, k, v)
	}

	tokens := []string{}
	tokenRe := regexp.MustCompile(`<[^>]+>|\S+`)
	modRe := regexp.MustCompile(`^<([CASMcasm])-([A-Za-z0-9]+)>$`)

	for _, tok := range tokenRe.FindAllString(key, -1) {
		// Modifier tokens expand to two tokens: Ctrl + n
		if m := modRe.FindStringSubmatch(tok); m != nil {
			mod := strings.ToUpper(m[1])
			var modName string
			switch mod {
			case "C":
				modName = "Ctrl"
			case "A":
				modName = "Alt"
			case "S":
				modName = "Shift"
			case "M":
				modName = "Meta"
			default:
				modName = m[1]
			}
			tokens = append(tokens, modName, m[2])
			continue
		}
		tokens = append(tokens, tok)
	}

	return tokens
}

// prettifyToken keeps angle tokens or special words readable
func prettifyToken(t string) string {
	if strings.HasPrefix(t, "<") && strings.HasSuffix(t, ">") {
		return strings.ToUpper(t)
	}
	switch strings.ToLower(t) {
	case "space":
		return "Space"
	case "enter":
		return "Enter"
	case "escape":
		return "Escape"
	case "tab":
		return "Tab"
	case "backspace":
		return "Backspace"
	default:
		return t
	}
}

// isSpecialWord prevents splitting of words like Space, Enter, Escape, Tab, Backspace
func isSpecialWord(s string) bool {
	switch strings.ToLower(s) {
	case "space", "enter", "escape", "tab", "backspace":
		return true
	default:
		return false
	}
}

// leaderDisplayToToken converts leader display into an angle-bracket token
func leaderDisplayToToken(s string) string {
	switch s {
	case "Space":
		return "<SPACE>"
	case "Comma":
		return "<,>"
	case "Semicolon":
		return "<;>"
	case "Slash":
		return "</>"
	case "Backslash":
		return "<\\>"
	case "Period":
		return "<.>"
	case "Minus":
		return "<->"
	case "Underscore":
		return "<_>"
	case "Equals":
		return "<=>"
	default:
		// Fallback: wrap in angle brackets, uppercased when single char alpha
		if len([]rune(s)) == 1 {
			return "<" + strings.ToUpper(s) + ">"
		}
		return "<" + s + ">"
	}
}

// ----- Neovim help integration -----

var (
	helpCache      = map[string]string{}
	fetchHelpTopic = fetchNvimHelpTopic
)

// lookupNvimHelpDescription tries to derive a topic from the action and fetch
// the first meaningful line(s) from Neovim help. Results are cached.
func lookupNvimHelpDescription(action string) string {
	action = strings.TrimSpace(action)
	if action == "" || action == "(function)" {
		return ""
	}

	topics := deriveHelpTopics(action)
	for _, t := range topics {
		if t == "" {
			continue
		}
		if desc, ok := helpCache[t]; ok {
			if desc != "" {
				return desc
			}
			continue
		}
		if d := fetchHelpTopic(t); d != "" {
			helpCache[t] = d
			return d
		}
		helpCache[t] = ""
	}
	return ""
}

// deriveHelpTopics extracts likely help topics from a command/action string.
// Examples:
//   - ":w"              => [":w", "write"]
//   - ":Telescope find_files" => [":Telescope", "telescope"]
//   - "gitsigns.stage_hunk"   => ["gitsigns", "gitsigns.nvim"]
func deriveHelpTopics(action string) []string {
	var topics []string
	s := strings.TrimSpace(action)

	// If action begins with ':' treat first token as Ex-command topic
	if strings.HasPrefix(s, ":") {
		first := strings.Fields(s)
		if len(first) > 0 {
			cmd := first[0]
			topics = append(topics, cmd)
			// Also lowercase variant without ':' for some docs
			topics = append(topics, strings.TrimPrefix(strings.ToLower(cmd), ":"))
		}
	}

	// If it looks like module.fn, try module docs
	if dot := strings.Index(s, "."); dot > 0 {
		mod := s[:dot]
		topics = append(topics, mod)
		topics = append(topics, mod+".nvim")
		topics = append(topics, mod+".txt")
	}

	// If nothing yet, fall back to first word
	if len(topics) == 0 {
		w := strings.Fields(s)
		if len(w) > 0 {
			topics = append(topics, w[0])
		}
	}
	// Deduplicate while preserving order
	seen := map[string]bool{}
	out := make([]string, 0, len(topics))
	for _, t := range topics {
		if !seen[t] {
			seen[t] = true
			out = append(out, t)
		}
	}
	return out
}

// fetchNvimHelpTopic runs Neovim headless to print the first lines of a help topic.
// Returns a short, single-line description.
func fetchNvimHelpTopic(topic string) string {
	// Construct a headless Neovim command sequence:
	//   :help {topic} | silent only | 1,30p | qall!
	// This prints first ~30 lines of the help buffer to stdout in ex-mode.
	args := []string{"-es", "--headless",
		"+silent helptags ALL",
		"+silent help " + topic,
		"+silent only",
		"+silent 1,30p",
		"+qall!",
	}

	cmd := exec.Command("nvim", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return ""
	}

	return parseHelpOutput(stdout.String())
}

func parseHelpOutput(text string) string {
	// Extract the first non-empty, non-tag line that looks like a summary
	lines := strings.Split(text, "\n")
	for _, ln := range lines {
		l := strings.TrimSpace(ln)
		if l == "" {
			continue
		}
		// Skip header delineations or tag anchors like *telescope*
		if strings.HasPrefix(l, "*") && strings.HasSuffix(l, "*") {
			continue
		}
		// Avoid section headers like "CONTENTS" or all-caps
		if l == strings.ToUpper(l) && len(l) > 3 {
			continue
		}
		// Return a concise line
		// Truncate if very long
		if len(l) > 140 {
			l = l[:137] + "..."
		}
		return l
	}
	return ""
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

// printPlain outputs keybindings as plain text rows
func printPlain(bindings []Keybinding) {
	// Stable ordering: by App, then Category, then Shortcut
	sort.Slice(bindings, func(i, j int) bool {
		if bindings[i].App != bindings[j].App {
			return bindings[i].App < bindings[j].App
		}
		if bindings[i].Category != bindings[j].Category {
			return bindings[i].Category < bindings[j].Category
		}
		return bindings[i].Shortcut < bindings[j].Shortcut
	})

	for _, kb := range bindings {
		fmt.Printf("%s\t%s\t%s\t%s\t%s\n", kb.App, kb.Category, kb.Shortcut, kb.Desc, kb.Action)
	}
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
