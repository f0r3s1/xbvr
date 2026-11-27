#!/usr/bin/env bash
# make check: privacy, comment length and translations of pending changes, then lint and tests (QUICK=1 skips those)
set -uo pipefail
cd "$(git rev-parse --show-toplevel)"

fail=0
red() { printf '\033[31m%s\033[0m\n' "$*"; fail=1; }
ok() { printf '\033[32m%s\033[0m\n' "$*"; }

# added lines of tracked changes plus untracked files, as "file:line"
added_lines() {
  git diff HEAD --unified=0 --no-color -- . ':!*.lock' ':!ui/bun.lock' | awk '
    /^\+\+\+ b\// { file = substr($0, 7); next }
    /^@@/ { split($3, a, ","); line = substr(a[1], 2); next }
    /^\+/ { print file ":" line ": " substr($0, 2); line++ }'
  git ls-files --others --exclude-standard -z | xargs -0 -r awk '{ print FILENAME ":" FNR ": " $0 }' 2>/dev/null
}
changes=$(added_lines)

echo "== privacy"
private=".git/info/private-words"
patterns='(/Users/|/home/)[A-Za-z0-9._-]+|[A-Za-z0-9._%+-]+@(gmail|outlook|hotmail|yahoo|icloud|proton)\.|192\.168\.[0-9]+\.[0-9]+|xbvr_[A-Za-z0-9_-]{20,}'
hits=$( {
  grep -E "$patterns" <<<"$changes" | grep -v '^scripts/check.sh:'
  [ -s "$private" ] && grep -iF -f <(grep -v '^\s*$' "$private") <<<"$changes"
} | sort -u)
if [ -n "$hits" ]; then red "personal info in changes:"; echo "$hits" | cut -c1-160; else ok "no personal info"; fi
git ls-files --others --exclude-standard | grep -E '^[^/]+\.png$|^\.playwright-mcp/|\.env$' | while read -r f; do red "must not be committed: $f"; done

echo "== comments"
long=$(awk -F': ' '
  { file = $1; sub(/:[0-9]+$/, "", file); text = substr($0, index($0, ": ") + 2) }
  text ~ /^[ \t]*(\/\/|#[^!]|\/\*|\*|<!--)/ { if (file == prev && ++run == 2) print $1; next }
  { run = 0 }
  { prev = file }' <<<"$(grep -E '\.(go|js|vue|sh|yml)[:]' <<<"$changes")")
if [ -n "$long" ]; then red "comments over one line (keep one line, or none):"; echo "$long"; else ok "comments are one line"; fi

echo "== translations"
python3 -I - <<'EOF' || fail=1
import glob, json, re, sys
loc = 'ui/src/locales'
files = {f: json.load(open(f)) for f in glob.glob(loc + '/*.json')}
base = files[loc + '/en-GB.json']
bad = [f for f, d in files.items() if set(d) != set(base)]
src = ''.join(open(f).read() for f in glob.glob('ui/src/**/*.vue', recursive=True) + glob.glob('ui/src/**/*.js', recursive=True))
used = {next(g for g in m.groups() if g is not None).replace("\\'", "'")
        for m in re.finditer(r"\$t\(\s*'((?:[^'\\]|\\.)*)'|\.t\(\s*'((?:[^'\\]|\\.)*)'|keypath=\"([^\"]+)\"", src)}
missing = sorted(used - set(base))
for f in bad: print(f'\033[31m{f} keys differ from en-GB\033[0m')
for k in missing: print(f'\033[31mnot in en-GB.json: {k}\033[0m')
if bad or missing: sys.exit(1)
print(f'\033[32m{len(base)} keys in {len(files)} languages\033[0m')
EOF

if [ "${QUICK:-}" != 1 ]; then
  echo "== lint"; make -s lint-go lint-js || fail=1
  echo "== tests"; make -s test || fail=1
fi

[ "$fail" = 0 ] && ok "all checks passed" || { red "checks failed"; exit 1; }
