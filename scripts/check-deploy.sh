#!/usr/bin/env bash
# Post-deploy contract for a depviz server (BRIEF-112).
#
# A green /api/health only proves the process is up. It says nothing about whether
# the embedded Live SPA actually embedded, so this also fetches / and asserts the
# app really shipped. Health-green + /-broken is the failure this exists to catch.
#
# Usage: scripts/check-deploy.sh [base-url]
#   BASE_URL           default https://depviz.1789.tech
#   DEPVIZ_BASIC_AUTH  "user:password" if the instance is gated (see README)
set -euo pipefail

BASE_URL="${1:-${BASE_URL:-https://depviz.1789.tech}}"
BASE_URL="${BASE_URL%/}"
CURL=(curl --silent --show-error --max-time 20)
if [[ -n "${DEPVIZ_BASIC_AUTH:-}" ]]; then
  CURL+=(--user "${DEPVIZ_BASIC_AUTH}")
fi

fail() { printf '  FAIL: %s\n' "$1" >&2; exit 1; }

printf 'checking %s\n' "${BASE_URL}"

# 1. health reports ok:true (never gated, so this works on a gated instance too)
health="$("${CURL[@]}" "${BASE_URL}/api/health")" || fail "/api/health unreachable"
case "${health}" in
  *'"ok":true'*) printf '  ok: /api/health reports ok:true\n' ;;
  *) fail "/api/health did not report ok:true: ${health}" ;;
esac

# 2. / serves the SPA. Separate the status code from the body so a 401 on a gated
#    instance is reported as "credentials needed", not as a missing embed.
root_body="$("${CURL[@]}" --write-out '\n%{http_code}' "${BASE_URL}/")" || fail "/ unreachable"
root_code="${root_body##*$'\n'}"
root_html="${root_body%$'\n'*}"
if [[ "${root_code}" == "401" ]]; then
  fail "/ returned 401 — instance is gated, set DEPVIZ_BASIC_AUTH to check it"
fi
[[ "${root_code}" == "200" ]] || fail "/ returned ${root_code}, want 200"
printf '  ok: / returned 200\n'

# 3. the response is really the Live app, not a stray index or an error page.
#    This is the "embed didn't embed" check: the binary serves live/app via go:embed.
case "${root_html}" in
  *'app.js'*) printf '  ok: / served the embedded Live SPA\n' ;;
  *) fail "/ returned 200 but no app.js reference — the embedded Live FS did not embed" ;;
esac

# 4. the SPA's own assets resolve, which a partial embed would break.
for asset in app.js style.css; do
  code="$("${CURL[@]}" --output /dev/null --write-out '%{http_code}' "${BASE_URL}/${asset}")" || fail "/${asset} unreachable"
  [[ "${code}" == "200" ]] || fail "/${asset} returned ${code}, want 200"
  printf '  ok: /%s returned 200\n' "${asset}"
done

# NOTE: the brief's fourth assertion — "the demo board renders >=1 card" — is not
# checked yet: there is no demo board to render. It is blocked on the access
# decision in BRIEF-112 and lands with that slice.

printf 'post-deploy contract passed\n'
