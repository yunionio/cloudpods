# Shared helpers for aiproxy functional test scripts (source only, do not execute directly).

die() { echo "ERROR: $*" >&2; exit 1; }

need_cmd() {
	command -v "$1" >/dev/null 2>&1 || die "missing command: $1"
}

aiproxy_ft_need_cmds() {
	need_cmd climc
	need_cmd curl
	need_cmd jq
}

climc_json() {
	climc --output-format json "$@"
}

aiproxy_ft_step() { echo; echo "==> $*"; }

catalog_model_id() {
	local provider="$1" model_key="$2"
	echo "${provider}-${model_key}"
}

default_model_for_provider() {
	case "$1" in
	aliyun) echo "qwen-turbo" ;;
	xiaomi) echo "mimo-v2-flash" ;;
	openai) echo "gpt-4o-mini" ;;
	*) echo "" ;;
	esac
}

default_prompt_for_provider() {
	case "$1" in
	aliyun) echo "用一句话介绍通义千问" ;;
	xiaomi) echo "用一句话介绍小米 MiMo" ;;
	*) echo "用一句话介绍这个模型" ;;
	esac
}

resolve_aiproxy_url() {
	if [[ -n "${AIPROXY_URL:-}" ]]; then
		echo "${AIPROXY_URL%/}"
		return
	fi
	local url
	url="$(climc_json endpoint-list --service aiproxy --interface public --limit 1 \
		| jq -r '.data[0].url // empty')"
	[[ -n "$url" ]] || die "cannot resolve aiproxy public URL; set AIPROXY_URL"
	echo "${url%/}"
}

list_catalog_provider_keys() {
	climc_json ai-provider-list --limit 500 \
		| jq -r '.data[] | .provider_key // empty' | sed '/^$/d' | sort -u
}

list_catalog_model_keys() {
	local provider_key="$1"
	climc_json ai-model-list --ai-provider-id "$provider_key" --limit 500 \
		| jq -r '.data[] | .model_key // empty' \
		| sed '/^$/d' | grep -vx 'default' | sort -u
}

# Resolve API key from env (generic or provider-specific legacy names).
resolve_api_key_from_env() {
	local provider_key="$1"
	if [[ -n "${AIPROXY_FT_API_KEY:-}" ]]; then
		echo "$AIPROXY_FT_API_KEY"
		return
	fi
	case "$provider_key" in
	aliyun)
		[[ -n "${DASHSCOPE_API_KEY:-}" ]] && echo "$DASHSCOPE_API_KEY" && return
		;;
	xiaomi)
		[[ -n "${MIMO_API_KEY:-}" ]] && echo "$MIMO_API_KEY" && return
		;;
	esac
	return 1
}

prompt_api_key() {
	local provider_key="$1" key
	if key="$(resolve_api_key_from_env "$provider_key")"; then
		echo "使用环境变量中的 API Key（未回显）" >&2
		echo "$key"
		return
	fi
	if [[ ! -t 0 ]]; then
		die "未设置 API Key：export AIPROXY_FT_API_KEY 或 ${provider_key} 对应的环境变量，或使用交互式终端"
	fi
	echo -n "请输入 ${provider_key} 的 API Key（不回显）: " >&2
	read -r -s key </dev/tty
	echo >&2
	[[ -n "$key" ]] || die "API Key 不能为空"
	echo "$key"
}

# Populates global AIPROXY_FT_LINES[] (bash 3.2 compatible).
_load_lines_into_array() {
	AIPROXY_FT_LINES=()
	while IFS= read -r line; do
		[[ -n "$line" ]] && AIPROXY_FT_LINES+=("$line")
	done
}

prompt_select_provider() {
	local -a keys=()
	local k i choice
	_load_lines_into_array < <(list_catalog_provider_keys)
	keys=("${AIPROXY_FT_LINES[@]}")
	[[ ${#keys[@]} -gt 0 ]] || die "catalog 中无 ai_provider，请先执行 aiproxy master InitDB"

	if [[ -n "${AIPROXY_FT_PROVIDER:-}" ]]; then
		for k in "${keys[@]}"; do
			[[ "$k" == "${AIPROXY_FT_PROVIDER}" ]] && echo "${AIPROXY_FT_PROVIDER}" && return
		done
		die "ai_provider ${AIPROXY_FT_PROVIDER} 不在 catalog 中"
	fi

	if [[ ! -t 0 ]]; then
		die "请设置 AIPROXY_FT_PROVIDER 或在交互式终端运行"
	fi

	echo "可用模型提供商 (catalog):" >&2
	for i in "${!keys[@]}"; do
		printf '  [%d] %s\n' "$((i + 1))" "${keys[$i]}" >&2
	done
	while true; do
		echo -n "请选择序号 [1-${#keys[@]}] 或直接输入 provider_key: " >&2
		read -r choice </dev/tty
		choice="${choice//[[:space:]]/}"
		[[ -z "$choice" ]] && continue
		if [[ "$choice" =~ ^[0-9]+$ ]] && ((choice >= 1 && choice <= ${#keys[@]})); then
			echo "${keys[$((choice - 1))]}"
			return
		fi
		for k in "${keys[@]}"; do
			[[ "$k" == "$choice" ]] && echo "$choice" && return
		done
		echo "无效选择，请重试。" >&2
	done
}

prompt_select_model() {
	local provider_key="$1"
	local -a models=()
	local m i choice default_m found
	_load_lines_into_array < <(list_catalog_model_keys "$provider_key")
	models=("${AIPROXY_FT_LINES[@]}")
	[[ ${#models[@]} -gt 0 ]] || die "provider ${provider_key} 下无可用 model_key（catalog 未 seed？）"

	if [[ -n "${AIPROXY_FT_MODEL:-}" ]]; then
		for m in "${models[@]}"; do
			[[ "$m" == "${AIPROXY_FT_MODEL}" ]] && echo "${AIPROXY_FT_MODEL}" && return
		done
		die "model_key ${AIPROXY_FT_MODEL} 不在 provider ${provider_key} 的 catalog 中"
	fi

	default_m="$(default_model_for_provider "$provider_key")"
	found=0
	if [[ -n "$default_m" ]]; then
		for m in "${models[@]}"; do
			if [[ "$m" == "$default_m" ]]; then
				found=1
				break
			fi
		done
	fi
	[[ "$found" -eq 1 ]] || default_m="${models[0]}"

	if [[ ! -t 0 ]]; then
		echo "$default_m"
		return
	fi

	echo "提供商 ${provider_key} 的模型:" >&2
	for i in "${!models[@]}"; do
		printf '  [%d] %s\n' "$((i + 1))" "${models[$i]}" >&2
	done
	while true; do
		echo -n "请选择序号 [1-${#models[@]}] 或输入 model_key [默认: ${default_m}]: " >&2
		read -r choice </dev/tty
		choice="${choice//[[:space:]]/}"
		if [[ -z "$choice" ]]; then
			echo "$default_m"
			return
		fi
		if [[ "$choice" =~ ^[0-9]+$ ]] && ((choice >= 1 && choice <= ${#models[@]})); then
			echo "${models[$((choice - 1))]}"
			return
		fi
		for m in "${models[@]}"; do
			[[ "$m" == "$choice" ]] && echo "$choice" && return
		done
		echo "无效选择，请重试。" >&2
	done
}

prompt_run_stream() {
	if [[ "${AIPROXY_FT_SKIP_STREAM:-}" == "1" ]]; then
		return 1
	fi
	if [[ "${AIPROXY_FT_SKIP_STREAM:-}" == "0" ]]; then
		return 0
	fi
	if [[ ! -t 0 ]]; then
		return 0
	fi
	local ans
	echo -n "是否执行流式测试 (stream=true)? [Y/n]: " >&2
	read -r ans </dev/tty
	case "$ans" in
	n|N|no|No|NO) return 1 ;;
	*) return 0 ;;
	esac
}

ensure_ai_key_enabled() {
	local key_name="$1"
	local enabled
	enabled="$(climc_json ai-key-show "$key_name" | jq -r '.enabled // false')"
	if [[ "$enabled" != "true" ]]; then
		echo "ai_key $key_name is disabled, enabling"
		climc ai-key-enable "$key_name"
	fi
}

ensure_ai_key() {
	local provider_key="$1" key_name="$2" api_secret="$3"
	local provider_id
	provider_id="$(climc_json ai-provider-show "$provider_key" | jq -r '.id // empty')"
	[[ -n "$provider_id" ]] || die "ai_provider $provider_key not found"

	if climc_json ai-key-show "$key_name" >/dev/null 2>&1; then
		echo "ai_key $key_name exists, syncing secret and ai_provider_id"
		climc ai-key-update "$key_name" \
			--ai-provider-id "$provider_id" \
			--secret "$api_secret" \
			--weight 10
	else
		climc ai-key-create \
			"$key_name" \
			--ai-provider-id "$provider_id" \
			--secret "$api_secret" \
			--weight 10 \
			--enabled
	fi
	ensure_ai_key_enabled "$key_name"
}

verify_ai_key_for_provider() {
	local provider_key="$1"
	local provider_id count
	provider_id="$(climc_json ai-provider-show "$provider_key" | jq -r '.id // empty')"
	[[ -n "$provider_id" ]] || die "ai_provider $provider_key not found"
	count="$(climc_json ai-key-list --ai-provider-id "$provider_id" | jq '[.data[] | select(.enabled == true)] | length')"
	[[ "$count" -gt 0 ]] || die "no enabled ai_key bound to ai_provider_id=$provider_id"
	echo "enabled ai_key rows for $provider_id: $count"
}

ensure_virtual_key() {
	local vk_name="$1"
	if climc_json ai-virtual-key-show "$vk_name" >/dev/null 2>&1; then
		echo "virtual key $vk_name already exists"
		return
	fi
	climc ai-virtual-key-create "$vk_name"
}

ensure_routing() {
	local routing_name="$1" provider_key="$2" catalog_model_id="$3"
	if climc_json ai-routing-show "$routing_name" >/dev/null 2>&1; then
		echo "routing $routing_name already exists"
		return
	fi
	climc ai-routing-create \
		"$routing_name" \
		--priority 10 \
		--models "[{\"ai_provider_id\":\"${provider_key}\",\"ai_model_id\":\"${catalog_model_id}\",\"priority\":1}]"
}

verify_stream_chat() {
	local base_url="$1" vk="$2" model="$3" out="$4" prompt="$5"
	local http_code aggregated delta payload

	http_code="$(curl -k -sS -N -o "$out" -w '%{http_code}' \
		"${base_url%/}/openai/v1/chat/completions" \
		-H "Authorization: Bearer ${vk}" \
		-H "Content-Type: application/json" \
		-d "{\"model\":\"${model}\",\"stream\":true,\"messages\":[{\"role\":\"user\",\"content\":$(jq -Rn --arg t "$prompt" '$t')}],\"max_tokens\":64}")"

	echo "HTTP $http_code (stream)"
	[[ "$http_code" == "200" ]] || {
		echo "--- stream body (first 40 lines) ---" >&2
		head -n 40 "$out" >&2 || true
		die "stream chat failed with HTTP $http_code"
	}

	aggregated=""
	while IFS= read -r line || [[ -n "$line" ]]; do
		[[ "$line" == data:* ]] || continue
		payload="${line#data: }"
		payload="${payload//$'\r'/}"
		[[ -z "$payload" ]] && continue
		[[ "$payload" == "[DONE]" ]] && continue
		if echo "$payload" | jq -e '.error // .message' >/dev/null 2>&1; then
			echo "upstream error chunk: $payload" >&2
			die "stream returned error event"
		fi
		delta="$(echo "$payload" | jq -r '.choices[0].delta.content // empty' 2>/dev/null || true)"
		aggregated+="$delta"
	done <"$out"

	[[ -n "$aggregated" ]] || {
		echo "--- stream body ---" >&2
		cat "$out" >&2
		die "empty aggregated stream content (no choices[0].delta.content)"
	}

	echo "stream content (${#aggregated} chars): ${aggregated:0:120}..."
}

# aiproxy_ft_run executes the full functional test for one provider/model/api key.
aiproxy_ft_run() {
	local provider_key="$1" chat_model="$2" api_secret="$3" chat_prompt="$4"
	local run_stream="${5:-1}"

	local key_name vk_name routing_name catalog_mid
	local chat_resp chat_stream_resp aiproxy_url vk http_code content

	key_name="${AIPROXY_FT_KEY_NAME:-aiproxy-ft-${provider_key}}"
	vk_name="${AIPROXY_FT_VK_NAME:-aiproxy-ft-${provider_key}-vk}"
	routing_name="${AIPROXY_FT_ROUTING_NAME:-aiproxy-ft-${provider_key}-routing}"
	chat_resp="${AIPROXY_FT_CHAT_RESP:-/tmp/aiproxy-ft-${provider_key}-chat.json}"
	chat_stream_resp="${AIPROXY_FT_STREAM_RESP:-/tmp/aiproxy-ft-${provider_key}-chat-stream.sse}"
	catalog_mid="$(catalog_model_id "$provider_key" "$chat_model")"

	echo
	echo "=== aiproxy 功能测试 ==="
	echo "provider: ${provider_key}  model: ${chat_model}  catalog_id: ${catalog_mid}"
	echo "ai_key: ${key_name}  virtual_key: ${vk_name}  routing: ${routing_name}"
	echo

	aiproxy_ft_step "1. Keystone aiproxy public endpoint"
	aiproxy_url="$(resolve_aiproxy_url)"
	echo "AIPROXY_URL=$aiproxy_url"

	aiproxy_ft_step "2. Catalog ${provider_key} / ${chat_model}"
	climc ai-provider-show "$provider_key" >/dev/null \
		|| die "ai_provider $provider_key missing; run aiproxy master InitDB first"
	climc ai-model-show "$catalog_mid" >/dev/null \
		|| die "ai_model $catalog_mid not in catalog (re-run aiproxy master InitDB)"

	aiproxy_ft_step "3. ai_key"
	ensure_ai_key "$provider_key" "$key_name" "$api_secret"
	verify_ai_key_for_provider "$provider_key"

	aiproxy_ft_step "4. ai_virtual_key"
	ensure_virtual_key "$vk_name"
	vk="$(climc_json ai-virtual-key-show "$vk_name" | jq -r '.virtual_key')"
	[[ -n "$vk" ]] || die "empty virtual_key from ai-virtual-key-show"
	echo "virtual_key=${vk:0:12}..."

	aiproxy_ft_step "5. ai_routing"
	ensure_routing "$routing_name" "$provider_key" "$catalog_mid"

	aiproxy_ft_step "6. POST /openai/v1/chat/completions"
	http_code="$(curl -k -sS -o "$chat_resp" -w '%{http_code}' \
		"${aiproxy_url}/openai/v1/chat/completions" \
		-H "Authorization: Bearer ${vk}" \
		-H "Content-Type: application/json" \
		-d "{\"model\":\"${chat_model}\",\"messages\":[{\"role\":\"user\",\"content\":$(jq -Rn --arg t "$chat_prompt" '$t')}],\"max_tokens\":128}")"

	echo "HTTP $http_code"
	jq . <"$chat_resp"
	[[ "$http_code" == "200" ]] || die "chat request failed with HTTP $http_code"

	content="$(jq -r '.choices[0].message.content // empty' <"$chat_resp")"
	[[ -n "$content" ]] || die "empty choices[0].message.content"

	if [[ "$run_stream" == "1" ]]; then
		aiproxy_ft_step "7. POST /openai/v1/chat/completions (stream=true)"
		verify_stream_chat "$aiproxy_url" "$vk" "$chat_model" "$chat_stream_resp" "$chat_prompt"
	fi

	echo
	echo "OK: aiproxy functional test passed for ${provider_key}/${chat_model} (non-stream$([[ "$run_stream" == "1" ]] && echo ' + stream' || echo ''))."
	echo "Cleanup (optional):"
	echo "  climc ai-routing-delete $routing_name"
	echo "  climc ai-virtual-key-delete $vk_name"
	echo "  climc ai-key-delete $key_name"
}
