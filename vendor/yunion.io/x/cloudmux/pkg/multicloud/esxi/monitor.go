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

package esxi

import (
	"fmt"
	"sync"
	"time"

	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/performance"
	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25/methods"
	"github.com/vmware/govmomi/vim25/types"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

const (
	MONTYPE_HOSTSYSTEM     = "HostSystem"
	MONTYPE_VIRTUALMACHINE = "VirtualMachine"
)

var (
	metricTimeSkewLogMu sync.Mutex
	metricTimeSkewLogAt = map[string]time.Time{}
)

// alignMetricTimeRange 将本地计算的查询窗口对齐到 vCenter 时钟。
// cloudmon 比 vCenter 快时，未对齐的时间窗对 vCenter 是「未来」，QueryPerf 会返回空。
func (self *SESXiClient) alignMetricTimeRange(start, end time.Time) (time.Time, time.Time) {
	startUTC := start.UTC()
	endUTC := end.UTC()
	vcNow, err := methods.GetCurrentTime(self.context, self.client.Client)
	if err != nil || vcNow == nil {
		return startUTC, endUTC
	}
	skew := time.Since(*vcNow)
	if skew > time.Minute {
		startUTC = startUTC.Add(-skew)
		endUTC = endUTC.Add(-skew)
		metricTimeSkewLogMu.Lock()
		if time.Since(metricTimeSkewLogAt[self.host]) >= time.Hour {
			log.Warningf("esxi metric time skew %s ahead of vCenter %s, aligned query window", skew.Round(time.Second), self.host)
			metricTimeSkewLogAt[self.host] = time.Now()
		}
		metricTimeSkewLogMu.Unlock()
	}
	return startUTC, endUTC
}

func (self *SESXiClient) GetEcsMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	metricName := ""
	switch opts.MetricType {
	case cloudprovider.VM_METRIC_TYPE_CPU_USAGE:
		metricName = "cpu.usage.average"
	case cloudprovider.VM_METRIC_TYPE_MEM_USAGE:
		metricName = "mem.usage.average"
	case cloudprovider.VM_METRIC_TYPE_NET_BPS_RX:
		metricName = "net.bytesRx.average"
	case cloudprovider.VM_METRIC_TYPE_NET_BPS_TX:
		metricName = "net.bytesTx.average"
	case cloudprovider.VM_METRIC_TYPE_DISK_IO_READ_BPS:
		metricName = "disk.read.average"
	case cloudprovider.VM_METRIC_TYPE_DISK_IO_WRITE_BPS:
		metricName = "disk.write.average"
	case cloudprovider.VM_METRIC_TYPE_DISK_USAGE:
		metricName = "disk.used.latest"
	default:
		return nil, errors.Wrapf(cloudprovider.ErrNotSupported, "%s", opts.MetricType)
	}
	return self.getEcsMetrics(metricName, opts.MetricType, opts.StartTime, opts.EndTime)
}

func (self *SESXiClient) getEcsMetrics(metricName string, metricType cloudprovider.TMetricType, start, end time.Time) ([]cloudprovider.MetricValues, error) {
	start, end = self.alignMetricTimeRange(start, end)

	m := view.NewManager(self.client.Client)
	v, err := m.CreateContainerView(self.context, self.client.Client.ServiceContent.RootFolder, nil, true)
	if err != nil {
		return nil, errors.Wrapf(err, "CreateContainerView")
	}
	vms, err := v.Find(self.context, []string{MONTYPE_VIRTUALMACHINE}, nil)
	if err != nil {
		return nil, err
	}
	perfManager := performance.NewManager(self.client.Client)
	counterInfo, err := perfManager.CounterInfoByName(self.context)
	if err != nil {
		return nil, errors.Wrapf(err, "CounterInfoByName")
	}

	counter, ok := counterInfo[metricName]
	if !ok {
		return nil, fmt.Errorf("not found %s", metricName)
	}

	queries := []types.PerfQuerySpec{}
	for i := range vms {
		s, e := start, end
		query := types.PerfQuerySpec{
			Entity: vms[i].Reference(),
			MetricId: []types.PerfMetricId{
				{
					CounterId: counter.Key,
				},
			},
			Format:     "normal",
			IntervalId: 20,
			StartTime:  &s,
			EndTime:    &e,
		}
		if metricType == cloudprovider.VM_METRIC_TYPE_DISK_USAGE {
			query.IntervalId = 0
		}
		queries = append(queries, query)
	}

	sample, err := perfManager.Query(self.context, queries)
	if err != nil {
		return nil, errors.Wrap(err, "Query")
	}

	result, err := perfManager.ToMetricSeries(self.context, sample)
	if err != nil {
		return nil, errors.Wrap(err, "ToMetricSeries")
	}

	ret := []cloudprovider.MetricValues{}
	for _, metric := range result {
		vm := object.NewVirtualMachine(self.client.Client, metric.Entity)
		metricValue := cloudprovider.MetricValues{}
		metricValue.MetricType = metricType
		metricValue.Id = vm.UUID(self.context)
		if len(metric.Value) == 0 {
			continue
		}
		for i, v := range metric.SampleInfo {
			if v.Timestamp.Second() != 0 {
				continue
			}
			var value float64
			if len(metric.Value[0].Value) > i {
				value = float64(metric.Value[0].Value[i])
			}
			switch counter.UnitInfo.GetElementDescription().Key {
			case "percent":
				value = value / 100.0
			case "kiloBytesPerSecond", "kiloBytes":
				value = value * 1024.0
			default:
				log.Errorf("unknow unit: %s", counter.UnitInfo.GetElementDescription().Key)
			}
			metricValue.Values = append(metricValue.Values, cloudprovider.MetricValue{
				Timestamp: v.Timestamp,
				Value:     value,
			})
		}
		ret = append(ret, metricValue)
	}
	return ret, nil
}

func (self *SESXiClient) GetHostMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	metricName := ""
	switch opts.MetricType {
	case cloudprovider.HOST_METRIC_TYPE_CPU_USAGE:
		metricName = "cpu.usage.average"
	case cloudprovider.HOST_METRIC_TYPE_MEM_USAGE:
		metricName = "mem.usage.average"
	case cloudprovider.HOST_METRIC_TYPE_NET_BPS_RX:
		metricName = "net.received.average"
	case cloudprovider.HOST_METRIC_TYPE_NET_BPS_TX:
		metricName = "net.transmitted.average"
	case cloudprovider.HOST_METRIC_TYPE_DISK_IO_READ_BPS:
		metricName = "disk.read.average"
	case cloudprovider.HOST_METRIC_TYPE_DISK_IO_WRITE_BPS:
		metricName = "disk.write.average"
	default:
		return nil, errors.Wrapf(cloudprovider.ErrNotSupported, "%s", opts.MetricType)
	}
	return self.getHostMetrics(metricName, opts.MetricType, opts.StartTime, opts.EndTime)
}

func (self *SESXiClient) getHostMetrics(metricName string, metricType cloudprovider.TMetricType, start, end time.Time) ([]cloudprovider.MetricValues, error) {
	start, end = self.alignMetricTimeRange(start, end)

	m := view.NewManager(self.client.Client)
	v, err := m.CreateContainerView(self.context, self.client.Client.ServiceContent.RootFolder, nil, true)
	if err != nil {
		return nil, errors.Wrapf(err, "CreateContainerView")
	}
	vms, err := v.Find(self.context, []string{"HostSystem"}, nil)
	if err != nil {
		return nil, err
	}
	perfManager := performance.NewManager(self.client.Client)
	counterInfo, err := perfManager.CounterInfoByName(self.context)
	if err != nil {
		return nil, errors.Wrapf(err, "CounterInfoByName")
	}

	counter, ok := counterInfo[metricName]
	if !ok {
		return nil, fmt.Errorf("not found %s", metricName)
	}
	keys := []types.PerfMetricId{}
	for _, v := range counterInfo {
		keys = append(keys, types.PerfMetricId{
			CounterId: v.Key,
		})
	}

	queries := []types.PerfQuerySpec{}
	for i := range vms {
		s, e := start, end
		query := types.PerfQuerySpec{
			Entity: vms[i].Reference(),
			MetricId: []types.PerfMetricId{
				{
					CounterId: counter.Key,
				},
			},
			Format:     "normal",
			IntervalId: 0,
			StartTime:  &s,
			EndTime:    &e,
		}
		queries = append(queries, query)
	}

	sample, err := perfManager.Query(self.context, queries)
	if err != nil {
		return nil, err
	}

	result, err := perfManager.ToMetricSeries(self.context, sample)
	if err != nil {
		return nil, err
	}

	dcs, err := self.GetDatacenters()
	if err != nil {
		return nil, err
	}
	hostMap := map[string]string{}
	for i := range dcs {
		if dcs[i] == nil {
			continue
		}
		part, err := dcs[i].GetIHosts()
		if err != nil {
			continue
		}
		for j := range part {
			host := part[j].(*SHost)
			hostMap[host.GetId()] = host.GetGlobalId()
		}
	}

	ret := []cloudprovider.MetricValues{}
	for _, metric := range result {
		metricValue := cloudprovider.MetricValues{}
		metricValue.MetricType = metricType
		hostId, ok := hostMap[metric.Entity.Value]
		if !ok {
			continue
		}
		metricValue.Id = hostId
		if len(metric.Value) == 0 {
			continue
		}
		for i, v := range metric.SampleInfo {
			if v.Timestamp.Second() != 0 {
				continue
			}
			var value float64
			if len(metric.Value[0].Value) > i {
				value = float64(metric.Value[0].Value[i])
			}
			switch counter.UnitInfo.GetElementDescription().Key {
			case "percent":
				value = value / 100.0
			case "kiloBytesPerSecond", "kiloBytes":
				value = value * 1024.0
			default:
				log.Errorf("unknow unit: %s", counter.UnitInfo.GetElementDescription().Key)
			}
			metricValue.Values = append(metricValue.Values, cloudprovider.MetricValue{
				Timestamp: v.Timestamp,
				Value:     value,
			})
		}
		ret = append(ret, metricValue)
	}
	return ret, nil
}

func (self *SESXiClient) GetMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	switch opts.ResourceType {
	case cloudprovider.METRIC_RESOURCE_TYPE_SERVER:
		return self.GetEcsMetrics(opts)
	case cloudprovider.METRIC_RESOURCE_TYPE_HOST:
		return self.GetHostMetrics(opts)
	default:
		return nil, errors.Wrapf(cloudprovider.ErrNotSupported, "%s", opts.ResourceType)
	}
}

type SEsxiMetricType struct {
	Key     string
	KeyId   int32
	Summary string
	Group   string
	Unit    string
}

func (self *SESXiClient) GetMetricTypes() ([]SEsxiMetricType, error) {
	perfManager := performance.NewManager(self.client.Client)
	counterInfo, err := perfManager.CounterInfoByName(self.context)
	if err != nil {
		return nil, errors.Wrapf(err, "CounterInfoByName")
	}
	ret := []SEsxiMetricType{}
	for k, v := range counterInfo {
		ret = append(ret, SEsxiMetricType{
			Key:     k,
			KeyId:   v.Key,
			Summary: v.NameInfo.GetElementDescription().Summary,
			Group:   v.GroupInfo.GetElementDescription().Key,
			Unit:    v.UnitInfo.GetElementDescription().Key,
		})
	}
	return ret, nil
}
