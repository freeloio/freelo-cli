# Freelo CLI v1.0.0 — release notes

The first public release of `freelo`, the official command-line interface
for [Freelo.io](https://www.freelo.io).

## Highlights

**Use Freelo from your terminal.** 27 command groups cover everything from
projects and tasks to time tracking, work reports, files, custom fields,
and the audit log. Run `freelo --help` to explore.

**Built for AI agents too.** Every command supports `--agent` for clean,
parseable JSON. The CLI ships an embedded agent skill — install it once
with `freelo skill install claude` (or `codex` / `opencode`) and Claude
Code automatically uses `freelo` whenever you mention Freelo in
conversation.

**Generated, not hand-rolled.** The API client is generated from Freelo's
OpenAPI spec by [oapi-codegen](https://github.com/oapi-codegen/oapi-codegen),
wrapped with a small middleware that handles Basic Auth, the required
`User-Agent` header, the 25 req/min rate limit, and retry-with-backoff
on transient failures. A weekly GitHub Actions cron refreshes the spec
and opens a pull request whenever Freelo changes the API — new endpoints
reach the CLI on autopilot.

**OS-native credential storage.** Logins go to your platform's keyring
(Keychain / Credential Manager / Secret Service) by default. Set
`FREELO_KEYRING=file` if you're on a headless server.

## Install

### Homebrew (macOS / Linux)

```bash
brew install freeloio/tap/freelo
```

### Curl install script

```bash
curl -fsSL https://raw.githubusercontent.com/freeloio/freelo-cli/main/install.sh | bash
```

### Manual download

Pre-built binaries for darwin/linux/windows × amd64/arm64 are attached to
this release. Verify against `checksums.txt`.

### Go install

```bash
go install github.com/freeloio/freelo-cli/cmd/freelo@latest
```

## Quick start

```bash
freelo auth login                                 # store credentials
freelo projects list                              # show your projects
freelo tasks list --project <id>                  # tasks in one project
freelo skill install claude                       # teach Claude Code about freelo
```

For agents and scripts:

```bash
freelo tasks list --project <id> --agent | jq '.[] | {id, name}'
```

## What's in 27 command groups

`projects · tasklists · tasks · subtasks · comments · search · users ·
workers · labels · notes · tracking · reports · files · custom-fields ·
templates · pinned · notifications · events · out-of-office · invoices ·
auth · api · skill · version`

Each subcommand in the catalog has `--help` listing flags and shapes. The
embedded skill (`freelo skill show`) documents server-side gotchas (color
whitelist, HTML sanitization, four pagination shapes, etc.) that even the
typed client can't paper over.

## Known limitations

A handful of operations the API doesn't actually support are intentionally
absent from the CLI surface — listing them so you don't waste time
looking:

- No `subtasks edit / finish / activate / delete` — Freelo's API only
  supports listing and creating subtasks.
- No `comments delete` — `POST /comment/{id}` (edit) is the only mutation
  endpoint on individual comments.
- No `tasklists edit / delete` — same reason.

For anything not covered, the `freelo api get/post/put/delete <path>`
escape hatch goes through the same wrapper (auth, retry, rate limit) and
talks to any path you specify.

## Auth quirk to know

Two Freelo accounts? The OS keyring uses separate namespaces — `freelo-cli`
for production and `freelo-cli-dev` (with `--dev`) — so they don't
collide. If you upgrade from a pre-1.0 file-based install, run
`freelo auth login` once to migrate, or set `FREELO_KEYRING=file` to keep
the old behavior.

## Thanks

This is the second public Freelo project after [`claude-freelo-skill`](https://github.com/freeloio/claude-freelo-skill).
Together they let you and AI agents on your machine drive Freelo
end-to-end with zero clicks. We hope you enjoy.

— The Freelo team
