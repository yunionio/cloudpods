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
	"context"
	"fmt"
	"io"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/google/cadvisor/utils/sysfs"
	"github.com/shirou/gopsutil/host"
	psnet "github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/utils"

	billing_api "yunion.io/x/onecloud/pkg/apis/billing"
	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/hostman/guestman"
	"yunion.io/x/onecloud/pkg/hostman/guestman/desc"
	"yunion.io/x/onecloud/pkg/hostman/hostinfo/hostconsts"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/pod/stats"
	"yunion.io/x/onecloud/pkg/util/timeutils2"
)

const (
	TelegrafServer = "http://127.0.0.1:8087/write"
)

type SHostMetricsCollector struct {
	ReportInterval    int // seconds
	running           bool
	LastCollectTime   time.Time
	waitingReportData []string
	guestMonitor      *SGuestMonitorCollector
}

var hostMetricsCollector *SHostMetricsCollector

type IHostInfo interface {
	GetContainerStatsProvider() stats.ContainerStatsProvider
	HasContainerNvidiaGpu() bool
	HasContainerVastaitechGpu() bool
	HasContainerCphAmdGpu() bool
	GetNvidiaGpuIndexMemoryMap() map[string]int
	ReportHostDmesg(data []compute.SKmsgEntry) error
}

var hostDmesgCollector *SHostDmesgCollector

func Init(hostInfo IHostInfo) {
	if hostMetricsCollector == nil {
		hostMetricsCollector = NewHostMetricsCollector(hostInfo)
	}
	if hostDmesgCollector == nil {
		hostDmesgCollector = NewHostDmesgCollector(hostInfo)
	}
}

func Start() {
	if hostMetricsCollector != nil {
		go hostMetricsCollector.Start()
	}
	if options.HostOptions.EnableDmesgCollect {
		timeutils2.AddTimeout(30*time.Second, hostDmesgCollector.Start)
	}
}

func Stop() {
	if hostMetricsCollector != nil {
		hostMetricsCollector.Stop()
	}
}

func (m *SHostMetricsCollector) Start() {
	m.running = true
	for m.running {
		m.runMain()
		time.Sleep(time.Second * 1)
	}
}

func (m *SHostMetricsCollector) Stop() {
	m.running = false
}

func (m *SHostMetricsCollector) runMain() {
	timeBegin := time.Now()
	elapse := timeBegin.Sub(m.LastCollectTime)
	if elapse < time.Second*time.Duration(m.ReportInterval) {
		return
	}

	m.runMonitor(timeBegin, m.LastCollectTime)

	m.LastCollectTime = timeBegin
}

func (m *SHostMetricsCollector) runMonitor(now, last time.Time) {
	reportData := m.collectReportData(now, last)
	if options.HostOptions.EnableTelegraf && len(reportData) > 0 {
		m.reportUsageToTelegraf(reportData)
	}
}

func (m *SHostMetricsCollector) reportUsageToTelegraf(data string) {
	body := strings.NewReader(data)
	res, err := httputils.Request(httputils.GetDefaultClient(), context.Background(), "POST", TelegrafServer, nil, body, false)
	if err != nil {
		log.Errorf("Upload guest metric failed: %s", err)
		return
	}
	defer res.Body.Close()
	if res.StatusCode != 204 {
		resBody, _ := io.ReadAll(res.Body)
		log.Errorf("upload guest metric failed with %d %s, data: %s", res.StatusCode, string(resBody), data)
		timestamp := time.Now().UnixNano()
		for _, line := range strings.Split(data, "\n") {
			m.waitingReportData = append(m.waitingReportData,
				fmt.Sprintf("%s %d", line, timestamp))
		}
	} else {
		if len(m.waitingReportData) > 0 {
			oldDatas := strings.Join(m.waitingReportData, "\n")
			body = strings.NewReader(oldDatas)
			res, err = httputils.Request(httputils.GetDefaultClient(), context.Background(), "POST", TelegrafServer, nil, body, false)
			if err == nil {
				defer res.Body.Close()
			}
			if res.StatusCode == 204 {
				m.waitingReportData = m.waitingReportData[len(m.waitingReportData):]
			} else {
				log.Errorf("upload guest metric failed code: %d", res.StatusCode)
			}
		}
	}
}

func (m *SHostMetricsCollector) collectReportData(now, last time.Time) string {
	if len(m.waitingReportData) > 60 {
		m.waitingReportData = m.waitingReportData[1:]
	}
	return m.guestMonitor.CollectReportData(now, last)
}

func NewHostMetricsCollector(hostInfo IHostInfo) *SHostMetricsCollector {
	return &SHostMetricsCollector{
		ReportInterval:    options.HostOptions.ReportInterval,
		waitingReportData: make([]string, 0, 10),
		guestMonitor:      NewGuestMonitorCollector(hostInfo),
	}
}

type SGuestMonitorCollector struct {
	monitors       map[string]*SGuestMonitor
	prevPids       map[string]int
	prevReportData map[string]*GuestMetrics
	hostInfo       IHostInfo
}

func NewGuestMonitorCollector(hostInfo IHostInfo) *SGuestMonitorCollector {
	return &SGuestMonitorCollector{
		monitors:       make(map[string]*SGuestMonitor, 0),
		prevPids:       make(map[string]int, 0),
		prevReportData: make(map[string]*GuestMetrics, 0),
		hostInfo:       hostInfo,
	}
}

func (s *SGuestMonitorCollector) GetGuests() map[string]*SGuestMonitor {
	var err error
	gms := make(map[string]*SGuestMonitor, 0)
	guestmanager := guestman.GetGuestManager()

	var podStats []stats.PodStats = nil
	var nvidiaGpuMetrics []NvidiaGpuProcessMetrics = nil
	var vastaitechGpuMetrics []VastaitechGpuProcessMetrics = nil
	var cphAmdGpuMetrics []CphAmdGpuProcessMetrics = nil
	var gpuPodProcs = s.collectGpuPodsProcesses()

	guestmanager.Servers.Range(func(k, v interface{}) bool {
		instance, ok := v.(guestman.GuestRuntimeInstance)
		if !ok {
			return false
		}
		if !instance.IsValid() {
			return false
		}
		hypervisor := instance.GetHypervisor()
		guestId := instance.GetId()
		guestName := instance.GetDesc().Name
		nicsDesc := instance.GetDesc().Nics
		vcpuCount := instance.GetDesc().Cpu
		switch hypervisor {
		case compute.HYPERVISOR_KVM:
			guest := instance.(*guestman.SKVMGuestInstance)
			pid := guest.GetPid()
			if pid > 0 {
				gm, ok := s.monitors[guestId]
				if ok && gm.Pid == pid {
					delete(s.monitors, guestId)
					gm.UpdateVmName(guestName)
					gm.UpdateNicsDesc(nicsDesc)
					gm.UpdateCpuCount(int(vcpuCount))
					gm.MemMB = instance.GetDesc().Mem
				} else {
					delete(s.monitors, guestId)
					gm, err = NewGuestMonitor(instance, guestName, guestId, pid, nicsDesc, int(vcpuCount))
					if err != nil {
						log.Errorf("NewGuestMonitor for %s(%s), pid: %d, nics: %#v", guestName, guestId, pid, nicsDesc)
						return true
					}
				}
				gm.ScalingGroupId = guest.GetDesc().ScalingGroupId
				gm.Tenant = guest.GetDesc().Tenant
				gm.TenantId = guest.GetDesc().TenantId
				gm.DomainId = guest.GetDesc().DomainId
				gm.ProjectDomain = guest.GetDesc().ProjectDomain

				gms[guestId] = gm
			}
			return true
		case compute.HYPERVISOR_POD:
			if podStats == nil {
				var err error
				csp := s.hostInfo.GetContainerStatsProvider()
				if csp == nil {
					log.Warningf("container stats provider is not ready")
					return true
				}
				podStats, err = csp.ListPodCPUAndMemoryStats()
				if err != nil {
					log.Errorf("ListPodCPUAndMemoryStats: %s", err)
					return true
				}
				if s.hostInfo.HasContainerNvidiaGpu() {
					nvidiaGpuMetrics, err = GetNvidiaGpuProcessMetrics()
					if err != nil {
						log.Errorf("GetNvidiaGpuProcessMetrics %s", err)
					}
				}
				if s.hostInfo.HasContainerVastaitechGpu() {
					vastaitechGpuMetrics, err = GetVastaitechGpuProcessMetrics()
					if err != nil {
						log.Errorf("GetVastaitechGpuProcessMetrics %s", err)
					}
				}
				if s.hostInfo.HasContainerCphAmdGpu() {
					cphAmdGpuMetrics, err = GetCphAmdGpuProcessMetrics()
					if err != nil {
						log.Errorf("GetCphAmdGpuProcessMetrics %s", err)
					}
				}
			}

			podStat, podProcs := GetPodStatsById(podStats, gpuPodProcs, guestId)
			if podStat != nil {
				gm, err := NewGuestPodMonitor(
					instance, guestName, guestId, podStat,
					nvidiaGpuMetrics, vastaitechGpuMetrics, cphAmdGpuMetrics,
					s.hostInfo, podProcs, nicsDesc, int(vcpuCount),
				)
				if err != nil {
					return true
				}

				gm.UpdateByInstance(instance)
				gms[guestId] = gm
				return true
			} else {
				delete(s.monitors, guestId)
			}
		}
		return true
	})
	s.monitors = gms
	return gms
}

func (s *SGuestMonitorCollector) CollectReportData(now, last time.Time) (ret string) {
	defer func() {
		if r := recover(); r != nil {
			log.Errorln(r)
			debug.PrintStack()
		}
	}()
	gms := s.GetGuests()
	s.cleanedPrevData(gms)
	reportData := make(map[string]*GuestMetrics)
	for _, gm := range gms {
		prevUsage := s.prevReportData[gm.Id]
		reportData[gm.Id] = s.collectGmReport(gm, prevUsage)
		s.prevPids[gm.Id] = gm.Pid
	}
	s.saveNicTraffics(reportData, gms, now, last)

	s.prevReportData = reportData
	ret = s.toTelegrafReportData(reportData)
	return
}

func (s *SGuestMonitorCollector) saveNicTraffics(reportData map[string]*GuestMetrics, gms map[string]*SGuestMonitor, now, last time.Time) {
	guestman.GetGuestManager().TrafficLock.Lock()
	defer guestman.GetGuestManager().TrafficLock.Unlock()
	isReset := now.Day() != last.Day() // across day
	var guestNicsTraffics = compute.NewGuestNicTrafficSyncInput(now, isReset)
	for guestId, data := range reportData {
		gm := gms[guestId]
		guestTrafficRecord, err := guestman.GetGuestManager().GetGuestTrafficRecord(gm.Id)
		if err != nil {
			log.Errorf("failed get guest traffic record %s", err)
			continue
		}
		guestTrafficsToSend := make(map[string]*compute.SNicTrafficRecord)
		guestTrafficsToSave := make(map[string]*compute.SNicTrafficRecord)
		for i := range gm.Nics {
			if gm.Nics[i].ChargeType != billing_api.NET_CHARGE_TYPE_BY_TRAFFIC {
				continue
			}

			var nicIo *NetIOMetric
			for j := range data.VmNetio {
				if gm.Nics[i].Mac == data.VmNetio[j].Meta.Mac {
					nicIo = data.VmNetio[j]
					break
				}
			}
			if nicIo == nil {
				log.Warningf("failed found report data for nic %s", gm.Nics[i].Ifname)
				continue
			}
			nicHasBeenSetDown := false
			nicTraffic := compute.SNicTrafficRecord{}
			for mac, record := range guestTrafficRecord {
				if mac == nicIo.Meta.Mac {
					nicTraffic.RxTraffic += record.RxTraffic
					nicTraffic.TxTraffic += record.TxTraffic
					nicHasBeenSetDown = record.HasBeenSetDown
				}
			}

			var nicDown = false
			nicTraffic.RxTraffic += int64(nicIo.TimeDiff * nicIo.BPSRecv / 8)
			if gm.Nics[i].RxTrafficLimit > 0 && nicTraffic.RxTraffic >= gm.Nics[i].RxTrafficLimit {
				// nic down
				nicDown = true
			}
			nicTraffic.TxTraffic += int64(nicIo.TimeDiff * nicIo.BPSSent / 8)
			if gm.Nics[i].TxTrafficLimit > 0 && nicTraffic.TxTraffic >= gm.Nics[i].TxTrafficLimit {
				// nic down
				nicDown = true
			}
			if !nicHasBeenSetDown && nicDown {
				log.Infof("guest %s nic %d traffic exceed tx: %d, tx_limit: %d, rx: %d, rx_limit: %d, set nic down", gm.Id, nicIo.Meta.Index, nicTraffic.TxTraffic, gm.Nics[i].TxTrafficLimit, nicTraffic.RxTraffic, gm.Nics[i].RxTrafficLimit)
				gm.SetNicDown(gm.Nics[i].Mac)
				nicTraffic.HasBeenSetDown = true
			}

			guestTrafficsToSend[nicIo.Meta.Mac] = &nicTraffic
			if gm.Nics[i].BillingType == billing_api.BILLING_TYPE_PREPAID || !isReset {
				guestTrafficsToSave[nicIo.Meta.Mac] = &nicTraffic
			}
		}
		if len(guestTrafficsToSend) > 0 {
			guestNicsTraffics.Traffic[gm.Id] = guestTrafficsToSend
		}
		if len(guestTrafficsToSave) > 0 {
			if err = guestman.GetGuestManager().SaveGuestTrafficRecord(gm.Id, guestTrafficsToSave); err != nil {
				log.Errorf("failed save guest %s traffic record %v", gm.Id, guestTrafficsToSave)
				continue
			}
		}
	}
	if len(guestNicsTraffics.Traffic) > 0 {
		guestman.SyncGuestNicsTraffics(guestNicsTraffics)
	}
}

func (s *SGuestMonitorCollector) toTelegrafReportData(data map[string]*GuestMetrics) string {
	ret := []string{}
	for guestId, report := range data {
		var vmName, vmIp, vmIp6, scalingGroupId, tenant, tenantId, domainId, projectDomain, hypervisor string
		if gm, ok := s.monitors[guestId]; ok {
			vmName = gm.Name
			vmIp = gm.Ip
			vmIp6 = gm.Ip6
			scalingGroupId = gm.ScalingGroupId
			tenant = gm.Tenant
			tenantId = gm.TenantId
			domainId = gm.DomainId
			projectDomain = gm.ProjectDomain
			hypervisor = gm.Hypervisor
		}

		tags := map[string]string{
			"id": guestId, "vm_id": guestId, "vm_name": vmName, "hypervisor": hypervisor,
			"is_vm": "true", hostconsts.TELEGRAF_TAG_KEY_BRAND: hostconsts.TELEGRAF_TAG_ONECLOUD_BRAND,
			hostconsts.TELEGRAF_TAG_KEY_RES_TYPE: "guest",
		}
		if len(vmIp) > 0 {
			tags["vm_ip"] = vmIp
		}
		if len(vmIp6) > 0 {
			tags["vm_ip6"] = vmIp6
		}
		if len(scalingGroupId) > 0 {
			tags["vm_scaling_group_id"] = scalingGroupId
		}
		if len(tenant) > 0 {
			tags["tenant"] = tenant
		}
		if len(tenantId) > 0 {
			tags["tenant_id"] = tenantId
		}
		if len(domainId) > 0 {
			tags["domain_id"] = domainId
		}
		if len(projectDomain) > 0 {
			tags["project_domain"] = projectDomain
		}
		ret = append(ret, report.toTelegrafData(tags)...)
	}
	return strings.Join(ret, "\n")
}

func (s *SGuestMonitorCollector) cleanedPrevData(gms map[string]*SGuestMonitor) {
	for guestId := range s.prevReportData {
		if gm, ok := gms[guestId]; !ok {
			delete(s.prevReportData, guestId)
			delete(s.prevPids, guestId)
		} else {
			if s.prevPids[guestId] != gm.Pid {
				delete(s.prevReportData, guestId)
				delete(s.prevPids, guestId)
			}
		}
	}
}

type GuestMetrics struct {
	VmCpu      *CpuMetric     `json:"vm_cpu"`
	VmMem      *MemMetric     `json:"vm_mem"`
	VmNetio    []*NetIOMetric `json:"vm_netio"`
	VmDiskio   *DiskIOMetric  `json:"vm_diskio"`
	PodMetrics *PodMetrics    `json:"pod_metrics"`
}

func (d *GuestMetrics) mapToStatStr(m map[string]interface{}) string {
	var statArr = []string{}
	for k, v := range m {
		if vs, ok := v.(string); ok && len(vs) == 0 {
			continue
		}
		statArr = append(statArr, fmt.Sprintf("%s=%v", k, v))
	}
	return strings.Join(statArr, ",")
}

func (d *GuestMetrics) netioToTelegrafData(measurement string, tagStr string) []string {
	res := []string{}
	for i := range d.VmNetio {
		netTagMap := d.VmNetio[i].ToTag()
		for k, v := range netTagMap {
			if len(k) == 0 || len(v) == 0 {
				continue
			}
			tagStr = fmt.Sprintf("%s,%s=%s", tagStr, k, v)
		}
		res = append(res, fmt.Sprintf("%s,%s %s", measurement, tagStr, d.mapToStatStr(d.VmNetio[i].ToMap())))
	}
	return res
}

func (d *GuestMetrics) toVmTelegrafData(tagStr string) []string {
	var res = []string{}
	res = append(res, fmt.Sprintf("%s,%s %s", "vm_cpu", tagStr, d.mapToStatStr(d.VmCpu.ToMap())))
	res = append(res, fmt.Sprintf("%s,%s %s", "vm_mem", tagStr, d.mapToStatStr(d.VmMem.ToMap())))
	res = append(res, fmt.Sprintf("%s,%s %s", "vm_diskio", tagStr, d.mapToStatStr(d.VmDiskio.ToMap())))
	res = append(res, d.netioToTelegrafData("vm_netio", tagStr)...)
	return res
}

func (d *GuestMetrics) toTelegrafData(tags map[string]string) []string {
	var tagArr = []string{}
	for k, v := range tags {
		tagArr = append(tagArr, fmt.Sprintf("%s=%s", k, strings.ReplaceAll(v, " ", "+")))
	}
	tagStr := strings.Join(tagArr, ",")
	if d.PodMetrics == nil {
		return d.toVmTelegrafData(tagStr)
	} else {
		return d.toPodTelegrafData(tagStr)
	}
}

func (s *SGuestMonitorCollector) collectGmReport(
	gm *SGuestMonitor, prevUsage *GuestMetrics,
) *GuestMetrics {
	if prevUsage == nil {
		prevUsage = new(GuestMetrics)
	}

	if !gm.HasPodMetrics() {
		return s.collectGuestMetrics(gm, prevUsage)
	} else {
		if isPodContainerStopped(prevUsage, gm.podStat) {
			log.Infof("pod %s(%s) has container(s) stopped, clear previous usage", gm.Name, gm.Id)
			prevUsage = new(GuestMetrics)
		}
		return s.collectPodMetrics(gm, prevUsage)
	}
}

func (s *SGuestMonitorCollector) collectGuestMetrics(gm *SGuestMonitor, prevUsage *GuestMetrics) *GuestMetrics {
	gmData := new(GuestMetrics)
	gmData.VmCpu = gm.Cpu()
	gmData.VmMem = gm.Mem()
	gmData.VmDiskio = gm.Diskio()
	gmData.VmNetio = gm.Netio()

	netio1 := gmData.VmNetio
	netio2 := prevUsage.VmNetio
	s.addNetio(netio1, netio2)

	diskio1 := gmData.VmDiskio
	diskio2 := prevUsage.VmDiskio
	s.addDiskio(diskio1, diskio2)
	return gmData
}

func (s *SGuestMonitorCollector) addDiskio(curInfo, prevInfo *DiskIOMetric) {
	if prevInfo != nil {
		s.reportDiskIo(curInfo, prevInfo)
	}
}

func (s *SGuestMonitorCollector) reportDiskIo(cur, prev *DiskIOMetric) {
	timeCur := cur.Meta.Uptime
	timeOld := prev.Meta.Uptime
	diffTime := float64(timeCur - timeOld)

	if diffTime > 0 {
		cur.ReadBPS = float64((cur.ReadBytes-prev.ReadBytes)*8) / diffTime
		cur.WriteBPS = float64((cur.WriteBytes-prev.WriteBytes)*8) / diffTime
		cur.ReadBps = float64(cur.ReadBytes-prev.ReadBytes) / diffTime
		cur.WriteBps = float64(cur.WriteBytes-prev.WriteBytes) / diffTime
		cur.ReadIOPS = float64(cur.ReadCount-prev.ReadCount) / diffTime
		cur.WriteIOPS = float64(cur.WriteCount-prev.WriteCount) / diffTime
	}
}

func (s *SGuestMonitorCollector) addNetio(curInfo, prevInfo []*NetIOMetric) {
	for _, v1 := range curInfo {
		for _, v2 := range prevInfo {
			if v1.Meta.Mac == v2.Meta.Mac {
				s.reportNetIo(v1, v2)
			}
		}
	}
}

func (s *SGuestMonitorCollector) reportNetIo(cur, prev *NetIOMetric) {
	timeCur := cur.Meta.Uptime
	timeOld := prev.Meta.Uptime
	diffTime := float64(timeCur - timeOld)

	cur.TimeDiff = diffTime
	if diffTime > 0 {
		if cur.BytesSent < prev.BytesSent {
			cur.BPSSent = float64(cur.BytesSent*8) / diffTime
		} else {
			cur.BPSSent = float64((cur.BytesSent-prev.BytesSent)*8) / diffTime
		}
		if cur.BytesRecv < prev.BytesRecv {
			cur.BPSRecv = float64(cur.BytesRecv*8) / diffTime
		} else {
			cur.BPSRecv = float64((cur.BytesRecv-prev.BytesRecv)*8) / diffTime
		}
		if cur.PacketsSent < prev.PacketsSent {
			cur.PPSSent = float64(cur.PacketsSent) / diffTime
		} else {
			cur.PPSSent = float64(cur.PacketsSent-prev.PacketsSent) / diffTime
		}
		if cur.PacketsRecv < prev.PacketsRecv {
			cur.PPSRecv = float64(cur.PacketsRecv) / diffTime
		} else {
			cur.PPSRecv = float64(cur.PacketsRecv-prev.PacketsRecv) / diffTime
		}
	}
}

type SGuestMonitor struct {
	Name                    string
	Id                      string
	Pid                     int
	Nics                    []*desc.SGuestNetwork
	CpuCnt                  int
	MemMB                   int64
	Ip                      string
	Ip6                     string
	Process                 *process.Process
	ScalingGroupId          string
	Tenant                  string
	TenantId                string
	DomainId                string
	ProjectDomain           string
	podStat                 *stats.PodStats
	nvidiaGpuMetrics        []NvidiaGpuProcessMetrics
	nvidiaGpuIndexMemoryMap map[string]int
	vastaitechGpuMetrics    []VastaitechGpuProcessMetrics
	cphAmdGpuMetrics        []CphAmdGpuProcessMetrics
	instance                guestman.GuestRuntimeInstance
	sysFs                   sysfs.SysFs

	Hypervisor string `json:"hypervisor"`
}

func NewGuestMonitor(instance guestman.GuestRuntimeInstance, name, id string, pid int, nics []*desc.SGuestNetwork, cpuCount int) (*SGuestMonitor, error) {
	proc, err := process.NewProcess(int32(pid))
	if err != nil {
		return nil, err
	}
	return newGuestMonitor(instance, name, id, proc, nics, cpuCount)
}

func NewGuestPodMonitor(
	instance guestman.GuestRuntimeInstance, name, id string, stat *stats.PodStats,
	nvidiaGpuMetrics []NvidiaGpuProcessMetrics, vastaitechGpuMetrics []VastaitechGpuProcessMetrics, cphAmdGpuMetrics []CphAmdGpuProcessMetrics,
	hostInstance IHostInfo, podProcs map[string]struct{}, nics []*desc.SGuestNetwork, cpuCount int,
) (*SGuestMonitor, error) {
	m, err := newGuestMonitor(instance, name, id, nil, nics, cpuCount)
	if err != nil {
		return nil, errors.Wrap(err, "new pod GuestMonitor")
	}
	m.podStat = stat
	podDesc := instance.GetDesc()

	hasNvGpu := false
	hasCphAmdGpu := false
	hasVastaitechGpu := false
	for i := range podDesc.IsolatedDevices {
		if utils.IsInStringArray(podDesc.IsolatedDevices[i].DevType, compute.NVIDIA_GPU_TYPES) {
			hasNvGpu = true
		} else if podDesc.IsolatedDevices[i].DevType == compute.CONTAINER_DEV_VASTAITECH_GPU {
			hasVastaitechGpu = true
		} else if podDesc.IsolatedDevices[i].DevType == compute.CONTAINER_DEV_CPH_AMD_GPU {
			hasCphAmdGpu = true
		}
	}

	if hasNvGpu {
		m.nvidiaGpuMetrics = GetPodNvidiaGpuMetrics(nvidiaGpuMetrics, podProcs)
		m.nvidiaGpuIndexMemoryMap = hostInstance.GetNvidiaGpuIndexMemoryMap()
	}
	if hasVastaitechGpu {
		m.vastaitechGpuMetrics = GetPodVastaitechGpuMetrics(vastaitechGpuMetrics, podProcs)
	}
	if hasCphAmdGpu {
		m.cphAmdGpuMetrics = GetPodCphAmdGpuMetrics(cphAmdGpuMetrics, podProcs)
	}
	return m, nil
}

func newGuestMonitor(instance guestman.GuestRuntimeInstance, name, id string, proc *process.Process, nics []*desc.SGuestNetwork, cpuCount int) (*SGuestMonitor, error) {
	var ip, ip6 string
	if len(nics) >= 1 {
		for i := range nics {
			if len(ip) == 0 && len(nics[i].Ip) > 0 {
				ip = nics[i].Ip
			}
			if len(ip6) == 0 && len(nics[i].Ip6) > 0 {
				ip6 = nics[i].Ip6
			}
		}
	}
	pid := 0
	if proc != nil {
		pid = int(proc.Pid)
	}
	return &SGuestMonitor{
		Name:     name,
		Id:       id,
		Pid:      pid,
		Nics:     nics,
		CpuCnt:   cpuCount,
		Ip:       ip,
		Ip6:      ip6,
		Process:  proc,
		instance: instance,
		sysFs:    sysfs.NewRealSysFs(),

		Hypervisor: instance.GetDesc().Hypervisor,
	}, nil
}

func (m *SGuestMonitor) UpdateByInstance(instance guestman.GuestRuntimeInstance) {
	guestName := instance.GetDesc().Name
	nicsDesc := instance.GetDesc().Nics
	vcpuCount := instance.GetDesc().Cpu
	m.UpdateVmName(guestName)
	m.UpdateNicsDesc(nicsDesc)
	m.UpdateCpuCount(int(vcpuCount))
	m.MemMB = instance.GetDesc().Mem
	m.ScalingGroupId = instance.GetDesc().ScalingGroupId
	m.Tenant = instance.GetDesc().Tenant
	m.TenantId = instance.GetDesc().TenantId
	m.DomainId = instance.GetDesc().DomainId
	m.ProjectDomain = instance.GetDesc().ProjectDomain
	m.Hypervisor = instance.GetDesc().Hypervisor
}

func (m *SGuestMonitor) SetNicDown(mac string) {
	guest, ok := guestman.GetGuestManager().GetKVMServer(m.Id)
	if !ok {
		return
	}
	if err := guest.SetNicDown(mac); err != nil {
		log.Errorf("guest %s SetNicDown failed %s", m.Id, err)
	}
}

func (m *SGuestMonitor) UpdateVmName(name string) {
	m.Name = name
}

func (m *SGuestMonitor) UpdateNicsDesc(nics []*desc.SGuestNetwork) {
	m.Nics = nics
}

func (m *SGuestMonitor) UpdateCpuCount(vcpuCount int) {
	if vcpuCount < 1 {
		vcpuCount = 1
	}
	m.CpuCnt = vcpuCount
}

func (m *SGuestMonitor) GetSriovNicStats(pfName string, virtfn int) (*psnet.IOCountersStat, error) {
	statsPath := fmt.Sprintf("/sys/class/net/%s/device/sriov/%d/stats", pfName, virtfn)
	if !fileutils2.Exists(statsPath) {
		return nil, nil
	}

	stats, err := fileutils2.FileGetContents(statsPath)
	if err != nil {
		return nil, errors.Wrapf(err, "read file %s", statsPath)
	}
	res := new(psnet.IOCountersStat)
	statsStr := string(stats)
	for _, line := range strings.Split(statsStr, "\n") {
		segs := strings.Split(line, ":")
		if len(segs) != 2 {
			continue
		}
		val, err := strconv.ParseUint(strings.TrimSpace(segs[1]), 10, 64)
		if err != nil {
			log.Errorf("failed parse %s", line)
			continue
		}

		switch strings.TrimSpace(segs[0]) {
		case "tx_packets":
			res.PacketsSent = val
		case "tx_bytes":
			res.BytesSent = val
		case "tx_dropped":
			res.Dropout = val
		case "rx_packets":
			res.PacketsRecv = val
		case "rx_bytes":
			res.BytesRecv = val
		case "rx_dropped":
			res.Dropin = val
		}
	}
	return res, nil
}

func (m *SGuestMonitor) Netio() []*NetIOMetric {
	if len(m.Nics) == 0 {
		return nil
	}
	netstats, err := psnet.IOCounters(true)
	if err != nil {
		return nil
	}

	var res = []*NetIOMetric{}
	for i, nic := range m.Nics {
		var ifname = nic.Ifname
		var nicStat *psnet.IOCountersStat
		if nic.Driver == "vfio-pci" {
			if guest, ok := guestman.GetGuestManager().GetKVMServer(m.Id); ok {
				dev, err := guest.GetSriovDeviceByNetworkIndex(nic.Index)
				if err != nil {
					log.Errorf("failed get sriov deivce by network index %s", err)
					continue
				}
				nicStat, err = m.GetSriovNicStats(dev.GetPfName(), dev.GetVirtfn())
				if err != nil {
					log.Errorf("failed get sriov nic stats: %s", err)
					continue
				}

			} else {
				continue
			}
		} else {
			for j, netstat := range netstats {
				if netstat.Name == ifname {
					nicStat = &netstats[j]
				}
			}
		}

		if nicStat == nil {
			continue
		}
		data := new(NetIOMetric)

		ip := nic.Ip
		if len(ip) > 0 {
			ipv4, _ := netutils.NewIPV4Addr(ip)
			if netutils.IsExitAddress(ipv4) {
				data.Meta.IpType = "external"
			} else {
				data.Meta.IpType = "internal"
			}
		} else {
			data.Meta.IpType = "none"
		}

		data.Meta.Ip = ip
		data.Meta.Ip6 = nic.Ip6
		data.Meta.Index = i
		data.Meta.Mac = nic.Mac
		data.Meta.Ifname = ifname
		data.Meta.NetId = nic.NetId
		data.Meta.Uptime, _ = host.Uptime()

		if nic.Driver == "vfio-pci" {
			data.BytesSent = nicStat.BytesSent
			data.BytesRecv = nicStat.BytesRecv
			data.PacketsRecv = nicStat.PacketsRecv
			data.PacketsSent = nicStat.PacketsSent
			data.ErrIn = nicStat.Errin
			data.ErrOut = nicStat.Errout
			data.DropIn = nicStat.Dropin
			data.DropOut = nicStat.Dropout
		} else {
			data.BytesSent = nicStat.BytesRecv
			data.BytesRecv = nicStat.BytesSent
			data.PacketsRecv = nicStat.PacketsSent
			data.PacketsSent = nicStat.PacketsRecv
			data.ErrIn = nicStat.Errout
			data.ErrOut = nicStat.Errin
			data.DropIn = nicStat.Dropout
			data.DropOut = nicStat.Dropin
		}

		res = append(res, data)
	}
	return res
}

type NetIOMetric struct {
	Meta NetMeta `json:"-"`

	BytesSent   uint64 `json:"bytes_sent"`
	BytesRecv   uint64 `json:"bytes_recv"`
	PacketsSent uint64 `json:"packets_sent"`
	PacketsRecv uint64 `json:"packets_recv"`
	ErrIn       uint64 `json:"err_in"`
	ErrOut      uint64 `json:"err_out"`
	DropIn      uint64 `json:"drop_in"`
	DropOut     uint64 `json:"drop_out"`

	// calculated on guest metrics report
	BPSRecv float64 `json:"bps_recv"`
	BPSSent float64 `json:"bps_sent"`
	PPSRecv float64 `json:"pps_recv"`
	PPSSent float64 `json:"pps_sent"`

	TimeDiff float64 `json:"-"`
}

func (n *NetIOMetric) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"bytes_sent":   n.BytesSent,
		"bytes_recv":   n.BytesRecv,
		"packets_sent": n.PacketsSent,
		"packets_recv": n.PacketsRecv,
		"err_in":       n.ErrIn,
		"err_out":      n.ErrOut,
		"drop_in":      n.DropIn,
		"drop_out":     n.DropOut,
		"bps_recv":     n.BPSRecv,
		"bps_sent":     n.BPSSent,
		"pps_recv":     n.PPSRecv,
		"pps_sent":     n.PPSSent,
	}
}

func (n *NetIOMetric) ToTag() map[string]string {
	tags := map[string]string{
		"interface":      fmt.Sprintf("eth%d", n.Meta.Index),
		"host_interface": n.Meta.Ifname,
		"mac":            n.Meta.Mac,
		"ip_type":        n.Meta.IpType,
	}
	if len(n.Meta.Ip) > 0 {
		tags["ip"] = n.Meta.Ip
	}
	if len(n.Meta.Ip6) > 0 {
		tags["ip6"] = n.Meta.Ip6
	}
	return tags
}

type NetMeta struct {
	IpType string `json:"ip_type"`
	Ip     string `json:"ip"`
	Mac    string `json:"mac"`
	Index  int    `json:"index"`
	Ifname string `json:"ifname"`
	NetId  string `json:"net_id"`
	Uptime uint64 `json:"uptime"`
	Ip6    string `json:"ip6"`
}

func (m *SGuestMonitor) Cpu() *CpuMetric {
	percent, _ := m.Process.Percent(time.Millisecond * 100)
	cpuTimes, _ := m.Process.Times()
	percent, _ = strconv.ParseFloat(fmt.Sprintf("%0.4f", percent/float64(m.CpuCnt)), 64)
	threadCnt, _ := m.Process.NumThreads()
	return &CpuMetric{
		UsageActive:       percent,
		CpuUsageIdlePcore: float64(100 - percent/float64(m.CpuCnt)),
		CpuUsagePcore:     float64(percent / float64(m.CpuCnt)),
		CpuTimeSystem:     cpuTimes.System,
		CpuTimeUser:       cpuTimes.User,
		CpuCount:          m.CpuCnt,
		ThreadCount:       threadCnt,
	}
}

type CpuMetric struct {
	UsageActive       float64 `json:"usage_active"`
	CpuUsageIdlePcore float64 `json:"cpu_usage_idle_pcore"`
	CpuUsagePcore     float64 `json:"cpu_usage_pcore"`
	CpuTimeUser       float64 `json:"cpu_time_user"`
	CpuTimeSystem     float64 `json:"cpu_time_system"`
	CpuCount          int     `json:"cpu_count"`
	ThreadCount       int32   `json:"thread_count"`
}

func (c *CpuMetric) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"usage_active":         c.UsageActive,
		"cpu_usage_idle_pcore": c.CpuUsageIdlePcore,
		"cpu_usage_pcore":      c.CpuUsagePcore,
		"cpu_time_user":        c.CpuTimeUser,
		"cpu_time_system":      c.CpuTimeSystem,
		"cpu_count":            c.CpuCount,
		"thread_count":         c.ThreadCount,
	}
}

func (m *SGuestMonitor) Diskio() *DiskIOMetric {
	io, err := m.Process.IOCounters()
	if err != nil {
		log.Errorln(err)
		return nil
	}
	ret := new(DiskIOMetric)

	ret.Meta.Uptime, _ = host.Uptime()
	ret.ReadCount = io.ReadCount
	ret.ReadBytes = io.ReadBytes
	ret.WriteBytes = io.WriteBytes
	ret.WriteCount = io.WriteCount
	return ret
}

type DiskIOMetric struct {
	Meta DiskIOMeta `json:"-"`

	ReadBytes  uint64 `json:"read_bytes"`
	WriteBytes uint64 `json:"write_bytes"`
	ReadCount  uint64 `json:"read_count"`
	WriteCount uint64 `json:"write_count"`

	// calculated on guest metrics report
	ReadBps   float64 `json:"read_Bps"`
	WriteBps  float64 `json:"write_Bps"`
	ReadBPS   float64 `json:"read_bps"`
	WriteBPS  float64 `json:"write_bps"`
	ReadIOPS  float64 `json:"read_iops"`
	WriteIOPS float64 `json:"write_iops"`
}

func (d *DiskIOMetric) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"read_bytes":  d.ReadBytes,
		"write_bytes": d.WriteBytes,
		"read_count":  d.ReadCount,
		"write_count": d.WriteCount,
		"read_Bps":    d.ReadBps,
		"write_Bps":   d.WriteBps,
		"read_bps":    d.ReadBPS,
		"write_bps":   d.WriteBPS,
		"read_iops":   d.ReadIOPS,
		"write_iops":  d.WriteIOPS,
	}
}

type DiskIOMeta struct {
	Uptime uint64 `json:"uptime"`
}

func (m *SGuestMonitor) Mem() *MemMetric {
	mem, err := m.Process.MemoryInfo()
	usedPercent, _ := m.Process.MemoryPercent()
	if err != nil {
		log.Errorln(err)
		return nil
	}
	ret := new(MemMetric)
	ret.RSS = mem.RSS
	ret.VMS = mem.VMS
	ret.UsedPercent = usedPercent
	return ret
}

type MemMetric struct {
	RSS         uint64  `json:"rss"`
	VMS         uint64  `json:"vms"`
	UsedPercent float32 `json:"used_percent"`
}

func (m *MemMetric) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"rss":          m.RSS,
		"vms":          m.VMS,
		"used_percent": m.UsedPercent,
	}
}
