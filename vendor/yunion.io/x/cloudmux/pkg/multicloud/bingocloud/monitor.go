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

package bingocloud

import (
	"fmt"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type MetricOutput struct {
	Datapoints Datapoints
	ObjName    string
	Period     int64
}

type DatapointMember struct {
	Average     float64
	Maximum     float64
	Minimum     float64
	SampleCount float64
	Sum         float64
	Timestamp   time.Time
	Unit        string
}

func (self DatapointMember) GetValue() float64 {
	return self.Average + self.Maximum + self.Minimum + self.Sum
}

type Datapoints struct {
	Member []DatapointMember
}

func (self *SBingoCloudClient) DescribeMetricList(ns, metricNm, dimensionName, dimensionValue string, since time.Time, until time.Time) (*MetricOutput, error) {
	params := map[string]string{}
	params["Namespace"] = ns
	params["MetricName"] = metricNm
	params["Dimensions.member.1.Name"] = dimensionName
	params["Dimensions.member.1.Value"] = dimensionValue
	params["StartTime"] = since.UTC().Format(time.RFC3339)
	params["EndTime"] = until.UTC().Format(time.RFC3339)
	params["Statistics.member.1"] = "Average"
	params["Period"] = "60"
	resp, err := self.invoke("GetMetricStatistics", params)
	if err != nil {
		return nil, errors.Wrap(err, "GetMetricStatistics err")
	}
	ret := &MetricOutput{}
	return ret, resp.Unmarshal(ret, "GetMetricStatisticsResult")
}

func (self *SBingoCloudClient) GetMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	if len(opts.ResourceId) == 0 {
		return nil, fmt.Errorf("missing resourceId")
	}
	switch opts.ResourceType {
	case cloudprovider.METRIC_RESOURCE_TYPE_SERVER:
		return self.GetEcsMetrics(opts)
	case cloudprovider.METRIC_RESOURCE_TYPE_HOST:
		return self.GetHostMetrics(opts)
	default:
		return nil, errors.Wrapf(cloudprovider.ErrNotSupported, "%s", opts.ResourceType)
	}
}

func (self *SBingoCloudClient) GetEcsMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	var ret []cloudprovider.MetricValues
	for metricType, metricName := range map[cloudprovider.TMetricType]string{
		cloudprovider.VM_METRIC_TYPE_CPU_USAGE:          "CPUUtilization",
		cloudprovider.VM_METRIC_TYPE_MEM_USAGE:          "MemoryUsage",
		cloudprovider.VM_METRIC_TYPE_NET_BPS_RX:         "NetworkIn",
		cloudprovider.VM_METRIC_TYPE_NET_BPS_TX:         "NetworkOut",
		cloudprovider.VM_METRIC_TYPE_DISK_IO_READ_BPS:   "DiskReadBytes",
		cloudprovider.VM_METRIC_TYPE_DISK_IO_WRITE_BPS:  "DiskWriteBytes",
		cloudprovider.VM_METRIC_TYPE_DISK_IO_READ_IOPS:  "DiskReadOps",
		cloudprovider.VM_METRIC_TYPE_DISK_IO_WRITE_IOPS: "DiskWriteOps",
	} {
		data, err := self.DescribeMetricList("AWS/EC2", metricName, "InstanceId", opts.ResourceId, opts.StartTime, opts.EndTime)
		if err != nil {
			log.Errorf("DescribeMetricList error: %v", err)
			continue
		}
		metric := cloudprovider.MetricValues{}
		metric.Id = opts.ResourceId
		metric.MetricType = metricType
		for _, value := range data.Datapoints.Member {
			metricValue := cloudprovider.MetricValue{}
			metricValue.Timestamp = value.Timestamp
			metricValue.Value = value.GetValue()
			metric.Values = append(metric.Values, metricValue)
		}
		ret = append(ret, metric)
	}
	return ret, nil
}

func (self *SBingoCloudClient) GetHostMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	var ret []cloudprovider.MetricValues
	for metricType, metricName := range map[cloudprovider.TMetricType]string{
		cloudprovider.HOST_METRIC_TYPE_CPU_USAGE:          "CPUUtilization",
		cloudprovider.HOST_METRIC_TYPE_MEM_USAGE:          "MemeryUsage",
		cloudprovider.HOST_METRIC_TYPE_NET_BPS_RX:         "NetworkIn",
		cloudprovider.HOST_METRIC_TYPE_NET_BPS_TX:         "NetworkOut",
		cloudprovider.HOST_METRIC_TYPE_DISK_IO_READ_BPS:   "DiskReadBytes",
		cloudprovider.HOST_METRIC_TYPE_DISK_IO_WRITE_BPS:  "DiskWriteBytes",
		cloudprovider.HOST_METRIC_TYPE_DISK_IO_READ_IOPS:  "DiskReadOps",
		cloudprovider.HOST_METRIC_TYPE_DISK_IO_WRITE_IOPS: "DiskWriteOps",
	} {
		data, err := self.DescribeMetricList("AWS/HOST", metricName, "HostId", opts.ResourceId, opts.StartTime, opts.EndTime)
		if err != nil {
			log.Errorf("DescribeMetricList error: %v", err)
			continue
		}
		metric := cloudprovider.MetricValues{}
		metric.Id = opts.ResourceId
		metric.MetricType = metricType
		for _, value := range data.Datapoints.Member {
			metricValue := cloudprovider.MetricValue{}
			metricValue.Timestamp = value.Timestamp
			metricValue.Value = value.GetValue()
			metric.Values = append(metric.Values, metricValue)
		}
		ret = append(ret, metric)
	}
	return ret, nil

}
