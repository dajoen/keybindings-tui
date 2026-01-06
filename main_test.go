package main

import (
	"bytes"
	"flag"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

func writeTempFile(t *testing.T, dir, rel, content string) string {
	t.Helper()
	path := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	return path
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	fn()
	_ = w.Close()
	os.Stdout = orig
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	return string(out)
}

func TestParseModifiers(t *testing.T) {
	got := parseModifiers("$mainMod SHIFT CONTROL ALT")
	if got != "Super + Shift + Ctrl + Alt" {
		t.Fatalf("unexpected modifiers: %q", got)
	}
}

func TestHumanizeAction(t *testing.T) {
	if got := humanizeAction(" exec,  \"foo   bar\" "); got != "foo bar" {
		t.Fatalf("unexpected action: %q", got)
	}
	if got := humanizeAction("   "); got != "(command)" {
		t.Fatalf("unexpected action: %q", got)
	}

	long := strings.Repeat("a", 100)
	if got := humanizeAction(long); len(got) != 80 {
		t.Fatalf("expected truncated action, got len=%d", len(got))
	}
}

func TestDisplayNameForLeaderChar(t *testing.T) {
	tests := map[string]string{
		" ":  "Space",
		",":  "Comma",
		";":  "Semicolon",
		"/":  "Slash",
		"\\": "Backslash",
		".":  "Period",
		"-":  "Minus",
		"_":  "Underscore",
		"=":  "Equals",
		"x":  "x",
	}
	for in, want := range tests {
		if got := displayNameForLeaderChar(in); got != want {
			t.Fatalf("leader %q: %q != %q", in, got, want)
		}
	}
}

func TestDisplayMode(t *testing.T) {
	if got := displayMode("n"); got != "Normal" {
		t.Fatalf("unexpected mode: %q", got)
	}
	if got := displayMode("t"); got != "Terminal" {
		t.Fatalf("unexpected mode: %q", got)
	}
	if got := displayMode("v"); got != "Visual" {
		t.Fatalf("unexpected mode: %q", got)
	}
	if got := displayMode("i"); got != "Insert" {
		t.Fatalf("unexpected mode: %q", got)
	}
	if got := displayMode("x"); got != "Visual-Block" {
		t.Fatalf("unexpected mode: %q", got)
	}
	if got := displayMode("o"); got != "Operator" {
		t.Fatalf("unexpected mode: %q", got)
	}
	if got := displayMode("c"); got != "Command" {
		t.Fatalf("unexpected mode: %q", got)
	}
	if got := displayMode("s"); got != "Select" {
		t.Fatalf("unexpected mode: %q", got)
	}
	if got := displayMode("z"); got != "z" {
		t.Fatalf("unexpected mode: %q", got)
	}
}

func TestLeaderDisplayToToken(t *testing.T) {
	if got := leaderDisplayToToken("Space"); got != "<SPACE>" {
		t.Fatalf("unexpected token: %q", got)
	}
	if got := leaderDisplayToToken("Comma"); got != "<,>" {
		t.Fatalf("unexpected token: %q", got)
	}
	if got := leaderDisplayToToken("Semicolon"); got != "<;>" {
		t.Fatalf("unexpected token: %q", got)
	}
	if got := leaderDisplayToToken("Slash"); got != "</>" {
		t.Fatalf("unexpected token: %q", got)
	}
	if got := leaderDisplayToToken("Period"); got != "<.>" {
		t.Fatalf("unexpected token: %q", got)
	}
	if got := leaderDisplayToToken("Minus"); got != "<->" {
		t.Fatalf("unexpected token: %q", got)
	}
	if got := leaderDisplayToToken("Underscore"); got != "<_>" {
		t.Fatalf("unexpected token: %q", got)
	}
	if got := leaderDisplayToToken("Equals"); got != "<=>" {
		t.Fatalf("unexpected token: %q", got)
	}
	if got := leaderDisplayToToken("k"); got != "<K>" {
		t.Fatalf("unexpected token: %q", got)
	}
	if got := leaderDisplayToToken("Leader"); got != "<Leader>" {
		t.Fatalf("unexpected token: %q", got)
	}
}

func TestTokenizeNeovimKey(t *testing.T) {
	tokens := tokenizeNeovimKey("<leader>gd", "Space", "Comma")
	want := []string{"<SPACE>", "gd"}
	if strings.Join(tokens, ",") != strings.Join(want, ",") {
		t.Fatalf("tokens: %v != %v", tokens, want)
	}

	tokens = tokenizeNeovimKey("<C-n>", "Space", "Comma")
	want = []string{"Ctrl", "n"}
	if strings.Join(tokens, ",") != strings.Join(want, ",") {
		t.Fatalf("tokens: %v != %v", tokens, want)
	}

	tokens = tokenizeNeovimKey("<A-k>", "Space", "Comma")
	want = []string{"Alt", "k"}
	if strings.Join(tokens, ",") != strings.Join(want, ",") {
		t.Fatalf("tokens: %v != %v", tokens, want)
	}

	tokens = tokenizeNeovimKey("<S-j>", "Space", "Comma")
	want = []string{"Shift", "j"}
	if strings.Join(tokens, ",") != strings.Join(want, ",") {
		t.Fatalf("tokens: %v != %v", tokens, want)
	}

	tokens = tokenizeNeovimKey("<M-x>", "Space", "Comma")
	want = []string{"Meta", "x"}
	if strings.Join(tokens, ",") != strings.Join(want, ",") {
		t.Fatalf("tokens: %v != %v", tokens, want)
	}

	tokens = tokenizeNeovimKey("<CR><Tab>", "Space", "Comma")
	want = []string{"<ENTER>", "<TAB>"}
	if strings.Join(tokens, ",") != strings.Join(want, ",") {
		t.Fatalf("tokens: %v != %v", tokens, want)
	}

	tokens = tokenizeNeovimKey("<localleader>x", "Space", "Comma")
	want = []string{"<,>", "x"}
	if strings.Join(tokens, ",") != strings.Join(want, ",") {
		t.Fatalf("tokens: %v != %v", tokens, want)
	}
}

func TestFormatNeovimKey(t *testing.T) {
	got := formatNeovimKey("<leader>ff", "Space", "Comma")
	if got != "<SPACE> + ff" {
		t.Fatalf("unexpected format: %q", got)
	}
	got = formatNeovimKey("gd", "Space", "Comma")
	if got != "g + d" {
		t.Fatalf("unexpected format: %q", got)
	}
	got = formatNeovimKey("Space", "Space", "Comma")
	if got != "Space" {
		t.Fatalf("unexpected format: %q", got)
	}
	got = formatNeovimKey("<TAB>", "Space", "Comma")
	if got != "<TAB>" {
		t.Fatalf("unexpected format: %q", got)
	}
}

func TestPrettifyTokenAndSpecialWord(t *testing.T) {
	if got := prettifyToken("<SPACE>"); got != "<SPACE>" {
		t.Fatalf("unexpected token: %q", got)
	}
	if got := prettifyToken("tab"); got != "Tab" {
		t.Fatalf("unexpected token: %q", got)
	}
	if got := prettifyToken("escape"); got != "Escape" {
		t.Fatalf("unexpected token: %q", got)
	}
	if got := prettifyToken("enter"); got != "Enter" {
		t.Fatalf("unexpected token: %q", got)
	}
	if got := prettifyToken("backspace"); got != "Backspace" {
		t.Fatalf("unexpected token: %q", got)
	}
	if !isSpecialWord("Space") {
		t.Fatalf("expected special word")
	}
}

func TestDeriveHelpTopics(t *testing.T) {
	topics := deriveHelpTopics(":w")
	if len(topics) == 0 || topics[0] != ":w" {
		t.Fatalf("unexpected topics: %v", topics)
	}

	topics = deriveHelpTopics("gitsigns.stage_hunk")
	if len(topics) < 2 || topics[0] != "gitsigns" {
		t.Fatalf("unexpected topics: %v", topics)
	}

	topics = deriveHelpTopics("write")
	if len(topics) != 1 || topics[0] != "write" {
		t.Fatalf("unexpected topics: %v", topics)
	}
}

func TestLookupNvimHelpDescription(t *testing.T) {
	origFetch := fetchHelpTopic
	origCache := helpCache
	helpCache = map[string]string{}
	fetchHelpTopic = func(topic string) string {
		if topic == ":w" {
			return "write file"
		}
		return ""
	}
	t.Cleanup(func() {
		fetchHelpTopic = origFetch
		helpCache = origCache
	})

	if got := lookupNvimHelpDescription(":w"); got != "write file" {
		t.Fatalf("unexpected description: %q", got)
	}
	if got := lookupNvimHelpDescription("(function)"); got != "" {
		t.Fatalf("unexpected description: %q", got)
	}
	if got := lookupNvimHelpDescription("nope"); got != "" {
		t.Fatalf("unexpected description: %q", got)
	}
	if got := lookupNvimHelpDescription("nope"); got != "" {
		t.Fatalf("unexpected description: %q", got)
	}

	helpCache["cached"] = "cached desc"
	if got := lookupNvimHelpDescription("cached"); got != "cached desc" {
		t.Fatalf("unexpected description: %q", got)
	}
	helpCache["empty"] = ""
	if got := lookupNvimHelpDescription("empty"); got != "" {
		t.Fatalf("unexpected description: %q", got)
	}
}

func TestParseHelpOutput(t *testing.T) {
	long := strings.Repeat("Line", 60)
	text := strings.Join([]string{
		"",
		"*topic*",
		"CONTENTS",
		long,
	}, "\n")
	desc := parseHelpOutput(text)
	if len(desc) != 140 {
		t.Fatalf("unexpected desc length: %d", len(desc))
	}
	if !strings.HasSuffix(desc, "...") {
		t.Fatalf("expected truncated suffix, got %q", desc)
	}
}

func TestParseHyprland(t *testing.T) {
	dir := t.TempDir()
	conf := `
# comment
bind = $mainMod, RETURN, exec, alacritty
bindl = , XF86AudioRaiseVolume, pamixer -i 5
bindm = $mainMod SHIFT, mouse:272, movewindow
invalid line
`
	path := writeTempFile(t, dir, "hyprland.conf", conf)
	bindings := parseHyprland(path)
	if len(bindings) != 3 {
		t.Fatalf("expected 3 bindings, got %d", len(bindings))
	}
	if bindings[0].Shortcut != "Super + RETURN" {
		t.Fatalf("unexpected shortcut: %q", bindings[0].Shortcut)
	}
	if bindings[1].Category != "Media" {
		t.Fatalf("unexpected category: %q", bindings[1].Category)
	}
}

func TestParseKitty(t *testing.T) {
	dir := t.TempDir()
	conf := `
kitty_mod ctrl+alt
map kitty_mod+k next_tab
map ctrl+shift+l previous_tab
`
	path := writeTempFile(t, dir, "kitty.conf", conf)
	bindings := parseKitty(path)
	if len(bindings) != 2 {
		t.Fatalf("expected 2 bindings, got %d", len(bindings))
	}
	if bindings[0].Shortcut != "Ctrl + Alt + K" {
		t.Fatalf("unexpected shortcut: %q", bindings[0].Shortcut)
	}
}

func TestParseKittyDefaultMod(t *testing.T) {
	dir := t.TempDir()
	conf := `
map kitty_mod+k next_tab
`
	path := writeTempFile(t, dir, "kitty.conf", conf)
	bindings := parseKitty(path)
	if len(bindings) != 1 {
		t.Fatalf("expected 1 binding, got %d", len(bindings))
	}
	if bindings[0].Shortcut != "Ctrl + Shift + K" {
		t.Fatalf("unexpected shortcut: %q", bindings[0].Shortcut)
	}
}

func TestDetectNvimLeaders(t *testing.T) {
	dir := t.TempDir()
	initLua := `vim.g.mapleader = " "
vim.g.maplocalleader = ","`
	path := writeTempFile(t, dir, "init.lua", initLua)
	leader, localLeader := detectNvimLeaders(path)
	if leader != "Space" || localLeader != "Comma" {
		t.Fatalf("unexpected leaders: %q %q", leader, localLeader)
	}
}

func TestDetectNvimLeadersMissing(t *testing.T) {
	leader, localLeader := detectNvimLeaders(filepath.Join(t.TempDir(), "missing.lua"))
	if leader != "Leader" || localLeader != "LocalLeader" {
		t.Fatalf("unexpected leaders: %q %q", leader, localLeader)
	}
}

func TestParseNeovimFile(t *testing.T) {
	origFetch := fetchHelpTopic
	origCache := helpCache
	helpCache = map[string]string{}
	fetchHelpTopic = func(topic string) string {
		if topic == "gT" {
			return "previous tab"
		}
		return ""
	}
	t.Cleanup(func() {
		fetchHelpTopic = origFetch
		helpCache = origCache
	})

	dir := t.TempDir()
	content := `
vim.keymap.set('n', '<leader>ff', ':Telescope find_files', { desc = 'Find files' })
vim.keymap.set('n', 'gr', 'gT')
map('n', '<leader>h', ':help', { desc = 'Help' })
`
	path := writeTempFile(t, dir, "init.lua", content)
	bindings := parseNeovimFile(path, "Core", "Space", "Comma")
	if len(bindings) != 3 {
		t.Fatalf("expected 3 bindings, got %d", len(bindings))
	}
	if bindings[0].Action != "(function)" {
		t.Fatalf("expected function action, got %q", bindings[0].Action)
	}
	if bindings[1].Desc != "previous tab" {
		t.Fatalf("unexpected desc: %q", bindings[1].Desc)
	}
}

func TestParseNeovimFileHumanized(t *testing.T) {
	dir := t.TempDir()
	content := `
vim.keymap.set('n', 'gx', ':!echo test<CR>')
`
	path := writeTempFile(t, dir, "init.lua", content)
	bindings := parseNeovimFile(path, "Core", "Space", "Comma")
	if len(bindings) != 1 {
		t.Fatalf("expected 1 binding, got %d", len(bindings))
	}
	if !strings.Contains(bindings[0].Desc, "echo test") {
		t.Fatalf("unexpected desc: %q", bindings[0].Desc)
	}
}

func TestParseNeovimAll(t *testing.T) {
	dir := t.TempDir()
	writeTempFile(t, dir, "init.lua", `vim.g.mapleader = " "`)
	writeTempFile(t, dir, "lua/plugins/test.lua", `vim.keymap.set('n', 'gd', 'gT')`)
	writeTempFile(t, dir, "lua/config/keys.lua", `vim.keymap.set('n', 'gD', 'gT')`)
	bindings := parseNeovimAll(dir)
	if len(bindings) == 0 {
		t.Fatalf("expected bindings")
	}
	found := false
	for _, b := range bindings {
		if b.Category == "Plugin: Test" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected plugin category binding")
	}
	found = false
	for _, b := range bindings {
		if b.Category == "Keys" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected non-plugin category binding")
	}
}

func TestParseNeovimAllMissingDirs(t *testing.T) {
	dir := t.TempDir()
	writeTempFile(t, dir, "init.lua", "vim.keymap.set('n', 'gd', 'gT')")
	bindings := parseNeovimAll(dir)
	if len(bindings) == 0 {
		t.Fatalf("expected bindings from init.lua")
	}
}

func TestCategorizeBinding(t *testing.T) {
	if got := categorizeBinding("PRINT", "hyprshot"); got != "Screenshots" {
		t.Fatalf("unexpected category: %q", got)
	}
	if got := categorizeBinding("x", "workspace"); got != "Workspaces" {
		t.Fatalf("unexpected category: %q", got)
	}
	if got := categorizeBinding("x", "togglefloating"); got != "Windows" {
		t.Fatalf("unexpected category: %q", got)
	}
	if got := categorizeBinding("x", "unknown"); got != "Other" {
		t.Fatalf("unexpected category: %q", got)
	}
}

func TestContains(t *testing.T) {
	if !contains([]string{"a", "b"}, "b") {
		t.Fatalf("expected item to be found")
	}
	if contains([]string{"a", "b"}, "c") {
		t.Fatalf("expected item to be missing")
	}
}

func TestBuildSidebarAndFilter(t *testing.T) {
	keys := []Keybinding{
		{App: "Kitty"},
		{App: "Hyprland"},
		{App: "Kitty"},
	}
	apps, items := buildSidebar(keys)
	if len(apps) != 3 || apps[0] != "All" {
		t.Fatalf("unexpected apps: %v", apps)
	}
	if _, ok := items[0].(appItem); !ok {
		t.Fatalf("unexpected item type")
	}
	if items[0].(appItem).Count != 3 {
		t.Fatalf("unexpected all count: %d", items[0].(appItem).Count)
	}

	mainList := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	m := model{
		main:         mainList,
		keybindings:  keys,
		selectedApp:  "Kitty",
		filteredKeys: keys,
	}
	m.applyFilter()
	if len(m.filteredKeys) != 2 {
		t.Fatalf("unexpected filtered keys: %d", len(m.filteredKeys))
	}

	m.selectedApp = "All"
	m.applyFilter()
	if len(m.filteredKeys) != 3 {
		t.Fatalf("unexpected filtered keys: %d", len(m.filteredKeys))
	}
}

func TestPrintPlainAndToListItems(t *testing.T) {
	keys := []Keybinding{
		{App: "B", Category: "Cat", Shortcut: "S", Desc: "D", Action: "A"},
		{App: "A", Category: "Cat", Shortcut: "S", Desc: "D", Action: "A"},
		{App: "A", Category: "Beta", Shortcut: "B", Desc: "D", Action: "A"},
		{App: "A", Category: "Beta", Shortcut: "A", Desc: "D", Action: "A"},
	}
	items := toListItems(keys)
	if len(items) != 4 {
		t.Fatalf("unexpected items: %d", len(items))
	}

	out := captureStdout(t, func() {
		printPlain(keys)
	})
	if !strings.Contains(out, "A\tCat\tS\tD\tA") {
		t.Fatalf("unexpected output: %q", out)
	}
}

func TestFilterValueAndDelegates(t *testing.T) {
	app := appItem{Name: "App", Count: 1}
	if app.FilterValue() != "App" {
		t.Fatalf("unexpected filter value: %q", app.FilterValue())
	}

	kb := Keybinding{Shortcut: "S", Desc: "D", Action: "A"}
	if kb.FilterValue() != "S D A" {
		t.Fatalf("unexpected filter value: %q", kb.FilterValue())
	}

	if newRowDelegate() == nil {
		t.Fatalf("expected delegate")
	}

	d := rowDelegate{}
	if d.Height() == 0 || d.Spacing() != 0 {
		t.Fatalf("unexpected delegate sizing")
	}
	if d.Update(nil, nil) != nil {
		t.Fatalf("expected nil update")
	}

	id := itemDelegate{}
	if id.Height() == 0 || id.Spacing() == 0 {
		t.Fatalf("unexpected item delegate sizing")
	}
	if id.Update(nil, nil) != nil {
		t.Fatalf("expected nil update")
	}
}

func TestModelUpdateAndView(t *testing.T) {
	selector := list.New([]list.Item{
		appItem{Name: "All", Count: 2},
		appItem{Name: "Kitty", Count: 1},
	}, list.NewDefaultDelegate(), 0, 0)
	mainList := list.New([]list.Item{Keybinding{Shortcut: "S", Desc: "D"}}, list.NewDefaultDelegate(), 0, 0)
	m := model{
		selector:    selector,
		main:        mainList,
		keybindings: []Keybinding{{App: "Kitty"}, {App: "Hyprland"}},
		screen:      screenSelect,
	}

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	m2 := updated.(model)
	if m2.width != 100 || m2.height != 40 {
		t.Fatalf("unexpected size: %d x %d", m2.width, m2.height)
	}

	updated, _ = m2.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m3 := updated.(model)
	if m3.screen != screenBrowse {
		t.Fatalf("expected browse screen")
	}

	updated, _ = m3.Update(tea.KeyMsg{Type: tea.KeyDown})
	m4 := updated.(model)
	view := m3.View()
	if !strings.Contains(view, "Keyboard Shortcuts") {
		t.Fatalf("unexpected view: %q", view)
	}

	updated, _ = m4.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m5 := updated.(model)
	if m5.screen != screenSelect {
		t.Fatalf("expected select screen")
	}

	updated, _ = m5.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m6 := updated.(model)
	if !m6.quitting {
		t.Fatalf("expected quitting")
	}
	if m6.View() != "" {
		t.Fatalf("expected empty view when quitting")
	}
}

func TestRowDelegateRender(t *testing.T) {
	var buf bytes.Buffer
	delegate := rowDelegate{}
	listModel := list.New([]list.Item{}, list.NewDefaultDelegate(), 80, 10)
	kb := Keybinding{Shortcut: "Ctrl + K", Desc: "Desc", Action: "Action"}
	delegate.Render(&buf, listModel, 0, kb)
	if buf.Len() == 0 {
		t.Fatalf("expected render output")
	}

	listModel.Select(0)
	buf.Reset()
	delegate.Render(&buf, listModel, 0, kb)
	if buf.Len() == 0 {
		t.Fatalf("expected render output")
	}

	listModel = list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 10)
	buf.Reset()
	delegate.Render(&buf, listModel, 0, kb)
	if buf.Len() == 0 {
		t.Fatalf("expected render output")
	}

	listModel = list.New([]list.Item{}, list.NewDefaultDelegate(), 30, 10)
	buf.Reset()
	delegate.Render(&buf, listModel, 0, kb)
	if buf.Len() == 0 {
		t.Fatalf("expected render output")
	}

	buf.Reset()
	delegate.Render(&buf, listModel, 0, appItem{Name: "App", Count: 1})
	if buf.Len() != 0 {
		t.Fatalf("unexpected render output")
	}
}

func TestItemDelegateRender(t *testing.T) {
	var buf bytes.Buffer
	delegate := itemDelegate{}
	listModel := list.New([]list.Item{}, list.NewDefaultDelegate(), 80, 10)
	kb := Keybinding{App: "App", Category: "Cat", Shortcut: "S", Action: "A"}
	delegate.Render(&buf, listModel, 0, kb)
	if buf.Len() == 0 {
		t.Fatalf("expected render output")
	}

	buf.Reset()
	delegate.Render(&buf, listModel, 0, appItem{Name: "App", Count: 1})
	if buf.Len() != 0 {
		t.Fatalf("unexpected render output")
	}
}

func TestModelInitAndMainUpdatePath(t *testing.T) {
	m := model{}
	if m.Init() != nil {
		t.Fatalf("expected nil init")
	}

	selector := list.New([]list.Item{}, list.NewDefaultDelegate(), 10, 5)
	mainList := list.New([]list.Item{Keybinding{Shortcut: "S", Desc: "D"}}, list.NewDefaultDelegate(), 10, 5)
	m = model{
		selector: selector,
		main:     mainList,
		screen:   screenBrowse,
	}
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m2 := updated.(model)
	if m2.screen != screenBrowse {
		t.Fatalf("expected browse screen")
	}
}

func TestMainPlain(t *testing.T) {
	dir := t.TempDir()
	if err := os.Setenv("HOME", dir); err != nil {
		t.Fatalf("set HOME: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Unsetenv("HOME")
	})

	writeTempFile(t, dir, ".config/hypr/hyprland.conf",
		"bind = $mainMod, RETURN, exec, alacritty\n")
	writeTempFile(t, dir, ".config/kitty/kitty.conf",
		"map kitty_mod+k next_tab\n")
	writeTempFile(t, dir, ".config/nvim/init.lua",
		"vim.keymap.set('n', '<leader>ff', ':Telescope find_files', { desc = 'Find files' })\n")

	origArgs := os.Args
	origFlags := flag.CommandLine
	os.Args = []string{"cmd", "-plain"}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	t.Cleanup(func() {
		os.Args = origArgs
		flag.CommandLine = origFlags
	})

	out := captureStdout(t, func() {
		main()
	})
	if !strings.Contains(out, "Hyprland") || !strings.Contains(out, "Kitty") || !strings.Contains(out, "Neovim") {
		t.Fatalf("unexpected output: %q", out)
	}
}
