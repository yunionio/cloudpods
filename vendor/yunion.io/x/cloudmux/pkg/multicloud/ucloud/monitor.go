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

package ucloud

import (
	"fmt"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

const (
	// https://docs.ucloud.cn/cloudwatch/metric/intro
	UCLOUD_PRODUCT_KEY_UHOST = "uhost"
)

type SMetricSample struct {
	Timestamp int64
	Value     float64
}

type SMetricResult struct {
	ResourceId   string
	ResourceName string
	TagMap       map[string]string
	Values       []SMetricSample
}

type SQueryMetricDataItem struct {
	Metric  string
	Results []SMetricResult
}

type SQueryMetricDataResp struct {
	InvalidResourceIds []string
	List               []SQueryMetricDataItem
}

var uhostMetricNames = map[cloudprovider.TMetricType]string{
	cloudprovider.VM_METRIC_TYPE_CPU_USAGE:          "uhost_cpu_used",
	cloudprovider.VM_METRIC_TYPE_MEM_USAGE:          "cloudwatch_memory_usage",
	cloudprovider.VM_METRIC_TYPE_DISK_USAGE:         "cloudwatch_sys_disk_used_per",
	cloudprovider.VM_METRIC_TYPE_NET_BPS_RX:         "uhost_net_in_flow",
	cloudprovider.VM_METRIC_TYPE_NET_BPS_TX:         "uhost_net_out_flow",
	cloudprovider.VM_METRIC_TYPE_DISK_IO_READ_BPS:   "uhost_disk_read",
	cloudprovider.VM_METRIC_TYPE_DISK_IO_WRITE_BPS:  "uhost_disk_write",
	cloudprovider.VM_METRIC_TYPE_DISK_IO_READ_IOPS:  "uhost_disk_read_times",
	cloudprovider.VM_METRIC_TYPE_DISK_IO_WRITE_IOPS: "uhost_disk_write_times",
}

var uhostMetricNameTypes map[string]cloudprovider.TMetricType

func init() {
	uhostMetricNameTypes = make(map[string]cloudprovider.TMetricType, len(uhostMetricNames))
	for metricType, name := range uhostMetricNames {
		uhostMetricNameTypes[name] = metricType
	}
}

func pickMetricPeriod(start, end time.Time, interval int) int {
	if interval > 0 {
		return interval
	}
	d := end.Sub(start)
	switch {
	case d <= time.Hour:
		return 60
	case d <= 12*time.Hour:
		return 60
	case d <= 24*time.Hour:
		return 300
	default:
		return 3600
	}
}

// https://docs.ucloud.cn/api/cloudwatch-api/query_metric_data_set
func (self *SRegion) QueryMetricDataSet(productKey string, resourceIds []string, metricNames []string, start, end time.Time, period int) (*SQueryMetricDataResp, error) {
	params := NewUcloudParams()
	params.Set("Region", self.GetId())
	params.Set("ProductKey", productKey)
	params.Set("StartTime", int(start.Unix()))
	params.Set("EndTime", int(end.Unix()))
	params.Set("CalcMethod", "avg")
	params.Set("Period", period)

	idx := 0
	for _, resourceId := range resourceIds {
		for _, metricName := range metricNames {
			params.Set(fmt.Sprintf("MetricInfos.%d.Metric", idx), metricName)
			params.Set(fmt.Sprintf("MetricInfos.%d.ResourceId", idx), resourceId)
			idx++
		}
	}
	if idx == 0 {
		return nil, errors.Wrap(cloudprovider.ErrMissingParameter, "metric infos")
	}

	params = self.client.commonParams(params)
	params.SetAction("QueryMetricDataSet")
	resp, err := jsonRequest(self.client, params)
	if err != nil {
		return nil, errors.Wrap(err, "QueryMetricDataSet")
	}
	ret := SQueryMetricDataResp{}
	err = resp.Unmarshal(&ret, "Data")
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal Data")
	}
	return &ret, nil
}

func (self *SRegion) GetUhostMetrics(resourceIds []string, start, end time.Time, interval int) ([]cloudprovider.MetricValues, error) {
	metricNames := make([]string, 0, len(uhostMetricNames))
	for _, name := range uhostMetricNames {
		metricNames = append(metricNames, name)
	}
	period := pickMetricPeriod(start, end, interval)
	data, err := self.QueryMetricDataSet(UCLOUD_PRODUCT_KEY_UHOST, resourceIds, metricNames, start, end, period)
	if err != nil {
		return nil, errors.Wrap(err, "QueryMetricDataSet")
	}
	ret := []cloudprovider.MetricValues{}
	for _, item := range data.List {
		metricType, ok := uhostMetricNameTypes[item.Metric]
		if !ok {
			log.Warningf("unknown uhost metric %s", item.Metric)
			continue
		}
		for _, result := range item.Results {
			metric := cloudprovider.MetricValues{
				Id:         result.ResourceId,
				MetricType: metricType,
				Values:     []cloudprovider.MetricValue{},
			}
			for _, sample := range result.Values {
				value := sample.Value
				switch metricType {
				case cloudprovider.VM_METRIC_TYPE_NET_BPS_RX, cloudprovider.VM_METRIC_TYPE_NET_BPS_TX:
					// uhost network metrics unit is bps (bits per second)
					value = value / 8.0
				}
				metric.Values = append(metric.Values, cloudprovider.MetricValue{
					Timestamp: time.Unix(sample.Timestamp, 0),
					Value:     value,
				})
			}
			if len(metric.Values) > 0 {
				ret = append(ret, metric)
			}
		}
	}
	return ret, nil
}

func (self *SUcloudClient) GetMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	switch opts.ResourceType {
	case cloudprovider.METRIC_RESOURCE_TYPE_SERVER:
		return self.GetEcsMetrics(opts)
	default:
		return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "%s", opts.ResourceType)
	}
}

func (self *SUcloudClient) GetEcsMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	region := self.GetRegion(opts.RegionExtId)
	if region == nil {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "region %s", opts.RegionExtId)
	}
	resourceIds := opts.ResourceIds
	if len(resourceIds) == 0 && len(opts.ResourceId) > 0 {
		resourceIds = []string{opts.ResourceId}
	}
	if len(resourceIds) == 0 {
		return nil, errors.Wrap(cloudprovider.ErrMissingParameter, "resource_id")
	}
	return region.GetUhostMetrics(resourceIds, opts.StartTime, opts.EndTime, opts.Interval)
}
