# Freelo CLI

Full access to [Freelo](https://www.freelo.io) from your terminal. `freelo` is the official command-line interface for Freelo. Manage projects, tasks, time tracking, and more from your terminal or through AI agents.

`freelo` works with any AI agent that can run shell commands.

## Quick Start

```bash
# Install
curl -fsSL https://raw.githubusercontent.com/freeloapp/freelo-cli/main/install.sh | bash

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
curl -fsSL https://raw.githubusercontent.com/freeloapp/freelo-cli/main/install.sh | bash
```

### Homebrew (macOS / Linux)

```bash
brew install freeloapp/tap/freelo
```

### Go install

```bash
go install github.com/freeloapp/freelo-cli/cmd/freelo@latest
```

### Manual download

Download the latest binary for your platform from [GitHub Releases](https://github.com/freeloapp/freelo-cli/releases).

## Authentication

Get your API key at [https://app.freelo.io/profil/nastaveni](https://app.freelo.io/profil/nastaveni).

```bash
# Interactive login (stores credentials securely)
freelo auth login

# Or use environment variables (great for CI/agents)
export FREELO_EMAIL=you@example.com
export FREELO_API_KEY=your-api-key
```

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
| `freelo subtasks` | Manage subtasks (list, show, create, finish, activate, delete) |
| `freelo tasklists` | Manage tasklists (list, show, create) |
| `freelo search` | Search across all entities |
| `freelo comments` | Manage comments (list, create, edit, delete) |
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
# Build
make build

# Install locally
make install

# Run tests against live API
make test-live

# Dry-run release (no publish)
make release-dry
```

## License

MIT
