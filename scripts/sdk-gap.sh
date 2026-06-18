#!/usr/bin/env bash
#
# sdk-gap.sh — SDK coverage + signature-drift checker.
#
# Source of truth is the generated SDK client in freelo-go. There are NO
# CLI<->SDK name pairs to maintain: the canonical key is the SDK method
# name itself. An SDK operation is "covered" if some command in
# internal/commands/ references FreeloClient.<Op>, OR it is explicitly
# declared in scripts/sdk-coverage-waived.txt with a reason. Anything else
# is a gap and fails `check`.
#
# Subcommands:
#   check      (default) report SDK ops with no command and no waiver. exit 1 if any.
#   drift      diff current SDK method signatures vs committed snapshot. exit 1 on drift.
#   snapshot   (re)write the signature snapshot. run after intentionally adopting SDK changes.
#
# The generated file is located via the go.mod `replace` directive if present,
# else the module cache. Override with FREELO_SDK_GEN=/path/to/freeloapi.gen.go.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

WAIVED="scripts/sdk-coverage-waived.txt"
SNAPSHOT="scripts/sdk-signatures.txt"

locate_gen() {
	if [[ -n "${FREELO_SDK_GEN:-}" ]]; then
		echo "$FREELO_SDK_GEN"; return
	fi
	local repl dir
	repl=$(go list -m -f '{{with .Replace}}{{.Dir}}{{end}}' github.com/freeloio/freelo-go 2>/dev/null || true)
	if [[ -n "$repl" ]]; then
		echo "$repl/freeloapi/freeloapi.gen.go"; return
	fi
	dir=$(go list -m -f '{{.Dir}}' github.com/freeloio/freelo-go 2>/dev/null || true)
	echo "$dir/freeloapi/freeloapi.gen.go"
}

SDK_GEN="$(locate_gen)"
[[ -f "$SDK_GEN" ]] || { echo "error: cannot find SDK gen file: $SDK_GEN" >&2; exit 2; }

# Exported raw client operations, WithBody body-variant duplicates dropped.
sdk_ops() {
	grep -oE 'func \(c \*Client\) [A-Z][A-Za-z0-9]+\(' "$SDK_GEN" \
		| sed -E 's/func \(c \*Client\) //; s/\(//' \
		| grep -vE 'WithBody$' | sort -u
}

# Drift surface: raw method signatures PLUS request-side struct fields
# (JSONBody + Params). Method signatures catch arity/param-type changes;
# struct fields catch added/changed/removed request parameters — the
# "SDK added a param you might want to expose as a flag" case that a
# `go build` stays green on. Response-body drift is intentionally NOT
# tracked: the CLI decodes responses as map[string]any, so new response
# fields neither break the build nor require a command change.
sdk_sigs() {
	# method signatures, normalized (receiver + reqEditors variadic stripped)
	grep -E '^func \(c \*Client\) [A-Z][A-Za-z0-9]+\(' "$SDK_GEN" \
		| sed -E 's/^func \(c \*Client\) /FUNC /; s/, reqEditors \.\.\.RequestEditorFn//; s/ \{$//' \
		| grep -vE '^FUNC [A-Za-z0-9]+WithBody\('
	# request-side struct fields, keyed by type name
	awk '
		/^type [A-Za-z0-9]+(JSONBody|Params) struct \{/ { t=$2; inb=1; next }
		inb && /^\}/ { inb=0; next }
		inb && /`(json|form):/ {
			line=$0; sub(/^[ \t]+/, "", line); sub(/[ \t]+`.*$/, "", line)
			print "FIELD " t " " line
		}
	' "$SDK_GEN"
} # output is sorted by the callers

cli_refs() {
	grep -rhoE 'FreeloClient\.[A-Z][A-Za-z0-9]+' internal/commands/*.go \
		| sed 's/FreeloClient\.//' | grep -vE 'WithBody$' | sort -u
}

waived_list() {
	[[ -f "$WAIVED" ]] && grep -vE '^[[:space:]]*(#|$)' "$WAIVED" | awk '{print $1}' | sort -u || true
}

case "${1:-check}" in
snapshot)
	sdk_sigs | sort -u > "$SNAPSHOT"
	echo "wrote $SNAPSHOT ($(wc -l < "$SNAPSHOT" | tr -d ' ') signatures)"
	;;
drift)
	[[ -f "$SNAPSHOT" ]] || { echo "no snapshot yet — run: $0 snapshot" >&2; exit 2; }
	tmp="$(mktemp)"; sdk_sigs | sort -u > "$tmp"
	if diff -u "$SNAPSHOT" "$tmp" >/dev/null; then
		echo "no SDK signature drift ($(wc -l < "$SNAPSHOT" | tr -d ' ') ops)"
		rm -f "$tmp"
	else
		echo ">> SDK method signatures changed (params/types). Review CLI callers, then:"
		echo "   $0 snapshot   # to adopt"
		echo
		diff -u "$SNAPSHOT" "$tmp" || true
		rm -f "$tmp"
		exit 1
	fi
	;;
check | *)
	covered="$(mktemp)"; { cli_refs; waived_list; } | sort -u > "$covered"
	gap="$(comm -23 <(sdk_ops) "$covered" || true)"
	rm -f "$covered"
	if [[ -z "$gap" ]]; then
		echo "SDK coverage OK — every SDK op is wrapped or explicitly waived."
	else
		echo ">> SDK ops with NO command and NO waiver:"
		echo "$gap" | sed 's/^/  - /'
		echo
		echo "For each: add a command in internal/commands/, or list it in"
		echo "$WAIVED with a reason."
		exit 1
	fi
	;;
esac
