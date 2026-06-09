# aiproxy 功能测试（climc + 小米 MiMo）

本文用 **climc** 配置 aiproxy 资源，并通过 **curl** 调用 `POST /v1/chat/completions` 验证 **xiaomi** catalog（`api.xiaomimimo.com`）。

> **安全**：请勿将 MiMo API Key 写入脚本或提交到 Git。使用环境变量 `MIMO_API_KEY`。若 Key 曾泄露，请到小米开放平台轮换。

通义千问（DashScope）测试见 [functional-test-climc.md](./functional-test-climc.md)。

## 前置条件

| 项 | 说明 |
|----|------|
| 服务 | aiproxy **主节点**已部署，Keystone 中已注册 `aiproxy` public endpoint |
| 数据库 | 主节点已执行 `InitDB`，catalog 含 `xiaomi` 及 `mimo-*` 模型 |
| 客户端 | 已 `source /etc/yunion/rcadmin`，`climc` 可用 |
| 工具 | `jq` |
| 网络 | aiproxy 节点能访问 `https://api.xiaomimimo.com` |

```bash
source /etc/yunion/rcadmin
export CLIMC_OUTPUT_FORMAT=json
bash scripts/test/aiproxy/aiproxy-functional-test.sh
# 交互菜单中选择 xiaomi，输入 MiMo API Key
```

或快捷入口（默认选中 `xiaomi`）：

```bash
export MIMO_API_KEY='你的 MiMo API Key'
bash scripts/test/aiproxy/aiproxy-functional-test-mimo.sh
```

预置模型：`export AIPROXY_FT_PROVIDER=xiaomi AIPROXY_FT_MODEL=mimo-v2.5-pro`

按 provider 自动命名的资源（可覆盖 `AIPROXY_FT_KEY_NAME` 等）：

| 变量 | 默认（xiaomi） |
|------|----------------|
| `AIPROXY_FT_KEY_NAME` | `aiproxy-ft-xiaomi` |
| `AIPROXY_FT_VK_NAME` | `aiproxy-ft-xiaomi-vk` |
| `AIPROXY_FT_ROUTING_NAME` | `aiproxy-ft-xiaomi-routing` |
| `AIPROXY_FT_MODEL` | 交互默认 `mimo-v2-flash` |

## 手动步骤摘要

### Catalog

```bash
climc ai-provider-show xiaomi
climc ai-model-show xiaomi-mimo-v2-flash
```

`config.base_url` 应为 `https://api.xiaomimimo.com`。

### ai_key

```bash
climc ai-key-create mimo-ft \
  --ai-provider-id xiaomi \
  --secret "${MIMO_API_KEY}" \
  --weight 10 \
  --enabled
```

### 路由与 chat

```bash
climc ai-virtual-key-create aiproxy-mimo-ft-vk
climc ai-routing-create aiproxy-mimo-ft-routing \
  --priority 10 \
  --models '[{"ai_provider_id":"xiaomi","ai_model_id":"mimo-v2-flash","priority":1}]'

AIPROXY_URL="$(climc endpoint-list --service aiproxy --interface public --limit 1 \
  --output-format json | jq -r '.data[0].url')"
VK="$(climc ai-virtual-key-show aiproxy-mimo-ft-vk --output-format json | jq -r '.virtual_key')"

curl -k -sS "${AIPROXY_URL%/}/v1/chat/completions" \
  -H "Authorization: Bearer ${VK}" \
  -H "Content-Type: application/json" \
  -d '{"model":"mimo-v2-flash","messages":[{"role":"user","content":"hi"}],"max_tokens":64}' | jq .
```

### 流式

`scripts/test/aiproxy/aiproxy-functional-test-mimo.sh` 在非流式通过后默认执行 step 7（`stream: true`）。跳过：`export AIPROXY_FT_SKIP_STREAM=1`。

```bash
curl -k -sS -N -o /tmp/aiproxy-mimo-stream.sse \
  "${AIPROXY_URL%/}/v1/chat/completions" \
  -H "Authorization: Bearer ${VK}" \
  -H "Content-Type: application/json" \
  -d '{"model":"mimo-v2-flash","stream":true,"messages":[{"role":"user","content":"hi"}],"max_tokens":64}'
```

catalog 中其它模型：`mimo-v2.5-pro`、`mimo-v2-pro`、`mimo-v2.5`、`mimo-v2-omni`（id 形如 `xiaomi-mimo-v2.5-pro`）。

## 常见问题

**上游 401**  
检查 `MIMO_API_KEY` 是否有效；确认 `ai_key` 已 `--enabled` 且 `ai_provider_id=xiaomi`。

**`no ai_routing matched`**  
virtual key 与 routing 须在同一 climc 项目下创建。

**与 DashScope 脚本冲突**  
MiMo 脚本使用独立的 vk/routing/key 名称；勿与 `aiproxy-ft-vk` 混用同一 routing 的 model 列表。

## 清理

```bash
climc ai-routing-delete aiproxy-mimo-ft-routing
climc ai-virtual-key-delete aiproxy-mimo-ft-vk
climc ai-key-delete mimo-ft
```
