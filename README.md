# Freelo CLI

Full access to [Freelo](https://www.freelo.io) from your terminal. `freelo` is the official command-line interface for Freelo. Manage projects, tasks, time tracking, and more from your terminal or through AI agents.

`freelo` works with any AI agent that can run shell commands.

## Quick Start

```bash
# Install
curl -fsSL https://raw.githubusercontent.com/freeloio/freelo-cli/main/install.sh | bash

# Authenticate
freelo auth login

# You're ready!
freelo projects list
freelo tasks list --project 123
freelo search "keyword"
```

## Installation

### curl (macOS / Linux)

```bash
curl -fsSL https://raw.githubusercontent.com/freeloio/freelo-cli/main/install.sh | bash
freelo --version   # verify: prints "freelo-cli vX.Y.Z"
```

### Homebrew (macOS / Linux)

Coming in a follow-up release. For now please use `install.sh`,
`go install`, or the GitHub Releases binary download below.

### Go install (macOS / Linux / Windows)

Requires [Go](https://go.dev/dl/) on your PATH.

```bash
go install github.com/freeloio/freelo-cli/cmd/freelo@latest   # or pin @v1.2.1
freelo --version   # prints the installed tag, e.g. "freelo-cli v1.2.1"
```

The binary lands in `$(go env GOBIN)` (defaults to `~/go/bin`, or
`%USERPROFILE%\go\bin` on Windows). Make sure that directory is on your PATH.

On Windows (PowerShell):

```powershell
go install github.com/freeloio/freelo-cli/cmd/freelo@latest
# add Go's bin dir to PATH if it isn't already, then open a new terminal:
setx PATH "$env:PATH;$env:USERPROFILE\go\bin"
freelo --version
```

### Manual download

Download the latest binary for your platform from [GitHub Releases](https://github.com/freeloio/freelo-cli/releases)
(Windows: `freelo_windows_amd64.exe`). Then run `freelo --version` to confirm.

> **Version reporting.** Released binaries (`install.sh`, GitHub Releases,
> Homebrew) and `go install <module>@<tag>` all report the real version via
> `freelo --version`. A plain `go build` of a local checkout reports a `-dev`
> / pseudo-version instead — use `make build` for a stamped local build.

## Authentication

Get your API key at [https://app.freelo.io/profil/nastaveni](https://app.freelo.io/profil/nastaveni).

```bash
# Interactive login (stores credentials in your OS keyring)
freelo auth login

# Or use environment variables (great for CI/agents)
export FREELO_EMAIL=you@example.com
export FREELO_API_KEY=your-api-key
```

### Credential storage

By default, `freelo auth login` stores credentials in your OS-native secrets store:

- **macOS:** Keychain
- **Windows:** Credential Manager
- **Linux:** Secret Service (gnome-keyring, kwallet, …)

On a headless Linux server or in a Docker container without DBus, set
`FREELO_KEYRING=file` to fall back to a 0600 JSON file at
`~/.config/freelo/credentials.json`.

## Environments

By default, the CLI connects to production (`api.freelo.io`). Use `--dev` to switch to a development environment:

```bash
# Set the dev API URL (get this from your team lead)
export FREELO_DEV_URL=https://your-dev-api-url/v1

# Login to dev environment (separate credentials)
freelo auth login --dev

# All commands with --dev go to the dev API
freelo projects list --dev
freelo tasks list --project 123 --dev
freelo search "keyword" --dev

# Without --dev → production (default)
freelo projects list
```

Production and dev credentials are stored separately — you can be logged into both at the same time. Add `FREELO_DEV_URL` to your `~/.zshrc` or `~/.bashrc` for persistence.

## Commands

| Command | Description |
|---------|-------------|
| `freelo projects` | Manage projects (list, show, create, archive, activate, delete) |
| `freelo tasks` | Manage tasks (list, show, create, edit, finish, activate, move) |
| `freelo subtasks` | Manage subtasks (list, create — Freelo API doesn't support edit/delete) |
| `freelo tasklists` | Manage tasklists (list, show, create — API doesn't support edit/delete) |
| `freelo search` | Search across all entities |
| `freelo comments` | Manage comments (list, create, edit — API doesn't support delete) |
| `freelo labels` | Manage task and project labels |
| `freelo tracking` | Time tracking (start, stop, status) |
| `freelo reports` | Work reports (list, create, edit, delete) |
| `freelo notes` | Project notes (create, show, edit, delete) |
| `freelo files` | Files (list, download, upload) |
| `freelo custom-fields` | Custom fields (types, list, create, rename, values, enums) |
| `freelo templates` | Create from templates (project, tasklist, task) |
| `freelo pinned` | Pinned items (list, create, delete) |
| `freelo users` | Users (me, list) |
| `freelo workers` | Project workers (list, invite, remove) |
| `freelo out-of-office` | Out-of-office (status, enable, disable) |
| `freelo notifications` | Notifications (list, read, unread) |
| `freelo invoices` | Invoices (list, show, mark-invoiced) |
| `freelo events` | Activity log |
| `freelo api` | Raw API access (get, post, put, delete) |

Use `freelo [command] --help` for detailed usage of any command.

## Output Formats

```bash
freelo projects list                # Styled table (terminal) or JSON (piped)
freelo projects list --agent        # Raw JSON — best for AI agents
freelo projects list --json         # JSON with envelope {ok, data, summary, breadcrumbs}
freelo projects list --quiet        # Minimal text
freelo projects list --ids-only     # Just IDs, one per line
freelo projects list --count        # Just the count
```

## AI Agent Integration

The CLI includes an embedded agent skill that teaches AI agents how to use Freelo:

```bash
# Install skill for your AI agent
freelo skill install claude         # Claude Code
freelo skill install codex          # OpenAI Codex
freelo skill install opencode       # OpenCode
freelo skill install all            # All agents
```

After installation, just talk naturally to your AI agent:

> "Show me my tasks in the TOPCORE project with this week's deadline"

The agent will automatically use `freelo` commands to fulfill your request.

## Development

```bash
make build              # build ./freelo with version ldflags
make install            # build + install to ~/bin + register skill with Claude Code
make test               # unit tests (no API required)
make test-integration   # end-to-end tests against real Freelo API (needs .env.freelo-test)
make release-dry        # GoReleaser snapshot — builds all 6 platform archives, no publish
```

The CLI builds on the [`freelo-go`](https://github.com/freeloio/freelo-go)
SDK — a sibling Go module that owns the generated OpenAPI client, transport
(rate limit + retry), and pluggable authentication. A weekly cron in that
repo regenerates the client from the upstream
[Freelo OpenAPI spec](https://api.freelo.io/docs/v1/freelo-api.yaml); the
CLI just bumps its SDK pin to pick up new endpoints.

## License

MIT
