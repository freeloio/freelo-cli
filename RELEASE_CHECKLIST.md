# Release checklist (internal)

Step-by-step for cutting a public Freelo CLI release. Internal doc — keep
it terse. Long-form launch notes for users live in `RELEASE_NOTES_v<x>.md`.

## Pre-flight

- [ ] CI green on `main` (build + goreleaser snapshot + gosec, both PR-mode
      and post-merge).
- [ ] Working tree clean: `git status` shows nothing.
- [ ] `make test` passes locally.
- [ ] `make test-integration` passes against the live test project (needs
      a fresh API key in `.env.freelo-test` — regenerate at
      <https://app.freelo.io/profil/nastaveni> if 401s appear).
- [ ] `make gen` is a no-op (or the generated diff is reviewed).
- [ ] `goreleaser release --snapshot --clean --skip=publish` succeeds and
      produces six archives + `checksums.txt` in `dist/`.
- [ ] `CHANGELOG.md` Unreleased section is renamed to the new version
      with today's date; the new comparison link is added at the bottom.
- [ ] `RELEASE_NOTES_v<x>.md` is up to date and proofread.

## Sanity-test the snapshot artifacts before tagging

Run from a temp dir, so we exercise the same path the user will:

```bash
SANDBOX=$(mktemp -d)
cp dist/freelo_*_darwin_arm64.tar.gz "$SANDBOX/freelo.tar.gz"
cd "$SANDBOX"
tar xzf freelo.tar.gz
./freelo version          # should print the SNAPSHOT version
./freelo --help | head    # smoke
./freelo skill show | head
cd -
rm -rf "$SANDBOX"
```

If anything in the smoke fails, do NOT proceed — find the regression first.

## Cut the release

```bash
# 1. Tag
git tag -a v<x.y.z> -m "v<x.y.z>"
git push origin v<x.y.z>

# 2. GoReleaser publishes — this is gated by GITHUB_TOKEN + the goreleaser
#    workflow we'll add in Phase 6. Until then, locally:
GITHUB_TOKEN=<gh-pat-with-repo-write> goreleaser release --clean

# 3. The release lands as a draft. Edit on GitHub:
#    - Paste RELEASE_NOTES_v<x>.md as the description
#    - Verify all 6 archives + checksums.txt + freelo.rb are attached
#    - Mark "Set as the latest release"
#    - Publish
```

## Post-release

- [ ] `freelo --version` reflects the new tag from a fresh `brew install`
      (after `brew update`).
- [ ] `curl -fsSL https://raw.githubusercontent.com/freeloio/freelo-cli/main/install.sh | bash`
      installs the new version on a clean macOS / Linux VM.
- [ ] Embedded skill version (`freelo skill show | head -3`) is the new
      release; users on the old version need `freelo skill install <target>`
      to refresh it.
- [ ] Sanity-check the [`update-api-spec.yml`](.github/workflows/update-api-spec.yml)
      cron is still scheduled for the following Monday.

## Smoke matrix (clean machine)

When v1.0.0 ships, run these on a previously-untouched macOS and Linux
to catch path / permission / DBus issues we can't see locally:

| Step | macOS (Apple Silicon) | Linux (Ubuntu desktop) | Linux (headless container) |
|------|------------------------|--------------------------|------------------------------|
| Install via `install.sh` | ✓ expected | ✓ expected | ✓ expected |
| `freelo auth login` | OS Keychain prompt | Secret Service prompt | error → `FREELO_KEYRING=file` |
| `freelo auth status` | authenticated | authenticated | authenticated |
| `freelo projects list` | ok | ok | ok |
| `freelo skill install claude` | writes `~/.claude/skills/freelo/SKILL.md` | same | same |
| `make gen` (in dev clone) | regenerates client | same | same |

If any cell breaks, file an issue and block the release until fixed.
