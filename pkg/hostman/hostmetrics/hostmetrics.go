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
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/shirou/gopsutil/host"
	psnet "github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"
	"yunion.io/x/pkg/util/netutils"

	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/hostman/guestman"
	"yunion.io/x/onecloud/pkg/hostman/guestman/desc"
	"yunion.io/x/onecloud/pkg/hostman/hostinfo/hostconsts"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
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

func Init() {
	if hostMetricsCollector == nil {
		hostMetricsCollector = NewHostMetricsCollector()
	}
}

func Start() {
	if hostMetricsCollector != nil {
		go hostMetricsCollector.Start()
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
	} else {
		m.LastCollectTime = timeBegin
	}
	m.runMonitor()
}

func (m *SHostMetricsCollector) runMonitor() {
	reportData := m.collectReportData()
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
		log.Errorf("upload guest metric failed %d", res.StatusCode)
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

func (m *SHostMetricsCollector) collectReportData() string {
	if len(m.waitingReportData) > 60 {
		m.waitingReportData = m.waitingReportData[1:]
	}
	return m.guestMonitor.CollectReportData()
}

func NewHostMetricsCollector() *SHostMetricsCollector {
	return &SHostMetricsCollector{
		ReportInterval:    options.HostOptions.ReportInterval,
		waitingReportData: make([]string, 0),
		guestMonitor:      NewGuestMonitorCollector(),
	}
}

type SGuestMonitorCollector struct {
	monitors       map[string]*SGuestMonitor
	prevPids       map[string]int
	prevReportData map[string]*GuestMetrics
}

func NewGuestMonitorCollector() *SGuestMonitorCollector {
	return &SGuestMonitorCollector{
		monitors:       make(map[string]*SGuestMonitor, 0),
		prevPids:       make(map[string]int, 0),
		prevReportData: make(map[string]*GuestMetrics, 0),
	}
}

func (s *SGuestMonitorCollector) GetGuests() map[string]*SGuestMonitor {
	var err error
	gms := make(map[string]*SGuestMonitor, 0)
	guestmanager := guestman.GetGuestManager()
	guestmanager.Servers.Range(func(k, v interface{}) bool {
		guest, ok := v.(*guestman.SKVMGuestInstance)
		if !ok {
			return false
		}
		if !guest.IsValid() {
			return false
		}
		pid := guest.GetPid()
		if pid > 0 {
			guestName := guest.Desc.Name
			guestId := guest.GetId()
			nicsDesc := guest.Desc.Nics
			vcpuCount := guest.Desc.Cpu
			gm, ok := s.monitors[guestId]
			if ok && gm.Pid == pid {
				delete(s.monitors, guestId)
				gm.UpdateVmName(guestName)
				gm.UpdateNicsDesc(nicsDesc)
				gm.UpdateCpuCount(int(vcpuCount))
			} else {
				delete(s.monitors, guestId)
				gm, err = NewGuestMonitor(guestName, guestId, pid, nicsDesc, int(vcpuCount))
				if err != nil {
					log.Errorf("NewGuestMonitor for %s(%s), pid: %d, nics: %#v", guestName, guestId, pid, nicsDesc)
					return true
				}
			}
			gm.ScalingGroupId = guest.Desc.ScalingGroupId
			gm.Tenant = guest.Desc.Tenant
			gm.TenantId = guest.Desc.TenantId
			gm.DomainId = guest.Desc.DomainId
			gm.ProjectDomain = guest.Desc.ProjectDomain

			gms[guestId] = gm
		}
		return true
	})
	s.monitors = gms
	return gms
}

func (s *SGuestMonitorCollector) CollectReportData() (ret string) {
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
		prevUsage, _ := s.prevReportData[gm.Id]
		reportData[gm.Id] = s.collectGmReport(gm, prevUsage)
		s.prevPids[gm.Id] = gm.Pid
	}
	s.saveNicTraffics(reportData, gms)

	s.prevReportData = reportData
	ret = s.toTelegrafReportData(reportData)
	return
}

func (s *SGuestMonitorCollector) saveNicTraffics(reportData map[string]*GuestMetrics, gms map[string]*SGuestMonitor) {
	guestman.GetGuestManager().TrafficLock.Lock()
	defer guestman.GetGuestManager().TrafficLock.Unlock()
	var guestNicsTraffics = make(map[string]map[string]compute.SNicTrafficRecord)
	for guestId, data := range reportData {
		gm := gms[guestId]
		guestTrafficRecord, err := guestman.GetGuestManager().GetGuestTrafficRecord(gm.Id)
		if err != nil {
			log.Errorf("failed get guest traffic record %s", err)
			continue
		}
		guestTraffics := make(map[string]compute.SNicTrafficRecord)
		for i := range gm.Nics {
			if gm.Nics[i].RxTrafficLimit <= 0 && gm.Nics[i].TxTrafficLimit <= 0 {
				continue
			}

			var nicIo *NetIOMetric
			for j := range data.VmNetio {
				if gm.Nics[i].Index == int8(data.VmNetio[j].Meta.Index) {
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
			for index, record := range guestTrafficRecord {
				if index == strconv.Itoa(nicIo.Meta.Index) {
					nicTraffic.RxTraffic += record.RxTraffic
					nicTraffic.TxTraffic += record.TxTraffic
					nicHasBeenSetDown = record.HasBeenSetDown
				}
			}

			if gm.Nics[i].RxTrafficLimit > 0 || gm.Nics[i].TxTrafficLimit > 0 {
				var nicDown = false
				if gm.Nics[i].RxTrafficLimit > 0 {
					nicTraffic.RxTraffic += int64(nicIo.TimeDiff * nicIo.BPSRecv / 8)
					if nicTraffic.RxTraffic >= gm.Nics[i].RxTrafficLimit {
						// nic down
						nicDown = true
					}
				}
				if gm.Nics[i].TxTrafficLimit > 0 {
					nicTraffic.TxTraffic += int64(nicIo.TimeDiff * nicIo.BPSSent / 8)
					if nicTraffic.TxTraffic >= gm.Nics[i].TxTrafficLimit {
						// nic down
						nicDown = true
					}
				}
				if !nicHasBeenSetDown && nicDown {
					log.Infof("guest %s nic %d traffic exceed, set nic down", gm.Id, nicIo.Meta.Index)
					gm.SetNicDown(nicIo.Meta.Index)
					nicTraffic.HasBeenSetDown = true
				}
				guestTraffics[strconv.Itoa(nicIo.Meta.Index)] = nicTraffic
			}
		}
		if len(guestTraffics) == 0 {
			continue
		}
		guestNicsTraffics[gm.Id] = guestTraffics
		if err = guestman.GetGuestManager().SaveGuestTrafficRecord(gm.Id, guestTraffics); err != nil {
			log.Errorf("failed save guest %s traffic record %v", gm.Id, guestTraffics)
			continue
		}
	}
	if len(guestNicsTraffics) > 0 {
		guestman.SyncGuestNicsTraffics(guestNicsTraffics)
	}
}

func (s *SGuestMonitorCollector) toTelegrafReportData(data map[string]*GuestMetrics) string {
	ret := []string{}
	for guestId, report := range data {
		var vmName, vmIp, scalingGroupId, tenant, tenantId, domainId, projectDomain string
		if gm, ok := s.monitors[guestId]; ok {
			vmName = gm.Name
			vmIp = gm.Ip
			scalingGroupId = gm.ScalingGroupId
			tenant = gm.Tenant
			tenantId = gm.TenantId
			domainId = gm.DomainId
			projectDomain = gm.ProjectDomain
		}

		tags := map[string]string{
			"id": guestId, "vm_id": guestId, "vm_name": vmName, "vm_ip": vmIp,
			"is_vm": "true", hostconsts.TELEGRAF_TAG_KEY_BRAND: hostconsts.TELEGRAF_TAG_ONECLOUD_BRAND,
			hostconsts.TELEGRAF_TAG_KEY_RES_TYPE: "guest",
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
	VmCpu    *CpuMetric     `json:"vm_cpu"`
	VmMem    *MemMetric     `json:"vm_mem"`
	VmNetio  []*NetIOMetric `json:"vm_netio"`
	VmDiskio *DiskIOMetric  `json:"vm_diskio"`
}

func (d *GuestMetrics) toTelegrafData(tags map[string]string) []string {
	var tagArr = []string{}
	for k, v := range tags {
		tagArr = append(tagArr, fmt.Sprintf("%s=%s", k, strings.ReplaceAll(v, " ", "+")))
	}
	tagStr := strings.Join(tagArr, ",")

	mapToStatStr := func(m map[string]interface{}) string {
		var statArr = []string{}
		for k, v := range m {
			statArr = append(statArr, fmt.Sprintf("%s=%v", k, v))
		}
		return strings.Join(statArr, ",")
	}

	var res = []string{}
	res = append(res, fmt.Sprintf("%s,%s %s", "vm_cpu", tagStr, mapToStatStr(d.VmCpu.ToMap())))
	res = append(res, fmt.Sprintf("%s,%s %s", "vm_mem", tagStr, mapToStatStr(d.VmMem.ToMap())))
	res = append(res, fmt.Sprintf("%s,%s %s", "vm_diskio", tagStr, mapToStatStr(d.VmDiskio.ToMap())))
	for i := range d.VmNetio {
		res = append(res, fmt.Sprintf("%s,%s %s", "vm_netio", tagStr, mapToStatStr(d.VmNetio[i].ToMap())))
	}
	return res
}

func (s *SGuestMonitorCollector) collectGmReport(
	gm *SGuestMonitor, prevUsage *GuestMetrics,
) *GuestMetrics {
	if prevUsage == nil {
		prevUsage = new(GuestMetrics)
	}
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
			if v1.Meta.Ip == v2.Meta.Ip {
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
	Name           string
	Id             string
	Pid            int
	Nics           []*desc.SGuestNetwork
	CpuCnt         int
	Ip             string
	Process        *process.Process
	ScalingGroupId string
	Tenant         string
	TenantId       string
	DomainId       string
	ProjectDomain  string
}

func NewGuestMonitor(name, id string, pid int, nics []*desc.SGuestNetwork, cpuCount int,
) (*SGuestMonitor, error) {
	var ip string
	if len(nics) >= 1 {
		ip = nics[0].Ip
	}
	proc, err := process.NewProcess(int32(pid))
	if err != nil {
		return nil, err
	}
	return &SGuestMonitor{name, id, pid, nics, cpuCount, ip, proc, "", "", "", "", ""}, nil
}

func (m *SGuestMonitor) SetNicDown(index int) {
	guest, ok := guestman.GetGuestManager().GetKVMServer(m.Id)
	if !ok {
		return
	}
	if err := guest.SetNicDown(int8(index)); err != nil {
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
		ipv4, _ := netutils.NewIPV4Addr(ip)
		if netutils.IsExitAddress(ipv4) {
			data.Meta.IpType = "external"
		} else {
			data.Meta.IpType = "internal"
		}
		data.Meta.Ip = ip
		data.Meta.Index = i
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

type NetMeta struct {
	IpType string `json:"ip_type"`
	Ip     string `json:"ip"`
	Index  int    `json:"index"`
	Ifname string `json:"ifname"`
	NetId  string `json:"net_id"`
	Uptime uint64 `json:"uptime"`
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
