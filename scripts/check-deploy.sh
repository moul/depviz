#!/usr/bin/env bash
# Post-deploy contract for a depviz server (BRIEF-112).
#
# A green /api/health only proves the process is up. It says nothing about whether
# the landing and embedded app actually shipped, so this checks /, /app/, and the
# authenticated demo-board data path when credentials are provided.
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

# 2. / serves the public landing. It must stay open even when /app/ is gated.
root_body="$("${CURL[@]}" --write-out '\n%{http_code}' "${BASE_URL}/")" || fail "/ unreachable"
root_code="${root_body##*$'\n'}"
root_html="${root_body%$'\n'*}"
[[ "${root_code}" == "200" ]] || fail "/ returned ${root_code}, want 200"
printf '  ok: / returned 200\n'
case "${root_html}" in
  *'DepViz'*) printf '  ok: / served the landing page\n' ;;
  *) fail "/ returned 200 but does not look like the DepViz landing page" ;;
esac

# 3. /app/ serves the embedded Live app. On a gated instance, missing
#    DEPVIZ_BASIC_AUTH should fail clearly.
app_body="$("${CURL[@]}" --write-out '\n%{http_code}' "${BASE_URL}/app/")" || fail "/app/ unreachable"
app_code="${app_body##*$'\n'}"
app_html="${app_body%$'\n'*}"
if [[ "${app_code}" == "401" ]]; then
  fail "/app/ returned 401 — instance is gated, set DEPVIZ_BASIC_AUTH to check it"
fi
[[ "${app_code}" == "200" ]] || fail "/app/ returned ${app_code}, want 200"
case "${app_html}" in
  *'app.js'*) printf '  ok: /app/ served the embedded Live SPA\n' ;;
  *) fail "/app/ returned 200 but no app.js reference — the embedded Live FS did not embed" ;;
esac

# 4. the SPA's own assets resolve, which a partial embed would break.
for asset in app.js style.css; do
  code="$("${CURL[@]}" --output /dev/null --write-out '%{http_code}' "${BASE_URL}/app/${asset}")" || fail "/app/${asset} unreachable"
  [[ "${code}" == "200" ]] || fail "/app/${asset} returned ${code}, want 200"
  printf '  ok: /app/%s returned 200\n' "${asset}"
done

# 5. The private board endpoint must not degrade to a public 200-empty response.
demo_code="$("${CURL[@]}" --output /tmp/depviz-demo-board.json --write-out '%{http_code}' "${BASE_URL}/api/demo-board")" || fail "/api/demo-board unreachable"
if [[ -n "${DEPVIZ_BASIC_AUTH:-}" ]]; then
  [[ "${demo_code}" == "200" ]] || fail "/api/demo-board returned ${demo_code}, want 200 with credentials"
  case "$(cat /tmp/depviz-demo-board.json)" in
    *'"brief_type":"board-status"'*|*'"brief_type": "board-status"'*) printf '  ok: authenticated demo board returned board-status data\n' ;;
    *) fail "/api/demo-board returned 200 but no board-status payload" ;;
  esac
else
  [[ "${demo_code}" == "401" || "${demo_code}" == "403" || "${demo_code}" == "404" ]] || fail "/api/demo-board returned ${demo_code}, want 401/403/404 without credentials"
  printf '  ok: anonymous demo board did not return private data (%s)\n' "${demo_code}"
fi

printf 'post-deploy contract passed\n'
