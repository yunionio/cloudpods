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
)

type PodMetrics struct {
	PodCpu     *PodCpuMetric       `json:"pod_cpu"`
	PodMemory  *PodMemoryMetric    `json:"pod_memory"`
	PodVolumes []*PodVolumeMetric  `json:"pod_volume"`
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

type ContainerMetrics struct {
	ContainerCpu    *ContainerCpuMetric    `json:"container_cpu"`
	ContainerMemory *ContainerMemoryMetric `json:"container_memory"`
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

func (m *SGuestMonitor) PodMetrics(prevUsage *GuestMetrics) *PodMetrics {
	stat := m.podStat
	podCpu := &PodCpuMetric{
		PodMetricMeta:        NewPodMetricMeta(stat.CPU.Time.Time),
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
		meta := NewContainerMetricMeta(m.Id, "", ctr.Name, ctr.CPU.Time.Time)
		cm := &ContainerMetrics{
			ContainerCpu: &ContainerCpuMetric{
				ContainerMetricMeta:  meta,
				CpuUsageSecondsTotal: float64(*ctr.CPU.UsageCoreNanoSeconds) / float64(time.Second),
			},
			ContainerMemory: &ContainerMemoryMetric{
				ContainerMetricMeta:   meta,
				MemoryWorkingSetBytes: float64(*ctr.Memory.WorkingSetBytes),
				MemoryUsageRate:       (float64(*ctr.Memory.WorkingSetBytes) / float64(m.MemMB*1024*1024)) * 100,
			},
		}
		if hasPrevUsage {
			for _, prevCtr := range prevUsage.PodMetrics.Containers {
				if prevCtr.ContainerCpu.ContainerName == ctr.Name {
					val := (cm.ContainerCpu.CpuUsageSecondsTotal - prevCtr.ContainerCpu.CpuUsageSecondsTotal) / cm.ContainerCpu.Time.Sub(prevCtr.ContainerCpu.Time).Seconds() * 100
					cm.ContainerCpu.CpuUsageRate = &val
					break
				}
			}
		}
		containers = append(containers, cm)
	}
	return &PodMetrics{
		PodCpu:     podCpu,
		PodMemory:  podMemory,
		PodVolumes: m.getVolumeMetrics(),
		Containers: containers,
	}
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
	for _, c := range m.Containers {
		ims = append(ims, c.ContainerCpu)
		ims = append(ims, c.ContainerMemory)
	}
	res := []string{}
	for _, im := range ims {
		tagMap := im.GetTag()
		if len(tagMap) != 0 {
			var newTagArr []string
			for k, v := range tagMap {
				newTagArr = append(newTagArr, fmt.Sprintf("%s=%s", k, v))
			}
			tagStr = strings.Join([]string{tagStr, strings.Join(newTagArr, ",")}, ",")
		}
		res = append(res, fmt.Sprintf("%s,%s %s", im.GetName(), tagStr, d.mapToStatStr(im.ToMap())))
	}
	return res
}
