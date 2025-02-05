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

package hostmetrics

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/log"

	apis "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/hostman/guestman"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/util/pod/stats"
)

const (
	// Cumulative cpu time consumed by the container in core-seconds
	CPU_USAGE_SECONDS_TOTAL = "usage_seconds_total"
	// cpu usage rate
	CPU_USAGE_RATE = "usage_rate"
	// current working set of memory size in bytes
	MEMORY_WORKING_SET_BYTES = "working_set_bytes"
	// memory usage rate
	MEMORY_USAGE_RATE = "usage_rate"

	VOLUME_TOTAL               = "total"
	VOLUME_FREE                = "free"
	VOLUME_USED                = "used"
	VOLUME_USED_PERCENT        = "used_percent"
	VOLUME_INODES_TOTAL        = "inodes_total"
	VOLUME_INODES_FREE         = "inodes_free"
	VOLUME_INODES_USED         = "inodes_used"
	VOLUME_INODES_USED_PERCENT = "inodes_used_percent"

	PROCESS_COUNT   = "process_count"
	FD_COUNT        = "fd_count"
	SOCKET_COUNT    = "socket_count"
	THREADS_CURRENT = "threads_current"
	THREADS_MAX     = "threads_max"

	NVIDIA_GPU_MEMORY_TOTAL   = "memory_total"
	NVIDIA_GPU_INDEX          = "index"
	NVIDIA_GPU_PHYSICAL_INDEX = "physical_index"
	NVIDIA_GPU_FRAME_BUFFER   = "frame_buffer"
	NVIDIA_GPU_CCPM           = "ccpm"
	NVIDIA_GPU_SM             = "sm"
	NVIDIA_GPU_MEM_UTIL       = "mem_util"
	NVIDIA_GPU_ENC            = "enc"
	NVIDIA_GPU_DEC            = "dec"
	NVIDIA_GPU_JPG            = "jpg"
	NVIDIA_GPU_OFA            = "ofa"

	VASTAITECH_GPU_DEV_ID   = "dev_id"
	VASTAITECH_GPU_ENC      = "enc"
	VASTAITECH_GPU_DEC      = "dec"
	VASTAITECH_GPU_GFX      = "gfx"
	VASTAITECH_GPU_MEM      = "mem"
	VASTAITECH_GPU_MEM_UTIL = "mem_util"

	CPH_AMD_GPU_DEV_ID   = "dev_id"
	CPH_AMD_GPU_MEM      = "mem"
	CPH_AMD_GPU_MEM_UTIL = "mem_util"
)

type CadvisorProcessMetric struct {
	// Number of processes
	ProcessCount uint64 `json:"process_count"`
	// Number of open file descriptors
	FdCount uint64 `json:"fd_count,omitempty"`
	// Number of sockets
	SocketCount uint64 `json:"socket_count"`
	// Number of threads currently in container
	ThreadsCurrent uint64 `json:"threads_current,omitempty"`
	// Maximum number of threads allowed in container
	ThreadsMax uint64 `json:"threads_max,omitempty"`
}

func (m CadvisorProcessMetric) ToMap() map[string]interface{} {
	return map[string]interface{}{
		PROCESS_COUNT:   m.ProcessCount,
		FD_COUNT:        m.FdCount,
		SOCKET_COUNT:    m.SocketCount,
		THREADS_CURRENT: m.ThreadsCurrent,
		THREADS_MAX:     m.ThreadsMax,
	}
}

type PodMetrics struct {
	PodCpu           *PodCpuMetric              `json:"pod_cpu"`
	PodMemory        *PodMemoryMetric           `json:"pod_memory"`
	PodProcess       *PodProcessMetric          `json:"pod_process"`
	PodVolumes       []*PodVolumeMetric         `json:"pod_volume"`
	PodDiskIos       PodDiskIoMetrics           `json:"pod_disk_ios"`
	PodNvidiaGpu     []*PodNvidiaGpuMetrics     `json:"pod_nvidia_gpu"`
	PodVastaitechGpu []*PodVastaitechGpuMetrics `json:"pod_vastaitech_gpu"`
	PodCphAmdGpu     []*PodCphAmdGpuMetrics     `json:"pod_cph_amd_gpu"`
	Containers       []*ContainerMetrics        `json:"containers"`
}

type PodMetricMeta struct {
	Time time.Time
}

func NewPodMetricMeta(time time.Time) PodMetricMeta {
	return PodMetricMeta{Time: time}
}

func (m PodMetricMeta) GetTag() map[string]string {
	return nil
}

type PodCphAmdGpuMetrics struct {
	PodMetricMeta

	DevId   string
	Mem     float64 // MB
	MemUtil float64
}

func (m PodCphAmdGpuMetrics) GetName() string {
	return "pod_cph_amd_gpu"
}

func (m PodCphAmdGpuMetrics) GetUniformName() string {
	return "pod_gpu"
}

func (m PodCphAmdGpuMetrics) GetTag() map[string]string {
	return map[string]string{
		"dev_id":   m.DevId,
		"dev_type": apis.CONTAINER_DEV_CPH_AMD_GPU,
	}
}

func (m PodCphAmdGpuMetrics) ToMap() map[string]interface{} {
	ret := map[string]interface{}{
		CPH_AMD_GPU_DEV_ID:   m.DevId,
		CPH_AMD_GPU_MEM:      m.Mem,
		CPH_AMD_GPU_MEM_UTIL: m.MemUtil,
	}
	return ret
}

type PodVastaitechGpuMetrics struct {
	PodMetricMeta

	PciAddr string
	DevId   string

	Mem     float64 // MB
	MemUtil float64
	Gfx     float64
	DecUtil float64
	EncUtil float64
}

func (m PodVastaitechGpuMetrics) GetName() string {
	return "pod_vastaitech_gpu"
}

func (m PodVastaitechGpuMetrics) GetUniformName() string {
	return "pod_gpu"
}

func (m PodVastaitechGpuMetrics) GetTag() map[string]string {
	return map[string]string{
		"dev_id":   m.DevId,
		"dev_type": apis.CONTAINER_DEV_VASTAITECH_GPU,
	}
}

func (m PodVastaitechGpuMetrics) ToMap() map[string]interface{} {
	ret := map[string]interface{}{
		VASTAITECH_GPU_DEC:      m.DecUtil,
		VASTAITECH_GPU_DEV_ID:   m.DevId,
		VASTAITECH_GPU_ENC:      m.EncUtil,
		VASTAITECH_GPU_GFX:      m.Gfx,
		VASTAITECH_GPU_MEM:      m.Mem,
		VASTAITECH_GPU_MEM_UTIL: m.MemUtil,
	}
	return ret
}

type PodNvidiaGpuMetrics struct {
	PodMetricMeta

	Index         int
	PhysicalIndex int
	MemTotal      int

	Framebuffer int     // Framebuffer Memory Usage
	Ccpm        int     // Current CUDA Contexts Per Measurement
	SmUtil      float64 // Streaming Multiprocessor Utilization
	Mem         int     // Mem Usage
	MemUtil     float64 // Memory Utilization
	EncUtil     float64 // Encoder Utilization
	DecUtil     float64 // Decoder Utilization
	JpgUtil     float64 // JPEG Decoder Utilization
	OfaUtil     float64 // Other Feature Utilization
}

func (m PodNvidiaGpuMetrics) GetName() string {
	return "pod_nvidia_gpu"
}

func (m PodNvidiaGpuMetrics) GetUniformName() string {
	return "pod_gpu"
}

func (m PodNvidiaGpuMetrics) GetTag() map[string]string {
	devType := apis.CONTAINER_DEV_NVIDIA_GPU
	if options.HostOptions.EnableCudaMPS {
		devType = apis.CONTAINER_DEV_NVIDIA_MPS
	}
	return map[string]string{
		"index":          strconv.Itoa(m.Index),
		"physical_index": strconv.Itoa(m.PhysicalIndex),
		"dev_type":       devType,
	}
}

func (m PodNvidiaGpuMetrics) ToMap() map[string]interface{} {
	ret := map[string]interface{}{
		NVIDIA_GPU_MEMORY_TOTAL:   m.MemTotal,
		NVIDIA_GPU_INDEX:          m.Index,
		NVIDIA_GPU_PHYSICAL_INDEX: m.PhysicalIndex,
		NVIDIA_GPU_FRAME_BUFFER:   m.Framebuffer,
		NVIDIA_GPU_CCPM:           m.Ccpm,
		NVIDIA_GPU_SM:             m.SmUtil,
		NVIDIA_GPU_MEM_UTIL:       m.MemUtil,
		NVIDIA_GPU_ENC:            m.EncUtil,
		NVIDIA_GPU_DEC:            m.DecUtil,
		NVIDIA_GPU_JPG:            m.JpgUtil,
		NVIDIA_GPU_OFA:            m.OfaUtil,
	}

	return ret
}

type PodCpuMetric struct {
	PodMetricMeta
	CpuUsageSecondsTotal float64  `json:"cpu_usage_seconds_total"`
	CpuUsageRate         *float64 `json:"cpu_usage_rate"`
}

func (m PodCpuMetric) GetName() string {
	return "pod_cpu"
}

func (m PodCpuMetric) ToMap() map[string]interface{} {
	ret := map[string]interface{}{
		CPU_USAGE_SECONDS_TOTAL: m.CpuUsageSecondsTotal,
	}
	if m.CpuUsageRate != nil {
		ret[CPU_USAGE_RATE] = *m.CpuUsageRate
	}
	return ret
}

type PodMemoryMetric struct {
	PodMetricMeta
	MemoryWorkingSetBytes float64 `json:"memory_working_set_bytes"`
	MemoryUsageRate       float64 `json:"memory_usage_rate"`
}

func (m PodMemoryMetric) GetName() string {
	return "pod_mem"
}

func (m PodMemoryMetric) ToMap() map[string]interface{} {
	return map[string]interface{}{
		MEMORY_WORKING_SET_BYTES: m.MemoryWorkingSetBytes,
		MEMORY_USAGE_RATE:        m.MemoryUsageRate,
	}
}

type PodProcessMetric struct {
	PodMetricMeta
	*CadvisorProcessMetric
}

func (m PodProcessMetric) GetName() string {
	return "pod_process"
}

type PodVolumeMetric struct {
	ContainerMetricMeta
	// 容器内挂载路径
	MountPath string `json:"mount_path"`
	// 宿主机路径
	HostPath          string            `json:"host_path"`
	Type              string            `json:"type"`
	Fstype            string            `json:"fstype"`
	Total             uint64            `json:"total"`
	Free              uint64            `json:"free"`
	Used              uint64            `json:"used"`
	UsedPercent       float64           `json:"used_percent"`
	InodesTotal       uint64            `json:"inodes_total"`
	InodesUsed        uint64            `json:"inodes_used"`
	InodesFree        uint64            `json:"inodes_free"`
	InodesUsedPercent float64           `json:"inodes_used_percent"`
	Tags              map[string]string `json:"tags"`
}

type CadvisorDiskIoMetric struct {
	Device string `json:"device"`
	//AsyncBytes   uint64 `json:"async_bytes"`
	//DiscardBytes uint64 `json:"discard_bytes"`
	ReadBytes  uint64 `json:"read_bytes"`
	WriteBytes uint64 `json:"write_bytes"`
	//TotalBytes   uint64 `json:"total_bytes"`
	//AsyncCount   uint64 `json:"async_count"`
	//DiscardCount uint64 `json:"discard_count"`
	ReadCount  uint64 `json:"read_count"`
	WriteCount uint64 `json:"write_count"`
	//TotalCount   uint64 `json:"total_count"`

	ReadIOPS  float64 `json:"read_iops"`
	WriteIOPS float64 `json:"write_iops"`
	ReadBPS   float64 `json:"read_Bps"`
	WriteBPS  float64 `json:"write_Bps"`
}

func (m CadvisorDiskIoMetric) GetTag() map[string]string {
	return map[string]string{
		"device": m.Device,
	}
}

func (m CadvisorDiskIoMetric) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"read_bytes":  m.ReadBytes,
		"write_bytes": m.WriteBytes,
		"read_Bps":    m.ReadBPS,
		"write_Bps":   m.WriteBPS,
		"read_count":  m.ReadCount,
		"write_count": m.WriteCount,
		"read_iops":   m.ReadIOPS,
		"write_iops":  m.WriteIOPS,
	}
}

type PodDiskIoMetrics map[string]*PodDiskIoMetric

func newPodDiskIoMetrics(metrics map[string]CadvisorDiskIoMetric, meta PodMetricMeta) PodDiskIoMetrics {
	ret := make(map[string]*PodDiskIoMetric)
	for k, v := range metrics {
		ret[k] = &PodDiskIoMetric{
			PodMetricMeta:        meta,
			CadvisorDiskIoMetric: v,
		}
	}
	return ret
}

func (m PodDiskIoMetrics) ToCadvisorDiskIoMetrics() map[string]CadvisorDiskIoMetric {
	ret := make(map[string]CadvisorDiskIoMetric)
	for k, v := range m {
		ret[k] = v.CadvisorDiskIoMetric
	}
	return ret
}

func (m PodDiskIoMetrics) GetTime() time.Time {
	for _, v := range m {
		return v.Time
	}
	return time.Time{}
}

type PodDiskIoMetric struct {
	PodMetricMeta
	CadvisorDiskIoMetric
}

func (m PodDiskIoMetric) GetName() string {
	return "pod_diskio"
}

func (m PodDiskIoMetric) GetTag() map[string]string {
	return m.CadvisorDiskIoMetric.GetTag()
}

func (m PodVolumeMetric) GetName() string {
	return "pod_volume"
}

func (m PodVolumeMetric) ToMap() map[string]interface{} {
	r := map[string]interface{}{
		VOLUME_TOTAL:               m.Total,
		VOLUME_FREE:                m.Free,
		VOLUME_USED:                m.Used,
		VOLUME_USED_PERCENT:        m.UsedPercent,
		VOLUME_INODES_TOTAL:        m.InodesTotal,
		VOLUME_INODES_FREE:         m.InodesFree,
		VOLUME_INODES_USED:         m.InodesUsed,
		VOLUME_INODES_USED_PERCENT: m.InodesUsedPercent,
	}
	return r
}

func (m PodVolumeMetric) GetTag() map[string]string {
	baseTags := m.ContainerMetricMeta.GetTag()
	curTags := map[string]string{
		"mount_path": m.MountPath,
		"host_path":  m.HostPath,
		"type":       m.Type,
	}
	for k, v := range curTags {
		baseTags[k] = v
	}
	for k, v := range m.Tags {
		baseTags[k] = v
	}
	return baseTags
}

type ContainerDiskIoMetrics map[string]*ContainerDiskIoMetric

func newContainerDiskioMetrics(metrics map[string]CadvisorDiskIoMetric, meta ContainerMetricMeta) ContainerDiskIoMetrics {
	ret := make(map[string]*ContainerDiskIoMetric)
	for k, v := range metrics {
		ret[k] = &ContainerDiskIoMetric{
			ContainerMetricMeta:  meta,
			CadvisorDiskIoMetric: v,
		}
	}
	return ret
}

func (m ContainerDiskIoMetrics) ToCadvisorDiskIoMetrics() map[string]CadvisorDiskIoMetric {
	ret := make(map[string]CadvisorDiskIoMetric)
	for k, v := range m {
		ret[k] = v.CadvisorDiskIoMetric
	}
	return ret
}

type ContainerMetrics struct {
	ContainerCpu     *ContainerCpuMetric     `json:"container_cpu"`
	ContainerMemory  *ContainerMemoryMetric  `json:"container_memory"`
	ContainerProcess *ContainerProcessMetric `json:"container_process"`
	ContainerDiskIos ContainerDiskIoMetrics  `json:"container_diskios"`
}

type ContainerMetricMeta struct {
	PodMetricMeta
	ContainerId   string `json:"container_id"`
	ContainerName string `json:"container_name"`
	PodId         string `json:"pod_id"`
}

func (m ContainerMetricMeta) GetTag() map[string]string {
	ret := map[string]string{
		"pod_id":         m.PodId,
		"container_name": strings.ReplaceAll(m.ContainerName, " ", "+"),
	}
	if m.ContainerId != "" {
		ret["container_id"] = m.ContainerId
	}
	return ret
}

type ContainerMemoryMetric struct {
	ContainerMetricMeta
	MemoryWorkingSetBytes float64 `json:"memory_working_set_bytes"`
	MemoryUsageRate       float64 `json:"memory_usage_rate"`
}

func (m ContainerMemoryMetric) GetName() string {
	return "container_mem"
}

func (m *ContainerMemoryMetric) ToMap() map[string]interface{} {
	return map[string]interface{}{
		MEMORY_WORKING_SET_BYTES: m.MemoryWorkingSetBytes,
		MEMORY_USAGE_RATE:        m.MemoryUsageRate,
	}
}

type ContainerCpuMetric struct {
	ContainerMetricMeta
	CpuUsageSecondsTotal float64  `json:"cpu_usage_seconds_total"`
	CpuUsageRate         *float64 `json:"cpu_usage_rate"`
}

func (m ContainerCpuMetric) GetName() string {
	return "container_cpu"
}

func (m *ContainerCpuMetric) ToMap() map[string]interface{} {
	ret := map[string]interface{}{
		CPU_USAGE_SECONDS_TOTAL: m.CpuUsageSecondsTotal,
	}
	if m.CpuUsageRate != nil {
		ret[CPU_USAGE_RATE] = *m.CpuUsageRate
	}
	return ret
}

type ContainerProcessMetric struct {
	ContainerMetricMeta
	*CadvisorProcessMetric
}

func (m ContainerProcessMetric) GetName() string {
	return "container_process"
}

type ContainerDiskIoMetric struct {
	ContainerMetricMeta
	CadvisorDiskIoMetric
}

func (m ContainerDiskIoMetric) GetName() string {
	return "container_diskio"
}

func (m *ContainerDiskIoMetric) GetTag() map[string]string {
	baseTags := m.ContainerMetricMeta.GetTag()
	for k, v := range m.CadvisorDiskIoMetric.GetTag() {
		baseTags[k] = v
	}
	return baseTags
}

func GetPodStatsById(ss []stats.PodStats, gpuPodProcs map[string]map[string]struct{}, podId string) (*stats.PodStats, map[string]struct{}) {
	var podStat *stats.PodStats
	for i := range ss {
		if ss[i].PodRef.UID == podId {
			podStat = &ss[i]
			break
		}
	}
	podProcs, _ := gpuPodProcs[podId]
	return podStat, podProcs
}

func GetPodNvidiaGpuMetrics(metrics []NvidiaGpuProcessMetrics, podProcs map[string]struct{}) []NvidiaGpuProcessMetrics {
	podMetrics := make([]NvidiaGpuProcessMetrics, 0)
	for i := range metrics {
		pid := metrics[i].Pid
		if _, ok := podProcs[pid]; ok {
			podMetrics = append(podMetrics, metrics[i])
		}
	}
	return podMetrics
}

func GetPodVastaitechGpuMetrics(metrics []VastaitechGpuProcessMetrics, podProcs map[string]struct{}) []VastaitechGpuProcessMetrics {
	podMetrics := make([]VastaitechGpuProcessMetrics, 0)
	for i := range metrics {
		pid := metrics[i].Pid
		if _, ok := podProcs[pid]; ok {
			podMetrics = append(podMetrics, metrics[i])
		}
	}
	return podMetrics
}

func GetPodCphAmdGpuMetrics(metrics []CphAmdGpuProcessMetrics, podProcs map[string]struct{}) []CphAmdGpuProcessMetrics {
	podMetrics := make([]CphAmdGpuProcessMetrics, 0)
	for i := range metrics {
		pid := metrics[i].Pid
		if _, ok := podProcs[pid]; ok {
			podMetrics = append(podMetrics, metrics[i])
		}
	}
	return podMetrics
}

func (s *SGuestMonitorCollector) collectPodMetrics(gm *SGuestMonitor, prevUsage *GuestMetrics) *GuestMetrics {
	gmData := new(GuestMetrics)
	s.hostInfo.GetContainerStatsProvider()
	gmData.PodMetrics = gm.PodMetrics(prevUsage)

	// netio
	gmData.VmNetio = gm.Netio()
	netio1 := gmData.VmNetio
	netio2 := prevUsage.VmNetio
	s.addNetio(netio1, netio2)

	return gmData
}

func NewContainerMetricMeta(serverId string, containerId string, containerName string, time time.Time) ContainerMetricMeta {
	return ContainerMetricMeta{
		PodMetricMeta: NewPodMetricMeta(time),
		ContainerId:   containerId,
		PodId:         serverId,
		ContainerName: containerName,
	}
}

func (m *SGuestMonitor) HasPodMetrics() bool {
	return m.podStat != nil
}

func (m *SGuestMonitor) getVolumeMetrics() []*PodVolumeMetric {
	pi := m.instance.(guestman.PodInstance)
	if !pi.IsRunning() {
		return nil
	}
	vus, err := pi.GetVolumeMountUsages()
	if err != nil {
		log.Warningf("get volume mount usages: %v", err)
	}
	result := make([]*PodVolumeMetric, 0)
	for i := range vus {
		vu := vus[i]
		ctr := pi.GetContainerById(vu.Id)
		if ctr == nil {
			log.Warningf("not found container by %s", vu.Id)
			continue
		}
		meta := NewContainerMetricMeta(pi.GetId(), vu.Id, ctr.Name, time.Now())
		result = append(result, &PodVolumeMetric{
			ContainerMetricMeta: meta,
			MountPath:           vu.MountPath,
			HostPath:            vu.HostPath,
			Type:                vu.VolumeType,
			Fstype:              vu.Usage.Fstype,
			Total:               vu.Usage.Total,
			Free:                vu.Usage.Free,
			Used:                vu.Usage.Used,
			UsedPercent:         vu.Usage.UsedPercent,
			InodesTotal:         vu.Usage.InodesTotal,
			InodesUsed:          vu.Usage.InodesUsed,
			InodesFree:          vu.Usage.InodesFree,
			InodesUsedPercent:   vu.Usage.InodesUsedPercent,
			Tags:                vu.Tags,
		})
	}
	return result
}

func (m *SGuestMonitor) getCadvisorProcessMetric(stat *stats.ProcessStats) *CadvisorProcessMetric {
	if stat == nil {
		return nil
	}
	return &CadvisorProcessMetric{
		ProcessCount:   stat.ProcessCount,
		FdCount:        stat.FdCount,
		SocketCount:    stat.SocketCount,
		ThreadsCurrent: stat.ThreadsCurrent,
		ThreadsMax:     stat.ThreadsMax,
	}
}

func (m *SGuestMonitor) getCadvisorDiskIoMetrics(cur stats.DiskIoStats, prev map[string]CadvisorDiskIoMetric, curTime, prevTime time.Time) map[string]CadvisorDiskIoMetric {
	ret := make(map[string]CadvisorDiskIoMetric)
	for devName, stat := range cur {
		devR := CadvisorDiskIoMetric{
			Device:     stat.DeviceName,
			ReadCount:  stat.ReadCount,
			WriteCount: stat.WriteCount,
			ReadBytes:  stat.ReadBytes,
			WriteBytes: stat.WriteBytes,
		}
		diffTime := float64(curTime.Sub(prevTime) / time.Second)
		if diffTime > 0 && prev != nil {
			prevStat, ok := prev[devName]
			if ok {
				devR.ReadBPS = float64(stat.ReadBytes-prevStat.ReadBytes) / diffTime
				devR.WriteBPS = float64(stat.WriteBytes-prevStat.WriteBytes) / diffTime
				devR.ReadIOPS = float64(stat.ReadCount-prevStat.ReadCount) / diffTime
				devR.WriteIOPS = float64(stat.WriteCount-prevStat.WriteCount) / diffTime
			}
		}
		ret[devName] = devR
	}
	return ret
}

func (m *SGuestMonitor) PodMetrics(prevUsage *GuestMetrics) *PodMetrics {
	stat := m.podStat
	curTime := stat.CPU.Time.Time
	podCpu := &PodCpuMetric{
		PodMetricMeta:        NewPodMetricMeta(curTime),
		CpuUsageSecondsTotal: float64(*stat.CPU.UsageCoreNanoSeconds) / float64(time.Second),
	}
	hasPrevUsage := prevUsage != nil && prevUsage.PodMetrics != nil
	if hasPrevUsage {
		pmPodCpu := prevUsage.PodMetrics.PodCpu
		val := (podCpu.CpuUsageSecondsTotal - pmPodCpu.CpuUsageSecondsTotal) / podCpu.Time.Sub(pmPodCpu.Time).Seconds() * 100
		podCpu.CpuUsageRate = &val
	}
	podMemory := &PodMemoryMetric{
		MemoryWorkingSetBytes: float64(*stat.Memory.WorkingSetBytes),
		MemoryUsageRate:       (float64(*stat.Memory.WorkingSetBytes) / float64(m.MemMB*1024*1024)) * 100,
	}

	containers := make([]*ContainerMetrics, 0)
	for _, ctr := range stat.Containers {
		ctrMeta := NewContainerMetricMeta(m.Id, "", ctr.Name, ctr.CPU.Time.Time)
		cm := &ContainerMetrics{
			ContainerCpu: &ContainerCpuMetric{
				ContainerMetricMeta:  ctrMeta,
				CpuUsageSecondsTotal: float64(*ctr.CPU.UsageCoreNanoSeconds) / float64(time.Second),
			},
			ContainerMemory: &ContainerMemoryMetric{
				ContainerMetricMeta:   ctrMeta,
				MemoryWorkingSetBytes: float64(*ctr.Memory.WorkingSetBytes),
				MemoryUsageRate:       (float64(*ctr.Memory.WorkingSetBytes) / float64(m.MemMB*1024*1024)) * 100,
			},
		}
		var prevCtrM *ContainerMetrics
		if hasPrevUsage {
			for _, prevCtr := range prevUsage.PodMetrics.Containers {
				if prevCtr.ContainerCpu.ContainerName == ctr.Name {
					prevCtrM = prevCtr
					val := (cm.ContainerCpu.CpuUsageSecondsTotal - prevCtr.ContainerCpu.CpuUsageSecondsTotal) / cm.ContainerCpu.Time.Sub(prevCtr.ContainerCpu.Time).Seconds() * 100
					cm.ContainerCpu.CpuUsageRate = &val
					break
				}
			}
		}
		if ctr.ProcessStats != nil {
			cm.ContainerProcess = &ContainerProcessMetric{
				ContainerMetricMeta:   ctrMeta,
				CadvisorProcessMetric: m.getCadvisorProcessMetric(ctr.ProcessStats),
			}
		}
		if ctr.DiskIo != nil {
			var prevStat map[string]CadvisorDiskIoMetric
			var prevTime = ctrMeta.Time
			if prevCtrM != nil {
				if prevCtrM.ContainerDiskIos != nil {
					prevStat = prevCtrM.ContainerDiskIos.ToCadvisorDiskIoMetrics()
					prevTime = prevCtrM.ContainerCpu.Time
				}
			}
			cm.ContainerDiskIos = newContainerDiskioMetrics(m.getCadvisorDiskIoMetrics(ctr.DiskIo, prevStat, ctrMeta.Time, prevTime), ctrMeta)
		}
		containers = append(containers, cm)
	}
	var podProcess *PodProcessMetric
	if stat.ProcessStats != nil {
		podProcess = &PodProcessMetric{
			CadvisorProcessMetric: m.getCadvisorProcessMetric(stat.ProcessStats),
		}
	}

	pm := &PodMetrics{
		PodCpu:           podCpu,
		PodMemory:        podMemory,
		PodProcess:       podProcess,
		PodVolumes:       m.getVolumeMetrics(),
		PodNvidiaGpu:     m.getPodNvidiaGpuMetrics(),
		PodVastaitechGpu: m.getPodVastaitechGpuMetrics(),
		PodCphAmdGpu:     m.getPodCphAmdGpuMetrics(),
		Containers:       containers,
	}

	if stat.DiskIo != nil {
		var prevStat map[string]CadvisorDiskIoMetric
		var prevTime = curTime
		if hasPrevUsage {
			pd := prevUsage.PodMetrics.PodDiskIos
			if pd != nil && len(pd) != 0 {
				prevStat = pd.ToCadvisorDiskIoMetrics()
				prevTime = pd.GetTime()
			}
		}
		podMeta := NewPodMetricMeta(curTime)
		pm.PodDiskIos = newPodDiskIoMetrics(m.getCadvisorDiskIoMetrics(stat.DiskIo, prevStat, curTime, prevTime), podMeta)
	}
	return pm
}

func (m *SGuestMonitor) getPodCphAmdGpuMetrics() []*PodCphAmdGpuMetrics {
	if len(m.cphAmdGpuMetrics) == 0 {
		return nil
	}
	addrGpuMap := map[string]*PodCphAmdGpuMetrics{}
	for i := range m.cphAmdGpuMetrics {
		devId := m.cphAmdGpuMetrics[i].DevId
		gms, ok := addrGpuMap[devId]
		if !ok {
			gms = new(PodCphAmdGpuMetrics)
			gms.DevId = devId
		}
		gms.Mem += m.cphAmdGpuMetrics[i].Mem
		gms.MemUtil += m.cphAmdGpuMetrics[i].MemUtil
		addrGpuMap[devId] = gms
	}
	res := make([]*PodCphAmdGpuMetrics, 0)
	for _, gms := range addrGpuMap {
		res = append(res, gms)
	}

	return res
}

func (m *SGuestMonitor) getPodVastaitechGpuMetrics() []*PodVastaitechGpuMetrics {
	if len(m.vastaitechGpuMetrics) == 0 {
		return nil
	}
	addrGpuMap := map[string]*PodVastaitechGpuMetrics{}
	for i := range m.vastaitechGpuMetrics {
		pciAddr := m.vastaitechGpuMetrics[i].PciAddr
		gms, ok := addrGpuMap[pciAddr]
		if !ok {
			gms = new(PodVastaitechGpuMetrics)
			gms.DevId = m.vastaitechGpuMetrics[i].DevId
			gms.PciAddr = m.vastaitechGpuMetrics[i].PciAddr
		}
		gms.Mem += m.vastaitechGpuMetrics[i].GfxMem
		gms.MemUtil += m.vastaitechGpuMetrics[i].GfxMemUsage
		gms.Gfx += m.vastaitechGpuMetrics[i].Gfx
		gms.DecUtil += m.vastaitechGpuMetrics[i].Dec
		gms.EncUtil += m.vastaitechGpuMetrics[i].Enc
		addrGpuMap[pciAddr] = gms
	}
	res := make([]*PodVastaitechGpuMetrics, 0)
	for _, gms := range addrGpuMap {
		res = append(res, gms)
	}

	return res
}

func (m *SGuestMonitor) getPodNvidiaGpuMetrics() []*PodNvidiaGpuMetrics {
	if len(m.nvidiaGpuMetrics) == 0 {
		return nil
	}
	indexGpuMap := map[int]*PodNvidiaGpuMetrics{}
	for i := range m.nvidiaGpuMetrics {
		index := m.nvidiaGpuMetrics[i].Index
		gms, ok := indexGpuMap[index]
		if !ok {
			gms = new(PodNvidiaGpuMetrics)
		}
		gms.Framebuffer += m.nvidiaGpuMetrics[i].FB
		gms.Ccpm += m.nvidiaGpuMetrics[i].Ccpm
		gms.SmUtil += m.nvidiaGpuMetrics[i].Sm
		gms.EncUtil += m.nvidiaGpuMetrics[i].Enc
		gms.DecUtil += m.nvidiaGpuMetrics[i].Dec
		gms.JpgUtil += m.nvidiaGpuMetrics[i].Jpg
		gms.OfaUtil += m.nvidiaGpuMetrics[i].Ofa
		indexGpuMap[index] = gms
	}

	indexs := make([]int, 0)
	for index, gms := range indexGpuMap {
		indexs = append(indexs, index)
		indexStr := strconv.Itoa(index)
		memSizeTotal, ok := m.nvidiaGpuIndexMemoryMap[indexStr]
		if !ok {
			continue
		}
		gms.MemTotal = memSizeTotal
		gms.Mem = gms.Framebuffer
		gms.MemUtil = float64(gms.Framebuffer) / float64(gms.MemTotal)
	}
	sort.Ints(indexs)
	res := make([]*PodNvidiaGpuMetrics, len(indexs))
	for i := range indexs {
		gms := indexGpuMap[indexs[i]]
		gms.PhysicalIndex = gms.Index
		gms.Index = i
		res[i] = gms
	}
	return res
}

type iPodMetric interface {
	GetName() string
	GetTag() map[string]string
	ToMap() map[string]interface{}
}

type iPodUniformName interface {
	GetUniformName() string
}

func (d *GuestMetrics) toPodTelegrafData(tagStr string) []string {
	m := d.PodMetrics
	ims := []iPodMetric{m.PodCpu, m.PodMemory}
	for i := range m.PodVolumes {
		ims = append(ims, m.PodVolumes[i])
	}
	if m.PodProcess != nil {
		ims = append(ims, m.PodProcess)
	}
	if m.PodDiskIos != nil {
		for _, d := range m.PodDiskIos {
			ims = append(ims, d)
		}
	}
	for i := range m.PodNvidiaGpu {
		ims = append(ims, m.PodNvidiaGpu[i])
	}
	for i := range m.PodVastaitechGpu {
		ims = append(ims, m.PodVastaitechGpu[i])
	}
	for i := range m.PodCphAmdGpu {
		ims = append(ims, m.PodCphAmdGpu[i])
	}

	for _, c := range m.Containers {
		ims = append(ims, c.ContainerCpu)
		ims = append(ims, c.ContainerMemory)
		if c.ContainerProcess != nil {
			ims = append(ims, c.ContainerProcess)
		}
		for _, cd := range c.ContainerDiskIos {
			ims = append(ims, cd)
		}
	}
	res := []string{}
	for _, im := range ims {
		tagMap := im.GetTag()
		newTagStr := tagStr
		if len(tagMap) != 0 {
			var newTagArr []string
			for k, v := range tagMap {
				newTagArr = append(newTagArr, fmt.Sprintf("%s=%s", k, v))
			}
			newTagStr = strings.Join([]string{tagStr, strings.Join(newTagArr, ",")}, ",")
		}
		res = append(res, fmt.Sprintf("%s,%s %s", im.GetName(), newTagStr, d.mapToStatStr(im.ToMap())))
		if imu, ok := im.(iPodUniformName); ok {
			if un := imu.GetUniformName(); un != "" {
				res = append(res, fmt.Sprintf("%s,%s %s", un, newTagStr, d.mapToStatStr(im.ToMap())))
			}
		}
	}
	res = append(res, d.netioToTelegrafData("pod_netio", tagStr)...)
	return res
}
