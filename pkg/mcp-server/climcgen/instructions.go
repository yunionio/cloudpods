// Copyright 2019 Yunion
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

package climcgen

import (
	"fmt"
	"strings"

	"yunion.io/x/onecloud/pkg/mcp-server/options"
)

// serverInstructionsTemplate 返回给 MCP 客户端的全局使用说明模板，%s 为 PlatformName。
// 细则写在本说明；各工具 mcp-desc 保持一行摘要，避免重复占 token。
const serverInstructionsTemplate = `%s MCP（climc tools）使用规则：

认证：initialize / tools/list 可匿名；tools/call 必须带 Header（X-Auth-Token，或 AK+SK，或 X-API-Key=base64(ak:sk)），不要在工具参数传密钥。

0. 严禁空口编造：未实际调用 climc_* 并拿到返回前，禁止声称“已查到/已创建成功”。
1. *-list 只用于准备参数，不等于任务完成；同一轮可并行多个查询。
2. 创建虚拟机（连续到 climc_server_create）：
   a) climc_cloud_region_list（公有云须 provider；创建时 usable=true）
   b) climc_cloud_region_capability → 取 storage_types2 作 disk.backend
   c) climc_image_list（KVM）或 climc_cached_image_list（公有云，须 provider+region=区域 id）
   d) climc_server_sku_list（口语 2c2g 用 spec=\"2c2g\"）或直接 ncpu/mem
   e) climc_server_create（name、disk 含 image+backend、规格；公有云 hypervisor+prefer-region）。net 可省略自动调度。工具会 forecast 预调度并等待 running/ready；若返回 wait_pending，用 climc_server_show 继续查，勿重复创建。
3. 启停/重启/删除/重置密码/改配/挂盘/绑 EIP：climc_server_list 定位 id 后立刻调用对应操作工具。
4. 监控指标用 climc_monitor_unifiedmonitor_query；climc_server_monitor 是 QEMU HMP/QMP，不是指标。
5. 缺参只追问真正缺失项；已有 id 直接下一步。
`

// BuildServerInstructions 用 platformName（BaseOptions.PlatformName）生成 MCP ServerInstructions。
// platformName 为空时使用 options.ResolvedPlatformName()。
func BuildServerInstructions(platformName string) string {
	name := strings.TrimSpace(platformName)
	if name == "" {
		name = options.ResolvedPlatformName()
	}
	return fmt.Sprintf(serverInstructionsTemplate, name)
}

// isCreateFlowListCommand：创建流程中间查询工具（Options mcp-desc 含该标记）。
func isCreateFlowListCommand(opt interface{}) bool {
	return strings.Contains(collectMcpDesc(opt), "创建流程中的中间步骤")
}
