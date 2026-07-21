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
	loggermodules "yunion.io/x/onecloud/pkg/mcclient/modules/logger"
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
		{[]string{"provider"}, "provider"},
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
	case compute.HYPERVISOR_CAS:
		return compute.CLOUD_PROVIDER_CAS
	case compute.HYPERVISOR_H3C:
		return compute.CLOUD_PROVIDER_H3C
	case compute.HYPERVISOR_CNWARE:
		return compute.CLOUD_PROVIDER_CNWARE
	case compute.HYPERVISOR_PROXMOX:
		return compute.CLOUD_PROVIDER_PROXMOX
	case compute.HYPERVISOR_SANGFOR:
		return compute.CLOUD_PROVIDER_SANGFOR
	case compute.HYPERVISOR_UIS:
		return compute.CLOUD_PROVIDER_UIS
	case compute.HYPERVISOR_ZETTAKIT:
		return compute.CLOUD_PROVIDER_ZETTAKIT
	default:
		// 多数公有云 hypervisor 与 provider 仅大小写不同
		if hv == "" {
			return ""
		}
		return strings.ToUpper(hv[:1]) + hv[1:]
	}
}

// ensureCreateProvider：私有云/公有云创建时注入 provider（与 dashboard GenCreateData 一致），
// 避免 forecast 把 CAS 等当成 OneCloud/KVM 调度。
func ensureCreateProvider(args map[string]interface{}) {
	var hv string
	if v, ok := argLookup(args, "hypervisor"); ok {
		hv = strings.ToLower(firstString(v))
	}
	if !isManagedHypervisor(hv) {
		return
	}
	prov := providerFromHypervisor(hv)
	if prov == "" {
		return
	}
	if v, ok := argLookup(args, "provider"); ok {
		cur := firstString(v)
		if cur != "" && !strings.EqualFold(cur, compute.CLOUD_PROVIDER_ONECLOUD) {
			return
		}
	}
	args["provider"] = prov
	log.Infof("auto-filled provider=%s from hypervisor=%s", prov, hv)
}

// isoOnlyPrivateHypervisor：dashboard 中 CAS/UIS/SangFor 仅支持 private_iso，系统盘不挂 image_id。
func isoOnlyPrivateHypervisor(hv string) bool {
	switch strings.ToLower(hv) {
	case compute.HYPERVISOR_CAS, compute.HYPERVISOR_UIS, compute.HYPERVISOR_SANGFOR:
		return true
	default:
		return false
	}
}

// ensureCasStyleCdrom：CAS 类 hypervisor 若 disk 带 image= 且未设 cdrom，则挪到 cdrom 并从 disk 去掉 image，
// 对齐前端「ISO → cdrom、系统盘仅 size+backend」。
func ensureCasStyleCdrom(args map[string]interface{}) {
	var hv string
	if v, ok := argLookup(args, "hypervisor"); ok {
		hv = firstString(v)
	}
	if !isoOnlyPrivateHypervisor(hv) {
		return
	}
	if v, ok := argLookup(args, "cdrom", "iso"); ok && firstString(v) != "" {
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
	imgID := diskImageID(disks[0])
	if imgID == "" {
		return
	}
	args["cdrom"] = imgID
	disks[0] = stripDiskImage(disks[0])
	args["disk"] = disks
	log.Infof("CAS-style create: moved disk image=%s to cdrom", imgID)
}

func diskImageID(disk string) string {
	for _, part := range strings.Split(disk, ",") {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) == 2 && (kv[0] == "image" || kv[0] == "image_id") {
			return strings.TrimSpace(kv[1])
		}
	}
	return ""
}

func stripDiskImage(disk string) string {
	parts := strings.Split(disk, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) >= 1 && (kv[0] == "image" || kv[0] == "image_id") {
			continue
		}
		if strings.TrimSpace(part) != "" {
			out = append(out, strings.TrimSpace(part))
		}
	}
	return strings.Join(out, ",")
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

// collectCreateFailDiagnostics 创建失败时拉取操作日志与关键失败字段，便于 agent 直接复述原因。
func collectCreateFailDiagnostics(session *mcclient.ClientSession, serverID string) map[string]interface{} {
	diag := map[string]interface{}{}
	if session == nil || strings.TrimSpace(serverID) == "" {
		return diag
	}

	params := jsonutils.NewDict()
	params.Set("show_fail_reason", jsonutils.JSONTrue)
	if obj, err := modules.Servers.Get(session, serverID, params); err == nil {
		for _, key := range []string{
			"status", "progress", "progress_desc", "host", "zone", "region",
			"instance_type", "hypervisor", "external_id",
		} {
			if v, _ := obj.GetString(key); v != "" {
				diag[key] = v
			}
		}
	} else {
		log.Warningf("collect fail diagnostics: server-show %s: %s", serverID, err)
	}

	logs, reason := listServerActionFailNotes(session, serverID, 15)
	if len(logs) > 0 {
		diag["action_logs"] = logs
	}
	if reason != "" {
		diag["fail_reason"] = reason
	}
	return diag
}

func listServerActionFailNotes(session *mcclient.ClientSession, serverID string, limit int64) ([]map[string]interface{}, string) {
	if limit <= 0 {
		limit = 15
	}
	params := jsonutils.NewDict()
	params.Set("obj_id", jsonutils.NewString(serverID))
	params.Set("obj_type", jsonutils.NewStringArray([]string{"server"}))
	params.Set("limit", jsonutils.NewInt(limit))
	params.Set("scope", jsonutils.NewString("system"))
	params.Set("order", jsonutils.NewString("desc"))

	list, err := loggermodules.Actions.List(session, params)
	if err != nil {
		log.Warningf("collect fail diagnostics: action-list %s: %s", serverID, err)
		return nil, ""
	}

	out := make([]map[string]interface{}, 0)
	var primaryReason string
	for _, raw := range list.Data {
		action, _ := raw.GetString("action")
		success, _ := raw.Bool("success")
		notes, _ := raw.GetString("notes")
		opsTime, _ := raw.GetString("ops_time")
		if success && !looksLikeFailNote(notes) && !strings.Contains(action, "fail") {
			// 成功日志里偶尔带失败痕迹（如 update_status 到 deploy_fail），仍保留含 fail 的
			if !strings.Contains(strings.ToLower(notes), "fail") && !strings.Contains(notes, "__reason__") {
				continue
			}
		}
		entry := map[string]interface{}{
			"action":   action,
			"success":  success,
			"ops_time": opsTime,
		}
		if notes != "" {
			entry["notes"] = truncateRunes(notes, 800)
			if r := extractActionFailReason(notes); r != "" && primaryReason == "" {
				primaryReason = r
				entry["reason"] = r
			}
		}
		out = append(out, entry)
		if len(out) >= 8 {
			break
		}
	}
	return out, primaryReason
}

func looksLikeFailNote(notes string) bool {
	n := strings.ToLower(notes)
	return strings.Contains(n, "__reason__") || strings.Contains(n, "error") || strings.Contains(n, "fail")
}

func extractActionFailReason(notes string) string {
	notes = strings.TrimSpace(notes)
	if notes == "" {
		return ""
	}
	// notes 可能是纯 JSON，或 "statusA=>statusB: {json}"
	jsonPart := notes
	if i := strings.Index(notes, "{"); i >= 0 {
		jsonPart = notes[i:]
	}
	if obj, err := jsonutils.ParseString(jsonPart); err == nil {
		if r, _ := obj.GetString("__reason__"); r != "" {
			return r
		}
		if r, _ := obj.GetString("reason"); r != "" {
			return r
		}
		if r, _ := obj.GetString("message"); r != "" {
			return r
		}
	}
	if looksLikeFailNote(notes) {
		return truncateRunes(notes, 400)
	}
	return ""
}

func truncateRunes(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	rs := []rune(s)
	if len(rs) <= max {
		return s
	}
	return string(rs[:max]) + "…"
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
	ensureCreateProvider(args)
	ensureCasStyleCdrom(args)
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
	forecastSummary := summarizeForecast(forecast)

	// 2) 真实创建（去掉 dry-run，避免走 suggestion 旁路）
	createArgs := cloneArgs(args)
	delete(createArgs, "dry-run")
	delete(createArgs, "dry_run")
	delete(createArgs, "DryRun")
	createOut, err := invokeCommand(t.cmd, session, createArgs)
	if err != nil {
		return "", fmt.Errorf("create failed after scheduler-forecast ok: %w\nforecast:\n%s\noutput:\n%s", err, forecastSummary, createOut)
	}

	serverID := extractServerID(createOut)
	result := map[string]interface{}{
		"preschedule": json.RawMessage(forecastSummary),
		"server":      summarizeServerJSON(string(extractJSONObject(createOut))),
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
			result["server"] = summarizeServerObject(obj)
			if status == compute.VM_SCHEDULE_FAILED || status == "sched_fail" {
				if progress, _ := obj.GetString("progress"); progress != "" {
					result["schedule_hint"] = progress
				}
				result["hint"] = "调度失败(sched_fail)。若未指定网络，应使用自动调度 net=[\"random\"]（等价 nets:[{exit:false}]）；也可检查 prefer-region、instance-type、disk.backend 是否与区域能力匹配后重试。"
			}
		}
		if isTerminalFailStatus(status) {
			if diag := collectCreateFailDiagnostics(session, serverID); len(diag) > 0 {
				result["fail_diagnostics"] = diag
				if reason, _ := diag["fail_reason"].(string); reason != "" {
					result["fail_reason"] = reason
					if result["hint"] == nil {
						result["hint"] = "创建失败。fail_reason / fail_diagnostics.action_logs 含平台操作日志原因；可再调 climc_action_show（type=server, id=<server_id>, fail=true）核对。"
					}
				}
			}
		}
		// 超时/取消：创建已提交，返回 server_id 让 agent 用 climc_server_show 继续查，不把整次 tool 判失败
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
		result["server"] = summarizeServerObject(obj)
	}
	b, err := json.Marshal(result)
	if err != nil {
		return createOut, nil
	}
	return string(b), nil
}

// summarizeForecast 压缩预调度结果，避免 SSE 大包被截断。
func summarizeForecast(forecast jsonutils.JSONObject) string {
	if forecast == nil {
		return "{}"
	}
	out := jsonutils.NewDict()
	for _, key := range []string{"allow_count", "req_count", "can_create"} {
		if forecast.Contains(key) {
			if v, err := forecast.Get(key); err == nil {
				out.Set(key, v)
			}
		}
	}
	if cands, err := forecast.GetArray("candidates"); err == nil {
		out.Set("candidate_count", jsonutils.NewInt(int64(len(cands))))
		brief := make([]jsonutils.JSONObject, 0, 3)
		for i, c := range cands {
			if i >= 3 {
				break
			}
			item := jsonutils.NewDict()
			for _, k := range []string{"id", "name", "host_id", "host_name", "zone_id"} {
				if c.Contains(k) {
					if v, err := c.Get(k); err == nil {
						item.Set(k, v)
					}
				}
			}
			brief = append(brief, item)
		}
		if len(brief) > 0 {
			out.Set("candidates_brief", jsonutils.NewArray(brief...))
		}
	}
	if filtered, err := forecast.GetArray("filtered_candidates"); err == nil && len(filtered) > 0 {
		out.Set("filtered_count", jsonutils.NewInt(int64(len(filtered))))
		reasons := make([]string, 0, 5)
		seen := map[string]struct{}{}
		for _, f := range filtered {
			r, _ := f.GetString("filter_name")
			if r == "" {
				r, _ = f.GetString("reason")
			}
			if r == "" {
				continue
			}
			if _, ok := seen[r]; ok {
				continue
			}
			seen[r] = struct{}{}
			reasons = append(reasons, r)
			if len(reasons) >= 5 {
				break
			}
		}
		if len(reasons) > 0 {
			out.Set("filter_reasons", jsonutils.NewStringArray(reasons))
		}
	}
	return out.String()
}

func summarizeServerJSON(raw string) map[string]interface{} {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return map[string]interface{}{}
	}
	obj, err := jsonutils.ParseString(raw)
	if err != nil {
		return map[string]interface{}{"raw": truncateRunes(raw, 200)}
	}
	return summarizeServerObject(obj)
}

func summarizeServerObject(obj jsonutils.JSONObject) map[string]interface{} {
	out := map[string]interface{}{}
	if obj == nil {
		return out
	}
	for _, key := range []string{
		"id", "name", "status", "hypervisor", "host", "zone", "region",
		"ips", "eip", "instance_type", "vcpu_count", "vmem_size",
		"billing_type", "external_id", "os_type", "progress",
	} {
		if v, err := obj.GetString(key); err == nil && v != "" {
			out[key] = v
			continue
		}
		if n, err := obj.Int(key); err == nil {
			out[key] = n
		}
	}
	return out
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
