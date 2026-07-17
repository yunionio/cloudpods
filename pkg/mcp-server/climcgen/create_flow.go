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
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/fileutils"
	"yunion.io/x/pkg/util/regutils"

	"yunion.io/x/onecloud/cmd/climc/shell"
	"yunion.io/x/onecloud/pkg/apis/compute"
	schedapi "yunion.io/x/onecloud/pkg/apis/scheduler"
	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	schedmodules "yunion.io/x/onecloud/pkg/mcclient/modules/scheduler"
	computeoptions "yunion.io/x/onecloud/pkg/mcclient/options/compute"
	"yunion.io/x/onecloud/pkg/mcp-server/options"
)

const (
	serverCreatePollInterval = 5 * time.Second
)

func cloneArgs(src map[string]interface{}) map[string]interface{} {
	dst := make(map[string]interface{}, len(src)+1)
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func argLookup(args map[string]interface{}, keys ...string) (interface{}, bool) {
	normalize := func(k string) string {
		return strings.ReplaceAll(k, "_", "-")
	}
	for _, key := range keys {
		if v, ok := args[key]; ok {
			return v, true
		}
		alt := strings.ReplaceAll(key, "-", "_")
		if v, ok := args[alt]; ok {
			return v, true
		}
		want := normalize(key)
		for k, v := range args {
			if normalize(k) == want {
				return v, true
			}
		}
	}
	return nil, false
}

func firstString(v interface{}) string {
	parts := valueToArgvParts(v)
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}

// mapCreateArgsToForecastArgs 将 server-create 参数映射为 scheduler-forecast CLI 参数。
func mapCreateArgsToForecastArgs(args map[string]interface{}) map[string]interface{} {
	out := map[string]interface{}{}
	// SchedulerForecastOptions 嵌入 ServerConfigs，CLI token 与 create 侧大多一致
	copyKeys := []struct {
		from []string
		to   string
	}{
		{[]string{"disk"}, "disk"},
		{[]string{"net"}, "net"},
		{[]string{"region", "prefer-region"}, "region"},
		{[]string{"zone", "prefer-zone"}, "zone"},
		{[]string{"host", "prefer-host"}, "host"},
		{[]string{"manager", "prefer-manager"}, "manager"},
		{[]string{"hypervisor"}, "hypervisor"},
		{[]string{"project", "tenant"}, "project"},
		{[]string{"count"}, "count"},
		{[]string{"schedtag"}, "schedtag"},
		{[]string{"ncpu"}, "ncpu"},
		{[]string{"cdrom", "iso"}, "cdrom"},
		{[]string{"sku", "instance-type"}, "sku"},
	}
	for _, item := range copyKeys {
		if v, ok := argLookup(args, item.from...); ok {
			out[item.to] = v
		}
	}
	// 无 sku 时，把 mem-spec（如 2048M/2G）转成 forecast 的 --mem（MB）
	if _, hasSku := out["sku"]; !hasSku {
		if v, ok := argLookup(args, "mem-spec"); ok {
			spec := firstString(v)
			if regutils.MatchSize(spec) {
				if mb, err := fileutils.GetSizeMb(spec, 'M', 1024); err == nil && mb > 0 {
					out["mem"] = strconv.Itoa(mb)
				}
			} else if n, err := strconv.Atoi(strings.TrimSpace(spec)); err == nil && n > 0 {
				out["mem"] = strconv.Itoa(n)
			}
		}
	}
	return out
}

func findCommand(name string) (shell.CMD, bool) {
	for _, cmd := range shell.CommandTable {
		if cmd.Command == name {
			return cmd, true
		}
	}
	return shell.CMD{}, false
}

func runSchedulerForecast(session *mcclient.ClientSession, createArgs map[string]interface{}) (jsonutils.JSONObject, error) {
	cmd, ok := findCommand("scheduler-forecast")
	if !ok {
		return nil, fmt.Errorf("scheduler-forecast command not registered; import climc shell/scheduler")
	}
	forecastArgs := mapCreateArgsToForecastArgs(createArgs)

	parser, optPtr, err := newArgumentParser(cmd)
	if err != nil {
		return nil, err
	}
	argv := mcpArgsToArgv(parser, forecastArgs)
	if err := parser.ParseArgs(argv, false); err != nil {
		return nil, fmt.Errorf("parse scheduler-forecast args: %w (argv=%v mapped=%v)", err, argv, forecastArgs)
	}
	filled := parser.Options()
	opts, ok := filled.(*computeoptions.SchedulerForecastOptions)
	if !ok {
		if o, ok2 := optPtr.(*computeoptions.SchedulerForecastOptions); ok2 {
			opts = o
		} else {
			return nil, fmt.Errorf("unexpected scheduler-forecast options type %T", filled)
		}
	}
	input, err := opts.Params(session)
	if err != nil {
		return nil, fmt.Errorf("build forecast input: %w", err)
	}
	prepareForecastInput(input)
	return schedmodules.SchedManager.DoForecast(session, input.JSON(input))
}

// prepareForecastInput 修正公有云 forecast 入参，避免被当成 KVM 去跑 host_cpu/host_memory。
// 公有云驱动 DoScheduleCPUFilter/MemoryFilter=false；但若 hypervisor/provider 对不上，
// GetHypervisorDriver() 为 nil，host_cpu 仍会执行，而阿里云宿主机是虚拟的，cpu total/free 常为 0。
func prepareForecastInput(input *schedapi.ScheduleInput) {
	if input == nil {
		return
	}
	if input.ServerConfigs == nil {
		input.ServerConfigs = &compute.ServerConfigs{}
	}
	hv := strings.ToLower(strings.TrimSpace(input.Hypervisor))
	if !isManagedHypervisor(hv) {
		return
	}
	if prov := providerFromHypervisor(hv); prov != "" {
		if input.Provider == "" || input.Provider == compute.CLOUD_PROVIDER_ONECLOUD {
			input.Provider = prov
		}
	}
	// 公有云按套餐调度：清掉 climc forecast Options 的默认 ncpu=1/mem=512，
	// 防止 driver 解析失败时 host_cpu 用默认核数把所有宿主机滤掉。
	if strings.TrimSpace(input.InstanceType) != "" {
		input.Ncpu = 0
		input.Memory = 0
	}
}

func isManagedHypervisor(hv string) bool {
	switch hv {
	case compute.HYPERVISOR_KVM, compute.HYPERVISOR_BAREMETAL, compute.HYPERVISOR_POD,
		"hypervisor", "":
		return false
	default:
		return hv != ""
	}
}

func providerFromHypervisor(hv string) string {
	switch strings.ToLower(hv) {
	case compute.HYPERVISOR_ALIYUN:
		return compute.CLOUD_PROVIDER_ALIYUN
	case compute.HYPERVISOR_AWS:
		return compute.CLOUD_PROVIDER_AWS
	case compute.HYPERVISOR_AZURE:
		return compute.CLOUD_PROVIDER_AZURE
	case compute.HYPERVISOR_QCLOUD:
		return compute.CLOUD_PROVIDER_QCLOUD
	case compute.HYPERVISOR_HUAWEI:
		return compute.CLOUD_PROVIDER_HUAWEI
	case compute.HYPERVISOR_GOOGLE:
		return compute.CLOUD_PROVIDER_GOOGLE
	case compute.HYPERVISOR_OPENSTACK:
		return compute.CLOUD_PROVIDER_OPENSTACK
	default:
		// 多数公有云 hypervisor 与 provider 仅大小写不同
		if hv == "" {
			return ""
		}
		return strings.ToUpper(hv[:1]) + hv[1:]
	}
}

// readyForecastCandidates 返回 forecast 中 error 为空的候选（与调度历史 Result.candidates 成功语义一致）。
func readyForecastCandidates(forecast jsonutils.JSONObject) []jsonutils.JSONObject {
	if forecast == nil {
		return nil
	}
	arr, err := forecast.GetArray("candidates")
	if err != nil || len(arr) == 0 {
		return nil
	}
	ready := make([]jsonutils.JSONObject, 0, len(arr))
	for _, c := range arr {
		errStr, _ := c.GetString("error")
		if strings.TrimSpace(errStr) != "" {
			continue
		}
		ready = append(ready, c)
	}
	return ready
}

// forecastSucceeded 判断预调度是否成功：优先看合格 candidates，兼容 can_create。
func forecastSucceeded(forecast jsonutils.JSONObject) error {
	if forecast == nil {
		return fmt.Errorf("scheduler-forecast returned empty result")
	}
	ready := readyForecastCandidates(forecast)
	reqCount, err := forecast.Int("req_count")
	if err != nil || reqCount <= 0 {
		reqCount = 1
	}
	if int64(len(ready)) >= reqCount {
		return nil
	}
	if jsonutils.QueryBoolean(forecast, "can_create", false) {
		return nil
	}
	reasons, _ := forecast.Get("not_allow_reasons")
	allow, _ := forecast.Int("allow_count")
	return fmt.Errorf(
		"scheduler-forecast not schedulable: ready_candidates=%d allow_count=%d req_count=%d can_create=false reasons=%s\nforecast=%s",
		len(ready), allow, reqCount, reasons, forecast.String(),
	)
}

func extractJSONObject(s string) json.RawMessage {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	var obj interface{}
	if err := json.Unmarshal([]byte(s), &obj); err == nil {
		b, _ := json.Marshal(obj)
		return b
	}
	b, _ := json.Marshal(map[string]string{"raw": s})
	return b
}

func extractServerID(createOut string) string {
	createOut = strings.TrimSpace(createOut)
	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(createOut), &obj); err != nil {
		return ""
	}
	if id, ok := obj["id"].(string); ok {
		return id
	}
	return ""
}

func isTerminalSuccessStatus(status string) bool {
	switch status {
	case compute.VM_RUNNING, compute.VM_READY:
		return true
	default:
		return false
	}
}

func isTerminalFailStatus(status string) bool {
	switch status {
	case compute.VM_CREATE_FAILED,
		compute.VM_SCHEDULE_FAILED,
		compute.VM_DEPLOY_FAILED,
		compute.VM_START_FAILED,
		compute.VM_DISK_FAILED,
		compute.VM_NETWORK_FAILED,
		compute.VM_DEVICE_FAILED,
		compute.VM_UNKNOWN:
		return true
	default:
		return strings.Contains(status, "fail")
	}
}

func waitServerRunningOrReady(ctx context.Context, session *mcclient.ClientSession, id string) (string, error) {
	deadline := time.Now().Add(options.ServerCreateWaitDuration())
	var lastStatus string
	for time.Now().Before(deadline) {
		if err := ctx.Err(); err != nil {
			return lastStatus, err
		}
		obj, err := modules.Servers.Get(session, id, nil)
		if err != nil {
			log.Warningf("wait server %s: get failed: %s", id, err)
			select {
			case <-ctx.Done():
				return lastStatus, ctx.Err()
			case <-time.After(serverCreatePollInterval):
			}
			continue
		}
		lastStatus, _ = obj.GetString("status")
		if isTerminalSuccessStatus(lastStatus) {
			return lastStatus, nil
		}
		if isTerminalFailStatus(lastStatus) {
			return lastStatus, fmt.Errorf("server %s entered failed status %q", id, lastStatus)
		}
		select {
		case <-ctx.Done():
			return lastStatus, ctx.Err()
		case <-time.After(serverCreatePollInterval):
		}
	}
	return lastStatus, fmt.Errorf("timeout waiting server %s to become running/ready, last status=%q", id, lastStatus)
}

func diskHasBackend(disk string) bool {
	for _, part := range strings.Split(disk, ",") {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(kv[0]))
		if (key == "backend" || key == "storage_type") && strings.TrimSpace(kv[1]) != "" {
			return true
		}
	}
	return false
}

func preferDiskBackend(types []string) string {
	if len(types) == 0 {
		return ""
	}
	prefer := []string{"cloud_essd", "cloud_ssd", "cloud_efficiency", "cloud", "local"}
	lower := make([]string, len(types))
	for i, t := range types {
		lower[i] = strings.ToLower(strings.TrimSpace(t))
	}
	for _, p := range prefer {
		for i, t := range lower {
			if t == p {
				return types[i]
			}
		}
	}
	return types[0]
}

func storageTypesFromCapability(capa jsonutils.JSONObject, hypervisor string) []string {
	if capa == nil {
		return nil
	}
	hv := strings.ToLower(strings.TrimSpace(hypervisor))
	asStrings := func(obj jsonutils.JSONObject) []string {
		if obj == nil {
			return nil
		}
		if arr, ok := obj.(*jsonutils.JSONArray); ok {
			out := arr.GetStringArray()
			if len(out) > 0 {
				return out
			}
		}
		if s, err := obj.GetString(); err == nil && strings.TrimSpace(s) != "" {
			return []string{s}
		}
		return nil
	}
	for _, key := range []string{"storage_types2", "StorageTypes2"} {
		m, err := capa.GetMap(key)
		if err != nil || len(m) == 0 {
			continue
		}
		if hv != "" {
			for k, v := range m {
				if strings.ToLower(k) == hv {
					if arr := asStrings(v); len(arr) > 0 {
						return arr
					}
				}
			}
		}
		for _, v := range m {
			if arr := asStrings(v); len(arr) > 0 {
				return arr
			}
		}
	}
	return nil
}

// ensureDiskBackend 公有云 disk 缺 backend 时，按 prefer-region 拉 cloud-region-capability 自动补全。
func ensureDiskBackend(session *mcclient.ClientSession, args map[string]interface{}) {
	var hv string
	if v, ok := argLookup(args, "hypervisor"); ok {
		hv = firstString(v)
	}
	if !isManagedHypervisor(strings.ToLower(hv)) {
		return
	}
	rawDisk, ok := argLookup(args, "disk")
	if !ok {
		return
	}
	disks := valueToArgvParts(rawDisk)
	if len(disks) == 0 {
		return
	}
	need := false
	for _, d := range disks {
		if !diskHasBackend(d) {
			need = true
			break
		}
	}
	if !need {
		return
	}
	var regionID string
	if v, ok := argLookup(args, "prefer-region", "region"); ok {
		regionID = firstString(v)
	}
	if regionID == "" {
		log.Warningf("disk missing backend but prefer-region empty; skip auto-fill")
		return
	}
	capa, err := modules.Cloudregions.GetSpecific(session, regionID, "capability", nil)
	if err != nil {
		log.Warningf("cloud-region-capability %s failed: %s", regionID, err)
		return
	}
	backend := preferDiskBackend(storageTypesFromCapability(capa, hv))
	if backend == "" {
		log.Warningf("cloud-region-capability %s has no storage_types2 for %s", regionID, hv)
		return
	}
	for i, d := range disks {
		if !diskHasBackend(d) {
			disks[i] = strings.TrimSuffix(d, ",") + ",backend=" + backend
		}
	}
	args["disk"] = disks
	log.Infof("auto-filled disk backend=%s from region %s capability", backend, regionID)
}

// ensureNetworkAutoSched：未指定网络时注入 CLI "random"（ParseNetworkConfig → Exit=false），
// 等价 API nets:[{"exit":false}]，由调度器自动选网。
func ensureNetworkAutoSched(args map[string]interface{}) {
	raw, ok := argLookup(args, "net", "nets")
	if ok {
		parts := valueToArgvParts(raw)
		if len(parts) > 0 {
			return
		}
	}
	args["net"] = []string{"random"}
	delete(args, "nets")
	log.Infof("auto-filled net=random (API equivalent nets:[{exit:false}]) for network auto schedule")
}

// ensureGenerateName：默认开启 generate-name，用 NAME 作为模板自动去重，避免 DuplicateNameError。
func ensureGenerateName(args map[string]interface{}) {
	if _, ok := argLookup(args, "generate-name", "generate_name", "GenerateName"); ok {
		return
	}
	args["generate-name"] = true
	log.Infof("auto-enabled generate-name to avoid DuplicateNameError")
}

// handleServerCreate：创建前 scheduler-forecast 预调度，创建后等待 running 或 ready（关机）。
// dry-run 仅做参数校验，不是预调度。等待超时不视为失败：返回 server_id 供 agent 继续查询。
func (t *ClimcTool) handleServerCreate(ctx context.Context, session *mcclient.ClientSession, args map[string]interface{}) (string, error) {
	ensureDiskBackend(session, args)
	ensureNetworkAutoSched(args)
	ensureGenerateName(args)

	// 1) 预调度（scheduler-forecast）
	forecast, err := runSchedulerForecast(session, args)
	if err != nil {
		return "", fmt.Errorf("scheduler-forecast failed: %w", err)
	}
	if err := forecastSucceeded(forecast); err != nil {
		return "", err
	}
	forecastRaw := json.RawMessage(forecast.String())

	// 2) 真实创建（去掉 dry-run，避免走 suggestion 旁路）
	createArgs := cloneArgs(args)
	delete(createArgs, "dry-run")
	delete(createArgs, "dry_run")
	delete(createArgs, "DryRun")
	createOut, err := invokeCommand(t.cmd, session, createArgs)
	if err != nil {
		return "", fmt.Errorf("create failed after scheduler-forecast ok: %w\nforecast:\n%s\noutput:\n%s", err, forecast.String(), createOut)
	}

	serverID := extractServerID(createOut)
	result := map[string]interface{}{
		"preschedule": forecastRaw,
		"server":      json.RawMessage(extractJSONObject(createOut)),
	}
	if serverID == "" {
		result["wait_error"] = "create succeeded but server id missing; skip wait"
		b, _ := json.Marshal(result)
		return string(b), nil
	}

	// 3) 等待 running / ready
	status, waitErr := waitServerRunningOrReady(ctx, session, serverID)
	result["final_status"] = status
	result["server_id"] = serverID
	if waitErr != nil {
		result["wait_error"] = waitErr.Error()
		if obj, gerr := modules.Servers.Get(session, serverID, nil); gerr == nil {
			result["server"] = json.RawMessage(obj.String())
			if status == compute.VM_SCHEDULE_FAILED || status == "sched_fail" {
				if progress, _ := obj.GetString("progress"); progress != "" {
					result["schedule_hint"] = progress
				}
				result["hint"] = "调度失败(sched_fail)。若未指定网络，应使用自动调度 net=[\"random\"]（等价 nets:[{exit:false}]）；也可检查 prefer-region、instance-type、disk.backend 是否与区域能力匹配后重试。"
			}
		}
		// 超时/取消：创建已成功，返回 server_id 让 agent 用 climc_server_show 继续查，不把整次 tool 判失败
		if isWaitTimeoutOrCanceled(waitErr) {
			result["wait_pending"] = true
			if result["hint"] == nil {
				result["hint"] = "创建已提交但尚未进入 running/ready。请用 climc_server_show 查询 status，勿重复创建。"
			}
			b, _ := json.Marshal(result)
			return string(b), nil
		}
		b, _ := json.Marshal(result)
		return string(b), waitErr
	}

	if obj, gerr := modules.Servers.Get(session, serverID, nil); gerr == nil {
		result["server"] = json.RawMessage(obj.String())
	}
	b, err := json.Marshal(result)
	if err != nil {
		return createOut, nil
	}
	return string(b), nil
}

func isWaitTimeoutOrCanceled(err error) bool {
	if err == nil {
		return false
	}
	if err == context.Canceled || err == context.DeadlineExceeded {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "timeout waiting server") || strings.Contains(msg, "context canceled") || strings.Contains(msg, "context deadline")
}
