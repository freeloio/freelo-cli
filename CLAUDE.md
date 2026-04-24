# CLAUDE.md — Freelo CLI project guide

Guidance for Claude Code (and future contributors) working in this repo.

## What this is

`freelo` is the official command-line interface for [Freelo.io](https://freelo.io).
It lets humans **and AI agents** manage projects, tasks, time tracking, comments,
files, reports, etc. from the terminal.

- **Language:** Go (`1.26.2`, minimal deps — Cobra + `x/term`)
- **Module path:** `github.com/freeloio/freelo-cli` (same org as the skill;
  org was consolidated from `freeloapp` in Phase 1 while the repo was still private)
- **Sibling project:** [`claude-freelo-skill`](https://github.com/freeloio/claude-freelo-skill) —
  the public Claude Code skill (shipped v1.0.0). Users who install both get
  skill-level knowledge of the API plus a typed CLI to execute against it.

## Current roadmap status

Working toward **v1.0.0 public launch** via a 7-phase plan approved 2026-04-24.
Full plan lives in auto-memory (`project_freelo_cli_goal.md`). Short version:

| Phase | Scope | Status |
|---|---|---|
| 1 | Cleanup + foundation (this file, CI, version) | **in progress** |
| 2 | Integration + unit test harness | pending |
| 3 | `oapi-codegen` migration (replace handwritten client) | pending |
| 4 | Rewrite embedded SKILL.md for CLI users | pending |
| 5 | OS keyring, goreleaser dry-run, docs polish | pending |
| 6 | Public launch v1.0.0 + Homebrew tap | pending |
| 7 | OAuth (deferred — see below) | blocked on Freelo backend |

**OAuth is deferred.** `identity.freelo.io` exists but is internal MCP/Claude/Slack
only; Dynamic Client Registration returns a shared `mcp-dcr` client_id that's
semantically wrong for a standalone CLI. Basic Auth remains the production path
until Freelo backend provisions a dedicated `client_id` for the CLI.

## Architecture

```
cmd/freelo/              main.go — thin entrypoint, calls cli.Execute()
internal/
  cli/root.go            root cobra.Command, wires all subcommands (lazy)
  commands/              27 command groups (tasks, projects, comments, ...)
  api/client.go          handwritten HTTP client (Phase 3 will replace this)
  auth/auth.go           Provider interface + BasicAuth impl
  auth/keyring.go        file-based keyring (0600 JSON; Phase 5 → OS keyring)
  config/config.go       layered config: flags > env > local > global > defaults
  output/output.go       envelope pattern — Format{Auto,JSON,Agent,Quiet,IDs,Count}
skills/
  embed.go               go:embed the SKILL.md bundled into the binary
  freelo/SKILL.md        224-line legacy skill (Phase 4 will rewrite from the
                         1613-line public skill, translating curl → freelo syntax)
spec/                    (Phase 3) vendored OpenAPI spec + generated client
```

Key design decisions already made:

- `auth.Provider` interface is **deliberately pluggable** — when OAuth unblocks,
  add `internal/auth/oauth.go` alongside `BasicAuth`, don't rework the interface.
- Output envelope lives in `internal/output/` and is intentionally boring/stable;
  **don't refactor it** during the oapi-codegen migration. It's Phase-3-safe.
- **Local config cannot override `base_url`** (security: prevents credential theft
  via a malicious repo's `.freelo/config.json`). See `internal/config/config.go`.
- **HTTPS enforced** for `base_url`; any non-https URL resets to default.

## Auth flow

1. Env vars win: `FREELO_EMAIL` + `FREELO_API_KEY` (for CI, agents, sandboxed runs).
2. Fallback to keyring: `~/.config/freelo/credentials.json` (or `credentials-dev.json`
   under `--dev`), mode `0600`.
3. `--dev` requires `FREELO_DEV_URL`; dev credentials are isolated from prod.
4. API key lives in Freelo settings → Profile → API key.

## Dev workflow

```bash
make build           # builds ./freelo with ldflags-injected version
make install         # builds + copies to ~/bin + installs bundled skill for Claude Code
make test-live       # smoke-tests ~7 commands against live API in --agent mode
make release-dry     # goreleaser --snapshot --clean
```

Integration tests (Phase 2) use `.env.freelo-test` (gitignored) with:
- `FREELO_EMAIL=info@byurban.cz` (test account)
- `FREELO_API_KEY=…`
- `FREELO_TEST_PROJECT_ID=580898` ("Vyzkoušej si Freelo" — the onboarding project)

Quirk to remember: project 580898 does **not** appear in `/projects` or
`/invited-projects` listings. Must be accessed by ID directly. This is a Freelo
server-side behavior, not a CLI bug — document in tests.

## Gotchas discovered during skill testing (apply to CLI too)

The public skill was hardened across 10 rounds of live testing (~225 API calls).
The full catalog of ~70 gotchas lives in
[`claude-freelo-skill`](https://github.com/freeloio/claude-freelo-skill) SKILL.md.
Highlights the CLI must respect:

- **Rate limit: 25 req/min** — current client enforces ~2.4 s min interval. Phase 3
  wrapper must keep this AND add retry-with-backoff on 429/5xx (3 tries).
- **HTML sanitization** on task/comment bodies — input gets stripped of certain
  tags server-side. Don't rely on round-trip equality in tests.
- **4 pagination shapes**: `data.tasks[]`, `data.items[]`, bare array, and dict
  keyed by name. Each endpoint has its own shape; oapi-codegen gives a typed
  struct per endpoint (fine — don't try to abstract a `Paginated[T]`).
- **Endpoints that 404 on non-paid plans or just don't exist server-side:**
  - `DELETE /tasklist/{id}`, `POST /tasklist/{id}` (edit), `DELETE /comment/{id}`
  - `/task/{id}/public-link`, `/task/{id}/user-time-estimate`
  - Notes `files` field is silently ignored by the server
  - Nested subtasks return `task_id: null` — unusable
- **User-Agent required** by Freelo API. CLI must send
  `User-Agent: FreeloCLI/<version>`; current handwritten client sends just
  `FreeloCLI` — Phase 3 will fix.

## Versioning

Version is injected at build time via ldflags:

```
-X github.com/freeloio/freelo-cli/internal/cli.Version=v1.0.0-dev
```

Default fallback (when built without ldflags) should always reflect the
**next target version with a `-dev` suffix**. Release tags drop the suffix.
Don't let "dev" leak into shipped binaries — `Makefile` `VERSION ?=` default
is what unaware builds pick up.

## Conventions

- **No emoji in source or commit messages** unless the user explicitly asks.
- **Commit style:** imperative subject ≤72 chars, optional body explaining the
  *why*. See recent commits (`git log`) for tone.
- **One concern per commit.** During the oapi-codegen migration, commit
  command-group by command-group (tasks, then projects, …) so bisect works.
- **No new deps without a strong reason.** Current tree is Cobra + `x/term` —
  keep it tight. Phase 3 adds `oapi-codegen`-generated code (no runtime dep on
  the generator); Phase 5 adds `github.com/zalando/go-keyring`.
- **Security mindset by default** — the existing client already thought about
  10 MB response cap, 500-char error truncation, HTTPS enforcement, local config
  base_url lockout. Keep that bar.

## What to **not** do without asking

- Don't rewrite `internal/output/` envelope pattern. It's stable on purpose.
- Don't re-open the OAuth scope unless the user brings new info from the Freelo
  backend team (dedicated client_id provisioned, partner program opened, etc.).
- Don't squash the `skills/freelo/SKILL.md` rewrite into Phase 3 — it's its own
  Phase 4 with a different audience (users *with* CLI vs. users *without*).
- Don't refactor the public skill. It's shipped and deliberately monolithic.
