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
	"reflect"
	"runtime/debug"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/shirou/gopsutil/host"
	psnet "github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/httputils"
	"yunion.io/x/pkg/util/netutils"

	"yunion.io/x/onecloud/pkg/hostman/guestman"
	"yunion.io/x/onecloud/pkg/hostman/guestman/desc"
	"yunion.io/x/onecloud/pkg/hostman/hostinfo/hostconsts"
	"yunion.io/x/onecloud/pkg/hostman/options"
)

const (
	TelegrafServer     = "http://127.0.0.1:8087/write"
	MeasurementsPrefix = "vm_"
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
	prevReportData *jsonutils.JSONDict
}

func NewGuestMonitorCollector() *SGuestMonitorCollector {
	return &SGuestMonitorCollector{
		monitors:       make(map[string]*SGuestMonitor, 0),
		prevPids:       make(map[string]int, 0),
		prevReportData: jsonutils.NewDict(),
	}
}

func (s *SGuestMonitorCollector) GetGuests() map[string]*SGuestMonitor {
	var err error
	gms := make(map[string]*SGuestMonitor, 0)
	guestmanager := guestman.GetGuestManager()
	guestmanager.Servers.Range(func(k, v interface{}) bool {
		guest := v.(*guestman.SKVMGuestInstance)
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
					log.Errorf("NewGuestMonitor for %s(%s), pid: %d, nics: %s", guestName, guestId, pid, jsonutils.Marshal(nicsDesc).String())
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
	reportData := jsonutils.NewDict()
	for _, gm := range gms {
		prevUsage, _ := s.prevReportData.Get(gm.Id)
		usage := s.collectGmReport(gm, prevUsage)
		reportData.Set(gm.Id, usage)
		s.prevPids[gm.Id] = gm.Pid
	}

	s.prevReportData = reportData.DeepCopy().(*jsonutils.JSONDict)
	ret = s.toTelegrafReportData(reportData)
	return
}

func (s *SGuestMonitorCollector) toTelegrafReportData(data *jsonutils.JSONDict) string {
	ret := []string{}
	vs, _ := data.GetMap()
	for guestId, report := range vs {
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

		rs, _ := report.(*jsonutils.JSONDict).GetMap()
		for metrics, stat := range rs {
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
			if val, ok := stat.(*jsonutils.JSONDict); ok {
				line := s.addTelegrafLine(metrics, tags, val)
				ret = append(ret, line)
			} else if val, ok := stat.(*jsonutils.JSONArray); ok {
				ss, _ := val.GetArray()
				for _, statItem := range ss {
					line := s.addTelegrafLine(metrics, tags, statItem.(*jsonutils.JSONDict))
					ret = append(ret, line)
				}
			}
		}
	}
	return strings.Join(ret, "\n")
}

func (s *SGuestMonitorCollector) addTelegrafLine(
	metrics string, tags map[string]string, stat *jsonutils.JSONDict,
) string {
	meta, _ := stat.GetMap("meta")
	stat.Remove("meta")
	if meta != nil {
		delete(meta, "uptime")
	}

	var tagArr = []string{}
	for k, v := range tags {
		tagArr = append(tagArr, fmt.Sprintf("%s=%s", k, strings.ReplaceAll(v, " ", "+")))
	}
	tagStr := strings.Join(tagArr, ",")

	var statArr = []string{}
	ss, _ := stat.GetMap()
	for k, v := range ss {
		statArr = append(statArr, fmt.Sprintf("%s=%s", k, v.String()))
	}
	statStr := strings.Join(statArr, ",")
	return fmt.Sprintf("%s,%s %s", metrics, tagStr, statStr)
}

func (s *SGuestMonitorCollector) cleanedPrevData(gms map[string]*SGuestMonitor) {
	rs, _ := s.prevReportData.GetMap()
	for guestId := range rs {
		if gm, ok := gms[guestId]; !ok {
			s.prevReportData.Remove(guestId)
			delete(s.prevPids, guestId)
		} else {
			if s.prevPids[guestId] != gm.Pid {
				s.prevReportData.Remove(guestId)
				delete(s.prevPids, guestId)
			}
		}
	}
}

func (s *SGuestMonitorCollector) collectGmReport(
	gm *SGuestMonitor, prevUsage jsonutils.JSONObject,
) *jsonutils.JSONDict {
	if prevUsage == nil {
		prevUsage = jsonutils.NewDict()
	}
	gmData := jsonutils.NewDict()
	v := reflect.ValueOf(gm)
	for _, k := range []string{"Netio", "Cpu", "Diskio", "Mem"} {
		res := v.MethodByName(k).Call(nil)
		if !res[0].IsNil() {
			val := res[0].Interface()
			in := []rune(k)
			in[0] = unicode.ToLower(in[0])
			key := MeasurementsPrefix + string(in)
			gmData.Set(key, val.(jsonutils.JSONObject))
		}
	}
	gmNetio := MeasurementsPrefix + "netio"
	netio1, err1 := gmData.Get(gmNetio)
	netio2, err2 := prevUsage.Get(gmNetio)
	if err1 == nil && err2 == nil {
		s.addNetio(netio1, netio2,
			[]string{"bits_recv", "bits_sent", "packets_sent", "packets_recv"})
	}

	gmDiskio := MeasurementsPrefix + "diskio"
	diskio1, err1 := gmData.Get(gmDiskio)
	diskio2, err2 := prevUsage.Get(gmDiskio)
	if err1 == nil && err2 == nil {
		s.addDiskio(diskio1, diskio2, []string{"read_bytes", "write_bytes", "read_bits", "write_bits", "read_count",
			"write_count"})
	}
	return gmData
}

func (s *SGuestMonitorCollector) GetIoFiledName(field string) string {
	kmap := map[string]string{
		"bits": "bps", "bytes": "Bps", "packets": "pps", "count": "iops",
	}
	for k, v := range kmap {
		if strings.Contains(field, k) {
			return strings.Replace(field, k, v, -1)
		}
	}
	return field + "_per_seconds"
}

func (s *SGuestMonitorCollector) reportIo(curInfo, prevInfo jsonutils.JSONObject, fields []string,
) *jsonutils.JSONDict {
	ioInfo := jsonutils.NewDict()

	var timeCur int64
	uptime, err := curInfo.Get("meta")
	if err == nil {
		timeCur, _ = uptime.Int("uptime")
	}

	var timeOld int64
	uptime, err = prevInfo.Get("meta")
	if err == nil {
		timeOld, _ = uptime.Int("uptime")
	}
	diffTime := timeCur - timeOld

	if diffTime > 0 {
		for _, field := range fields {
			cur, _ := curInfo.GetString(field)
			prev, _ := prevInfo.GetString(field)
			fcur, _ := strconv.ParseFloat(cur, 64)
			fprev, _ := strconv.ParseFloat(prev, 64)
			ioInfo.Set(s.GetIoFiledName(field), jsonutils.NewFloat64((fcur-fprev)/float64(diffTime)))
		}
	}
	return ioInfo
}

func (s *SGuestMonitorCollector) addDiskio(curInfo, prevInfo jsonutils.JSONObject, fields []string) {
	ioInfo := s.reportIo(curInfo, prevInfo, fields)
	curInfo.(*jsonutils.JSONDict).Update(ioInfo)
}

func (s *SGuestMonitorCollector) addNetio(curInfo, prevInfo jsonutils.JSONObject, fields []string) {
	curArray, _ := curInfo.GetArray()
	prevArray, _ := prevInfo.GetArray()
	for _, v1 := range curArray {
		for _, v2 := range prevArray {
			if v1.Contains("meta", "ip") && v2.Contains("meta", "ip") {
				ip1, _ := v1.GetString("meta", "ip")
				ip2, _ := v2.GetString("meta", "ip")
				if ip1 == ip2 {
					ioInfo := s.reportIo(v1, v2, fields)
					v1.(*jsonutils.JSONDict).Update(ioInfo)
				}
			}
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

func (m *SGuestMonitor) Netio() jsonutils.JSONObject {
	if len(m.Nics) == 0 {
		return nil
	}
	netstats, err := psnet.IOCounters(true)
	if err != nil {
		return nil
	}

	var res = jsonutils.NewArray()
	for i, nic := range m.Nics {
		ifname := nic.Ifname
		var nicStat *psnet.IOCountersStat
		for j, netstat := range netstats {
			if netstat.Name == ifname {
				nicStat = &netstats[j]
			}
		}
		if nicStat == nil {
			continue
		}
		data := jsonutils.NewDict()
		meta := jsonutils.NewDict()

		ip := nic.Ip
		ipv4, _ := netutils.NewIPV4Addr(ip)
		if netutils.IsExitAddress(ipv4) {
			meta.Set("ip_type", jsonutils.NewString("external"))
		} else {
			meta.Set("ip_type", jsonutils.NewString("internal"))
		}

		netId := nic.NetId
		meta.Set("ip", jsonutils.NewString(ip))
		meta.Set("index", jsonutils.NewInt(int64(i)))
		meta.Set("ifname", jsonutils.NewString(ifname))
		meta.Set("net_id", jsonutils.NewString(netId))
		uptime, _ := host.Uptime()
		meta.Set("uptime", jsonutils.NewInt(int64(uptime)))
		data.Set("meta", meta)
		data.Set("bits_recv", jsonutils.NewInt(int64(nicStat.BytesSent*8)))
		data.Set("bits_sent", jsonutils.NewInt(int64(nicStat.BytesRecv*8)))
		data.Set("packets_recv", jsonutils.NewInt(int64(nicStat.PacketsSent)))
		data.Set("packets_sent", jsonutils.NewInt(int64(nicStat.PacketsRecv)))
		data.Set("err_out", jsonutils.NewInt(int64(nicStat.Errin)))
		data.Set("err_in", jsonutils.NewInt(int64(nicStat.Errout)))
		data.Set("drop_out", jsonutils.NewInt(int64(nicStat.Dropin)))
		data.Set("drop_in", jsonutils.NewInt(int64(nicStat.Dropout)))
		res.Add(data)
	}
	return res
}

func (m *SGuestMonitor) Cpu() jsonutils.JSONObject {
	percent, _ := m.Process.Percent(time.Millisecond * 100)
	cpuTimes, _ := m.Process.Times()
	ret := jsonutils.NewDict()
	percent, _ = strconv.ParseFloat(fmt.Sprintf("%0.4f", percent/float64(m.CpuCnt)), 64)
	ret.Set("usage_active", jsonutils.NewFloat64(percent))
	ret.Set("cpu_usage_idle_pcore", jsonutils.NewFloat64(100-percent/float64(m.CpuCnt)))
	ret.Set("cpu_usage_pcore", jsonutils.NewFloat64(percent/float64(m.CpuCnt)))
	ret.Set("cpu_time_user", jsonutils.NewFloat64(cpuTimes.User))
	ret.Set("cpu_time_system", jsonutils.NewFloat64(cpuTimes.System))
	ret.Set("cpu_count", jsonutils.NewInt(int64(m.CpuCnt)))

	threadCnt, _ := m.Process.NumThreads()
	ret.Set("thread_count", jsonutils.NewInt(int64(threadCnt)))
	return ret
}

func (m *SGuestMonitor) Diskio() jsonutils.JSONObject {
	io, err := m.Process.IOCounters()
	if err != nil {
		log.Errorln(err)
		return nil
	}
	ret := jsonutils.NewDict()
	meta := jsonutils.NewDict()

	uptime, _ := host.Uptime()
	meta.Set("uptime", jsonutils.NewInt(int64(uptime)))
	ret.Set("meta", meta)
	ret.Set("read_bytes", jsonutils.NewInt(int64(io.ReadBytes)))
	ret.Set("write_bytes", jsonutils.NewInt(int64(io.WriteBytes)))
	ret.Set("read_bits", jsonutils.NewInt(int64(io.ReadBytes)*8))
	ret.Set("write_bits", jsonutils.NewInt(int64(io.WriteBytes)*8))
	ret.Set("read_count", jsonutils.NewInt(int64(io.ReadCount)))
	ret.Set("write_count", jsonutils.NewInt(int64(io.WriteCount)))
	return ret
}

func (m *SGuestMonitor) Mem() jsonutils.JSONObject {
	mem, err := m.Process.MemoryInfo()
	used_percent, _ := m.Process.MemoryPercent()
	if err != nil {
		log.Errorln(err)
		return nil
	}
	ret := jsonutils.NewDict()
	ret.Set("rss", jsonutils.NewInt(int64(mem.RSS)))
	ret.Set("vms", jsonutils.NewInt(int64(mem.VMS)))
	ret.Set("used_percent", jsonutils.NewFloat64(float64(used_percent)))
	return ret
}
