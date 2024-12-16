package stats

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	cadvisorapiv1 "github.com/google/cadvisor/info/v1"
	info "github.com/google/cadvisor/info/v1"
	cadvisorapiv2 "github.com/google/cadvisor/info/v2"
	"github.com/google/cadvisor/utils/sysfs"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"

	"yunion.io/x/log"
)

// defaultNetworkInterfaceName is used for collectng network stats.
// This logic relies on knowledge of the container runtime implementation and
// is not reliable.
const defaultNetworkInterfaceName = "eth0"

func getUint64Value(value *uint64) uint64 {
	if value == nil {
		return 0
	}

	return *value
}

func uint64Ptr(i uint64) *uint64 {
	return &i
}

func cadvisorInfoToCPUandMemoryStats(info *cadvisorapiv2.ContainerInfo) (*CPUStats, *MemoryStats) {
	cstat, found := latestContainerStats(info)
	if !found {
		return nil, nil
	}
	var cpuStats *CPUStats
	var memoryStats *MemoryStats
	cpuStats = &CPUStats{
		Time:                 metav1.NewTime(cstat.Timestamp),
		UsageNanoCores:       uint64Ptr(0),
		UsageCoreNanoSeconds: uint64Ptr(0),
	}
	if info.Spec.HasCpu {
		if cstat.CpuInst != nil {
			cpuStats.UsageNanoCores = &cstat.CpuInst.Usage.Total
		}
		if cstat.Cpu != nil {
			cpuStats.UsageCoreNanoSeconds = &cstat.Cpu.Usage.Total
		}
	}
	if info.Spec.HasMemory && cstat.Memory != nil {
		pageFaults := cstat.Memory.ContainerData.Pgfault
		majorPageFaults := cstat.Memory.ContainerData.Pgmajfault
		memoryStats = &MemoryStats{
			Time:            metav1.NewTime(cstat.Timestamp),
			UsageBytes:      &cstat.Memory.Usage,
			WorkingSetBytes: &cstat.Memory.WorkingSet,
			RSSBytes:        &cstat.Memory.RSS,
			PageFaults:      &pageFaults,
			MajorPageFaults: &majorPageFaults,
		}
		// availableBytes = memory limit (if known) - workingset
		if !isMemoryUnlimited(info.Spec.Memory.Limit) {
			availableBytes := info.Spec.Memory.Limit - cstat.Memory.WorkingSet
			memoryStats.AvailableBytes = &availableBytes
		}
	} else {
		memoryStats = &MemoryStats{
			Time:            metav1.NewTime(cstat.Timestamp),
			WorkingSetBytes: uint64Ptr(0),
		}
	}
	return cpuStats, memoryStats
}

// latestContainerStats returns the latest container stats from cadvisor, or nil if none exist
func latestContainerStats(info *cadvisorapiv2.ContainerInfo) (*cadvisorapiv2.ContainerStats, bool) {
	stats := info.Stats
	if len(stats) < 1 {
		return nil, false
	}
	latest := stats[len(stats)-1]
	if latest == nil {
		return nil, false
	}
	return latest, true
}

func isMemoryUnlimited(v uint64) bool {
	// Size after which we consider memory to be "unlimited". This is not
	// MaxInt64 due to rounding by the kernel.
	// TODO: cadvisor should export this https://github.com/google/cadvisor/blob/master/metrics/prometheus.go#L596
	const maxMemorySize = uint64(1 << 62)

	return v > maxMemorySize
}

// cadvisorInfoToNetworkStats returns the statsapi.NetworkStats converted from
// the container info from cadvisor.
func cadvisorInfoToNetworkStats(info *cadvisorapiv2.ContainerInfo) *NetworkStats {
	if !info.Spec.HasNetwork {
		return nil
	}
	cstat, found := latestContainerStats(info)
	if !found {
		return nil
	}

	if cstat.Network == nil {
		return nil
	}

	iStats := NetworkStats{
		Time: metav1.NewTime(cstat.Timestamp),
	}

	for i := range cstat.Network.Interfaces {
		inter := cstat.Network.Interfaces[i]
		iStat := InterfaceStats{
			Name:     inter.Name,
			RxBytes:  &inter.RxBytes,
			RxErrors: &inter.RxErrors,
			TxBytes:  &inter.TxBytes,
			TxErrors: &inter.TxErrors,
		}

		if inter.Name == defaultNetworkInterfaceName {
			iStats.InterfaceStats = iStat
		}

		iStats.Interfaces = append(iStats.Interfaces, iStat)
	}

	return &iStats
}

// cadvisorInfoToUserDefinedMetrics returns the statsapi.UserDefinedMetric
// converted from the container info from cadvisor.
func cadvisorInfoToUserDefinedMetrics(info *cadvisorapiv2.ContainerInfo) []UserDefinedMetric {
	type specVal struct {
		ref     UserDefinedMetricDescriptor
		valType cadvisorapiv1.DataType
		time    time.Time
		value   float64
	}
	udmMap := map[string]*specVal{}
	for _, spec := range info.Spec.CustomMetrics {
		udmMap[spec.Name] = &specVal{
			ref: UserDefinedMetricDescriptor{
				Name:  spec.Name,
				Type:  UserDefinedMetricType(spec.Type),
				Units: spec.Units,
			},
			valType: spec.Format,
		}
	}
	for _, stat := range info.Stats {
		for name, values := range stat.CustomMetrics {
			specVal, ok := udmMap[name]
			if !ok {
				klog.Warningf("spec for custom metric %q is missing from cAdvisor output. Spec: %+v, Metrics: %+v", name, info.Spec, stat.CustomMetrics)
				continue
			}
			for _, value := range values {
				// Pick the most recent value
				if value.Timestamp.Before(specVal.time) {
					continue
				}
				specVal.time = value.Timestamp
				specVal.value = value.FloatValue
				if specVal.valType == cadvisorapiv1.IntType {
					specVal.value = float64(value.IntValue)
				}
			}
		}
	}
	var udm []UserDefinedMetric
	for _, specVal := range udmMap {
		udm = append(udm, UserDefinedMetric{
			UserDefinedMetricDescriptor: specVal.ref,
			Time:                        metav1.NewTime(specVal.time),
			Value:                       specVal.value,
		})
	}
	return udm
}

func cadvisorInfoToProcessStats(info *cadvisorapiv2.ContainerInfo) *ProcessStats {
	cstat, found := latestContainerStats(info)
	if !found || cstat.Processes == nil {
		return nil
	}
	return &ProcessStats{
		ProcessCount:   cstat.Processes.ProcessCount,
		FdCount:        cstat.Processes.FdCount,
		SocketCount:    cstat.Processes.SocketCount,
		ThreadsCurrent: cstat.Processes.ThreadsCurrent,
		ThreadsMax:     cstat.Processes.ThreadsMax,
	}
}

func convertToDiskIoStats(cStats *cadvisorapiv1.DiskIoStats) DiskIoStats {
	if cStats == nil {
		return nil
	}
	diskInfos, err := GetBlockDeviceInfo(sysfs.NewRealSysFs())
	if err != nil {
		log.Warningf("get block device info: %v", err)
		return nil
	}
	result := make(map[string]*DiskIoStat)
	var (
		keyServiced     = "service"
		keyServiceBytes = "serviceBytes"
	)
	infos := map[string][]cadvisorapiv1.PerDiskStats{
		keyServiced:     cStats.IoServiced,
		keyServiceBytes: cStats.IoServiceBytes,
	}
	for key, info := range infos {
		isByte := key == keyServiceBytes
		for i := range info {
			svc := info[i]
			devName := svc.Device
			if devName == "" {
				// fill devName by major and minor number
				key := fmt.Sprintf("%d:%d", svc.Major, svc.Minor)
				disk, ok := diskInfos[key]
				if !ok {
					log.Warningf("not found disk by %s from %#v", key, diskInfos)
					continue
				}
				devName = fmt.Sprintf("/dev/%s", disk.Name)
			}
			diskResult, ok := result[devName]
			if !ok {
				diskResult = NewDiskIoStat(devName, svc.Stats, isByte)
			} else {
				diskResult.fillStats(svc.Stats, isByte)
			}
			result[devName] = diskResult
		}
	}
	return result
}

func cadvisorInfoToDiskIoStats(info *cadvisorapiv2.ContainerInfo) DiskIoStats {
	cstat, found := latestContainerStats(info)
	if !found || cstat.DiskIo == nil {
		return nil
	}
	return convertToDiskIoStats(cstat.DiskIo)
}

var schedulerRegExp = regexp.MustCompile(`.*\[(.*)\].*`)

// Get information about block devices present on the system.
// Uses the passed in system interface to retrieve the low level OS information.
func GetBlockDeviceInfo(sysfs sysfs.SysFs) (map[string]info.DiskInfo, error) {

	disks, err := sysfs.GetBlockDevices()
	if err != nil {
		return nil, err
	}

	diskMap := make(map[string]info.DiskInfo)
	for _, disk := range disks {
		name := disk.Name()
		// Ignore non-disk devices.
		// TODO(rjnagal): Maybe just match hd, sd, loop, and dm prefixes.
		if strings.HasPrefix(name, "ram") || strings.HasPrefix(name, "sr") {
			continue
		}
		diskInfo := info.DiskInfo{
			Name: name,
		}
		dev, err := sysfs.GetBlockDeviceNumbers(name)
		if err != nil {
			return nil, err
		}
		n, err := fmt.Sscanf(dev, "%d:%d", &diskInfo.Major, &diskInfo.Minor)
		if err != nil || n != 2 {
			return nil, fmt.Errorf("could not parse device numbers from %s for device %s", dev, name)
		}
		out, err := sysfs.GetBlockDeviceSize(name)
		if err != nil {
			return nil, err
		}
		// Remove trailing newline before conversion.
		size, err := strconv.ParseUint(strings.TrimSpace(out), 10, 64)
		if err != nil {
			return nil, err
		}
		// size is in 512 bytes blocks.
		diskInfo.Size = size * 512

		diskInfo.Scheduler = "none"
		blkSched, err := sysfs.GetBlockDeviceScheduler(name)
		if err == nil {
			matches := schedulerRegExp.FindSubmatch([]byte(blkSched))
			if len(matches) >= 2 {
				diskInfo.Scheduler = string(matches[1])
			}
		}
		device := fmt.Sprintf("%d:%d", diskInfo.Major, diskInfo.Minor)
		diskMap[device] = diskInfo
	}
	return diskMap, nil
}
