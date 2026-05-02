# Changelog

All notable changes to the Freelo CLI are documented here.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.1.0] — 2026-05-02

### Changed

- **HTTP/auth layer extracted to [`freelo-go`](https://github.com/freeloio/freelo-go).**
  The generated OpenAPI client, transport (rate limit + retry), and
  pluggable auth provider now live in a dedicated SDK module that other
  Go projects can `go get`. The CLI imports it and contributes a
  CLI-specific credential store (OS keyring + 0600 file fallback) via a
  `CredentialsFunc` adapter.
- `make gen` and the weekly spec-refresh workflow moved to the SDK
  repo; this repo no longer owns the generated client or vendored spec.
- New `freelotime.Time` SDK type parses Freelo's timezone-less wire
  format (`"2026-04-24T11:12:38"`) as Europe/Prague and normalizes to
  UTC, so typed `*WithResponse` decoders work out of the box for any
  consumer (including CLI commands that adopt them).
- `freelo api get/post/put/delete` passthrough now routes through
  `app.SDK.Do` (a new SDK helper), preserving Content-Type for JSON
  bodies — previous hand-rolled `RawClientFromResponses` is gone.

### Removed (internal)

- `internal/api/`, `internal/auth/`, `spec/freelo-api.yaml`. Replaced by
  the SDK + `internal/credstore/`.

## [1.0.0] — 2026-04-28

First public release.

### Added

- **OS keyring** for credential storage by default (Keychain on macOS,
  Credential Manager on Windows, Secret Service on Linux desktop). Headless
  Linux / Docker without DBus can opt back into the previous file-based
  store with `FREELO_KEYRING=file`.
- **Generated API client** from the Freelo OpenAPI spec (`oapi-codegen`).
  All 27 command groups now hit the typed client through one shared HTTP
  wrapper that handles Basic Auth, the required `User-Agent: FreeloCLI/<version>`,
  the 25 req/min rate limit, and retry-with-backoff on 429 / 5xx (3 tries,
  honoring `Retry-After`).
- **Weekly automated spec refresh** — a GitHub Actions cron runs `make gen`
  every Monday and opens a pull request whenever the upstream Freelo
  OpenAPI spec changes.
- **Embedded agent skill** rewritten as a CLI-native 759-line `SKILL.md`,
  installable into Claude Code / Codex / OpenCode via `freelo skill install`.
- **Test foundation**: 35 unit tests (auth, output, wrapper, keyring) +
  5 integration tests against a live test project, plus `make test`,
  `make test-integration`, and `make gen` Makefile targets.
- **CI hardening**: `go vet`, `gofmt`, `go test`, `goreleaser --snapshot`
  (validates all six release archives on every PR), and `gosec` security
  scan with results uploaded to GitHub code scanning.

### Changed

- `freelo projects create` now defaults `--currency` to `CZK` (the OpenAPI
  spec marks `currency_iso` as required).
- `freelo notes edit` now requires `--name` (the spec's `EditNote`
  endpoint treats `name` as mandatory; partial bodies were silently
  rejected by the server before).
- `freelo workers remove` body migrated to the spec field name
  (`users_emails`); the legacy `emails` key was being ignored server-side.
- `freelo out-of-office enable` body wraps dates under `out_of_office`
  per spec instead of sending them at the top level.
- `freelo custom-fields enum-create` / `enum-edit` send `value` per
  spec (was `name`); the legacy `--color` flag is now a no-op (the
  server schema has no color field on enum options).
- The OS keyring uses separate service namespaces for prod
  (`freelo-cli`) and dev (`freelo-cli-dev`) so `freelo --dev` cannot
  clobber prod credentials in the same store.
- Module path consolidated to `github.com/freeloio/freelo-cli` to match
  the [`claude-freelo-skill`](https://github.com/freeloio/claude-freelo-skill)
  org.

### Removed

- `freelo subtasks show / finish / activate / delete` — `/subtask/{id}`
  endpoints return 404 from the Freelo API; the CLI no longer advertises
  commands that cannot succeed.
- `freelo comments delete` — `/comment/{id}` only supports POST (edit);
  there is no delete endpoint server-side.

### Security

- All credentials default to the OS keyring; the file fallback uses
  mode 0600.
- gosec scan in CI excluding the auto-generated client; remaining 5
  `#nosec` annotations all carry rule code + reason in-line.
- Generated archives are restricted to the binary + LICENSE +
  README.md; no source bundling.

## [0.1.0] — 2026-04-10

Initial private release. See `git log` for the full history.

[Unreleased]: https://github.com/freeloio/freelo-cli/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/freeloio/freelo-cli/releases/tag/v1.0.0
[0.1.0]: https://github.com/freeloio/freelo-cli/releases/tag/v0.1.0
