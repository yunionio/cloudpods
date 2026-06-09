#!/usr/bin/env bash
# Legacy entry: pre-select xiaomi provider, then run interactive / env-based test.
# Prefer: bash scripts/test/aiproxy/aiproxy-functional-test.sh

set -euo pipefail

export AIPROXY_FT_PROVIDER="${AIPROXY_FT_PROVIDER:-xiaomi}"
[[ -n "${MIMO_API_KEY:-}" && -z "${AIPROXY_FT_API_KEY:-}" ]] && export AIPROXY_FT_API_KEY="${MIMO_API_KEY}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
exec bash "${SCRIPT_DIR}/aiproxy-functional-test.sh" "$@"
