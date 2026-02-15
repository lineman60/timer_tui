# timer_tui

A small terminal-based timer application written in Go. It provides a simple text user interface (TUI) for creating, running, and tracking timers. Timer data is persisted to a local SQLite database (`timer_tui.db`) so you can keep a history of timer sessions across runs.

## Features

- Create, start, pause, and stop named timers from a terminal UI
- Persist timers and history in a local SQLite database (`timer_tui.db`)
- Keyboard-driven, minimal TUI for quick control and visibility
- Implemented in Go for easy cross-platform builds and distribution

## Prerequisites

- Go 1.20+ (or the version indicated in `go.mod`)
- A terminal that supports ANSI escape sequences
- (Optional) `sqlite3` command-line client if you want to inspect the DB directly

## Quick start

Clone the repository, build, and run:

1. Build
```bash
go build -o timer_tui
```

2. Run
```bash
./timer_tui
```

Alternatively, run directly with `go run` during development:

```bash
go run main.go
```

## Files and layout

- `main.go` — application entry point
- `internal/` — internal Go packages and application logic
  - `internal/timer` — core timer logic
  - `internal/timelog` — logic for time logging and persistence
  - `internal/project` — project/repository helpers
  - other internal helpers and models
- `timer_tui` — binary output / runtime artifacts (this may appear after a build)
- `timer_tui.db` — SQLite database created/used by the app to persist timers (placed next to the binary)
- `go.mod`, `go.sum` — Go module metadata and dependency lockfiles

Note: the repository may place the SQLite DB file (`timer_tui.db`) next to the binary. Back this file up if you need to preserve timer history.

## Usage

- Start the application and follow the on-screen TUI instructions. The UI shows available keyboard commands for creating and manipulating timers.
- Timers and timestamps are automatically saved to `timer_tui.db`.

If you need to reset the database while developing or testing, stop the app and remove the `timer_tui.db` file (e.g. `rm timer_tui.db`). The application should recreate or reinitialize the database as needed.

## Development

- Code for core logic lives under `internal/`.
- Follow standard Go project structure and naming.
- Run tests (if present) with:
```bash
go test ./...
```
- Add new features as focused changes and open pull requests.

Recommended workflow:
1. Create a feature branch for work.
2. Keep commits small and focused.
3. Run `go vet`, `golint` (if desired), and `go test` before submitting PRs.

## Debugging & Inspection

- The app persists data to `timer_tui.db`. You can inspect this SQLite DB with any SQLite client, for example:
```bash
sqlite3 timer_tui.db
```
- If you see UI rendering issues, verify your terminal supports ANSI colors and UTF-8.

## Security & Privacy

- The app stores timer names and timestamps in a local SQLite file. The repository does not transmit data externally by default.
- If you add integrations (e.g., remote sync), treat secrets (API keys) carefully and do not hardcode them into the repository.
