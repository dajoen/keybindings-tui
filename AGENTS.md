# Repository Guidelines

## Project Structure & Module Organization
The codebase is intentionally small and lives at the repo root. `main.go` is the
primary entry point and includes parsing, UI rendering, and helper logic.
Tests live alongside the code in `main_test.go`. Dependency metadata is in
`go.mod` and `go.sum`, and build automation is in `Makefile`. `README.md`
documents usage and installation. Build artifacts are placed in `bin/` (for
example, `bin/keybindings-tui` from `make build`).

## Build, Test, and Development Commands
- `go run .` runs the TUI locally.
- `go run . -plain` prints parsed bindings to stdout for quick inspection.
- `make build` compiles `bin/keybindings-tui`.
- `make install` installs to `~/.local/bin`.
- `make install-go` installs via `go install` to `$GOBIN` (or `~/go/bin`).
- `make clean` removes the compiled binary.
- `make test-local-cache` runs tests using a repo-local Go build cache.
- `make setup` installs the pre-commit hooks.
- `go test ./...` runs all tests.
- If your Go build cache is restricted, use a repo-local cache:
  `GOCACHE=$PWD/.gocache go test ./...`.
- Pre-commit hooks run `gofmt` and `go test` along with basic whitespace checks.

## Coding Style & Naming Conventions
Use standard Go formatting (`gofmt` / `go fmt ./...`) and tabs for indentation.
Follow Go naming conventions: exported identifiers use `CamelCase`, unexported
use `camelCase`, and short, descriptive names are preferred. Keep logic in
small, focused helpers to maintain readability in `main.go`.

## Testing Guidelines
Tests use Go’s built-in `testing` package and should live in `*_test.go` files
in the same package. Prefer table-driven tests for parsing helpers. Avoid
reading real user config files; use temp files or inline fixtures instead.

## Commit & Pull Request Guidelines
Recent history uses short, imperative commit messages (e.g. “Add plain output
mode”, “Improve responsive layout”). Keep subject lines concise and action-led.
Pull requests should include a brief description, manual test steps, and
screenshots or recordings for UI changes when possible.

## Configuration & Runtime Notes
The app reads keybindings from `~/.config/hypr/hyprland.conf`,
`~/.config/kitty/kitty.conf`, and `~/.config/nvim`. For Neovim help enrichment,
`nvim` must be available in `PATH`; if not, descriptions fall back to best-effort
humanized text.
