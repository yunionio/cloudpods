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

package ctyun

import (
	"fmt"
	"time"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/pkg/errors"
)

func (self *SCtyunClient) GetMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	switch opts.ResourceType {
	case cloudprovider.METRIC_RESOURCE_TYPE_SERVER:
		return self.GetEcsMetrics(opts)
	default:
		return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "%s", opts.ResourceType)
	}
}

type Metric struct {
	Value        float64
	SamplingTime int64
}

func (v Metric) Values() cloudprovider.MetricValue {
	ret := cloudprovider.MetricValue{}
	ret.Timestamp = time.Unix(v.SamplingTime, 0)
	ret.Value = v.Value
	return ret
}

type SMetric struct {
	RegionId          string
	Fuid              string
	FuserLastUpdated  time.Time
	DeviceUUID        string
	ItemAggregateList struct {
		CpuIdleTime           []Metric
		CpuUtil               []Metric
		MemUtil               []Metric
		DiskUtil              []Metric
		DiskReadBytesRate     []Metric
		DiskWriteBytesRate    []Metric
		DiskReadRequestsRate  []Metric
		DiskWriteRequestsRate []Metric
		NetInBytesRate        []Metric
		NetOutBytesRate       []Metric
	}
}

func (self *SCtyunClient) getEcsCpuMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	metrics, err := self.getMetrics("/v4/ecs/vm-cpu-history-metric-data", opts)
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.MetricValues{}
	for _, metric := range metrics {
		data := cloudprovider.MetricValues{}
		data.Id = metric.DeviceUUID
		data.MetricType = cloudprovider.VM_METRIC_TYPE_CPU_USAGE
		data.Values = []cloudprovider.MetricValue{}
		for _, v := range metric.ItemAggregateList.CpuUtil {
			data.Values = append(data.Values, v.Values())
		}
		ret = append(ret, data)
	}
	return ret, nil
}

func (self *SCtyunClient) getEcsMemMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	metrics, err := self.getMetrics("/v4/ecs/vm-mem-history-metric-data", opts)
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.MetricValues{}
	for _, metric := range metrics {
		data := cloudprovider.MetricValues{}
		data.Id = metric.DeviceUUID
		data.MetricType = cloudprovider.VM_METRIC_TYPE_MEM_USAGE
		data.Values = []cloudprovider.MetricValue{}
		for _, v := range metric.ItemAggregateList.MemUtil {
			data.Values = append(data.Values, v.Values())
		}
		ret = append(ret, data)
	}
	return ret, nil
}

func (self *SCtyunClient) getMetrics(apiName string, opts *cloudprovider.MetricListOptions) ([]SMetric, error) {
	region, err := self.GetRegion(opts.RegionExtId)
	if err != nil {
		return nil, err
	}
	metrics, pageNo := []SMetric{}, 1
	for {
		resp, err := self.post(SERVICE_ECS, apiName, map[string]interface{}{
			"regionID":     region.RegionId,
			"deviceIDList": opts.ResourceIds,
			"period":       300,
			"startTime":    fmt.Sprintf("%d", opts.StartTime.Unix()),
			"endTime":      fmt.Sprintf("%d", opts.EndTime.Unix()),
			"pageNo":       pageNo,
		})
		if err != nil {
			return nil, err
		}
		part := struct {
			Result    []SMetric
			TotalPage int
			Page      int
		}{}
		err = resp.Unmarshal(&part, "returnObj")
		if err != nil {
			return nil, errors.Wrapf(err, "Unmarshal")
		}
		metrics = append(metrics, part.Result...)
		pageNo++
		if pageNo > part.Page {
			break
		}
	}
	return metrics, nil
}

func (self *SCtyunClient) getEcsDiskMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	metrics, err := self.getMetrics("/v4/ecs/vm-disk-history-metric-data", opts)
	if err != nil {
		return nil, err
	}
	var getMetric = func(id string, metricType cloudprovider.TMetricType, values []Metric) cloudprovider.MetricValues {
		data := cloudprovider.MetricValues{}
		data.Id = id
		data.MetricType = metricType
		data.Values = []cloudprovider.MetricValue{}
		for _, v := range values {
			data.Values = append(data.Values, v.Values())
		}
		return data
	}
	ret := []cloudprovider.MetricValues{}
	for _, metric := range metrics {
		data := getMetric(metric.DeviceUUID, cloudprovider.VM_METRIC_TYPE_DISK_USAGE, metric.ItemAggregateList.DiskUtil)
		ret = append(ret, data)

		data = getMetric(metric.DeviceUUID, cloudprovider.VM_METRIC_TYPE_DISK_IO_READ_BPS, metric.ItemAggregateList.DiskReadBytesRate)
		ret = append(ret, data)

		data = getMetric(metric.DeviceUUID, cloudprovider.VM_METRIC_TYPE_DISK_IO_WRITE_BPS, metric.ItemAggregateList.DiskWriteBytesRate)
		ret = append(ret, data)

		data = getMetric(metric.DeviceUUID, cloudprovider.VM_METRIC_TYPE_DISK_IO_READ_IOPS, metric.ItemAggregateList.DiskReadRequestsRate)
		ret = append(ret, data)

		data = getMetric(metric.DeviceUUID, cloudprovider.VM_METRIC_TYPE_DISK_IO_WRITE_IOPS, metric.ItemAggregateList.DiskWriteRequestsRate)
		ret = append(ret, data)
	}
	return ret, nil
}

func (self *SCtyunClient) getEcsNicMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	metrics, err := self.getMetrics("/v4/ecs/vm-network-history-metric-data", opts)
	if err != nil {
		return nil, err
	}
	var getMetric = func(id string, metricType cloudprovider.TMetricType, values []Metric) cloudprovider.MetricValues {
		data := cloudprovider.MetricValues{}
		data.Id = id
		data.MetricType = metricType
		data.Values = []cloudprovider.MetricValue{}
		for _, v := range values {
			data.Values = append(data.Values, v.Values())
		}
		return data
	}
	ret := []cloudprovider.MetricValues{}
	for _, metric := range metrics {
		data := getMetric(metric.DeviceUUID, cloudprovider.VM_METRIC_TYPE_NET_BPS_RX, metric.ItemAggregateList.NetInBytesRate)
		ret = append(ret, data)

		data = getMetric(metric.DeviceUUID, cloudprovider.VM_METRIC_TYPE_NET_BPS_TX, metric.ItemAggregateList.NetOutBytesRate)
		ret = append(ret, data)
	}
	return ret, nil
}

func (self *SCtyunClient) GetEcsMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	metrics, err := self.getEcsCpuMetrics(opts)
	if err != nil {
		return nil, err
	}
	memMetrics, err := self.getEcsMemMetrics(opts)
	if err != nil {
		return nil, err
	}
	metrics = append(metrics, memMetrics...)
	diskMetris, err := self.getEcsDiskMetrics(opts)
	if err != nil {
		return nil, err
	}
	metrics = append(metrics, diskMetris...)
	nicMetrics, err := self.getEcsNicMetrics(opts)
	if err != nil {
		return nil, err
	}
	metrics = append(metrics, nicMetrics...)
	return metrics, nil
}
