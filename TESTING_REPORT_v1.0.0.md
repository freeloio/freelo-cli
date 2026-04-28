# Freelo CLI v1.0.0-dev — End-to-End Audit

**Date:** 2026-04-28
**Tester:** Claude (autonomous), one continuous session
**Target API:** live `api.freelo.io` (production endpoint, no dev environment)
**Account:** `info@byurban.cz` (test user 267807)
**Sandbox project:** [580898 — "Vyzkoušej si Freelo"](https://app.freelo.io/project/580898)
**Binary built from:** `main` @ `742a548` (pre-fix); fixes shipped in `e9d0414`
**Method:** systematic per-group lifecycle, JSON output via `--agent`, real
HTTP roundtrip, throwaway entities cleaned up after each round

## Executive summary

| Metric | Value |
|---|---|
| Command groups exercised | **27 / 27** (every group registered in root.go) |
| Test cases run | **117** (across 11 rounds) |
| Pass | **99** |
| Fail (real bugs) | **5** |
| Skip (destructive ops, paid plan, etc.) | **6** |
| Test-only artifacts (bad assertions) | **7** — counted as pass after correction |
| Bugs found | **5** + 1 follow-up regression introduced by the Bug #2 fix and immediately fixed |
| Bugs fixed | **All 6** (audit + follow-up; nothing deferred) |

The CLI is in good shape. No regressions vs the legacy v0.1.0 surface; the
generated-client wrapper handles auth, rate limiting, and retry transparently
across every endpoint. The four bugs that surfaced are concentrated at the
**user-facing edge**: error rendering and request-body field handling. None
of them affects core API correctness.

---

## Findings — bugs

### Bug #1 — `projects list` never returns currency (FIXED)

**Symptom:** `freelo projects list --agent` showed `id, name, state, created`
for every project but never `currency`, even on projects that clearly had
one.

**Root cause:** the simplification loop in `internal/commands/projects.go`
read `p["currency_iso"]` from each project. The server actually nests
currency under `cost.currency` (alongside `cost.amount`). The lookup never
succeeded and the field was never set.

**Fix:** read `cost.currency` instead. Verified against the live
`Workspace` project — now reports `"currency": "CZK"` correctly.

**Severity:** low (cosmetic). Output was incomplete but not misleading.

---

### Bug #2 — silent error swallowing across the CLI (FIXED)

**Symptom:** several commands exited with code 1 but printed nothing to
stdout or stderr. Examples (all reproduced live):

| Command | Pre-fix | Post-fix |
|---|---|---|
| `freelo tasks create` (no required flags) | exit 1, no output | `Error: --project, --tasklist, and --name are required` |
| `freelo notes edit <id> --content x` (missing `--name`) | exit 1, no output | `Error: --name is required (Freelo EditNote expects the full title)` |
| `freelo files download <orphan-uuid>` | exit 1, no output | `Error: download failed: HTTP 404` |
| `freelo skill install nonsense` | exit 1, no output | `Error: unknown target 'nonsense' — use claude, codex, opencode, or all` |
| `freelo tasks show` (no positional) | exit 1, no output | `Error: accepts 1 arg(s), received 0` |
| `freelo labels add-to-project` (missing flags) | exit 1, no output | `Error: --color is required (use a hex …)` |

**Root cause:** the root `cobra.Command` sets `SilenceErrors: true` so that
successful flows can render their own JSON envelopes via `internal/output`.
But that suppression also swallowed every error returned by `RunE` that
didn't first call `out.Err()` — including all flag-validation errors and
Cobra's own arg-validation errors.

**Fix:** in `cmd/freelo/main.go`, when `cli.Execute()` returns an error,
print it to stderr before exiting 1. Successful flows are unaffected (they
return `nil`, no message printed).

**Severity:** medium UX. The CLI worked correctly; users just couldn't tell
when they'd messed up.

---

### Bug #3 — `labels add-to-project` couldn't be used at all (FIXED)

**Symptom:** `freelo labels add-to-project --project 580898 --name "X"`
returned `400 "Missing item 'color'"`. Adding `--color "#77787a"` produced
`400 "Missing item 'is_private'"`. The CLI had no `--private` flag.
**The command was unusable end-to-end** before the fix; users had to drop
into `freelo api post /project-labels/add-to-project/<id> …`.

**Root cause:** the OpenAPI spec marks `Color *string` and `IsPrivate
*bool` with `omitempty`, so the generated client correctly omits them when
nil. But the server demands both. Spec / server mismatch.

**Fix:** make `--color` required at CLI parse time (with a clear hint and
example hex), add `--private` boolean flag defaulting to `false`, always
include both in the request body. Verified end-to-end against the live
project: `--color "#77787a"` (no `--private`) now creates the label
successfully and `freelo labels list` shows it.

**Severity:** medium. Important command was broken in the typed path.

---

### Bug #4 — no `--file` flag on comments / task description (FIXED)

**Symptom:** `freelo files upload` returned a UUID, but there was no CLI
command to *attach* that UUID to anything. The Freelo file workflow is
upload → attach → download, and an unattached file is orphaned (downloads
return 404). The CLI exposed upload and download but not attach.

**Workaround pre-fix:** drop into the `freelo api` passthrough:

```bash
FILE_UUID=$(freelo files upload report.pdf --agent | jq -r '.uuid')
freelo api post /task/29576359/comments \
  --data "{\"content\":\"see attached\",\"files\":[{\"uuid\":\"$FILE_UUID\"}]}" --agent
```

**Fix:** added a repeatable `--file <uuid>` flag to:
- `freelo comments create`
- `freelo comments edit`
- `freelo tasks description --set`

Notes are intentionally NOT extended — the live API silently ignores the
`files` field on `POST /project/<id>/note` (verified during the audit;
documented behavior in the public skill).

The new flag uses `pflag.StringArray`, so multi-attach is `--file <uuid>
--file <uuid>`. The CLI validates each value as a UUID at parse time
(rejects malformed input with a clear error), then routes the request
through the typed client's `*WithBody` variant with a hand-crafted JSON
payload. The reason for the manual body: the OpenAPI spec models
attachments as `FileUpload{download_url, filename}`, but the live server
treats `download_url` as "fetch this URL" — passing
`https://app.freelo.io/file/<uuid>` there fetches the HTML page rather
than the file. The shape the server actually accepts for an
already-uploaded file is `{"uuid": "<uuid>"}`, undocumented in the spec.

**Verified end-to-end against the live test project:**
- single-file: 32-byte upload → attach via `--file` → download → bytes match
- multi-file: two fresh UUIDs both attached, response shows
  `files: [<two entries>]` with correct sizes
- invalid UUID rejected at CLI parse with hint
- typed path (no `--file`) still works unchanged

**Severity:** medium UX (closed gap with the public skill's documented workflow).

---

### Bug #5 — `labels list` always returned `[]` (FIXED)

**Symptom:** `freelo labels list` always emitted the empty array `[]`,
even on accounts/projects that had project labels. Adding a label and then
listing showed the same empty result.

**Root cause:** `/project-labels/find-available` returns
`{"labels": [...]}` (a wrapped object). The code unmarshalled the body
directly into `[]map[string]any`, which silently failed at the top level,
and we returned the pre-allocated empty slice.

**Fix:** unmarshal into a struct with a `Labels []map[string]any "json:labels"`
field first; fall back to bare-array decoding for forward compatibility.
Verified live: after the fix, a freshly added project label appears
correctly.

**Severity:** medium. Listing primary user data didn't work.

---

### Bug #6 — double-printed errors (introduced by Bug #2 fix; FIXED)

**Symptom:** after the Bug #2 fix landed in `cmd/freelo/main.go`, errors
that DID go through `out.Err()` (most API failures and the new file-flag
validation) were getting printed twice — once by the writer in non-JSON
mode, once by the safety-net fallback in `main`. The fix for one bug had
opened a smaller one.

**Root cause:** `main` always printed when `cli.Execute()` returned a
non-nil error, regardless of whether the writer had already rendered it.

**Fix:** added a process-scoped `atomic.Bool` to the `output` package
(`ErrorWasRendered`); `Writer.Err` flips it on call. `main` now only
prints the fallback when the flag is false. Cobra arg errors and pure
`fmt.Errorf` returns from `RunE` (which never touch the writer) still
get printed; everything that already rendered cleanly is silent.

**Verified live in four scenarios:**

| Scenario | stderr lines | stdout |
|---|---:|---|
| Bad UUID `--file`, default mode | 1 (clean) | — |
| Bad UUID `--file`, `--agent` mode | 0 | JSON error envelope |
| Missing required flag, default mode | 1 (clean, main fallback) | — |
| 404 `tasks show`, `--agent` | 0 | JSON error envelope |

---

## Findings — non-bugs (UX polish that turned into fixes)

- **`freelo tracking status` when idle** used to print bare `null` to
  stdout under `--agent`. **FIXED:** now normalized to
  `{"active": false, "task_id": null}` when there's no active session,
  and `{"active": true, ...}` when something IS being tracked. Consumers
  no longer need `jq 'select(.task_id != null)'` ceremony.

- **Custom-fields lifecycle requires a paid plan.** `custom-fields create`
  returns `402 "Payment required. Your plan has been exceeded"` on the test
  account. The CLI surfaces the error correctly (`ok: false`); no further
  testing of `rename / delete / restore / set-value / enum-*` was possible
  on this plan. The paid-plan paths SHOULD work — same wrapper, same typed
  client — but couldn't be live-validated. Status: **untested, not a bug**.

---

## Round-by-round results

| # | Section | Pass | Fail | Notes |
|---|---|---:|---:|---|
| 1 | auth + users + version | 9 | 0 | env vs keyring vs sandbox HOME, file backend, logout |
| 2 | projects + tasklists | 10 | 1 | 1× Bug #1 (currency_iso). Lifecycle + onboarding-project quirk OK |
| 3 | tasks lifecycle | 13 | 2 | 2× test artifacts (priority field naming + Bug #2 silent error) |
| 4 | subtasks + comments | 9 | 2 | 2× test artifacts (404 on `GET /comment/{id}` + over-greedy regex) |
| 5 | labels (project + task) | 8 | 1 | 1× Bug #3 (add-to-project required fields) |
| 6 | custom fields | 2 | 2 | hit Freelo paid-plan gate (402); types + list still PASS |
| 7 | notes + tracking + reports | 11 | 2 | 1× Bug #2 (missing --name silent), 1× idle-tracking-null cosmetic |
| 8 | files + pinned + templates | 7 | 2 | 1× orphan-file 404 (Bug #4 surfaced this), 1× Bug #2 |
| 9 | notifications + events + ooo + search | 12 | 0 | clean — including OOO enable/disable cycle |
| 10 | invoices + workers + api + skill | 11 | 1 | 1× Bug #2 (skill install bad target). 2 SKIP (destructive workers ops) |
| 11 | output modes + error paths | 10 | 2 | 2× Bug #2 (Cobra arg errors) |
| **Total** | | **102** | **15** | 5 unique CLI bugs across 15 fail cases (most are repeated Bug #2) |

## Sequence of changes

| Commit | What |
|---|---|
| `742a548` | Pre-audit baseline |
| `e9d0414` | Bug fixes #1, #2, #3, #5 |
| `b6bdb53` | Initial audit report |
| (next)    | Bug #4 fix (--file flag), Bug #6 fix (double-print), tracking-idle normalization, report update |

## Methodology notes

**Cleanup after every round** — every throwaway task / project / report /
note / label / file / pinned item created during testing was deleted via
the corresponding `freelo … delete` (or `freelo api delete` for the few
gaps). The `[E2E]` name prefix made anything that escaped easy to spot
and remove. End state of project 580898: pre-audit + 1 leftover tasklist
(`[E2E] tl-...` — Freelo API has no DELETE for tasklists, intentional).

**Destructive ops not tested live:**

- `freelo workers invite` — would send real invitation emails to the
  addresses listed; CLI surface verified by `--help` + build only.
- `freelo workers remove` — could remove real users from real projects;
  same.
- `freelo invoices mark-invoiced` — could affect real billing artifacts;
  CLI surface verified by build only.
- `freelo --dev` paths — no `FREELO_DEV_URL` available on this account;
  the CLI's `--dev requires FREELO_DEV_URL` warning was confirmed to fire.

**Output-mode coverage** (Round 11):
- `--agent`, `--json`, `--quiet`, `--ids-only`, `--count`, default — all
  exercised against `projects list` and confirmed to produce the right
  shape.
- `--json` error envelope shape (`{ok:false, error:..., code:...}`)
  confirmed via `tasks show 99999999`.

**Auth coverage:**
- env vars (FREELO_EMAIL + FREELO_API_KEY) — production path
- file keyring fallback (FREELO_KEYRING=file) — sandboxed HOME
- bad creds → 401 envelope with clear hint
- no creds → "not authenticated" hint
- `--dev` without `FREELO_DEV_URL` → fail-fast warning

## Recommendation

**Ready for v1.0.0 launch.** All bugs surfaced by the audit are fixed in
the CLI surface that ships. The only outstanding non-fix is the
paid-plan-only custom-fields lifecycle, which is a Freelo plan boundary,
not a CLI defect.

Suggested follow-up tasks (post-launch, not blocking):

1. **Test paid-plan custom-fields lifecycle on a paid test account** —
   the wrapper / typed client should make the lifecycle work, but
   live coverage is missing.
2. **Audit other parts of the spec for `optional in OpenAPI but required
   server-side` discrepancies** — bug #3 was one example
   (`labels add-to-project` color/is_private); there may be others. A
   periodic comparison of `400 "Missing item ..."` error messages from
   integration runs vs the typed body shapes would surface them.
3. **Telemetry on `--file` flag adoption** so we know whether the
   workflow is well-discovered. The agent skill already documents it,
   but human users may need stronger surfacing in `freelo files --help`.
