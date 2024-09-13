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

package google

import (
	"fmt"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type MetricDataValue struct {
	DoubleValue float64
	Int64Value  float64
}

func (self *MetricDataValue) GetValue() float64 {
	return self.DoubleValue + self.Int64Value
}

type MetricData struct {
	Resource struct {
		Labels struct {
			InstanceId string
		}
	}
	MetricKind string
	ValueType  string
	Points     []struct {
		Interval struct {
			StartTime time.Time
			EndTime   time.Time
		}
		Value MetricDataValue
	}
}

func (self *SGoogleClient) GetMonitorData(metricName string, since time.Time, until time.Time) (*jsonutils.JSONArray, error) {
	params := map[string]string{
		"filter":             fmt.Sprintf(`metric.type="%s"`, metricName),
		"interval.startTime": since.Format(time.RFC3339),
		"interval.endTime":   until.Format(time.RFC3339),
	}
	resource := fmt.Sprintf("%s/%s/%s", "projects", self.projectId, "timeSeries")
	timeSeries, err := self.monitorListAll(resource, params)
	if err != nil {
		return nil, err
	}
	return timeSeries, nil
}

func (self *SGoogleClient) GetMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	switch opts.ResourceType {
	case cloudprovider.METRIC_RESOURCE_TYPE_SERVER:
		return self.GetEcsMetrics(opts)
	default:
		return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "%s", opts.ResourceType)
	}
}

func (self *SGoogleClient) GetEcsMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	metricName := ""
	switch opts.MetricType {
	case cloudprovider.VM_METRIC_TYPE_CPU_USAGE:
		metricName = "compute.googleapis.com/instance/cpu/utilization"
	case cloudprovider.VM_METRIC_TYPE_NET_BPS_RX:
		metricName = "compute.googleapis.com/instance/network/received_bytes_count"
	case cloudprovider.VM_METRIC_TYPE_NET_BPS_TX:
		metricName = "compute.googleapis.com/instance/network/sent_bytes_count"
	case cloudprovider.VM_METRIC_TYPE_DISK_IO_READ_BPS:
		metricName = "compute.googleapis.com/instance/disk/read_bytes_count"
	case cloudprovider.VM_METRIC_TYPE_DISK_IO_WRITE_BPS:
		metricName = "compute.googleapis.com/instance/disk/write_bytes_count"
	case cloudprovider.VM_METRIC_TYPE_DISK_IO_READ_IOPS:
		metricName = "compute.googleapis.com/instance/disk/read_ops_count"
	case cloudprovider.VM_METRIC_TYPE_DISK_IO_WRITE_IOPS:
		metricName = "compute.googleapis.com/instance/disk/write_ops_count"
	default:
		return nil, errors.Wrapf(cloudprovider.ErrNotSupported, "%s", opts.MetricType)
	}
	metrics, err := self.GetMonitorData(metricName, opts.StartTime, opts.EndTime)
	if err != nil {
		return nil, errors.Wrapf(err, "GetMonitorData")
	}
	data := []MetricData{}
	err = metrics.Unmarshal(&data)
	if err != nil {
		return nil, errors.Wrapf(err, "metrics.Unmarshal")
	}
	ret := []cloudprovider.MetricValues{}
	for i := range data {
		value := cloudprovider.MetricValues{}
		value.Id = data[i].Resource.Labels.InstanceId
		value.MetricType = opts.MetricType
		for j := range data[i].Points {
			metricValue := cloudprovider.MetricValue{}
			metricValue.Timestamp = data[i].Points[j].Interval.StartTime
			metricValue.Value = data[i].Points[j].Value.GetValue()
			if opts.MetricType == cloudprovider.VM_METRIC_TYPE_CPU_USAGE {
				metricValue.Value *= 100
			}
			value.Values = append(value.Values, metricValue)
		}
		ret = append(ret, value)
	}
	return ret, nil
}
