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
	"strings"
	"testing"

	"yunion.io/x/jsonutils"

	_ "yunion.io/x/onecloud/cmd/climc/shell/scheduler"
	"yunion.io/x/onecloud/pkg/apis/compute"
	schedapi "yunion.io/x/onecloud/pkg/apis/scheduler"
)

func TestMapCreateArgsToForecastArgs(t *testing.T) {
	got := mapCreateArgsToForecastArgs(map[string]interface{}{
		"name":          "mcp-test",
		"disk":          []interface{}{"size=40g,image=img-1"},
		"net":           []interface{}{"net-1"},
		"prefer-region": "reg-1",
		"hypervisor":    "aliyun",
		"ncpu":          4,
		"instance-type": "ecs.c7t.xlarge",
		"mem-spec":      "8G", // 有 sku 时不应再映射 mem
		"dry-run":       true,
	})
	if got["region"] != "reg-1" {
		t.Fatalf("region=%v", got["region"])
	}
	if got["sku"] != "ecs.c7t.xlarge" {
		t.Fatalf("sku=%v", got["sku"])
	}
	if _, ok := got["mem"]; ok {
		t.Fatalf("mem should be omitted when sku present, got %v", got["mem"])
	}
	if _, ok := got["dry-run"]; ok {
		t.Fatalf("dry-run must not be forwarded to forecast")
	}

	got = mapCreateArgsToForecastArgs(map[string]interface{}{
		"ncpu":     2,
		"mem-spec": "2048M",
		"disk":     []string{"size=30g,image=x"},
		"net":      []string{"n1"},
		"region":   "cn-beijing",
	})
	if got["mem"] != "2048" {
		t.Fatalf("mem=%v want 2048", got["mem"])
	}
	if got["region"] != "cn-beijing" {
		t.Fatalf("region=%v", got["region"])
	}
}

func TestSchedulerForecastCommandRegistered(t *testing.T) {
	cmd, ok := findCommand("scheduler-forecast")
	if !ok {
		t.Fatal("scheduler-forecast not in CommandTable; imports.go must blank-import shell/scheduler")
	}
	if cmd.Options == nil {
		t.Fatal("scheduler-forecast Options is nil")
	}
}

func TestPrepareForecastInputManagedCloud(t *testing.T) {
	input := &schedapi.ScheduleInput{}
	input.ServerConfigs = &compute.ServerConfigs{
		Hypervisor:   compute.HYPERVISOR_ALIYUN,
		InstanceType: "ecs.c7t.xlarge",
		Provider:     compute.CLOUD_PROVIDER_ONECLOUD, // 错误默认
	}
	input.Ncpu = 2
	input.Memory = 2048
	prepareForecastInput(input)
	if input.Provider != compute.CLOUD_PROVIDER_ALIYUN {
		t.Fatalf("provider=%q want Aliyun", input.Provider)
	}
	if input.Ncpu != 0 || input.Memory != 0 {
		t.Fatalf("managed+sku should clear default ncpu/mem, got ncpu=%d mem=%d", input.Ncpu, input.Memory)
	}
}

func TestForecastSucceededByCandidates(t *testing.T) {
	okForecast := jsonutils.Marshal(map[string]interface{}{
		"can_create": true,
		"req_count":  1,
		"candidates": []map[string]interface{}{
			{"host_id": "h1", "name": "qx-aliyun-cn-beijing-i", "error": ""},
		},
	})
	if err := forecastSucceeded(okForecast); err != nil {
		t.Fatalf("expected success: %v", err)
	}
	failForecast := jsonutils.Marshal(map[string]interface{}{
		"can_create":        false,
		"req_count":         1,
		"candidates":        nil,
		"not_allow_reasons": []string{"Out of resource"},
	})
	if err := forecastSucceeded(failForecast); err == nil {
		t.Fatal("expected failure")
	}
}

func TestPreferDiskBackend(t *testing.T) {
	if got := preferDiskBackend([]string{"cloud", "cloud_essd", "cloud_ssd"}); got != "cloud_essd" {
		t.Fatalf("prefer=%q", got)
	}
	capa := jsonutils.Marshal(map[string]interface{}{
		"storage_types2": map[string][]string{
			"aliyun": {"cloud_efficiency", "cloud_ssd"},
		},
	})
	types := storageTypesFromCapability(capa, "aliyun")
	if preferDiskBackend(types) != "cloud_ssd" {
		t.Fatalf("types=%v prefer=%q", types, preferDiskBackend(types))
	}
	if diskHasBackend("size=30g,image=x") {
		t.Fatal("should not have backend")
	}
	if !diskHasBackend("size=30g,image=x,backend=cloud_essd") {
		t.Fatal("should have backend")
	}
}

func TestEnsureNetworkAutoSched(t *testing.T) {
	args := map[string]interface{}{"name": "t"}
	ensureNetworkAutoSched(args)
	parts := valueToArgvParts(args["net"])
	if len(parts) != 1 || parts[0] != "random" {
		t.Fatalf("expected net=[random], got %v", args["net"])
	}

	args = map[string]interface{}{"net": []interface{}{}}
	ensureNetworkAutoSched(args)
	parts = valueToArgvParts(args["net"])
	if len(parts) != 1 || parts[0] != "random" {
		t.Fatalf("empty net should become random, got %v", args["net"])
	}

	args = map[string]interface{}{"net": []string{"net-abc"}}
	ensureNetworkAutoSched(args)
	parts = valueToArgvParts(args["net"])
	if len(parts) != 1 || parts[0] != "net-abc" {
		t.Fatalf("explicit net must be kept, got %v", args["net"])
	}
}

func TestServerCreateDescriptionMentionsForecast(t *testing.T) {
	cmd, ok := findCommand("server-create")
	if !ok {
		t.Fatal("server-create not registered")
	}
	desc := buildDescription(cmd)
	if !strings.Contains(desc, "scheduler-forecast") {
		t.Fatalf("mcp-desc should mention scheduler-forecast, got %q", desc)
	}
	if strings.Contains(desc, "dry-run 预调度") {
		t.Fatalf("mcp-desc must not claim dry-run is preschedule: %q", desc)
	}
}
