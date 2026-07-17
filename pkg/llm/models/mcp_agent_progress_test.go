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

package models

import (
	"strings"
	"testing"
)

func TestSummarizeToolProgressRegionList(t *testing.T) {
	result := `{
  "total": 2,
  "data": [
    {"id": "r1", "name": "北京", "external_id": "cn-beijing"},
    {"id": "r2", "name": "上海", "external_id": "cn-shanghai"}
  ]
}
[MCP下一步] ignore`
	got := summarizeToolProgress("climc_cloud_region_list", nil, result, nil)
	if !strings.Contains(got, "已找到区域") || !strings.Contains(got, "北京") {
		t.Fatalf("got %q", got)
	}
	if strings.Contains(got, "正在调用工具") {
		t.Fatalf("should not mention tool name style: %q", got)
	}
}

func TestSummarizeToolProgressCreate(t *testing.T) {
	args := map[string]interface{}{
		"name":          "ubuntu-22-04",
		"hypervisor":    "aliyun",
		"prefer-region": "r1",
		"instance-type": "ecs.t6-c1m1.large",
		"disk":          []interface{}{"size=30g,image=img1,backend=cloud_essd"},
	}
	result := `{"server_id":"s1","final_status":"running","server":{"id":"s1","status":"running"}}`
	got := summarizeToolProgress("climc_server_create", args, result, nil)
	if !strings.Contains(got, "选用资源创建虚拟机") || !strings.Contains(got, "ubuntu-22-04") {
		t.Fatalf("got %q", got)
	}
	if !strings.Contains(got, "状态=running") {
		t.Fatalf("expected final status in %q", got)
	}
}

func TestSummarizeToolProgressCapability(t *testing.T) {
	labels := newProgressLabelCache()
	labels.regions["reg-1"] = "华东1（杭州）"
	result := `{"storage_types2":{"aliyun":["cloud_essd/ssd","cloud_ssd/ssd"]}}`
	got := summarizeToolProgress("climc_cloud_region_capability", map[string]interface{}{"id": "reg-1"}, result, labels)
	if !strings.Contains(got, "cloud_essd") || !strings.Contains(got, "华东1（杭州）") {
		t.Fatalf("got %q", got)
	}
	if strings.Contains(got, "reg-1") {
		t.Fatalf("should show region name not id: %q", got)
	}
}
