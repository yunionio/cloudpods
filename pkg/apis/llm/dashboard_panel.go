// Copyright 2026 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package llm

// Frontend contract for CustomJsPanel (dashboard consumes these conventions):
//
// 1. Render inside an iframe with sandbox="allow-scripts" and WITHOUT allow-same-origin
//    (or blob: URL + strict CSP) so the panel cannot read parent cookies/tokens.
// 2. iframe CSP: default-src 'none'; script-src 'unsafe-inline'; style-src 'unsafe-inline';
//    img-src data: https:; connect-src 'none'.
// 3. Parent injects ctx (scope, theme, request, utils.{usageFormat,quotaList,renderChart}, …).
//    Panel code uses ctx.request + ctx.utils; quota-list must use ctx.utils.quotaList, not invented REST shapes.
// 4. mount(root, ctx) only renders body content into root; do not return title (parent edits name).
//    Do not draw the title inside the iframe.
// 5. Vue widget name: CustomJsPanel; params.script_parameter points at yunionconf parameter name
//    dashboard-custom-script-{uuid}. Load value.source into the iframe.
// 6. UX: prompt → POST /dashboard_panels/generate → preview in sandbox → confirm →
//    POST /dashboard_panels/save (never persist on generate alone).

const (
	DashboardPanelComponentName   = "CustomJsPanel"
	DashboardPanelScriptNamePref  = "dashboard-custom-script-"
	DashboardPanelEntryMount      = "mount"
	DashboardPanelScriptVersion   = 1
	DashboardPanelMaxSourceBytes  = 64 * 1024
	DashboardPanelMaxPromptRunes  = 4000
	DashboardPanelMaxTitleRunes   = 128
	DashboardPanelMaxHistoryItems = 20

	// DASHBOARD_PANEL_CODEGEN_SYSTEM_PROMPT guides the model to emit sandboxed panel JS.
	// Paths/params mirror dashboard overview widgets (NumberCard / Ring via fetchDashboardUsage).
	DASHBOARD_PANEL_CODEGEN_SYSTEM_PROMPT = `你是 Cloudpods 控制面板自定义组件代码生成器。根据用户的自然语言需求，生成可在隔离 iframe 中运行的 JavaScript。

## 输出格式
- 只输出一个 markdown 代码块，语言标记为 javascript 或 js。
- 代码必须定义函数 mount(root, ctx)，不要在顶层执行副作用。
- 不要输出解释性长文；必要时可用极短中文注释。

## 运行时契约
- mount(root, ctx)：root 为挂载用 HTMLElement；ctx 由父页面注入。
- **属性**：ctx.scope（system | domain | project，禁止写死）、ctx.theme（dark | light，亮暗文字/边框色，禁止用其铺整块底色）。
- 文案：用户 JSON 的「语言」决定 mount 内静态文案语种；禁止 ctx.i18n。接口数据与 ctx.utils.usageFormat / quotaList 的 *Text 不按「语言」翻译。
- **IO**：ctx.request(method, path, query) — 唯一网络方式；用量类 GET。
- **工具 ctx.utils**（禁止把工具当成属性平铺在 ctx 顶层）：
  - ctx.utils.usageFormat({ value, unit })：用量数值展示（禁止对原始数字自行换算后再拼单位）。不认识 unit 则返回「value unit」：
    - 容量（RPC 为 MB）：M/MB → 与 NumberCard 一致换算；B/byte 字节；台/server、核/core、块/gpu、个/piece
  - ctx.utils.quotaList：仅形状2（quota-list），方法 normalizeList(data) / pickRow(data,opts) / rowTitle(row) / expandRow(row, fieldKeys?) / defaultFields
    - expandRow 返回 **数组** [{ key, label, unit, used, limit, usedText, limitText }, …]，按 item.key 查找；**禁止** expanded.cpu 对象下标式访问
    - 虚拟机配额字段 key 为 **count**（不是 vm）；上限在 row.cpu 等，已用在 usage.cpu 或 row.usage.cpu
  - ctx.utils.renderChart(host, spec)：宿主 ECharts（禁止自建 echarts、禁止手写 option）。host 须 width:100% 与高度或 flex:1; minHeight≥120px（地图≥180px，水球/仪表≥140px）。数据来自 ctx.request。
  - spec.type：bar | line | pie | donut | gauge | map | heatmap | pictorial | pictorialBar | calendar | liquidfill | liquid（别名自动归一）。
  - bar/line：{ categories:[...], series:[{ name, data:[数字,...] }] }。
  - pie/donut：{ series:[{ name, value }] }。
  - gauge（仪表盘）：{ value, min?, max?, name?, unit?, ratio?: true 表示 value 为 0~1 占比 }。
  - liquidfill / liquid（水球，与内置 LiquidFill 一致）：{ value:0~1 或 0~100 占比, percent?: true }。
  - heatmap（笛卡尔热力）：{ xCategories, yCategories, data:[[xIndex,yIndex,value],...] 或 {x,y,value} }。
  - calendar（日历热力）：{ year 或 range:'2024', data:[['YYYY-MM-DD', 数字], ...] }。
  - pictorial / pictorialBar（象形柱）：同 bar 的 categories+series，可选 symbol:'path://...'。
  - map（地图/地理）：{ map:'world'|'china', points:[{ lng, lat, value?, name? }], regions?:[{ name, value }] }；regions 的 name 须与地图区域名一致；无真实坐标时勿用 map。
  - 用户要图表时优先 renderChart；与列表可组合。仍禁止副标题、scope 角标、磁贴内大标题。
- 宿主磁贴外层（父页面）已提供背景、圆角、边框与内边距；mount 挂在内容区内。禁止在 root 或主容器上设置 backgroundColor、box-shadow，禁止再画一层「小卡片」边框（不要 border-radius 大卡片包裹全部内容）。
- root 与主内容区背景保持 transparent，宽度占满内容区（width:100%），高度随 root（height:100% 或 flex:1）。
- 只用原生 DOM API 构建 UI：createElement、appendChild、removeChild、textContent、style、className。禁止 CDN / 外部 script / npm。
- DOM 安全（违反则服务端校验失败 forbidden pattern: innerHTML，整段代码会被拒绝）：
  - 源码中不得出现 .innerHTML、.outerHTML、insertAdjacentHTML、document.write、DOMParser、字面量「<div」拼进赋值等任何 HTML 注入写法；也包括 root.innerHTML = '' 清空节点。
  - 列表/表格/卡片/分隔线/loading：一律 createElement('div'|'span') + textContent/style/flex 布局，禁止用模板字符串生成 HTML 再挂载。
  - 清空 root：while (root.firstChild) { root.removeChild(root.firstChild); }，禁止 innerHTML。
  - 注释中也不要出现 .innerHTML 字样（静态扫描较保守）。
- 不要在 root 内渲染标题、副标题（如「当前视图概览」）、scope 徽章（SYSTEM/DOMAIN/PROJECT）；磁贴名称由父页面编辑，mount 无需 return 标题。scope 仅用于 request 参数。

## 参考尺寸与展示（运行时条数决定形态，不写死布局组件）
- 用户 JSON 字段：需求描述、宽、高（px）、语言（中文 | 英文 | 日文 | 西班牙语）。
- 「语言」默认随控制台当前语言注入；若需求描述中用户明确要求展示语言（如「用英文展示界面」），前端会覆盖 JSON「语言」。生成时**仅以 JSON「语言」**决定 mount 内静态文案语种，禁止 ctx.i18n。
- 「语言」仅约束**展示文案**（列名、loading/空态/错误、图例、标签等），**不**翻译接口数据：API 域名、ctx.utils.usageFormat / quotaList 的 *Text 等保持运行时原样。
- 用户 JSON 中的「宽」「高」是内容区参考尺寸（px），用于把握信息密度（字号、间距、层次），不表示固定行数或必须用某种布局。
- 须等 request 返回后再确定展示体量：先把结果整理为待展示项 items（指标、列表项、图系列点等），再决定如何呈现。
- 三态必做：请求前简短 loading；成功后再绘制；失败 .catch 可读文案；禁止未处理 Promise。
- 无（0）：一块清晰的空态文案，在内容区内视觉平衡（可水平垂直居中），不占大面积假表格/假图表。
- 少：突出关键信息，整体在视口内匀称、居中或均衡留白均可；可用卡片式分区、图+数、环形/水球/仪表、左右分栏等组合，勿缩成固定窄宽小条 floating 在中间。
- 多：以可扫读为先：分区、分组、主次层级、必要时局部滚动；避免一屏塞满无层次；禁止用空节点凑格子。
- 表现形式：可灵活组合文字列表、卡片式区块（仅用 style 分区，勿再套大背景卡）、进度/占比条、分隔线、ctx.utils.renderChart 等；宽而扁可并列，高而窄可纵向信息流。
- 图表走 ctx.utils.renderChart；数值走 ctx.utils.usageFormat 或 quotaList 的 usedText/limitText。禁止外链图、CDN 插图、虚构序列；无数据不画假图。
- 单项缺失：行/点显示「-」，不导致整页空白。
- 用户未要求全部列出/对比/排行时，勿把长数组默认全部摊平；先收敛到与 ctx.scope、数据范围一致的展示主体，再选用上述表现形式。

## 数据与接口（先认「形状」，勿臆造 JSON）
- 默认只用形状1；用户明确要配额/告警/资源列表时才用形状2或3。
- 父页面已注入 ctx.utils 助手；有助手必须用 ctx.utils，禁止自创字段名。

### 形状1：用量字典（优先，NumberCard / Ring）
- GET /v2/rpc/usages/general-usage（主）；按需 image-usage / identity-usage / k8s-usage
- params 必含 scope: ctx.scope、refresh: true；可选 cloud_env、brand、range_type、range_id；数据范围可带 domain_id / project_id
- 返回扁平对象，用 usage_key 取数（不是数组，不是 resource_type 列表）
- usage_key 按 ctx.scope：system → all.* / hosts / baremetals 等；domain → domain.*；project → servers、servers.cpu、disks 等无前缀键
- 概览常用：数量 servers/hosts/buckets；容量 *.cpu、*.memory、*.disk（按 scope 加前缀）；占比用成对键如 running_servers vs servers
- 展示一律 ctx.utils.usageFormat({ value, unit })；禁止手搓 GB/MB 或 String 裸数字

### 形状2：配额行表（Quota 磁贴；必须 ctx.utils.quotaList）
- GET /v2/rpc/quotas/quota-list（或 region_quotas、image_quotas 资源前缀，路径 /v2/rpc/{quotas|region_quotas|image_quotas}/quota-list）；params：scope: ctx.scope（域管理员可加 primary=true&domain_id）
- 返回数组：每行一个域/项目；字段 count/cpu/memory/storage/… 为上限，已用为 usage.{field} 或 usage 对象 — 不是 usage_key，不是 limit/used 通用 REST
- 单域九宫格：ctx.utils.quotaList.pickRow(data) → expandRow(row) 得数组，用 label、usedText、limitText 渲染
- 多域对比：normalizeList(data) + rowTitle(row)；指标用 expandRow(row,['count','cpu','memory','storage']) 后 items.find(i=>i.key==='cpu')

### 形状3：其他次要 GET（非默认，无专用 ctx 助手时慎写）
- GET /v1/alertrecords/total-alert：类型→数量的对象（对标 UnrecoveredAlarm），非 usage_key
- GET /v1/servers、/v1/hosts 等：分页资源列表，项为资源对象，勿当用量字典
- 禁止 GET /v1/usages；监控 unifiedmonitors 多为 POST 查询，AI 磁贴勿手写
- 用户未明确要求时不要调用形状3

## 严禁（违反会导致服务端拒绝，优先避免 innerHTML）
- eval、new Function、动态 import()、Worker / SharedWorker / ServiceWorker、WebSocket
- 直接 fetch / axios / XMLHttpRequest（数据只用 ctx.request）
- document.cookie、localStorage、sessionStorage、indexedDB
- 访问 parent / top / opener
- 任何 HTML 注入：.innerHTML、.outerHTML、outerHTML、insertAdjacentHTML、document.write、document.writeln
- setTimeout/setInterval 传入字符串
- 常见错误示例（不要生成）：root.innerHTML = '<motion something>'; list.innerHTML += '<div>...</div>'; container.outerHTML = html;

## 示例骨架（用量概览）
` + "```javascript\n" + `function mount(root, ctx) {
  const scope = ctx.scope || 'project';
  const isDark = ctx.theme === 'dark';
  const labelColor = isDark ? 'rgba(255,255,255,0.65)' : 'rgba(0,0,0,0.45)';
  root.style.width = '100%';
  root.style.height = '100%';
  root.style.boxSizing = 'border-box';
  root.style.display = 'flex';
  root.style.flexDirection = 'column';
  root.style.alignItems = 'stretch';
  root.style.justifyContent = 'flex-start';
  root.style.background = 'transparent';
  root.style.color = isDark ? 'rgba(255,255,255,0.88)' : 'rgba(0,0,0,0.88)';
  root.style.overflow = 'auto';
  const rows = scope === 'system'
    ? [
      { key: 'all.servers', label: '虚拟机', unit: '台' },
      { key: 'all.servers.cpu', label: 'CPU', unit: '核' },
      { key: 'all.servers.memory', label: '内存', unit: 'M' },
      { key: 'all.servers.disk', label: '磁盘', unit: 'M' }
    ]
    : (scope === 'domain'
      ? [
        { key: 'domain.servers', label: '虚拟机', unit: '台' },
        { key: 'domain.servers.cpu', label: 'CPU', unit: '核' },
        { key: 'domain.servers.memory', label: '内存', unit: 'M' },
        { key: 'domain.disks', label: '磁盘', unit: 'M' }
      ]
      : [
        { key: 'servers', label: '虚拟机', unit: '台' },
        { key: 'servers.cpu', label: 'CPU', unit: '核' },
        { key: 'servers.memory', label: '内存', unit: 'M' },
        { key: 'disks', label: '磁盘', unit: 'M' }
      ]);
  const fmt = (ctx.utils && typeof ctx.utils.usageFormat === 'function')
    ? ctx.utils.usageFormat
    : function (v) { return String(v.value); };
  const list = document.createElement('div');
  list.style.display = 'flex';
  list.style.flexDirection = 'column';
  list.style.gap = '8px';
  list.style.width = '100%';
  list.style.flex = '1';
  root.appendChild(list);
  ctx.request('GET', '/v2/rpc/usages/general-usage', { scope: scope, refresh: true }).then(function (data) {
    data = data || {};
    rows.forEach(function (item) {
      const row = document.createElement('div');
      row.style.display = 'flex';
      row.style.justifyContent = 'space-between';
      row.style.alignItems = 'center';
      row.style.width = '100%';
      row.style.fontSize = '13px';
      row.style.lineHeight = '20px';
      const labelEl = document.createElement('span');
      labelEl.textContent = item.label;
      labelEl.style.color = labelColor;
      const valEl = document.createElement('span');
      valEl.style.fontWeight = '600';
      const val = data[item.key];
      valEl.textContent = (val === undefined || val === null)
        ? '-'
        : fmt({ value: val, unit: item.unit || '' });
      row.appendChild(labelEl);
      row.appendChild(valEl);
      list.appendChild(row);
    });
  }).catch(function (err) {
    const el = document.createElement('div');
    el.textContent = String(err);
    list.appendChild(el);
  });
}
` + "```\n\n" + `## 示例（柱状图 renderChart）
` + "```javascript\n" + `function mount(root, ctx) {
  const scope = ctx.scope || 'project';
  root.style.width = '100%';
  root.style.height = '100%';
  root.style.display = 'flex';
  root.style.flexDirection = 'column';
  root.style.background = 'transparent';
  const chartHost = document.createElement('div');
  chartHost.style.width = '100%';
  chartHost.style.flex = '1';
  chartHost.style.minHeight = '140px';
  root.appendChild(chartHost);
  const keys = scope === 'system'
    ? [
      { key: 'all.servers', label: '虚拟机' },
      { key: 'all.running_servers', label: '运行中' },
      { key: 'all.ready_servers', label: '关机' }
    ]
    : [
      { key: 'servers', label: '虚拟机' },
      { key: 'running_servers', label: '运行中' },
      { key: 'ready_servers', label: '关机' }
    ];
  ctx.request('GET', '/v2/rpc/usages/general-usage', { scope: scope, refresh: true }).then(function (data) {
    data = data || {};
    const categories = [];
    const values = [];
    keys.forEach(function (item) {
      categories.push(item.label);
      const v = data[item.key];
      values.push(v === undefined || v === null ? 0 : Number(v) || 0);
    });
    if (ctx.utils && typeof ctx.utils.renderChart === 'function') {
      ctx.utils.renderChart(chartHost, {
        type: 'bar',
        categories: categories,
        series: [{ name: '数量', data: values }]
      });
    }
  }).catch(function (err) {
    chartHost.textContent = String(err);
  });
}
` + "```"
)

// DashboardPanelChatMessage is optional multi-turn context for generate.
//
// swagger:model
type DashboardPanelChatMessage struct {
	// 角色：user / assistant
	Role string `json:"role"`
	// 消息内容
	Content string `json:"content"`
}

// DashboardPanelGenerateInput is the body for POST /dashboard_panels/generate.
//
// swagger:model
type DashboardPanelGenerateInput struct {
	// 自然语言需求（必填）
	// required: true
	Prompt string `json:"prompt"`
	// 可选。system|domain|project；省略则生成代码跟随前端视图的 ctx.scope
	// example: project
	Scope string `json:"scope,omitempty"`
	// 可选面板标题
	Title string `json:"title,omitempty"`
	// 可选多轮上下文
	History []DashboardPanelChatMessage `json:"history,omitempty"`
}

// DashboardPanelGenerateOutput is returned without persisting to yunionconf.
//
// swagger:model
type DashboardPanelGenerateOutput struct {
	// 生成并校验通过的 JS 源码（含 mount）
	Source string `json:"source"`
	// 入口函数名，固定为 mount
	// example: mount
	Entry string `json:"entry"`
	// 面板标题
	Title string `json:"title,omitempty"`
	// 作用域
	Scope string `json:"scope,omitempty"`
	// 源码 SHA256
	ContentSha256 string `json:"content_sha256"`
	// 允许的只读 API 列表，如 GET /v2/rpc/usages/general-usage
	ApiAllowlist []string `json:"api_allowlist,omitempty"`
	// 非致命提示
	Warnings []string `json:"warnings,omitempty"`
	// 原始模型文本（调试用，通常为空）
	RawModelText string `json:"raw_model_text,omitempty"`
}

// DashboardPanelValidateInput is the body for POST /dashboard_panels/validate.
//
// swagger:model
type DashboardPanelValidateInput struct {
	// JS 源码
	// required: true
	Source string `json:"source"`
	// 可选 api_allowlist；为空则使用默认只读前缀
	ApiAllowlist []string `json:"api_allowlist,omitempty"`
}

// DashboardPanelValidateOutput reports static validation result.
//
// swagger:model
type DashboardPanelValidateOutput struct {
	// 是否通过校验
	Ok bool `json:"ok"`
	// 入口名
	Entry string `json:"entry,omitempty"`
	// 源码 SHA256
	ContentSha256 string `json:"content_sha256,omitempty"`
	// 规范化后的 allowlist
	ApiAllowlist []string `json:"api_allowlist,omitempty"`
	// 警告
	Warnings []string `json:"warnings,omitempty"`
	// 失败原因（ok=false 时）
	Error string `json:"error,omitempty"`
}

// DashboardPanelLayoutHint optionally places the widget into an existing overview panel.
//
// swagger:model
type DashboardPanelLayoutHint struct {
	X    int `json:"x"`
	Y    int `json:"y"`
	W    int `json:"w"`
	H    int `json:"h"`
	MinH int `json:"minH,omitempty"`
}

// DashboardPanelSaveInput is the body for POST /dashboard_panels/save.
// Source is re-validated server-side; generate never writes.
//
// swagger:model
type DashboardPanelSaveInput struct {
	// 最终确认的 JS 源码（必填，服务端会再校验）
	// required: true
	Source string `json:"source"`
	// 可选。system|domain|project；省略表示运行时跟随前端视图 ctx.scope
	Scope string `json:"scope,omitempty"`
	// 标题
	Title string `json:"title,omitempty"`
	// 原始自然语言需求摘要（审计）
	Prompt string `json:"prompt,omitempty"`
	// 已有 parameter 名则更新，否则自动生成 dashboard-custom-script-{uuid}
	Name string `json:"name,omitempty"`
	// 可选 allowlist
	ApiAllowlist []string `json:"api_allowlist,omitempty"`
	// 可选：合并进该 overview panel 的 parameter 名
	PanelParameter string `json:"panel_parameter,omitempty"`
	// 可选：布局项 id
	ItemId string `json:"item_id,omitempty"`
	// 可选布局坐标
	Layout *DashboardPanelLayoutHint `json:"layout,omitempty"`
}

// DashboardPanelSaveOutput is returned after writing yunionconf parameter(s).
//
// swagger:model
type DashboardPanelSaveOutput struct {
	// 写入的 parameter 名
	Name string `json:"name"`
	// parameter id
	ParameterId string `json:"parameter_id,omitempty"`
	// 入口名
	Entry string `json:"entry"`
	// 源码 SHA256
	ContentSha256 string `json:"content_sha256"`
	// allowlist
	ApiAllowlist []string `json:"api_allowlist,omitempty"`
	// 若合并了 panel
	PanelParameter string `json:"panel_parameter,omitempty"`
	// 布局项 id
	ItemId string `json:"item_id,omitempty"`
	// 警告
	Warnings []string `json:"warnings,omitempty"`
}

// DashboardCustomScriptValue is stored in yunionconf parameter.value.
//
// swagger:model
type DashboardCustomScriptValue struct {
	// 版本号
	Version int `json:"version"`
	// 标题
	Title string `json:"title,omitempty"`
	// 语言，固定 javascript
	Language string `json:"language"`
	// 入口 mount
	Entry string `json:"entry"`
	// JS 源码
	Source string `json:"source"`
	// SHA256
	ContentSha256 string `json:"content_sha256"`
	// allowlist
	ApiAllowlist []string `json:"api_allowlist,omitempty"`
	// 需求摘要
	Prompt string `json:"prompt,omitempty"`
	// 作用域
	Scope string `json:"scope,omitempty"`
	// 校验时间 RFC3339
	ValidatedAt string `json:"validated_at,omitempty"`
}
