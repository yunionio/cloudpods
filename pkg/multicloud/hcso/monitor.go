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

package hcso

import (
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud/huawei/client/modules"
)

func (r *SRegion) GetMetrics() ([]modules.SMetricMeta, error) {
	return r.ecsClient.CloudEye.ListMetrics()
}

func (r *SRegion) GetMetricsData(metrics []modules.SMetricMeta, since time.Time, until time.Time) ([]modules.SMetricData, error) {
	return r.ecsClient.CloudEye.GetMetricsData(metrics, since, until)
}

func (self *SHuaweiClient) getModelartsPoolMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	resp, err := self.modelartsPoolMonitor(opts.ResourceId, nil)
	if err != nil {
		return nil, err
	}
	metricData := []SModelartsMetric{}
	err = resp.Unmarshal(&metricData, "metrics")
	if err != nil {
		return nil, errors.Wrapf(err, "resp.Unmarshal")
	}
	result := []cloudprovider.MetricValues{}
	for i := range metricData {
		isMB := false
		if metricData[i].Datapoints[0].Unit == "Megabytes" {
			isMB = true
			metricData[i].Datapoints[0].Unit = "Bytes"
		}
		ret := cloudprovider.MetricValues{
			Id:     opts.ResourceId,
			Unit:   metricData[i].Datapoints[0].Unit,
			Values: []cloudprovider.MetricValue{},
		}
		tags := map[string]string{}
		switch metricData[i].Metric.MetricName {
		case "cpuUsage":
			ret.MetricType = cloudprovider.MODELARTS_POOL_METRIC_TYPE_CPU_USAGE
		case "memUsedRate":
			ret.MetricType = cloudprovider.MODELARTS_POOL_METRIC_TYPE_MEM_USAGE
		case "gpuUtil":
			ret.MetricType = cloudprovider.MODELARTS_POOL_METRIC_TYPE_GPU_UTIL
		case "gpuMemUsage":
			ret.MetricType = cloudprovider.MODELARTS_POOL_METRIC_TYPE_GPU_MEM_USAGE
		case "npuUtil":
			ret.MetricType = cloudprovider.MODELARTS_POOL_METRIC_TYPE_NPU_UTIL
		case "npuMemUsage":
			ret.MetricType = cloudprovider.MODELARTS_POOL_METRIC_TYPE_NPU_MEM_USAGE
		case "diskAvailableCapacity":
			ret.MetricType = cloudprovider.MODELARTS_POOL_METRIC_TYPE_DISK_AVAILABLE_CAPACITY
		case "diskCapacity":
			ret.MetricType = cloudprovider.MODELARTS_POOL_METRIC_TYPE_DISK_CAPACITY
		case "diskUsedRate":
			ret.MetricType = cloudprovider.MODELARTS_POOL_METRIC_TYPE_DISK_USAGE
		default:
			log.Warningf("invalid metricName %s for %s %s", metricData[i].Metric.MetricName, opts.ResourceType, opts.ResourceId)
			continue
		}
		for _, value := range metricData[i].Datapoints {
			if isMB {
				value.Statistics[0].Value *= 1024
			}
			if value.Statistics[0].Value == -1 {
				value.Statistics[0].Value = 0
			}
			metricValue := cloudprovider.MetricValue{
				Value:     value.Statistics[0].Value,
				Timestamp: time.UnixMilli(value.Timestamp),
				Tags:      tags,
			}
			ret.Values = append(ret.Values, metricValue)
		}
		result = append(result, ret)
	}
	return result, nil
}

func (self *SHuaweiClient) GetMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	switch opts.ResourceType {
	case cloudprovider.METRIC_RESOURCE_TYPE_MODELARTS_POOL:
		return self.getModelartsPoolMetrics(opts)
	default:
		return nil, errors.Wrapf(cloudprovider.ErrNotSupported, "%s", opts.ResourceType)
	}
}
