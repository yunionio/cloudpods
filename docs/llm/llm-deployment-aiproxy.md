# llm_deployment 与 aiproxy 自动关联

创建 `llm_deployment` 时**默认开启** `auto_register_aiproxy`，running 的 `llm` 副本会自动在 aiproxy 中创建 catalog 与路由资源。

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

## 查看对应关系

deployment 详情中的 `aiproxy_instances` 列出每个副本的客户端 model 别名与 provider id：

```bash
climc llm-deployment-show my-qwen
```

aiproxy 侧也可按来源 id 反查：

```bash
climc ai-provider-list --llm-deployment-id <deployment_id>
climc ai-provider-list --llm-id <llm_id>
climc ai-routing-list --llm-deployment-id <deployment_id>
```

## 客户端访问

需先创建项目级 `ai_virtual_key`，再通过 aiproxy OpenAI 兼容接口访问：

```bash
curl -k "$AIPROXY/openai/v1/chat/completions" \
  -H "Authorization: Bearer $VK" \
  -d '{"model":"<llm_name>-<upstream_model_key>","messages":[{"role":"user","content":"hi"}]}'
```

`model` 填 deployment 详情里 `aiproxy_instances[].client_model_alias`。

## 删除 deployment

删除 `llm_deployment` 时会自动清理关联的 aiproxy 资源（`ai_provider`、`ai_model`、`ai_routing`），按 `llm_deployment_id` 及子 `llm` 实例 id 查找，无需 `auto_register_aiproxy` 为 true。

## 字段对应关系

| llm 侧 | aiproxy 侧 |
|--------|-----------|
| `llm_deployment.id` | `ai_routing.llm_deployment_id` |
| `llm.id` | `ai_provider.llm_id`、`ai_routing_model.llm_id` |
| `llm_deployment.id` | `ai_provider.llm_deployment_id` |
| upstream served model name（vLLM/SGLang 为 model 目录 basename，Ollama 为 `name:tag`） | `ai_model.model_key` |
| `{llm_name}-{upstream_model_key}` | `ai_routing_model.model_pattern` |
