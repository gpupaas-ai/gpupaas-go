#!/usr/bin/env bash
# Preflight for changing gpupaas-go. Prints OK / NOTE / MISSING lines.
# Exit 1 if something required is missing. Safe to run any time: it only reads.
# Works from any directory: it finds its own repo from the script's location.
cd "$(dirname "$0")/../../.." 2>/dev/null && git rev-parse --git-dir >/dev/null 2>&1 || { echo "MISSING  cannot find the repo root from this script (expected .claude/skills/<name>/preflight.sh)"; exit 2; }

bad=0
ok()   { echo "OK       $1"; }
note() { echo "NOTE     $1"; }
miss() { echo "MISSING  $1"; bad=1; }

if command -v go >/dev/null 2>&1; then ok "go $(go version | awk '{print $3}')"; else miss "go"; fi

if [ -d ../terraform-provider-gpupaas ]; then
  ok "provider checkout ../terraform-provider-gpupaas (used to prove it still builds against your change)"
else
  note "provider not checked out beside this repo: you cannot check that the provider still builds. Clone gpupaas-ai/terraform-provider-gpupaas next to this repo."
fi

branch="$(git branch --show-current)"
[ -n "$branch" ] || branch="(detached HEAD)"
case "$branch" in
  main|master|release/*) note "on branch $branch: work on a feature branch" ;;
  *) ok "branch $branch" ;;
esac

if [ -n "$(git status --porcelain)" ]; then
  note "uncommitted changes present:"
  git status --short | head -5 | sed 's/^/           /'
else
  ok "working tree clean"
fi

if go build ./... >/dev/null 2>&1; then
  ok "go build ./... passes"
else
  miss "go build ./... fails: fix the build before starting"
fi

echo
if [ "$bad" = 0 ]; then echo "PREFLIGHT PASS"; else echo "PREFLIGHT FAIL"; fi
exit "$bad"
