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
	"strings"
	"time"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/hostman/guestman"
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
	PodCpu     *PodCpuMetric       `json:"pod_cpu"`
	PodMemory  *PodMemoryMetric    `json:"pod_memory"`
	PodProcess *PodProcessMetric   `json:"pod_process"`
	PodVolumes []*PodVolumeMetric  `json:"pod_volume"`
	PodDiskIos PodDiskIoMetrics    `json:"pod_disk_ios"`
	Containers []*ContainerMetrics `json:"containers"`
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
		"container_name": m.ContainerName,
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

func GetPodStatsById(stats []stats.PodStats, podId string) *stats.PodStats {
	for _, stat := range stats {
		if stat.PodRef.UID == podId {
			tmp := stat
			return &tmp
		}
	}
	return nil
}

func (s *SGuestMonitorCollector) collectPodMetrics(gm *SGuestMonitor, prevUsage *GuestMetrics) *GuestMetrics {
	gmData := new(GuestMetrics)
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
		PodCpu:     podCpu,
		PodMemory:  podMemory,
		PodProcess: podProcess,
		PodVolumes: m.getVolumeMetrics(),
		Containers: containers,
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

type iPodMetric interface {
	GetName() string
	GetTag() map[string]string
	ToMap() map[string]interface{}
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
	}
	res = append(res, d.netioToTelegrafData("pod_netio", tagStr)...)
	return res
}
