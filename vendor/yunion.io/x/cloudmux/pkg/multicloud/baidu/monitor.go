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

package baidu

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

func (self *SBaiduClient) GetMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	switch opts.ResourceType {
	case cloudprovider.METRIC_RESOURCE_TYPE_SERVER:
		return self.GetEcsMetrics(opts)
	default:
		return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "%s", opts.ResourceType)
	}
}

type SMetric struct {
	Average   float64
	Timestamp time.Time
}

func (self *SBaiduClient) GetEcsMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	ret := []cloudprovider.MetricValues{}
	metricNames := map[cloudprovider.TMetricType]string{
		cloudprovider.VM_METRIC_TYPE_CPU_USAGE:  "CPUUsagePercent",
		cloudprovider.VM_METRIC_TYPE_MEM_USAGE:  "MemUsedPercent",
		cloudprovider.VM_METRIC_TYPE_DISK_USAGE: "RootUsedPercent",
		cloudprovider.VM_METRIC_TYPE_NET_BPS_RX: "VNicInBPS",
		cloudprovider.VM_METRIC_TYPE_NET_BPS_TX: "VNicOutBPS",
	}
	if strings.EqualFold(opts.OsType, "windows") {
		metricNames[cloudprovider.VM_METRIC_TYPE_DISK_USAGE] = "DiskCUsedPercent"
	}
	metricName, ok := metricNames[opts.MetricType]
	if !ok {
		return nil, errors.Wrapf(cloudprovider.ErrNotSupported, "metric type %s", opts.MetricType)
	}
	res := fmt.Sprintf("json-api/v1/metricdata/%s/BCE_BCC/%s", self.ownerId, metricName)
	params := url.Values{}
	params.Set("dimensions", fmt.Sprintf("InstanceId:%s", opts.ResourceId))
	params.Set("statistics[]", "average")
	params.Set("startTime", opts.StartTime.UTC().Format(time.RFC3339))
	params.Set("endTime", opts.EndTime.UTC().Format(time.RFC3339))
	params.Set("periodInSecond", "60")
	resp, err := self.list(SERVICE_BCM, opts.RegionExtId, res, params)
	if err != nil {
		return nil, err
	}
	metrics := []SMetric{}
	err = resp.Unmarshal(&metrics, "dataPoints")
	if err != nil {
		return nil, err
	}
	metric := cloudprovider.MetricValues{
		Id:         opts.ResourceId,
		MetricType: opts.MetricType,
		Values:     []cloudprovider.MetricValue{},
	}
	for _, v := range metrics {
		metric.Values = append(metric.Values, cloudprovider.MetricValue{
			Timestamp: v.Timestamp,
			Value:     v.Average,
		})
	}
	ret = append(ret, metric)
	return ret, nil
}
