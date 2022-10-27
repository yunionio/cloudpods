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

package zstack

import (
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type SDataPoint struct {
	DataPoints []DataPoint `json:"data"`
}

type DataPoint struct {
	Value  float64 `json:"value"`
	Time   int64   `json:"time"`
	Labels Label   `json:"labels"`
}

type Label struct {
	VMUuid   string `json:"VMUuid"`
	HostUuid string `json:"HostUuid"`
}

func (self *SZStackClient) GetMonitorData(name string, namespace string, since time.Time, until time.Time) ([]DataPoint, error) {
	datas := SDataPoint{}
	param := jsonutils.NewDict()
	param.Add(jsonutils.NewString(namespace), "namespace")
	param.Add(jsonutils.NewString(name), "metricName")
	param.Add(jsonutils.NewString("60"), "period")
	param.Add(jsonutils.NewInt(since.Unix()), "startTime")
	param.Add(jsonutils.NewInt(until.Unix()), "endTime")
	rep, err := self.getMonitor("zwatch/metrics", param)
	if err != nil {
		return nil, err
	}
	return datas.DataPoints, rep.Unmarshal(&datas)
}

func (self *SZStackClient) GetMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	switch opts.ResourceType {
	case cloudprovider.METRIC_RESOURCE_TYPE_SERVER:
		return self.GetEcsMetrics(opts)
	case cloudprovider.METRIC_RESOURCE_TYPE_HOST:
		return self.GetHostMetrics(opts)
	default:
		return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "%s", opts.ResourceType)
	}
}

func (self *SZStackClient) GetHostMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	ns, metricName := "ZStack/Host", ""
	switch opts.MetricType {
	case cloudprovider.VM_METRIC_TYPE_CPU_USAGE:
		metricName = "CPUAverageUsedUtilization"
	case cloudprovider.VM_METRIC_TYPE_DISK_IO_READ_BPS:
		metricName = "DiskReadBytes"
	case cloudprovider.VM_METRIC_TYPE_DISK_IO_WRITE_BPS:
		metricName = "DiskWriteBytes"
	case cloudprovider.VM_METRIC_TYPE_NET_BPS_RX:
		metricName = "NetworkInBytes"
	case cloudprovider.VM_METRIC_TYPE_NET_BPS_TX:
		metricName = "NetworkOutBytes"
	case cloudprovider.VM_METRIC_TYPE_DISK_IO_READ_IOPS:
		metricName = "DiskReadOps"
	case cloudprovider.VM_METRIC_TYPE_DISK_IO_WRITE_IOPS:
		metricName = "DiskWriteOps"
	default:
		return nil, errors.Wrapf(cloudprovider.ErrNotSupported, "%s", opts.MetricType)
	}
	data, err := self.GetMonitorData(metricName, ns, opts.StartTime, opts.EndTime)
	if err != nil {
		return nil, errors.Wrapf(err, "GetMonitorData")
	}
	ret := []cloudprovider.MetricValues{}
	for i := range data {
		metric := cloudprovider.MetricValues{}
		metric.MetricType = opts.MetricType
		metric.Id = data[i].Labels.HostUuid
		metric.Values = append(metric.Values, cloudprovider.MetricValue{
			Timestamp: time.Unix(data[i].Time, 0),
			Value:     data[i].Value,
		})
		ret = append(ret, metric)
	}
	return ret, nil
}

func (self *SZStackClient) GetEcsMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	ns, metricName := "ZStack/VM", ""
	switch opts.MetricType {
	case cloudprovider.VM_METRIC_TYPE_CPU_USAGE:
		metricName = "CPUAverageUsedUtilization"
	case cloudprovider.VM_METRIC_TYPE_DISK_IO_READ_BPS:
		metricName = "DiskReadBytes"
	case cloudprovider.VM_METRIC_TYPE_DISK_IO_WRITE_BPS:
		metricName = "DiskWriteBytes"
	case cloudprovider.VM_METRIC_TYPE_NET_BPS_RX:
		metricName = "NetworkInBytes"
	case cloudprovider.VM_METRIC_TYPE_NET_BPS_TX:
		metricName = "NetworkOutBytes"
	case cloudprovider.VM_METRIC_TYPE_DISK_IO_READ_IOPS:
		metricName = "DiskReadOps"
	case cloudprovider.VM_METRIC_TYPE_DISK_IO_WRITE_IOPS:
		metricName = "DiskWriteOps"
	default:
		return nil, errors.Wrapf(cloudprovider.ErrNotSupported, "%s", opts.MetricType)
	}
	data, err := self.GetMonitorData(metricName, ns, opts.StartTime, opts.EndTime)
	if err != nil {
		return nil, errors.Wrapf(err, "GetMonitorData")
	}
	ret := []cloudprovider.MetricValues{}
	for i := range data {
		metric := cloudprovider.MetricValues{}
		metric.MetricType = opts.MetricType
		metric.Id = data[i].Labels.VMUuid
		metric.Values = append(metric.Values, cloudprovider.MetricValue{
			Timestamp: time.Unix(data[i].Time, 0),
			Value:     data[i].Value,
		})
		ret = append(ret, metric)
	}
	return ret, nil
}
