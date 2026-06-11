# llm_deployment 与 aiproxy 自动关联

创建 `llm_deployment` 时**默认开启** `auto_register_aiproxy`。创建任务完成后会调度 `LLMAiproxySyncTask`；副本变为 running 后自动在 aiproxy 中创建 catalog 与路由资源。

## 创建 deployment（默认已关联 aiproxy）

```bash
climc llm-deployment-create my-qwen \
  --llm-sku-id <sku> \
  --net <net> \
  --replicas 2
```

关闭自动关联：

```bash
climc llm-deployment-create my-qwen \
  --auto-register-aiproxy=false \
  --llm-sku-id <sku> \
  --net <net>
```

## 手动重同步 / 取消注册

```bash
climc llm-deployment-register-aiproxy my-qwen
climc llm-deployment-unregister-aiproxy my-qwen
```

`unregister-aiproxy` 会清理 aiproxy 侧资源，并将 `auto_register_aiproxy` 置为 false，同时清空 `aiproxy_bindings`。deployment `status` 恢复为副本健康态（`ready` / `partial` / `deploying` 等）。

网关同步阶段通过 deployment `status` 表达：`aiproxy_pending`、`aiproxy_syncing`、`aiproxy_partial`、`aiproxy_sync_failed`；全部同步成功后恢复为 `ready` 或 `partial`。

## 查看对应关系

deployment 详情中的 `aiproxy_bindings` 列出每个副本的客户端 model 别名与 provider id：

```bash
climc llm-deployment-show my-qwen
```

aiproxy 侧也可按来源 id 反查 provider：

```bash
climc ai-provider-list --llm-deployment-id <deployment_id>
climc ai-provider-list --llm-id <llm_id>
```

deployment 与 routing 的关联保存在 `llm_deployment.aiproxy_routing_id`。

## 客户端访问

需先创建项目级 `ai_virtual_key`，再通过 aiproxy OpenAI 兼容接口访问：

```bash
curl -k "$AIPROXY/openai/v1/chat/completions" \
  -H "Authorization: Bearer $VK" \
  -d '{"model":"<deployment_name>-<upstream_model_key>","messages":[{"role":"user","content":"hi"}]}'
```

`model` 填 deployment 详情里 `aiproxy_bindings[].client_model_alias`（格式 `{deployment_name}-{upstream_model_key}`，同一 deployment 下各副本相同），与 `ai_routing.model_key` 同值。LLM 自动注册/重同步会写入 `model_key` 并清空 `model_pattern`。

## 删除 deployment

删除 `llm_deployment` 时会自动清理关联的 aiproxy 资源（`ai_provider`、`ai_model`、`ai_routing`），provider 按 `llm_deployment_id` / `llm_id` 查找，routing 按 `llm_deployment.aiproxy_routing_id` 删除，无需 `auto_register_aiproxy` 为 true。

## 字段对应关系

| llm 侧 | aiproxy 侧 |
|--------|-----------|
| `llm_deployment.aiproxy_routing_id` | `ai_routing.id` |
| `llm.id` | `ai_provider.llm_id` |
| `llm_deployment.id` | `ai_provider.llm_deployment_id` |
| upstream served model name（vLLM/SGLang 为 model 目录 basename，Ollama 为 `name:tag`） | `ai_model.model_key` |
| `{deployment_name}-{upstream_model_key}` | **`ai_routing.model_key`** 与 **`aiproxy_bindings[].client_model_alias`**（同值；客户端 model 精确匹配，选路与列表优先） |
| （手工通配/前缀规则） | `ai_routing.model_pattern` |
| （LLM 自动注册时留空） | `ai_routing_model.model_pattern` |

`ai_routing` / `ai_routing_model` 不再保存 `llm_deployment_id` / `llm_id`；通过 `ai_provider` 与 `ai_routing_model.ai_provider_id` 间接关联推理实例。

## 命名规则

自动注册时 aiproxy catalog 资源名称基于推理部署名与副本实例名（`{deployment_name}-{index}`），不再使用 UUID 后缀：

| aiproxy 资源 | 命名规则 | 示例（部署 `my-qwen`，副本 `my-qwen-0`） |
|--------------|----------|------------------------------------------|
| `ai_routing.name` | `llm-dep-{slug(deployment.name)}` | `llm-dep-my-qwen` |
| `ai_provider.name` | `llm-{slug(llm.name)}` | `llm-my-qwen-0` |
| `ai_model.name` | `llm-{slug(llm.name)}-{slug(model_key)}` | `llm-my-qwen-0-qwen3-0-6b` |

`slug(...)` 将名称规范为小写并将 `/`、`.` 等特殊字符转为 `-`。部署名或副本名为空时回退为对应 id。

若 deployment 尚未记录 `aiproxy_routing_id`，同步时会按 `llm-dep-{slug(name)}` 查找已有 `ai_routing` 并更新，避免同名 deployment 重建或并发同步时的 409 重名错误。

对已注册资源执行 `climc llm-deployment-register-aiproxy` 会同步更新上述名称。
