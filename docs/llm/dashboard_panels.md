# 自定义控制面板 JS API

服务：`llm`  
鉴权：`X-Auth-Token`（用户 session）  
交互：自然语言 → **generate**（预览，不落库）→ **validate**（可选）→ **save**（确认写入 yunionconf）

请求/响应均使用 **KeywordPlural** 包裹（与 `PerformClassAction` / 前端 Manager 一致）：

```json
{ "dashboard_panels": { /* 业务字段 */ } }
```

前端约定：沙箱 iframe 渲染 `mount(root, ctx)`（只画内容，标题由 `return { title }` 交给父页面）；用量数据对齐 overview 的 `GET /v2/rpc/usages/general-usage`。

---

## POST `/dashboard_panels/generate`

用 default MCP Agent 生成并静态校验 JS，**不写库**。

### Request

```json
{
  "prompt": "展示用量概览",
  "title": "用量概览",
  "history": []
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| prompt | string | 是 | 自然语言需求 |
| scope | string | 否 | `system` / `domain` / `project`；**可省略**，生成代码跟随前端视图 `ctx.scope` |
| title | string | 否 | 面板标题建议 |
| history | array | 否 | 多轮上下文 `{role,content}` |

### Response `200`

```json
{
  "dashboard_panels": {
    "source": "function mount(root, ctx) { ... }",
    "entry": "mount",
    "title": "用量概览",
    "scope": "",
    "content_sha256": "...",
    "api_allowlist": ["GET /v2/rpc/usages/", "..."],
    "warnings": []
  }
}
```

### climc

```bash
climc dashboard-panel-generate --prompt "展示用量概览"
```

---

## POST `/dashboard_panels/validate`

仅静态校验（预览前可调；`save` 仍会再校验）。

### Request

```json
{
  "source": "function mount(root, ctx) { ... }",
  "api_allowlist": ["GET /v2/rpc/usages/general-usage"]
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| source | string | 是 | JS 源码 |
| api_allowlist | string[] | 否 | 为空则用默认只读前缀 |

### Response `200`

```json
{
  "dashboard_panels": {
    "ok": true,
    "entry": "mount",
    "content_sha256": "...",
    "api_allowlist": ["GET /v2/rpc/usages/general-usage"],
    "warnings": [],
    "error": ""
  }
}
```

`ok=false` 时看 `error`（如命中 `eval` / 缺少 `mount` / allowlist 非法路径）。

### climc

```bash
climc dashboard-panel-validate --source 'function mount(root, ctx) {}'
```

---

## POST `/dashboard_panels/save`

再次校验后写入用户 namespace 的 yunionconf `parameter`。

### Request

```json
{
  "source": "function mount(root, ctx) { ... }",
  "title": "用量概览",
  "prompt": "展示用量概览",
  "name": "",
  "panel_parameter": "dashboard-project-panel-xxx",
  "item_id": "dashboard-item-xxx",
  "layout": { "x": 0, "y": 0, "w": 4, "h": 4, "minH": 2 }
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| source | string | 是 | 最终确认源码 |
| scope | string | 否 | system/domain/project；可省略，运行时跟前端视图 |
| title | string | 否 | 标题 |
| prompt | string | 否 | 需求摘要（审计） |
| name | string | 否 | 已有则更新；否则 `dashboard-custom-script-{uuid}` |
| api_allowlist | string[] | 否 | 只读 API 白名单 |
| panel_parameter | string | 否 | 合并进该 overview panel |
| item_id | string | 否 | 布局项 id |
| layout | object | 否 | x/y/w/h/minH |

落库 `parameter.value` 形状见 `DashboardCustomScriptValue`（`source` / `entry` / `api_allowlist` / `content_sha256` 等）。  
挂到概览时组件名为 `CustomJsPanel`，`params.script_parameter` 指向上述 parameter 名。

### Response `200`

```json
{
  "dashboard_panels": {
    "name": "dashboard-custom-script-...",
    "parameter_id": "123",
    "entry": "mount",
    "content_sha256": "...",
    "api_allowlist": ["GET /v2/rpc/usages/"],
    "panel_parameter": "dashboard-project-panel-xxx",
    "item_id": "dashboard-item-xxx",
    "warnings": []
  }
}
```

### climc

```bash
climc dashboard-panel-save --source '...' --title '用量概览' --prompt '展示用量概览'
```

---

## Swagger

```bash
# 生成 llm swagger 代码与 yaml，并启动文档站（默认端口 40900）
make swagger-serve
# 或只刷 llm：
./scripts/codegen.py swagger-code -s llm
./scripts/codegen.py swagger-yaml -s llm
```

生成物：

- `pkg/generated/swagger/llm/`
- `_output/swagger/swagger_llm.yaml`
- `docs/swagger/swagger_llm.yaml`（仓库内文档副本）
