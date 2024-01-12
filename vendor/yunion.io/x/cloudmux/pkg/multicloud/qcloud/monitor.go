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

package qcloud

import (
	"fmt"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/timeutils"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

const (
	QCLOUD_API_VERSION_METRICS = "2018-07-24"
)

type SQcMetricDimension struct {
	Name  string `json:"Name"`
	Value string `json:"Value"`
}

type SQcMetricConditionDimension struct {
	Key      string `json:"Key"`
	Value    string `json:"Value"`
	Operator string `json:"Operator"`
}

type SQcInstanceMetricDimension struct {
	Dimensions []SQcMetricDimension
}

type SDataPoint struct {
	Dimensions []SQcMetricDimension `json:"Dimensions"`
	Timestamps []float64            `json:"Timestamps"`
	Values     []float64            `json:"Values"`
}

type SK8SDataPoint struct {
	MetricName string      `json:"MetricName"`
	Points     []SK8sPoint `json:"Points"`
}

type SK8sPoint struct {
	Dimensions []SQcMetricDimension `json:"Dimensions"`
	Values     []SK8sPointValue     `json:"Values"`
}

type SK8sPointValue struct {
	Timestamp float64 `json:"Timestamp"`
	Value     float64 `json:"Value"`
}

type SBatchQueryMetricDataInput struct {
	MetricName string               `json:"MetricName"`
	Namespace  string               `json:"Namespace"`
	Metrics    []SQcMetricDimension `json:"Metrics"`
	StartTime  int64                `json:"StartTime"`
	EndTime    int64                `json:"EndTime"`
	Period     string               `json:"Period"`
}

func (self *SQcloudClient) metricsRequest(action string, params map[string]string) (jsonutils.JSONObject, error) {
	cli, err := self.getDefaultClient(params)
	if err != nil {
		return nil, err
	}
	return monitorRequest(cli, action, params, self.cpcfg.UpdatePermission, self.debug)
}

func (self *SQcloudClient) GetMonitorData(ns string, name string, since time.Time, until time.Time, regionId string, dimensionName string, resIds []string) ([]SDataPoint, error) {
	params := make(map[string]string)
	params["Region"] = regionId
	params["MetricName"] = name
	params["Namespace"] = ns
	params["StartTime"] = since.Format(timeutils.IsoTimeFormat)
	params["EndTime"] = until.Format(timeutils.IsoTimeFormat)
	for idx, resId := range resIds {
		params[fmt.Sprintf("Instances.%d.Dimensions.0.Name", idx)] = dimensionName
		params[fmt.Sprintf("Instances.%d.Dimensions.0.Value", idx)] = resId
	}
	body, err := self.metricsRequest("GetMonitorData", params)
	if err != nil {
		return nil, errors.Wrapf(err, "MetricRequest for %s", resIds)
	}
	ret := []SDataPoint{}
	err = body.Unmarshal(&ret, "DataPoints")
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return ret, nil
}

func (self *SQcloudClient) GetK8sMonitorData(ns, name, resourceId string, since time.Time, until time.Time, regionId string) ([]SK8SDataPoint, error) {
	params := make(map[string]string)
	params["Module"] = "monitor"
	params["Region"] = regionId
	params["MetricNames.0"] = name
	params["Namespace"] = ns
	params["Period"] = "60"
	params["StartTime"] = since.Format(timeutils.IsoTimeFormat)
	params["EndTime"] = until.Format(timeutils.IsoTimeFormat)
	params["Conditions.0.Key"] = "tke_cluster_instance_id"
	params["Conditions.0.Operator"] = "in"
	params["Conditions.0.Value.0"] = resourceId
	body, err := self.metricsRequest("DescribeStatisticData", params)
	if err != nil {
		return nil, errors.Wrap(err, "region.MetricRequest")
	}
	ret := []SK8SDataPoint{}
	return ret, body.Unmarshal(&ret, "Data")
}

func (self *SQcloudClient) GetMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	switch opts.ResourceType {
	case cloudprovider.METRIC_RESOURCE_TYPE_SERVER:
		return self.GetEcsMetrics(opts)
	case cloudprovider.METRIC_RESOURCE_TYPE_REDIS:
		return self.GetRedisMetrics(opts)
	case cloudprovider.METRIC_RESOURCE_TYPE_RDS:
		return self.GetRdsMetrics(opts)
	case cloudprovider.METRIC_RESOURCE_TYPE_K8S:
		return self.GetK8sMetrics(opts)
	default:
		return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "%s", opts.ResourceType)
	}
}

func (self *SQcloudClient) GetEcsMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	ret := []cloudprovider.MetricValues{}
	for metricType, metricNames := range map[cloudprovider.TMetricType]map[string]string{
		cloudprovider.VM_METRIC_TYPE_CPU_USAGE: {
			"CPUUsage": "",
		},
		cloudprovider.VM_METRIC_TYPE_MEM_USAGE: {
			"MemUsage": "",
		},
		cloudprovider.VM_METRIC_TYPE_NET_BPS_TX: {
			"lanOuttraffic": cloudprovider.METRIC_TAG_NET_TYPE + ":" + cloudprovider.METRIC_TAG_NET_TYPE_INTRANET,
			"WanOuttraffic": cloudprovider.METRIC_TAG_NET_TYPE + ":" + cloudprovider.METRIC_TAG_NET_TYPE_INTERNET,
		},
		cloudprovider.VM_METRIC_TYPE_NET_BPS_RX: {
			"lanIntraffic": cloudprovider.METRIC_TAG_NET_TYPE + ":" + cloudprovider.METRIC_TAG_NET_TYPE_INTRANET,
			"WanIntraffic": cloudprovider.METRIC_TAG_NET_TYPE + ":" + cloudprovider.METRIC_TAG_NET_TYPE_INTERNET,
		},
	} {
		for metricName, tag := range metricNames {
			metrics, err := self.GetMonitorData("QCE/CVM", metricName, opts.StartTime, opts.EndTime, opts.RegionExtId, "InstanceId", opts.ResourceIds)
			if err != nil {
				log.Errorf("GetMonitorData error: %v", err)
				continue
			}
			for i := range metrics {
				metric := cloudprovider.MetricValues{}
				if len(metrics[i].Dimensions) < 1 || metrics[i].Dimensions[0].Name != "InstanceId" {
					continue
				}
				metric.Id = metrics[i].Dimensions[0].Value
				metric.MetricType = metricType
				metricValue := cloudprovider.MetricValue{}
				metricValue.Tags = map[string]string{}
				idx := strings.Index(tag, ":")
				if idx > 0 {
					metricValue.Tags[tag[:idx]] = tag[idx+1:]
				}
				if len(metrics[i].Timestamps) == 0 {
					continue
				}
				for j := range metrics[i].Timestamps {
					metricValue.Value = metrics[i].Values[j]
					if strings.Contains(metricName, "traffic") { //Mbps
						metricValue.Value *= 1024 * 1024
					}
					metricValue.Timestamp = time.Unix(int64(metrics[i].Timestamps[j]), 0)
					metric.Values = append(metric.Values, metricValue)
				}
				ret = append(ret, metric)
			}
		}
	}
	return ret, nil
}

func (self *SQcloudClient) GetRedisMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	ret := []cloudprovider.MetricValues{}
	for metricType, metricNames := range map[cloudprovider.TMetricType]map[string]string{
		cloudprovider.REDIS_METRIC_TYPE_CPU_USAGE: {
			"CpuUtil": "",
		},
		cloudprovider.REDIS_METRIC_TYPE_MEM_USAGE: {
			"MemUtil": "",
		},
		cloudprovider.REDIS_METRIC_TYPE_NET_BPS_RX: {
			"InFlow": "",
		},
		cloudprovider.REDIS_METRIC_TYPE_NET_BPS_TX: {
			"OutFlow": "",
		},
		cloudprovider.REDIS_METRIC_TYPE_USED_CONN: {
			"Connections": "",
		},
		cloudprovider.REDIS_METRIC_TYPE_OPT_SES: {
			"Commands": "",
		},
		cloudprovider.REDIS_METRIC_TYPE_CACHE_KEYS: {
			"Keys": "",
		},
		cloudprovider.REDIS_METRIC_TYPE_CACHE_EXP_KEYS: {
			"Expired": "",
		},
		cloudprovider.REDIS_METRIC_TYPE_DATA_MEM_USAGE: {
			"MemUsed": "",
		},
	} {
		for metricName, tag := range metricNames {
			metrics, err := self.GetMonitorData("QCE/REDIS_MEM", metricName, opts.StartTime, opts.EndTime, opts.RegionExtId, "instanceid", opts.ResourceIds)
			if err != nil {
				log.Errorf("GetMonitorData error: %v", err)
				continue
			}
			for i := range metrics {
				metric := cloudprovider.MetricValues{}
				if len(metrics[i].Dimensions) < 1 || metrics[i].Dimensions[0].Name != "instanceid" {
					continue
				}
				metric.Id = metrics[i].Dimensions[0].Value
				metric.MetricType = metricType
				metricValue := cloudprovider.MetricValue{}
				metricValue.Tags = map[string]string{}
				idx := strings.Index(tag, ":")
				if idx > 0 {
					metricValue.Tags[tag[:idx]] = tag[idx+1:]
				}
				if len(metrics[i].Timestamps) == 0 {
					continue
				}
				for j := range metrics[i].Timestamps {
					metricValue.Value = metrics[i].Values[j]
					metricValue.Timestamp = time.Unix(int64(metrics[i].Timestamps[j]), 0)
					metric.Values = append(metric.Values, metricValue)
				}
				ret = append(ret, metric)
			}
		}
	}
	return ret, nil
}

func (self *SQcloudClient) GetRdsMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	ret := []cloudprovider.MetricValues{}
	for metricType, metricNames := range map[cloudprovider.TMetricType]map[string]string{
		cloudprovider.RDS_METRIC_TYPE_CPU_USAGE: {
			"CPUUseRate": "",
		},
		cloudprovider.RDS_METRIC_TYPE_MEM_USAGE: {
			"MemoryUseRate": "",
		},
		cloudprovider.RDS_METRIC_TYPE_NET_BPS_TX: {
			"BytesSent": cloudprovider.METRIC_TAG_NET_TYPE + ":" + cloudprovider.METRIC_TAG_NET_TYPE_INTRANET,
		},
		cloudprovider.RDS_METRIC_TYPE_NET_BPS_RX: {
			"BytesReceived": cloudprovider.METRIC_TAG_NET_TYPE + ":" + cloudprovider.METRIC_TAG_NET_TYPE_INTRANET,
		},
		cloudprovider.RDS_METRIC_TYPE_DISK_USAGE: {
			"VolumeRate": "",
		},
		cloudprovider.RDS_METRIC_TYPE_CONN_COUNT: {
			"ThreadsConnected": "",
		},
		cloudprovider.RDS_METRIC_TYPE_CONN_USAGE: {
			"ConnectionUseRate": "",
		},
		cloudprovider.RDS_METRIC_TYPE_QPS: {
			"QPS": "",
		},
		cloudprovider.RDS_METRIC_TYPE_TPS: {
			"TPS": "",
		},
		cloudprovider.RDS_METRIC_TYPE_INNODB_READ_BPS: {
			"InnodbDataRead": "",
		},
		cloudprovider.RDS_METRIC_TYPE_INNODB_WRITE_BPS: {
			"InnodbDataWritten": "",
		},
	} {
		for metricName, tag := range metricNames {
			metrics, err := self.GetMonitorData("QCE/CDB", metricName, opts.StartTime, opts.EndTime, opts.RegionExtId, "InstanceId", opts.ResourceIds)
			if err != nil {
				log.Errorf("GetMonitorData error: %v", err)
				continue
			}
			for i := range metrics {
				metric := cloudprovider.MetricValues{}
				if len(metrics[i].Dimensions) < 1 || metrics[i].Dimensions[0].Name != "InstanceId" {
					continue
				}
				metric.Id = metrics[i].Dimensions[0].Value
				metric.MetricType = metricType
				metricValue := cloudprovider.MetricValue{}
				metricValue.Tags = map[string]string{}
				idx := strings.Index(tag, ":")
				if idx > 0 {
					metricValue.Tags[tag[:idx]] = tag[idx+1:]
				}
				if len(metrics[i].Timestamps) == 0 {
					continue
				}
				for j := range metrics[i].Timestamps {
					metricValue.Value = metrics[i].Values[j]
					metricValue.Timestamp = time.Unix(int64(metrics[i].Timestamps[j]), 0)
					metric.Values = append(metric.Values, metricValue)
				}
				ret = append(ret, metric)
			}
		}
	}
	return ret, nil
}

func (self *SQcloudClient) GetK8sMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	ret := []cloudprovider.MetricValues{}
	for metricType, metricName := range map[cloudprovider.TMetricType]string{
		cloudprovider.K8S_NODE_METRIC_TYPE_CPU_USAGE: "K8sNodeCpuUsage",
		cloudprovider.K8S_NODE_METRIC_TYPE_MEM_USAGE: "K8sNodeMemUsage",
	} {
		metrics, err := self.GetK8sMonitorData("QCE/TKE2", metricName, opts.ResourceId, opts.StartTime, opts.EndTime, opts.RegionExtId)
		if err != nil {
			log.Errorf("GetMonitorData error: %v", err)
			continue
		}
		for i := range metrics {
			for _, point := range metrics[i].Points {
				tags := map[string]string{}
				for _, dim := range point.Dimensions {
					if dim.Name == "node" {
						tags[cloudprovider.METRIC_TAG_NODE] = dim.Value
					}
				}
				metric := cloudprovider.MetricValues{}
				metric.Id = opts.ResourceId
				metric.MetricType = metricType
				for _, value := range point.Values {
					metric.Values = append(metric.Values, cloudprovider.MetricValue{
						Value:     value.Value,
						Timestamp: time.Unix(int64(value.Timestamp), 0),
						Tags:      tags,
					})
				}
				if len(metric.Values) > 0 {
					ret = append(ret, metric)
				}
			}
		}
	}
	return ret, nil
}
