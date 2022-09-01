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

package apsara

import (
	"fmt"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudprovider"
)

const (
	APSARA_API_VERSION_METRICS = "2019-01-01"
)

func (self *SApsaraClient) metricsRequest(action string, params map[string]string) (jsonutils.JSONObject, error) {
	client, err := self.getDefaultClient("")
	if err != nil {
		return nil, errors.Wrap(err, "getDefaultClient")
	}
	domain := self.getDomain(APSARA_PRODUCT_METRICS)
	return productRequest(client, APSARA_PRODUCT_METRICS, domain, APSARA_API_VERSION_METRICS, action, params, self.debug)
}

type MetricData struct {
	Timestamp  int64
	BucketName string
	InstanceId string
	UserId     string
	Value      float64
	Average    float64
	Minimum    float64
	Maximum    float64
	Diskname   string
	Device     string
}

func (d MetricData) GetValue() float64 {
	if d.Average > 0 {
		return d.Average
	}
	if d.Maximum > 0 {
		return d.Maximum
	}
	if d.Minimum > 0 {
		return d.Minimum
	}
	if d.Value > 0 {
		return d.Value
	}
	return 0.0
}

func (d MetricData) GetTags() map[string]string {
	ret := map[string]string{}
	if len(d.Device) > 0 {
		ret[cloudprovider.METRIC_TAG_DEVICE] = fmt.Sprintf("%s(%s)", d.Device, d.Diskname)
	}
	return ret
}

func (self *SApsaraClient) ListMetrics(ns, metricName string, start, end time.Time) ([]MetricData, error) {
	result := []MetricData{}
	nextToken := ""
	for {
		part, next, err := self.listMetrics(ns, metricName, nextToken, start, end)
		if err != nil {
			return nil, errors.Wrap(err, "listMetrics")
		}
		result = append(result, part...)
		if len(next) == 0 {
			break
		}
		nextToken = next
	}
	return result, nil
}

func (self *SApsaraClient) listMetrics(ns, metricName, nextToken string, start, end time.Time) ([]MetricData, string, error) {
	params := make(map[string]string)
	params["MetricName"] = metricName
	params["Namespace"] = ns
	params["Length"] = "2000"
	if len(nextToken) > 0 {
		params["NextToken"] = nextToken
	}
	if len(self.organizationId) > 0 {
		params["Department"] = self.organizationId
	}
	params["StartTime"] = fmt.Sprintf("%d", start.UnixMilli())
	params["EndTime"] = fmt.Sprintf("%d", end.UnixMilli())
	resp, err := self.metricsRequest("DescribeMetricList", params)
	if err != nil {
		return nil, "", errors.Wrap(err, "DescribeMetricList")
	}
	ret := struct {
		NextToken  string
		Datapoints string
	}{}
	err = resp.Unmarshal(&ret)
	if err != nil {
		return nil, "", errors.Wrapf(err, "resp.Unmarshal")
	}
	obj, err := jsonutils.ParseString(ret.Datapoints)
	if err != nil {
		return nil, "", errors.Wrap(err, "jsonutils.ParseString")
	}
	result := []MetricData{}
	err = obj.Unmarshal(&result)
	if err != nil {
		return nil, "", errors.Wrapf(err, "obj.Unmarshal")
	}
	return result, ret.NextToken, nil
}

func (self *SApsaraClient) GetMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	switch opts.ResourceType {
	case cloudprovider.METRIC_RESOURCE_TYPE_SERVER:
		return self.GetEcsMetrics(opts)
	case cloudprovider.METRIC_RESOURCE_TYPE_BUCKET:
		return self.GetOssMetrics(opts)
	case cloudprovider.METRIC_RESOURCE_TYPE_REDIS:
		return self.GetRedisMetrics(opts)
	case cloudprovider.METRIC_RESOURCE_TYPE_RDS:
		return self.GetRdsMetrics(opts)
	case cloudprovider.METRIC_RESOURCE_TYPE_LB:
		return self.GetElbMetrics(opts)
	default:
		return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "%s", opts.ResourceType)
	}
}

func (self *SApsaraClient) GetEcsMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	metricTags, tagKey := map[string]string{}, ""
	switch opts.MetricType {
	case cloudprovider.VM_METRIC_TYPE_CPU_USAGE:
		metricTags = map[string]string{
			"CPUUtilization": "",
		}
	case cloudprovider.VM_METRIC_TYPE_NET_BPS_RX:
		metricTags = map[string]string{
			"InternetInRate": cloudprovider.METRIC_TAG_NET_TYPE_INTERNET,
			"IntranetInRate": cloudprovider.METRIC_TAG_NET_TYPE_INTRANET,
		}
		tagKey = cloudprovider.METRIC_TAG_NET_TYPE
	case cloudprovider.VM_METRIC_TYPE_NET_BPS_TX:
		metricTags = map[string]string{
			"InternetOutRate": cloudprovider.METRIC_TAG_NET_TYPE_INTERNET,
			"IntranetOutRate": cloudprovider.METRIC_TAG_NET_TYPE_INTRANET,
		}
		tagKey = cloudprovider.METRIC_TAG_NET_TYPE
	case cloudprovider.VM_METRIC_TYPE_DISK_IO_READ_BPS:
		metricTags = map[string]string{
			"DiskReadBPS": "",
		}
	case cloudprovider.VM_METRIC_TYPE_DISK_IO_WRITE_BPS:
		metricTags = map[string]string{
			"DiskWriteBPS": "",
		}
	case cloudprovider.VM_METRIC_TYPE_DISK_IO_READ_IOPS:
		metricTags = map[string]string{
			"DiskReadIOPS": "",
		}
	case cloudprovider.VM_METRIC_TYPE_DISK_IO_WRITE_IOPS:
		metricTags = map[string]string{
			"DiskWriteIOPS": "",
		}
	case cloudprovider.VM_METRIC_TYPE_MEM_USAGE:
		metricTags = map[string]string{
			"memory_usedutilization": "",
		}
	case cloudprovider.VM_METRIC_TYPE_DISK_USAGE:
		metricTags = map[string]string{
			"diskusage_utilization": "",
		}
	default:
		return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "%s", opts.MetricType)
	}
	ret := []cloudprovider.MetricValues{}
	for metric, tag := range metricTags {
		result, err := self.ListMetrics("acs_ecs_dashboard", metric, opts.StartTime, opts.EndTime)
		if err != nil {
			log.Errorf("ListMetric(%s) error: %v", metric, err)
			continue
		}
		tags := map[string]string{}
		if len(tag) > 0 && len(tagKey) > 0 {
			tags[tagKey] = tag
		}
		for i := range result {
			dataTag := result[i].GetTags()
			for k, v := range tags {
				dataTag[k] = v
			}
			ret = append(ret, cloudprovider.MetricValues{
				Id:         result[i].InstanceId,
				MetricType: opts.MetricType,
				Values: []cloudprovider.MetricValue{
					{
						Timestamp: time.UnixMilli(result[i].Timestamp),
						Value:     result[i].GetValue(),
						Tags:      dataTag,
					},
				},
			})
		}
	}
	return ret, nil
}

func (self *SApsaraClient) GetOssMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	metricTags, tagKey := map[string]string{}, ""
	switch opts.MetricType {
	case cloudprovider.BUCKET_METRIC_TYPE_LATECY:
		metricTags = map[string]string{
			"GetObjectE2eLatency":  cloudprovider.METRIC_TAG_REQUST_GET,
			"PostObjectE2eLatency": cloudprovider.METRIC_TAG_REQUST_GET,
		}
		tagKey = cloudprovider.METRIC_TAG_REQUST
	case cloudprovider.BUCKET_METRIC_TYPE_NET_BPS_TX:
		metricTags = map[string]string{
			"InternetSend": cloudprovider.METRIC_TAG_NET_TYPE_INTERNET,
			"IntranetSend": cloudprovider.METRIC_TAG_NET_TYPE_INTRANET,
		}
		tagKey = cloudprovider.METRIC_TAG_REQUST
	case cloudprovider.BUCKET_METRIC_TYPE_NET_BPS_RX:
		metricTags = map[string]string{
			"InternetRecv": cloudprovider.METRIC_TAG_NET_TYPE_INTERNET,
			"IntranetRecv": cloudprovider.METRIC_TAG_NET_TYPE_INTRANET,
		}
		tagKey = cloudprovider.METRIC_TAG_NET_TYPE
	case cloudprovider.BUCKET_METRYC_TYPE_REQ_COUNT:
		metricTags = map[string]string{
			"GetObjectCount":   cloudprovider.METRIC_TAG_REQUST_GET,
			"PostObjectCount":  cloudprovider.METRIC_TAG_REQUST_POST,
			"ServerErrorCount": cloudprovider.METRIC_TAG_REQUST_5XX,
		}
		tagKey = cloudprovider.METRIC_TAG_REQUST
	default:
		return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "%s", opts.MetricType)
	}
	ret := []cloudprovider.MetricValues{}
	for metric, tag := range metricTags {
		result, err := self.ListMetrics("acs_oss_dashboard", metric, opts.StartTime, opts.EndTime)
		if err != nil {
			log.Errorf("ListMetric(%s) error: %v", metric, err)
			continue
		}
		for i := range result {
			ret = append(ret, cloudprovider.MetricValues{
				Id:         result[i].BucketName,
				MetricType: opts.MetricType,
				Values: []cloudprovider.MetricValue{
					{
						Timestamp: time.UnixMilli(result[i].Timestamp),
						Value:     result[i].GetValue(),
						Tags: map[string]string{
							tagKey: tag,
						},
					},
				},
			})
		}
	}
	return ret, nil
}

func (self *SApsaraClient) GetRedisMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	metricTags, tagKey := map[string]string{}, ""
	switch opts.MetricType {
	case cloudprovider.REDIS_METRIC_TYPE_CPU_USAGE:
		metricTags = map[string]string{
			"CpuUsage": "",
		}
	case cloudprovider.REDIS_METRIC_TYPE_MEM_USAGE:
		metricTags = map[string]string{
			"MemoryUsage": "",
		}
	case cloudprovider.REDIS_METRIC_TYPE_NET_BPS_RX:
		metricTags = map[string]string{
			"IntranetIn": "",
		}
	case cloudprovider.REDIS_METRIC_TYPE_NET_BPS_TX:
		metricTags = map[string]string{
			"IntranetOut": "",
		}
	case cloudprovider.REDIS_METRIC_TYPE_USED_CONN:
		metricTags = map[string]string{
			"UsedConnection": "",
		}
	case cloudprovider.REDIS_METRIC_TYPE_OPT_SES:
		metricTags = map[string]string{
			"UsedQPS": "",
		}
	case cloudprovider.REDIS_METRIC_TYPE_CACHE_KEYS:
		metricTags = map[string]string{
			"Keys": "",
		}
	case cloudprovider.REDIS_METRIC_TYPE_CACHE_EXP_KEYS:
		metricTags = map[string]string{
			"ExpiredKeys": "",
		}
	case cloudprovider.REDIS_METRIC_TYPE_DATA_MEM_USAGE:
		metricTags = map[string]string{
			"UsedMemory": "",
		}
	default:
		return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "%s", opts.MetricType)
	}
	ret := []cloudprovider.MetricValues{}
	for metric, tag := range metricTags {
		result, err := self.ListMetrics("acs_kvstore", metric, opts.StartTime, opts.EndTime)
		if err != nil {
			log.Errorf("ListMetric(%s) error: %v", metric, err)
			continue
		}
		tags := map[string]string{}
		if len(tag) > 0 && len(tagKey) > 0 {
			tags[tagKey] = tag
		}
		for i := range result {
			ret = append(ret, cloudprovider.MetricValues{
				Id:         result[i].InstanceId,
				MetricType: opts.MetricType,
				Values: []cloudprovider.MetricValue{
					{
						Timestamp: time.UnixMilli(result[i].Timestamp),
						Value:     result[i].GetValue(),
						Tags:      tags,
					},
				},
			})
		}
	}
	return ret, nil
}

func (self *SApsaraClient) GetRdsMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	metrics := map[string]string{}
	switch opts.MetricType {
	case cloudprovider.RDS_METRIC_TYPE_CPU_USAGE:
		metrics = map[string]string{
			"CpuUsage": "",
		}
	case cloudprovider.RDS_METRIC_TYPE_MEM_USAGE:
		metrics = map[string]string{
			"MemoryUsage": "",
		}
	case cloudprovider.RDS_METRIC_TYPE_NET_BPS_RX:
		metrics = map[string]string{
			"MySQL_NetworkInNew":     "",
			"SQLServer_NetworkInNew": "",
		}
	case cloudprovider.RDS_METRIC_TYPE_NET_BPS_TX:
		metrics = map[string]string{
			"MySQL_NetworkOutNew":     "",
			"SQLServer_NetworkOutNew": "",
		}
	case cloudprovider.RDS_METRIC_TYPE_DISK_USAGE:
		metrics = map[string]string{
			"DiskUsage": "",
		}
	case cloudprovider.RDS_METRIC_TYPE_CONN_USAGE:
		metrics = map[string]string{
			"ConnectionUsage": "",
		}
	default:
		return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "%s", opts.MetricType)
	}
	ret := []cloudprovider.MetricValues{}
	for metric := range metrics {
		result, err := self.ListMetrics("acs_rds_dashboard", metric, opts.StartTime, opts.EndTime)
		if err != nil {
			log.Errorf("ListMetric(%s) error: %v", metric, err)
			continue
		}
		for i := range result {
			ret = append(ret, cloudprovider.MetricValues{
				Id:         result[i].InstanceId,
				MetricType: opts.MetricType,
				Values: []cloudprovider.MetricValue{
					{
						Timestamp: time.UnixMilli(result[i].Timestamp),
						Value:     result[i].GetValue(),
					},
				},
			})
		}
	}
	return ret, nil
}

func (self *SApsaraClient) GetElbMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	metricTags, tagKey := map[string]string{}, ""
	switch opts.MetricType {
	case cloudprovider.LB_METRIC_TYPE_NET_BPS_RX:
		metricTags = map[string]string{
			"InstanceTrafficRX": "",
		}
	case cloudprovider.LB_METRIC_TYPE_NET_BPS_TX:
		metricTags = map[string]string{
			"InstanceTrafficTX": "",
		}
	case cloudprovider.LB_METRIC_TYPE_HRSP_COUNT:
		metricTags = map[string]string{
			"InstanceStatusCode2xx": cloudprovider.METRIC_TAG_REQUST_2XX,
			"InstanceStatusCode3xx": cloudprovider.METRIC_TAG_REQUST_3XX,
			"InstanceStatusCode4xx": cloudprovider.METRIC_TAG_REQUST_4XX,
			"InstanceStatusCode5xx": cloudprovider.METRIC_TAG_REQUST_5XX,
		}
		tagKey = cloudprovider.METRIC_TAG_REQUST
	default:
		return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "%s", opts.MetricType)
	}
	ret := []cloudprovider.MetricValues{}
	for metric, tag := range metricTags {
		result, err := self.ListMetrics("acs_slb_dashboard", metric, opts.StartTime, opts.EndTime)
		if err != nil {
			log.Errorf("ListMetric(%s) error: %v", metric, err)
			continue
		}
		tags := map[string]string{}
		if len(tag) > 0 && len(tagKey) > 0 {
			tags[tagKey] = tag
		}
		for i := range result {
			ret = append(ret, cloudprovider.MetricValues{
				Id:         result[i].InstanceId,
				MetricType: opts.MetricType,
				Values: []cloudprovider.MetricValue{
					{
						Timestamp: time.UnixMilli(result[i].Timestamp),
						Value:     result[i].GetValue(),
						Tags:      tags,
					},
				},
			})
		}
	}
	return ret, nil
}
