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

package pod

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"github.com/opencontainers/runtime-spec/specs-go"

	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

type TCgroupController string

const (
	CGROUP_PATH_SYSFS = "/sys/fs/cgroup"

	CgroupControllerMemory TCgroupController = "memory"
)

type CgroupUtil interface {
	SetMemoryLimitBytes(ctrId string, bytes int64) error
	SetCPUCfs(ctrId string, quota int64, period int64) error
	SetDevicesAllow(ctrId string, allows []string) error
	SetPidsMax(ctrId string, max int) error
	SetCpusetCloneChildren(ctrId string) error
	SetCgroupKeyValue(ctrId string, ctrler TCgroupController, key, value string) error
}

type podCgroupV1Util struct {
	parentPath string
}

func NewPodCgroupV1Util(parentPath string) CgroupUtil {
	return &podCgroupV1Util{
		parentPath: parentPath,
	}
}

func (p podCgroupV1Util) getContainerControllerPath(controller string, ctrId string) string {
	return filepath.Join(CGROUP_PATH_SYSFS, controller, p.parentPath, ctrId)
}

func (p podCgroupV1Util) getContainerCGFilePath(controller string, ctrId string, filename string) string {
	return filepath.Join(p.getContainerControllerPath(controller, ctrId), filename)
}

func (p podCgroupV1Util) write(fp string, content string) error {
	cmd := fmt.Sprintf("echo %q > %s", content, fp)
	out, err := procutils.NewRemoteCommandAsFarAsPossible("sh", "-c", cmd).Output()
	if err != nil {
		return errors.Wrapf(err, "%s: %s", cmd, out)
	}
	return nil
}

func (p podCgroupV1Util) SetMemoryLimitBytes(ctrId string, bytes int64) error {
	memFp := p.getContainerCGFilePath("memory", ctrId, "memory.limit_in_bytes")
	return p.write(memFp, fmt.Sprintf("%d", bytes))
}

func (p podCgroupV1Util) SetCgroupKeyValue(ctrId string, ctrler TCgroupController, key, value string) error {
	memFp := p.getContainerCGFilePath(string(ctrler), ctrId, key)
	return p.write(memFp, value)
}

func (p podCgroupV1Util) SetCPUCfs(ctrId string, quota int64, period int64) error {
	quotaFp := p.getContainerCGFilePath("cpu,cpuacct", ctrId, "cpu.cfs_quota_us")
	periodFp := p.getContainerCGFilePath("cpu,cpuacct", ctrId, "cpu.cfs_period_us")
	if err := p.write(quotaFp, fmt.Sprintf("%d", quota)); err != nil {
		return errors.Wrapf(err, "write quota: %d", quota)
	}
	if err := p.write(periodFp, fmt.Sprintf("%d", period)); err != nil {
		return errors.Wrapf(err, "write period: %d", period)
	}
	return nil
}

func (p podCgroupV1Util) SetDevicesAllow(ctrId string, allows []string) error {
	devicesFp := p.getContainerCGFilePath("devices", ctrId, "devices.allow")
	for _, allowStr := range allows {
		if err := p.write(devicesFp, allowStr); err != nil {
			return errors.Wrapf(err, "write: %s", allowStr)
		}
	}
	return nil
}

func (p podCgroupV1Util) SetPidsMax(ctrId string, max int) error {
	pidFp := p.getContainerCGFilePath("pids", ctrId, "pids.max")
	return p.write(pidFp, fmt.Sprintf("%d", max))
}

func (p podCgroupV1Util) SetCpusetCloneChildren(ctrId string) error {
	ccFp := p.getContainerCGFilePath("cpuset", ctrId, "cgroup.clone_children")
	return p.write(ccFp, "1")
}

type podCgroupV2Util struct {
	parentPath string
}

func NewPodCgroupV2Util(parentPath string) CgroupUtil {
	return &podCgroupV2Util{
		parentPath: parentPath,
	}
}

// DetectCgroupVersion 检测当前系统的 cgroup 版本
// 返回 true 表示 cgroup v2，false 表示 cgroup v1
func DetectCgroupVersion() (bool, error) {
	cgroupPath := CGROUP_PATH_SYSFS
	if !fileutils2.Exists(cgroupPath) {
		// 如果 /sys/fs/cgroup 不存在，尝试 /cgroup
		cgroupPath = "/cgroup"
		if !fileutils2.Exists(cgroupPath) {
			return false, errors.Errorf("cgroup path not found")
		}
	}

	// 使用 stat 命令检测文件系统类型
	// cgroup v2 的文件系统类型是 "cgroup2fs"
	output, err := procutils.NewCommand("stat", "-fc", "%T", cgroupPath).Output()
	if err != nil {
		return false, errors.Wrapf(err, "stat cgroup path %s", cgroupPath)
	}
	cgroupfs := strings.TrimSpace(string(output))
	return cgroupfs == "cgroup2fs", nil
}

// NewPodCgroupUtil 根据系统自动检测 cgroup 版本并返回相应的实现
func NewPodCgroupUtil(parentPath string) (CgroupUtil, error) {
	isV2, err := DetectCgroupVersion()
	if err != nil {
		// 如果检测失败，默认使用 v1（向后兼容）
		return NewPodCgroupV1Util(parentPath), nil
	}
	if isV2 {
		return NewPodCgroupV2Util(parentPath), nil
	}
	return NewPodCgroupV1Util(parentPath), nil
}

func (p podCgroupV2Util) getContainerPath(ctrId string) string {
	return filepath.Join(CGROUP_PATH_SYSFS, p.parentPath, ctrId)
}

func (p podCgroupV2Util) getContainerCGFilePath(ctrId string, filename string) string {
	return filepath.Join(p.getContainerPath(ctrId), filename)
}

func (p podCgroupV2Util) write(fp string, content string) error {
	cmd := fmt.Sprintf("echo %q > %s", content, fp)
	out, err := procutils.NewRemoteCommandAsFarAsPossible("sh", "-c", cmd).Output()
	if err != nil {
		return errors.Wrapf(err, "%s: %s", cmd, out)
	}
	return nil
}

func (p podCgroupV2Util) getParentPath() string {
	return filepath.Join(CGROUP_PATH_SYSFS, p.parentPath)
}

func (p podCgroupV2Util) ensureController(ctrId string, controller string) error {
	containerPath := p.getContainerPath(ctrId)
	// 确保 cgroup 目录存在
	cmd := fmt.Sprintf("mkdir -p %s", containerPath)
	out, err := procutils.NewRemoteCommandAsFarAsPossible("sh", "-c", cmd).Output()
	if err != nil {
		return errors.Wrapf(err, "create cgroup directory: %s", out)
	}

	// 首先检查当前 cgroup 的 controllers，看是否已经启用了该控制器
	controllersFp := filepath.Join(containerPath, "cgroup.controllers")
	cmd = fmt.Sprintf("cat %s 2>/dev/null || echo ''", controllersFp)
	out, err = procutils.NewRemoteCommandAsFarAsPossible("sh", "-c", cmd).Output()
	if err != nil {
		return errors.Wrapf(err, "read cgroup.controllers: %s", out)
	}
	currentControllers := strings.TrimSpace(string(out))

	// 如果当前 cgroup 已经启用了该控制器，直接返回
	if containsController(currentControllers, controller) {
		return nil
	}

	// 根据文档，需要在父 cgroup 的 subtree_control 中启用控制器
	// 这样当前 cgroup 才能使用该控制器
	parentPath := p.getParentPath()

	// 确保父 cgroup 目录存在
	cmd = fmt.Sprintf("mkdir -p %s", parentPath)
	out, err = procutils.NewRemoteCommandAsFarAsPossible("sh", "-c", cmd).Output()
	if err != nil {
		return errors.Wrapf(err, "create parent cgroup directory: %s", out)
	}

	parentSubtreeControlFp := filepath.Join(parentPath, "cgroup.subtree_control")

	// 读取父 cgroup 的 subtree_control
	cmd = fmt.Sprintf("cat %s 2>/dev/null || echo ''", parentSubtreeControlFp)
	out, err = procutils.NewRemoteCommandAsFarAsPossible("sh", "-c", cmd).Output()
	if err != nil {
		return errors.Wrapf(err, "read parent subtree_control: %s", out)
	}
	parentSubtreeControl := strings.TrimSpace(string(out))

	// 如果父 cgroup 的 subtree_control 中没有该控制器，添加它
	if !containsController(parentSubtreeControl, controller) {
		cmd = fmt.Sprintf("echo +%s > %s", controller, parentSubtreeControlFp)
		out, err = procutils.NewRemoteCommandAsFarAsPossible("sh", "-c", cmd).Output()
		if err != nil {
			return errors.Wrapf(err, "enable controller %s in parent cgroup: %s", controller, out)
		}
	}

	// 再次检查当前 cgroup 的 controllers，确认控制器已启用
	cmd = fmt.Sprintf("cat %s 2>/dev/null || echo ''", controllersFp)
	out, err = procutils.NewRemoteCommandAsFarAsPossible("sh", "-c", cmd).Output()
	if err != nil {
		return errors.Wrapf(err, "read cgroup.controllers after enable: %s", out)
	}
	finalControllers := strings.TrimSpace(string(out))
	if !containsController(finalControllers, controller) {
		return errors.Errorf("controller %s not available in cgroup %s after enabling in parent. Available controllers: %s", controller, containerPath, finalControllers)
	}

	return nil
}

func containsController(controllers string, controller string) bool {
	// controllers 格式类似 "cpu memory pids" (空格分隔)
	controllers = strings.TrimSpace(controllers)
	if len(controllers) == 0 {
		return false
	}
	for _, c := range strings.Fields(controllers) {
		if strings.TrimSpace(c) == controller {
			return true
		}
	}
	return false
}

func (p podCgroupV2Util) SetMemoryLimitBytes(ctrId string, bytes int64) error {
	if err := p.ensureController(ctrId, "memory"); err != nil {
		return errors.Wrap(err, "ensure memory controller")
	}
	// cgroup v2 使用 memory.max
	memFp := p.getContainerCGFilePath(ctrId, "memory.max")
	return p.write(memFp, fmt.Sprintf("%d", bytes))
}

func (p podCgroupV2Util) SetCPUCfs(ctrId string, quota int64, period int64) error {
	if err := p.ensureController(ctrId, "cpu"); err != nil {
		return errors.Wrap(err, "ensure cpu controller")
	}
	// cgroup v2 使用 cpu.max，格式为 "quota period" 或 "max" 表示无限制
	// 如果 quota 为 -1，表示无限制
	var cpuMaxValue string
	if quota == -1 {
		cpuMaxValue = "max"
	} else {
		cpuMaxValue = fmt.Sprintf("%d %d", quota, period)
	}
	cpuFp := p.getContainerCGFilePath(ctrId, "cpu.max")
	return p.write(cpuFp, cpuMaxValue)
}

// normalizePermissions 规范化权限字符串，将任意顺序的权限（如 "rmw"）转换为标准格式 "rwm"
// DevicePermissions.IsValid() 要求权限字符串必须是 "rwm" 的标准顺序
func normalizePermissions(permStr string) string {
	var hasR, hasW, hasM bool
	for _, c := range permStr {
		switch c {
		case 'r':
			hasR = true
		case 'w':
			hasW = true
		case 'm':
			hasM = true
		}
	}
	var normalized strings.Builder
	if hasR {
		normalized.WriteRune('r')
	}
	if hasW {
		normalized.WriteRune('w')
	}
	if hasM {
		normalized.WriteRune('m')
	}
	return normalized.String()
}

// GetDeviceAllowRuleFromPath 从设备路径获取设备号并生成设备规则字符串
// 返回格式: "c 226:128 rwm" 或 "b 8:0 rwm"
// devicePath: 设备路径，如 "/dev/dri/renderD128"
// permissions: 权限字符串，如 "rwm" 或 "rmw"（会被规范化）
func GetDeviceAllowRuleFromPath(devicePath string, permissions string) (string, error) {
	// 获取设备信息（使用 unix.Stat 避免类型转换问题）
	var stat unix.Stat_t
	if err := unix.Stat(devicePath, &stat); err != nil {
		return "", errors.Wrapf(err, "stat device: %s", devicePath)
	}

	// 获取设备号
	major := unix.Major(uint64(stat.Rdev))
	minor := unix.Minor(uint64(stat.Rdev))

	// 判断设备类型（通过 mode 判断）
	var devType string
	mode := stat.Mode
	if mode&unix.S_IFCHR != 0 {
		devType = "c" // 字符设备
	} else if mode&unix.S_IFBLK != 0 {
		devType = "b" // 块设备
	} else {
		return "", errors.Errorf("not a device file: %s (mode: %o)", devicePath, mode)
	}

	// 规范化权限字符串
	normalizedPerms := normalizePermissions(permissions)
	if normalizedPerms == "" {
		normalizedPerms = "rwm" // 默认权限
	}

	// 生成设备规则字符串
	ruleStr := fmt.Sprintf("%s %d:%d %s", devType, major, minor, normalizedPerms)
	return ruleStr, nil
}

// parseDeviceRule 解析设备规则字符串，格式如 "c 13:* rwm" 或 "b 8:0 rwm"
// 格式: <type> <major>:<minor> <permissions>
// type: 'c' (char), 'b' (block), 'a' (all)
// major/minor: 数字或 '*' 表示通配符
// permissions: 'r' (read), 'w' (write), 'm' (mknod) 的组合
func parseDeviceRule(ruleStr string) (*specs.LinuxDeviceCgroup, error) {
	parts := strings.Fields(ruleStr)
	if len(parts) != 3 {
		return nil, errors.Errorf("invalid device rule format: %s, expected format: <type> <major>:<minor> <permissions>", ruleStr)
	}

	// 解析设备类型
	var devType string
	switch parts[0] {
	case "c":
		devType = "c"
	case "b":
		devType = "b"
	case "a":
		devType = "a"
	default:
		return nil, errors.Errorf("invalid device type: %s, must be 'c', 'b', or 'a'", parts[0])
	}

	// 解析 major:minor
	majorMinor := strings.Split(parts[1], ":")
	if len(majorMinor) != 2 {
		return nil, errors.Errorf("invalid major:minor format: %s", parts[1])
	}

	var major, minor int64 = -1, -1
	if majorMinor[0] != "*" {
		var err error
		major, err = strconv.ParseInt(majorMinor[0], 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid major number: %s", majorMinor[0])
		}
	}
	if majorMinor[1] != "*" {
		var err error
		minor, err = strconv.ParseInt(majorMinor[1], 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid minor number: %s", majorMinor[1])
		}
	}

	// 解析权限：先规范化权限字符串为标准格式 "rwm"
	normalizedPerms := normalizePermissions(parts[2])
	if normalizedPerms == "" {
		normalizedPerms = "rwm" // 默认权限
	}

	return &specs.LinuxDeviceCgroup{
		Type:   devType,
		Major:  &major,
		Minor:  &minor,
		Access: normalizedPerms,
		Allow:  true, // devices.allow 表示允许
	}, nil
}

// ConvertDeviceRulesToSpecsDevices 将设备规则字符串和容器设备配置转换为 specs.LinuxDeviceCgroup 列表
// 用于通过 containerd API 更新 container spec 中的 devices
func ConvertDeviceRulesToSpecsDevices(allows []string) ([]*specs.LinuxDeviceCgroup, error) {
	// 解析设备规则
	deviceRules := make([]*specs.LinuxDeviceCgroup, 0, len(allows))
	for _, allowStr := range allows {
		rule, err := parseDeviceRule(allowStr)
		if err != nil {
			return nil, errors.Wrapf(err, "parse device rule: %s", allowStr)
		}
		deviceRules = append(deviceRules, rule)
	}

	return deviceRules, nil
}

func (p podCgroupV2Util) SetDevicesAllow(ctrId string, allows []string) error {
	if len(allows) == 0 {
		return nil
	}
	log.Warningf("=========skip set devices allow for container %s", ctrId)
	return nil

	/*log.Infof("[SetDevicesAllow] Setting device allow rules for container %s: %v", ctrId, allows)

	// 解析设备规则
	deviceRules := make([]*configs.DeviceRule, 0, len(allows))
	for _, allowStr := range allows {
		rule, err := parseDeviceRule(allowStr)
		if err != nil {
			return errors.Wrapf(err, "parse device rule: %s", allowStr)
		}
		deviceRules = append(deviceRules, rule)
		log.Infof("[SetDevicesAllow] Parsed rule: type=%v, major=%d, minor=%d, permissions=%s, allow=%v",
			rule.Type, rule.Major, rule.Minor, rule.Permissions, rule.Allow)
	}

	// 转换为 specs.LinuxDeviceCgroup
	specDevices := make([]specs.LinuxDeviceCgroup, 0, len(deviceRules))
	for _, rule := range deviceRules {
		specDevices = append(specDevices, convertDeviceRuleToSpec(rule))
	}

	// 使用 containerd 的 DeviceFilter 函数生成 eBPF 设备过滤程序
	insts, license, err := devicefilter.DeviceFilter(specDevices)
	if err != nil {
		return errors.Wrap(err, "generate eBPF device filter program using containerd DeviceFilter")
	}

	// 获取 cgroup 路径
	containerPath := p.getContainerPath(ctrId)
	log.Infof("[SetDevicesAllow] Using cgroup path: %s, specDevices: %s", containerPath, jsonutils.Marshal(specDevices).PrettyString())

	// 打开 cgroup 目录（参考 containerd 实现）
	dirFD, err := unix.Open(containerPath, unix.O_DIRECTORY|unix.O_RDONLY|unix.O_CLOEXEC, 0o600)
	if err != nil {
		return errors.Wrapf(err, "cannot get dir FD for %s", containerPath)
	}
	defer unix.Close(dirFD)

	// 加载并附加 eBPF 程序（参考 containerd 的 setDevices 逻辑）
	if _, err := LoadAttachCgroupDeviceFilter(insts, license, dirFD); err != nil {
		if !canSkipEBPFError(specDevices) {
			return errors.Wrap(err, "load and attach eBPF device filter")
		}
		log.Warningf("[SetDevicesAllow] Failed to attach eBPF device filter, but error can be skipped: %v", err)
	}

	log.Infof("[SetDevicesAllow] Successfully set device allow rules for container %s", ctrId)*/
	return nil
}

func (p podCgroupV2Util) SetPidsMax(ctrId string, max int) error {
	if err := p.ensureController(ctrId, "pids"); err != nil {
		return errors.Wrap(err, "ensure pids controller")
	}
	// cgroup v2 和 v1 都使用 pids.max
	pidFp := p.getContainerCGFilePath(ctrId, "pids.max")
	return p.write(pidFp, fmt.Sprintf("%d", max))
}

func (p podCgroupV2Util) SetCpusetCloneChildren(ctrId string) error {
	// cgroup v2 不支持 cgroup.clone_children
	// 在 v2 中，子 cgroup 会自动继承父 cgroup 的 cpuset 配置
	// 这个操作在 v2 中是 no-op，但为了接口兼容性，我们不做任何操作
	return nil
}

func (p podCgroupV2Util) SetCgroupKeyValue(ctrId string, ctrler TCgroupController, key, value string) error {
	// 对于通用键值设置，需要确保相应的控制器已启用
	controller := string(ctrler)
	if controller != "" {
		if err := p.ensureController(ctrId, controller); err != nil {
			return errors.Wrapf(err, "ensure %s controller", controller)
		}
	}
	// cgroup v2 中，文件路径直接在统一路径下
	fp := p.getContainerCGFilePath(ctrId, key)
	return p.write(fp, value)
}

// checkEbpfSupport 检查系统是否支持 eBPF
func (p podCgroupV2Util) checkEbpfSupport() error {
	// 检查内核版本（需要 >= 4.15）
	// 这里只做基本检查，详细的版本检查可能需要解析 /proc/version
	// 实际的内核版本检查在 eBPF 库加载时会进行

	// 检查 /sys/fs/bpf 是否存在（如果使用 pinning）
	if !fileutils2.Exists("/sys/fs/bpf") {
		log.Debugf("[checkEbpfSupport] /sys/fs/bpf does not exist (may be normal if not using pinning)")
	}

	// 检查 memlock 限制
	cmd := "ulimit -l"
	out, err := procutils.NewRemoteCommandAsFarAsPossible("sh", "-c", cmd).Output()
	if err == nil {
		memlock := strings.TrimSpace(string(out))
		if memlock != "unlimited" {
			log.Debugf("[checkEbpfSupport] memlock limit: %s (recommended: unlimited)", memlock)
		}
	}

	return nil
}

// verifyEbpfAttached 验证 eBPF 程序是否成功附加到 cgroup
func (p podCgroupV2Util) verifyEbpfAttached(cgroupPath string) error {
	// 方法1: 尝试使用 bpftool 检查（如果可用）
	// bpftool cgroup tree /sys/fs/cgroup/<path>
	cmd := fmt.Sprintf("bpftool cgroup tree %s 2>/dev/null | grep -q device || echo 'not found'", cgroupPath)
	out, err := procutils.NewRemoteCommandAsFarAsPossible("sh", "-c", cmd).Output()
	if err == nil {
		output := strings.TrimSpace(string(out))
		if output != "not found" {
			log.Debugf("[verifyEbpfAttached] Verified eBPF program attached via bpftool")
			return nil
		}
		log.Debugf("[verifyEbpfAttached] bpftool did not find device filter (may be normal)")
	} else {
		log.Debugf("[verifyEbpfAttached] bpftool not available or failed: %v", err)
	}

	// 方法2: 检查是否有进程在该 cgroup 中
	// 如果有进程，可以尝试实际测试设备访问
	cgroupProcsPath := filepath.Join(cgroupPath, "cgroup.procs")
	if fileutils2.Exists(cgroupProcsPath) {
		cmd := fmt.Sprintf("cat %s 2>/dev/null | wc -l", cgroupProcsPath)
		out, err := procutils.NewRemoteCommandAsFarAsPossible("sh", "-c", cmd).Output()
		if err == nil {
			procCount := strings.TrimSpace(string(out))
			log.Debugf("[verifyEbpfAttached] Processes in cgroup: %s", procCount)
		}
	}

	// 注意：由于 eBPF 程序附加是异步的，并且验证可能需要特殊权限，
	// 这里只做基本的检查，不返回错误
	return nil
}
