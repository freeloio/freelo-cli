# Changelog

All notable changes to the Freelo CLI are documented here.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.3.0] — 2026-07-13

### Added

- `freelo labels colors` — lists the accepted task-label color palette from
  the public `GET /api/v1/task-label-colors` endpoint. Mirrors `labels list`:
  the `{"colors":[...]}` response is normalized to a flat slice, so `--agent`
  and `jq` consumers get a stable shape.

### Changed

- Bumped `freelo-go` `v0.2.0` → `v0.2.1` for the generated `GetTaskLabelColors`
  method.

### Fixed

- `labels create --color` help no longer suggests `#ff0000`, which is not in
  the palette and is rejected with HTTP 400. It now shows `#e9483a` (red) and
  points to `freelo labels colors` for the valid values.

## [1.2.1] — 2026-06-16

### Added

- `freelo --version` / `-v` now work. Previously only the `freelo version`
  subcommand existed and the flag errored with `unknown flag: --version`.

### Fixed

- `go install github.com/freeloio/freelo-cli/cmd/freelo@vX.Y.Z` (and `@latest`)
  now reports the installed tag instead of the `-dev` fallback. `go install`
  cannot pass ldflags, so the version is recovered from the module build info
  (`runtime/debug.ReadBuildInfo`) when ldflags were absent. Priority is
  ldflags > build-info module version > `-dev` fallback, so `make build` and
  goreleaser artifacts are unaffected. The `--version` flag, the `version`
  subcommand, and the `FreeloCLI/<version>` User-Agent all report the same
  resolved value.

## [1.2.0] — 2026-05-15

Pre-production hardening pass. Behavior is unchanged for the happy path
of every command — this release tightens what happens when something
goes wrong (network blip mid-write, schema drift, malformed input).

### Fixed

- **Decode helpers** (`decodeAPIObject`, `consumeAPIAny`, customfields /
  pinned list paths) now surface JSON parse errors instead of silently
  returning empty data on a non-JSON 2xx body. Schema drift used to be
  invisible.
- **TCP connection leak** when the SDK returned both a non-nil response
  and an error: `consumeAPIObject` / `consumeAPIBody` now close
  `resp.Body` on every transport error.
- **SDK init failures** propagate from `PersistentPreRunE` — process
  exits 1 instead of 0 after a setup error.
- **`config.Load` no longer calls `os.Exit`** — `--help` and `version`
  work even when `--dev` is set without `FREELO_DEV_URL`.
- **`tasklists list`** returns a clear error if `project.tasklists`
  isn't the expected array shape (was a silent empty success on
  schema drift).
- **`freelo tasks show abc`** (non-numeric ID) now returns
  `task-id must be a number` instead of silently issuing `GET /task/0`.
  Same parse-and-validate fix across 17 sites — replaces `mustInt`
  with `parseIntArg`.
- **`printCount`** returns `0` for non-array payloads (was misleadingly
  returning `1`, including on error envelopes).
- **`printJSON`** surfaces marshal errors to stderr instead of
  producing empty output.

### Changed

- **Command wiring rewritten.** The lazy-wrapper indirection (24
  `*Lazy` constructors, `wrapLazy`, `patchRunE`, `Use`-string-based
  `findSubCmd` lookup) is gone. Commands now capture a single `*App`
  populated in-place by `PersistentPreRunE`. The old approach also
  had a latent name-collision bug in subcommand lookup.
- **Pagination boilerplate consolidated.** New `setProjectsFilter` /
  `setUsersFilter` / `setPageFilter` helpers replace ~150 lines of
  duplication across 9 commands. `--page` is now 1-indexed with
  validation everywhere; help text uniform.
- **`tracking status`** returns a clean `{active, server}` envelope;
  no longer mutates the server response map in place.
- **Default ldflags-free build version** is now `v1.2.0-dev` (was a
  stale `v1.0.0-dev` in `cli/root.go`).
- **`get_description_failed` → `api_error`** — last outlier among the
  `*_failed` error codes.

### Removed

- **`freelo tasks list --worker` flag** — was parsed and silently
  dropped because `/all-tasks` has no `worker_id` parameter. Use
  `--project + --tasklist` filtering instead.

### Security / durability

- **Atomic credential writes.** `fileKeyring` writes via tempfile +
  chmod 0600 + Sync + Rename. A crash, full disk, or signal mid-write
  can no longer wipe stored credentials.
- **Transactional credential store.** `credstore.Store` rolls back
  the email key if the api_key write fails — never sits in a
  half-saved state.
- **Atomic downloads.** Files write to a `.partial` sibling and
  rename on success; cleanup removes the partial on any error.
  Failed downloads no longer leave truncated files under the final
  name.
- **Streaming uploads.** Multipart body streams through `io.Pipe`
  instead of being held twice in memory. Peak RAM is ~32KB
  regardless of file size — previously ~200MB for the 100MB upload
  ceiling.
- **`google/uuid.Parse`** replaces the hand-rolled UUID regex in
  `validateAndWrapFileUUIDs`; keyring errors are checked via
  `errors.Is`.

### Tests

- `helpers_test.go` — coverage for `parsePaginatedItems` (4 shapes +
  edge cases), `parseIntArg`, `validateAndWrapFileUUIDs`.
- `output_test.go` — updated for the new `--count` semantics.

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

[Unreleased]: https://github.com/freeloio/freelo-cli/compare/v1.3.0...HEAD
[1.3.0]: https://github.com/freeloio/freelo-cli/compare/v1.2.1...v1.3.0
[1.0.0]: https://github.com/freeloio/freelo-cli/releases/tag/v1.0.0
[0.1.0]: https://github.com/freeloio/freelo-cli/releases/tag/v0.1.0
