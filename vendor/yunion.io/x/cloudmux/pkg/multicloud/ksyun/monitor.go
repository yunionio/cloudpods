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

package ksyun

import (
	"time"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
)

func (cli *SKsyunClient) monitorRequest(regionId, action string, params map[string]interface{}) (jsonutils.JSONObject, error) {
	return cli.request("monitor", regionId, action, KSYUN_MONITOR_API_VERSION, params)
}

func (cli *SKsyunClient) GetMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	switch opts.ResourceType {
	case cloudprovider.METRIC_RESOURCE_TYPE_SERVER:
		return cli.GetEcsMetrics(opts)
	default:
		return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "%s", opts.ResourceType)
	}
}

type SMetric struct {
	Instance   string
	Datapoints struct {
		Member []struct {
			Average       float64
			Timestamp     time.Time
			UnixTimestamp int64
		}
	}
	Label string
}

func (cli *SKsyunClient) GetEcsMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	params := map[string]interface{}{
		"Namespace": "KEC",
		"StartTime": opts.StartTime.Format("2006-01-02T15:04:05Z"),
		"EndTime":   opts.EndTime.Format("2006-01-02T15:04:05Z"),
		"Aggregate": []string{"Average"},
		"Metrics": []map[string]interface{}{
			{
				"InstanceID": opts.ResourceId,
				"MetricName": "cpu.utilizition.total",
			},
			{
				"InstanceID": opts.ResourceId,
				"MetricName": "memory.utilizition.total",
			},
			{
				"InstanceID": opts.ResourceId,
				"MetricName": "vfs.fs.size",
			},
		},
	}

	resp, err := cli.monitorRequest(opts.RegionExtId, "GetMetricStatisticsBatch", params)
	if err != nil {
		return nil, err
	}

	metrics := struct {
		GetMetricStatisticsBatchResults []SMetric
	}{}

	err = resp.Unmarshal(&metrics)
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.MetricValues{}
	for _, metric := range metrics.GetMetricStatisticsBatchResults {
		values := []cloudprovider.MetricValue{}
		for _, value := range metric.Datapoints.Member {
			values = append(values, cloudprovider.MetricValue{
				Timestamp: value.Timestamp,
				Value:     value.Average,
			})
		}
		metricValue := cloudprovider.MetricValues{
			Id:     metric.Instance,
			Values: values,
		}
		switch metric.Label {
		case "cpu.utilizition.total":
			metricValue.MetricType = cloudprovider.VM_METRIC_TYPE_CPU_USAGE
		case "memory.utilizition.total":
			metricValue.MetricType = cloudprovider.VM_METRIC_TYPE_MEM_USAGE
		case "vfs.fs.size":
			metricValue.MetricType = cloudprovider.VM_METRIC_TYPE_DISK_USAGE
		default:
			log.Errorf("invalid metric label %s", metric.Label)
			continue
		}
		ret = append(ret, metricValue)
	}

	return ret, nil
}
