#!/usr/bin/env bash
# aiproxy interactive functional test: select catalog provider/model, enter API key, run chat + stream.
#
# Usage:
#   source /etc/yunion/rcadmin
#   bash scripts/test/aiproxy/aiproxy-functional-test.sh
#
# Non-interactive (CI):
#   export AIPROXY_FT_PROVIDER=aliyun
#   export AIPROXY_FT_MODEL=qwen-turbo
#   export AIPROXY_FT_API_KEY='...'
#   export AIPROXY_FT_NONINTERACTIVE=1
#   bash scripts/test/aiproxy/aiproxy-functional-test.sh
#
# Legacy env (still supported):
#   DASHSCOPE_API_KEY + AIPROXY_FT_PROVIDER=aliyun
#   MIMO_API_KEY + AIPROXY_FT_PROVIDER=xiaomi

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=scripts/test/aiproxy/aiproxy-functional-test-common.sh
source "${SCRIPT_DIR}/aiproxy-functional-test-common.sh"

CLIMC_OUTPUT_FORMAT="${CLIMC_OUTPUT_FORMAT:-json}"
export CLIMC_OUTPUT_FORMAT

aiproxy_ft_need_cmds

if [[ "${AIPROXY_FT_NONINTERACTIVE:-}" == "1" && -z "${AIPROXY_FT_PROVIDER:-}" ]]; then
	die "AIPROXY_FT_NONINTERACTIVE=1 时需设置 AIPROXY_FT_PROVIDER"
fi

PROVIDER_KEY="$(prompt_select_provider)"
CHAT_MODEL="$(prompt_select_model "$PROVIDER_KEY")"
API_SECRET="$(prompt_api_key "$PROVIDER_KEY")"
CHAT_PROMPT="${AIPROXY_FT_PROMPT:-$(default_prompt_for_provider "$PROVIDER_KEY")}"

RUN_STREAM=1
if prompt_run_stream; then
	RUN_STREAM=1
else
	RUN_STREAM=0
fi

aiproxy_ft_run "$PROVIDER_KEY" "$CHAT_MODEL" "$API_SECRET" "$CHAT_PROMPT" "$RUN_STREAM"
