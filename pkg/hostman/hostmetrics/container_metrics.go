package hostmetrics

import (
	"fmt"
	"strings"
	"time"

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
)

type PodMetrics struct {
	PodCpu     *PodCpuMetric       `json:"pod_cpu"`
	PodMemory  *PodMemoryMetric    `json:"pod_memory"`
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
