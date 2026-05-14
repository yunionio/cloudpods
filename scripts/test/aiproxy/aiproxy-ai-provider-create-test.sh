#!/usr/bin/env bash
# Test climc ai-provider-create: create a custom ai_provider and verify with show/list.
#
# Usage:
#   source /etc/yunion/rcadmin
#   bash scripts/test/aiproxy/aiproxy-ai-provider-create-test.sh
#
# Non-interactive:
#   export AIPROXY_PROVIDER_FT_NONINTERACTIVE=1
#   export AIPROXY_PROVIDER_FT_NAME=my-custom-provider
#   export AIPROXY_PROVIDER_FT_PROVIDER_KEY=my-custom-key
#   export AIPROXY_PROVIDER_FT_BASE_URL=https://api.example.com/v1
#   bash scripts/test/aiproxy/aiproxy-ai-provider-create-test.sh
#
# Or pass full config JSON:
#   export AIPROXY_PROVIDER_FT_CONFIG='{"base_url":"https://api.example.com"}'

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=scripts/test/aiproxy/aiproxy-functional-test-common.sh
source "${SCRIPT_DIR}/aiproxy-functional-test-common.sh"

CLIMC_OUTPUT_FORMAT="${CLIMC_OUTPUT_FORMAT:-json}"
export CLIMC_OUTPUT_FORMAT

aiproxy_ft_need_cmds

PROVIDER_RESOURCE_NAME="${AIPROXY_PROVIDER_FT_NAME:-}"
PROVIDER_KEY="${AIPROXY_PROVIDER_FT_PROVIDER_KEY:-}"
BASE_URL="${AIPROXY_PROVIDER_FT_BASE_URL:-}"
CONFIG_JSON="${AIPROXY_PROVIDER_FT_CONFIG:-}"
ENABLED_FLAG="${AIPROXY_PROVIDER_FT_ENABLED:-true}"
DELETE_IF_EXISTS="${AIPROXY_PROVIDER_FT_DELETE_EXISTING:-}"

prompt_line() {
	local prompt="$1" default="${2:-}" varname="$3"
	local value
	if [[ -n "$default" ]]; then
		echo -n "${prompt} [${default}]: " >&2
	else
		echo -n "${prompt}: " >&2
	fi
	if [[ ! -t 0 ]]; then
		value="$default"
	else
		read -r value </dev/tty
	fi
	value="${value:-$default}"
	[[ -n "$value" ]] || die "empty input for ${varname}"
	printf -v "$varname" '%s' "$value"
}

prompt_yes_no() {
	local prompt="$1" default_yes="${2:-1}"
	local ans
	if [[ "${AIPROXY_PROVIDER_FT_NONINTERACTIVE:-}" == "1" ]]; then
		[[ "$default_yes" == "1" ]]
		return
	fi
	if [[ ! -t 0 ]]; then
		[[ "$default_yes" == "1" ]]
		return
	fi
	echo -n "${prompt} [Y/n]: " >&2
	read -r ans </dev/tty
	case "$ans" in
	n|N|no|No|NO) return 1 ;;
	*) return 0 ;;
	esac
}

build_config_json() {
	if [[ -n "$CONFIG_JSON" ]]; then
		echo "$CONFIG_JSON" | jq -c .
		return
	fi
	[[ -n "$BASE_URL" ]] || die "set AIPROXY_PROVIDER_FT_BASE_URL or AIPROXY_PROVIDER_FT_CONFIG"
	jq -nc --arg u "$BASE_URL" '{base_url:$u}'
}

collect_inputs() {
	local suffix
	suffix="$(date +%Y%m%d%H%M%S)"

	if [[ "${AIPROXY_PROVIDER_FT_NONINTERACTIVE:-}" == "1" ]]; then
		PROVIDER_RESOURCE_NAME="${PROVIDER_RESOURCE_NAME:-aiproxy-provider-ft-${suffix}}"
		PROVIDER_KEY="${PROVIDER_KEY:-custom-ft-${suffix}}"
		[[ -n "$CONFIG_JSON" || -n "$BASE_URL" ]] || die "non-interactive mode requires BASE_URL or CONFIG"
		return
	fi

	echo "=== ai_provider 创建测试 ===" >&2
	echo "将创建自定义 ai_provider（provider_key 不可与 catalog 重复）。" >&2
	echo

	if [[ -z "$PROVIDER_RESOURCE_NAME" ]]; then
		prompt_line "资源名称 (climc 第一个参数 NAME)" "aiproxy-provider-ft-${suffix}" PROVIDER_RESOURCE_NAME
	fi
	if [[ -z "$PROVIDER_KEY" ]]; then
		prompt_line "provider_key (唯一标识)" "${PROVIDER_RESOURCE_NAME}" PROVIDER_KEY
	fi
	if [[ -z "$CONFIG_JSON" && -z "$BASE_URL" ]]; then
		prompt_line "config.base_url (OpenAI 兼容上游)" "https://api.openai.com" BASE_URL
	fi
	if prompt_yes_no "创建后启用 (--enabled)?" 1; then
		ENABLED_FLAG=true
	else
		ENABLED_FLAG=false
	fi
}

delete_existing_provider() {
	local name="$1"
	if ! climc_json ai-provider-show "$name" >/dev/null 2>&1; then
		return 0
	fi
	if [[ "$DELETE_IF_EXISTS" != "1" ]]; then
		die "ai_provider $name already exists; set AIPROXY_PROVIDER_FT_DELETE_EXISTING=1 to delete first"
	fi
	echo "deleting existing ai_provider $name" >&2
	climc ai-provider-delete "$name"
}

create_provider() {
	local config enabled_args=()
	config="$(build_config_json)"
	if [[ "$ENABLED_FLAG" == "true" ]]; then
		enabled_args=(--enabled)
	fi
	climc ai-provider-create \
		"$PROVIDER_RESOURCE_NAME" \
		--provider-key "$PROVIDER_KEY" \
		--config "$config" \
		"${enabled_args[@]}"
}

verify_provider() {
	local row pk base enabled
	row="$(climc_json ai-provider-show "$PROVIDER_RESOURCE_NAME")"
	pk="$(echo "$row" | jq -r '.provider_key // empty')"
	base="$(echo "$row" | jq -r '.config.base_url // empty')"
	enabled="$(echo "$row" | jq -r '.enabled // false')"
	[[ "$pk" == "$PROVIDER_KEY" ]] || die "provider_key mismatch: got $pk want $PROVIDER_KEY"
	if [[ -n "$BASE_URL" ]]; then
		[[ "$base" == "$BASE_URL" ]] || die "base_url mismatch: got $base want $BASE_URL"
	fi
	[[ "$enabled" == "true" ]] || [[ "$ENABLED_FLAG" != "true" ]] || die "expected enabled=true"
	echo "$row" | jq '{id, name, provider_key, enabled, config}'
}

collect_inputs
delete_existing_provider "$PROVIDER_RESOURCE_NAME"

aiproxy_ft_step "create ai_provider"
create_provider

aiproxy_ft_step "verify ai-provider-show"
verify_provider

aiproxy_ft_step "verify ai-provider-list filter"
cnt="$(climc_json ai-provider-list --provider-key "$PROVIDER_KEY" \
	| jq --arg n "$PROVIDER_RESOURCE_NAME" '[.data[] | select(.name == $n)] | length')"
[[ "$cnt" -ge 1 ]] || die "ai-provider-list --provider-key did not return created row"

echo
echo "OK: ai_provider create test passed."
echo "  name:         $PROVIDER_RESOURCE_NAME"
echo "  provider_key: $PROVIDER_KEY"
echo "Cleanup:"
echo "  climc ai-provider-delete $PROVIDER_RESOURCE_NAME"
