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

package aws

import (
	"fmt"
	"strings"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

func (self *SAwsClient) GetMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	if len(opts.ResourceId) == 0 {
		return nil, fmt.Errorf("missing resource id")
	}
	switch opts.ResourceType {
	case cloudprovider.METRIC_RESOURCE_TYPE_SERVER:
		return self.GetEcsMetrics(opts)
	case cloudprovider.METRIC_RESOURCE_TYPE_RDS:
		return self.GetRdsMetrics(opts)
	case cloudprovider.METRIC_RESOURCE_TYPE_REDIS:
		return self.GetRedisMetrics(opts)
	default:
		return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "%s", opts.ResourceType)
	}
}

func (self *SAwsClient) GetEcsMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	ret := []cloudprovider.MetricValues{}
	for metricType, metricNames := range map[cloudprovider.TMetricType]map[string]string{
		cloudprovider.VM_METRIC_TYPE_CPU_USAGE: {
			"CPUUtilization": "",
		},
		cloudprovider.VM_METRIC_TYPE_DISK_IO_READ_IOPS: {
			"DiskReadOps": "",
			"EBSReadOps":  cloudprovider.METRIC_TAG_TYPE_DISK_TYPE + ":" + cloudprovider.METRIC_TAG_TYPE_DISK_TYPE,
		},
		cloudprovider.VM_METRIC_TYPE_DISK_IO_WRITE_IOPS: {
			"DiskWriteOps": "",
			"EBSWriteOps":  cloudprovider.METRIC_TAG_TYPE_DISK_TYPE + ":" + cloudprovider.METRIC_TAG_TYPE_DISK_TYPE,
		},
		cloudprovider.VM_METRIC_TYPE_DISK_IO_READ_BPS: {
			"DiskReadBytes": "",
			"EBSReadBytes":  cloudprovider.METRIC_TAG_TYPE_DISK_TYPE + ":" + cloudprovider.METRIC_TAG_TYPE_DISK_TYPE,
		},
		cloudprovider.VM_METRIC_TYPE_DISK_IO_WRITE_BPS: {
			"DiskWriteBytes": "",
			"EBSWriteBytes":  cloudprovider.METRIC_TAG_TYPE_DISK_TYPE + ":" + cloudprovider.METRIC_TAG_TYPE_DISK_TYPE,
		},
		cloudprovider.VM_METRIC_TYPE_NET_BPS_RX: {
			"NetworkIn": "",
		},
		cloudprovider.VM_METRIC_TYPE_NET_BPS_TX: {
			"NetworkOut": "",
		},
	} {
		part, err := self.getMetrics(opts.RegionExtId, "AWS/EC2", metricType, metricNames, "InstanceId", opts.ResourceId, opts.StartTime, opts.EndTime)
		if err != nil {
			log.Errorf("getMetrics error: %v", err)
			continue
		}
		ret = append(ret, part...)
	}
	return ret, nil
}

type Datapoint struct {
	Average            float64 `xml:"Average"`
	ExtendedStatistics struct {
		Key   string `xml:"Key"`
		Value string `xml:"Value"`
	} `xml:"ExtendedStatistics>entry"`
	Maximum     float64   `xml:"Maximum"`
	Minimum     float64   `xml:"Minimum"`
	SampleCount float64   `xml:"SampleCount"`
	Sum         float64   `xml:"Sum"`
	Timestamp   time.Time `xml:"Timestamp"`
	Unit        string    `xml:"Unit"`
}

func (self Datapoint) GetValue() float64 {
	return self.Average + self.Minimum + self.Sum
}

type Datapoints struct {
	Datapoints []Datapoint `xml:"Datapoints>member"`
	Label      string      `xml:"Label"`
}

func (self *SAwsClient) getMetrics(regionId, ns string, metricType cloudprovider.TMetricType, metrics map[string]string, dimensionName, dimensionValue string, start, end time.Time) ([]cloudprovider.MetricValues, error) {
	result := []cloudprovider.MetricValues{}
	for metricName, tagValue := range metrics {
		params := map[string]string{
			"EndTime":                   end.Format(time.RFC3339),
			"MetricName":                metricName,
			"Dimensions.member.1.Name":  dimensionName,
			"Dimensions.member.1.Value": dimensionValue,
			"Namespace":                 ns,
			"Period":                    "1",
			"StartTime":                 start.Format(time.RFC3339),
			"Statistics.member.1":       "Average",
		}
		ret := Datapoints{}
		err := self.monitorRequest(regionId, "GetMetricStatistics", params, &ret)
		if err != nil {
			log.Errorf("GetMetricStatistics error: %v", err)
			continue
		}
		metric := cloudprovider.MetricValues{}
		metric.Id = dimensionValue
		metric.MetricType = metricType
		tags := map[string]string{}
		idx := strings.Index(tagValue, ":")
		if idx > 0 {
			tags[tagValue[:idx]] = tagValue[idx+1:]
		}
		for _, data := range ret.Datapoints {
			metricValue := cloudprovider.MetricValue{}
			metricValue.Tags = tags
			metricValue.Timestamp = data.Timestamp
			metricValue.Value = data.GetValue()
			metric.Values = append(metric.Values, metricValue)
		}
		result = append(result, metric)
	}
	return result, nil
}

func (self *SAwsClient) GetRdsMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	ret := []cloudprovider.MetricValues{}
	for metricType, metricNames := range map[cloudprovider.TMetricType]map[string]string{
		cloudprovider.RDS_METRIC_TYPE_CPU_USAGE: {
			"CPUUtilization": "",
		},
		cloudprovider.RDS_METRIC_TYPE_NET_BPS_RX: {
			"NetworkReceiveThroughput": "",
		},
		cloudprovider.RDS_METRIC_TYPE_NET_BPS_TX: {
			"NetworkTransmitThroughput": "",
		},
		cloudprovider.RDS_METRIC_TYPE_CONN_COUNT: {
			"DatabaseConnections": "",
		},
	} {
		part, err := self.getMetrics(opts.RegionExtId, "AWS/RDS", metricType, metricNames, "DBInstanceIdentifier", opts.ResourceId, opts.StartTime, opts.EndTime)
		if err != nil {
			log.Errorf("getMetrics error: %v", err)
			continue
		}
		ret = append(ret, part...)
	}
	return ret, nil

}

func (self *SAwsClient) GetRedisMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	ret := []cloudprovider.MetricValues{}
	for metricType, metricNames := range map[cloudprovider.TMetricType]map[string]string{
		cloudprovider.REDIS_METRIC_TYPE_CPU_USAGE: {
			"CPUUtilization": "",
		},
		cloudprovider.REDIS_METRIC_TYPE_USED_CONN: {
			"CurrConnections": "",
		},
		cloudprovider.REDIS_METRIC_TYPE_CACHE_EXP_KEYS: {
			"Reclaimed": "",
		},
	} {
		part, err := self.getMetrics(opts.RegionExtId, "AWS/ElastiCache", metricType, metricNames, "CacheClusterId", opts.ResourceId, opts.StartTime, opts.EndTime)
		if err != nil {
			log.Errorf("getMetrics error: %v", err)
			continue
		}
		ret = append(ret, part...)
	}
	return ret, nil
}
