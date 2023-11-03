// Copyright 2023 Yunion
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

package volcengine

import (
	"time"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/pkg/errors"
)

func (self *SVolcEngineClient) GetMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	switch opts.ResourceType {
	case cloudprovider.METRIC_RESOURCE_TYPE_SERVER:
		return self.GetEcsMetrics(opts)
	//case cloudprovider.METRIC_RESOURCE_TYPE_BUCKET:
	//	return self.GetOssMetrics(opts)
	//case cloudprovider.METRIC_RESOURCE_TYPE_EIP:
	//	return self.GetEipMetrics(opts)
	default:
		return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "%s", opts.ResourceType)
	}
}

func (self *SVolcEngineClient) GetEcsMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	metricName, namespace, subNamespace := "", "VCM_ECS", "Instance"
	switch opts.MetricType {
	case cloudprovider.VM_METRIC_TYPE_CPU_USAGE:
		metricName = "CpuTotal"
	case cloudprovider.VM_METRIC_TYPE_NET_BPS_RX:
		metricName = "NetworkInRate"
	case cloudprovider.VM_METRIC_TYPE_NET_BPS_TX:
		metricName = "NetworkOutRate"
	case cloudprovider.VM_METRIC_TYPE_DISK_IO_READ_BPS:
		metricName, subNamespace = "DiskReadBytes", "Storage"
	case cloudprovider.VM_METRIC_TYPE_DISK_IO_WRITE_BPS:
		metricName, subNamespace = "DiskWriteBytes", "Storage"
	case cloudprovider.VM_METRIC_TYPE_DISK_IO_READ_IOPS:
		metricName, subNamespace = "DiskReadIOPS", "Storage"
	case cloudprovider.VM_METRIC_TYPE_DISK_IO_WRITE_IOPS:
		metricName, subNamespace = "DiskWriteIOPS", "Storage"
	case cloudprovider.VM_METRIC_TYPE_MEM_USAGE:
		metricName = "MemoryUsedUtilization"
	case cloudprovider.VM_METRIC_TYPE_DISK_USAGE:
		//metricName = "DiskUsageUtilization"
		return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "%s", opts.MetricType)
	case cloudprovider.VM_METRIC_TYPE_NET_TCP_CONNECTION:
		metricName = "NetTcpConnectionStatus"
	default:
		return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "%s", opts.MetricType)
	}
	ret := []cloudprovider.MetricValues{}
	result, err := self.GetMetricData(opts.RegionExtId, opts.StartTime, opts.EndTime, namespace, subNamespace, metricName, opts.ResourceIds)
	if err != nil {
		return nil, errors.Wrapf(err, "GetMetricData")
	}
	for i := range result {
		id := ""
		for _, dim := range result[i].Dimensions {
			if dim.Name == "ResourceID" {
				id = dim.Value
			}
		}
		if len(id) == 0 {
			continue
		}
		values := []cloudprovider.MetricValue{}
		for _, v := range result[i].DataPoints {
			values = append(values, cloudprovider.MetricValue{
				Timestamp: time.Unix(v.Timestamp, 0),
				Value:     v.Value,
			})
		}
		ret = append(ret, cloudprovider.MetricValues{
			Id:         id,
			MetricType: opts.MetricType,
			Values:     values,
		})
	}
	return ret, nil
}

type SMetricData struct {
	Legend     string
	Dimensions []struct {
		Name  string
		Value string
	}
	DataPoints []struct {
		Timestamp int64
		Value     float64
	}
}

func (self *SVolcEngineClient) GetMetricData(regionId string, start, end time.Time, namespace, subNamespace, metricName string, ids []string) ([]SMetricData, error) {
	instances := []interface{}{}
	for _, id := range ids {
		instances = append(instances, map[string]interface{}{
			"Dimensions": []map[string]interface{}{
				{
					"Name":  "ResourceID",
					"Value": id,
				},
			},
		})
	}
	params := map[string]interface{}{
		"StartTime":    start.UTC().Unix(),
		"EndTime":      end.UTC().Unix(),
		"MetricName":   metricName,
		"Namespace":    namespace,
		"SubNamespace": subNamespace,
		"Period":       "1m",
		"Instances":    instances,
	}
	resp, err := self.monitorRequest(regionId, "GetMetricData", params)
	if err != nil {
		return nil, errors.Wrapf(err, "GetMetricData")
	}
	ret := []SMetricData{}
	err = resp.Unmarshal(&ret, "Data", "MetricDataResults")
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (self *SVolcEngineClient) GetOssMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	metricName, namespace, subNamespace := "", "VCM_TOS", ""
	switch opts.MetricType {
	case cloudprovider.BUCKET_METRIC_TYPE_REQ_2XX_COUNT:
		metricName, subNamespace = "2xxQPS", "bucket_status_code"
	case cloudprovider.BUCKET_METRIC_TYPE_REQ_3XX_COUNT:
		metricName, subNamespace = "3xxQPS", "bucket_status_code"
	case cloudprovider.BUCKET_METRIC_TYPE_REQ_4XX_COUNT:
		metricName, subNamespace = "4xxQPS", "bucket_status_code"
	case cloudprovider.BUCKET_METRIC_TYPE_REQ_5XX_COUNT:
		metricName, subNamespace = "5xxQPS", "bucket_status_code"
	case cloudprovider.BUCKET_METRIC_TYPE_STORAGE_SIZE:
		metricName, subNamespace = "BucketTotalStorage", "bucket_overview"
	default:
		return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "%s", opts.MetricType)
	}
	ret := []cloudprovider.MetricValues{}
	result, err := self.GetMetricData(opts.RegionExtId, opts.StartTime, opts.EndTime, namespace, subNamespace, metricName, opts.ResourceIds)
	if err != nil {
		return nil, errors.Wrapf(err, "GetMetricData")
	}
	for i := range result {
		id := ""
		for _, dim := range result[i].Dimensions {
			if dim.Name == "ResourceID" {
				id = dim.Value
			}
		}
		if len(id) == 0 {
			continue
		}
		values := []cloudprovider.MetricValue{}
		for _, v := range result[i].DataPoints {
			values = append(values, cloudprovider.MetricValue{
				Timestamp: time.Unix(v.Timestamp, 0),
				Value:     v.Value,
			})
		}
		ret = append(ret, cloudprovider.MetricValues{
			Id:         id,
			MetricType: opts.MetricType,
			Values:     values,
		})
	}
	return ret, nil
}
