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

package aliyun

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

const (
	ALIYUN_API_VERSION_METRICS = "2019-01-01"
)

func (r *SRegion) metricsRequest(action string, params map[string]string) (jsonutils.JSONObject, error) {
	return r.client.metricsRequest(action, params)
}

func (self *SAliyunClient) metricsRequest(action string, params map[string]string) (jsonutils.JSONObject, error) {
	client, err := self.getSdkClient("")
	if err != nil {
		return nil, errors.Wrap(err, "self.getSdkClient")
	}
	return jsonRequest(client, "metrics.aliyuncs.com", ALIYUN_API_VERSION_METRICS, action, params, self.debug)
}

type SResourceLabel struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type SResource struct {
	Description string `json:"Description"`
	Labels      string `json:"Labels"`
	Namespace   string `json:"Namespace"`
}

func (r *SRegion) DescribeProjectMeta(limit, offset int) (int, []SResource, error) {
	params := make(map[string]string)
	if limit <= 0 {
		limit = 30
	}
	params["PageSize"] = strconv.FormatInt(int64(limit), 10)
	if offset > 0 {
		pageNum := (offset / limit) + 1
		params["PageNumber"] = strconv.FormatInt(int64(pageNum), 10)
	}
	body, err := r.metricsRequest("DescribeProjectMeta", params)
	if err != nil {
		return 0, nil, errors.Wrap(err, "r.metricsRequest DescribeProjectMeta")
	}
	total, _ := body.Int("Total")
	res := make([]SResource, 0)
	err = body.Unmarshal(&res, "Resources", "Resource")
	if err != nil {
		return 0, nil, errors.Wrap(err, "body.Unmarshal Resources Resource")
	}
	return int(total), res, nil
}

func (r *SRegion) FetchNamespaces() ([]SResource, error) {
	resources := make([]SResource, 0)
	total := -1
	for total < 0 || len(resources) < total {
		ntotal, res, err := r.DescribeProjectMeta(1000, len(resources))
		if err != nil {
			return nil, errors.Wrap(err, "r.DescribeProjectMeta")
		}
		if len(res) == 0 {
			break
		}
		resources = append(resources, res...)
		total = ntotal
	}
	return resources, nil
}

type SMetricMeta struct {
	Description string `json:"Description"`
	MetricName  string `json:"MetricName"`
	Statistics  string `json:"Statistics"`
	Labels      string `json:"Labels"`
	Dimensions  string `json:"Dimensions"`
	Namespace   string `json:"Namespace"`
	Periods     string `json:"Periods"`
	Unit        string `json:"Unit"`
}

func (r *SRegion) DescribeMetricMetaList(ns string, limit, offset int) (int, []SMetricMeta, error) {
	params := make(map[string]string)
	if limit <= 0 {
		limit = 30
	}
	params["Namespace"] = ns
	params["PageSize"] = strconv.FormatInt(int64(limit), 10)
	if offset > 0 {
		pageNum := (offset / limit) + 1
		params["PageNumber"] = strconv.FormatInt(int64(pageNum), 10)
	}
	body, err := r.metricsRequest("DescribeMetricMetaList", params)
	if err != nil {
		return 0, nil, errors.Wrap(err, "r.metricsRequest DescribeMetricMetaList")
	}
	total, _ := body.Int("TotalCount")
	res := make([]SMetricMeta, 0)
	err = body.Unmarshal(&res, "Resources", "Resource")
	if err != nil {
		return 0, nil, errors.Wrap(err, "body.Unmarshal Resources Resource")
	}
	return int(total), res, nil
}

func (r *SRegion) FetchMetrics(ns string) ([]SMetricMeta, error) {
	metrics := make([]SMetricMeta, 0)
	total := -1
	for total < 0 || len(metrics) < total {
		ntotal, res, err := r.DescribeMetricMetaList(ns, 1000, len(metrics))
		if err != nil {
			return nil, errors.Wrap(err, "r.DescribeMetricMetaList")
		}
		if len(res) == 0 {
			break
		}
		metrics = append(metrics, res...)
		total = ntotal
	}
	return metrics, nil
}

func (r *SRegion) DescribeMetricList(name string, ns string, since time.Time, until time.Time, nextToken string, dimensions []SResourceLabel) ([]jsonutils.JSONObject, string, error) {
	params := make(map[string]string)
	params["MetricName"] = name
	params["Namespace"] = ns
	params["Length"] = "2000"
	if len(nextToken) > 0 {
		params["NextToken"] = nextToken
	}
	if !since.IsZero() {
		params["StartTime"] = strconv.FormatInt(since.Unix()*1000, 10)
	}
	if !until.IsZero() {
		params["EndTime"] = strconv.FormatInt(until.Unix()*1000, 10)
	}
	if len(dimensions) > 0 {
		for _, dimension := range dimensions {
			params[dimension.Name] = dimension.Value
		}
	}
	body, err := r.metricsRequest("DescribeMetricList", params)
	if err != nil {
		return nil, "", errors.Wrap(err, "region.MetricRequest")
	}
	nToken, _ := body.GetString("NextToken")
	dataStr, _ := body.GetString("Datapoints")
	if len(dataStr) == 0 {
		return nil, "", nil
	}
	dataJson, err := jsonutils.ParseString(dataStr)
	if err != nil {
		return nil, "", errors.Wrap(err, "jsonutils.ParseString")
	}
	dataArray, err := dataJson.GetArray()
	if err != nil {
		return nil, "", errors.Wrap(err, "dataJson.GetArray")
	}
	return dataArray, nToken, nil
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

	Diskname string
	Device   string

	// k8s
	Cluster string
	Node    string
	Pod     string
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

func (self *SAliyunClient) ListMetrics(ns, metricName string, start, end time.Time) ([]MetricData, error) {
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

func (self *SAliyunClient) listMetrics(ns, metricName, nextToken string, start, end time.Time) ([]MetricData, string, error) {
	params := make(map[string]string)
	params["MetricName"] = metricName
	params["Namespace"] = ns
	params["Length"] = "2000"
	if len(nextToken) > 0 {
		params["NextToken"] = nextToken
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

func (r *SRegion) FetchMetricData(name string, ns string, since time.Time, until time.Time) ([]jsonutils.JSONObject, error) {
	data := make([]jsonutils.JSONObject, 0)
	nextToken := ""
	for {
		datArray, next, err := r.DescribeMetricList(name, ns, since, until, nextToken, nil)
		if err != nil {
			return nil, errors.Wrap(err, "r.DescribeMetricList")
		}
		data = append(data, datArray...)
		if len(next) == 0 {
			break
		}
		nextToken = next
	}
	return data, nil
}

func (self *SAliyunClient) GetMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
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
	case cloudprovider.METRIC_RESOURCE_TYPE_K8S:
		return self.GetK8sMetrics(opts)
	default:
		return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "%s", opts.ResourceType)
	}
}

func (self *SAliyunClient) GetEcsMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
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

func (self *SAliyunClient) GetOssMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
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

func (self *SAliyunClient) GetRedisMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	metrics := map[cloudprovider.TMetricType]string{
		cloudprovider.REDIS_METRIC_TYPE_CPU_USAGE:      "CpuUsage",
		cloudprovider.REDIS_METRIC_TYPE_MEM_USAGE:      "MemoryUsage",
		cloudprovider.REDIS_METRIC_TYPE_NET_BPS_RX:     "IntranetIn",
		cloudprovider.REDIS_METRIC_TYPE_NET_BPS_TX:     "IntranetOut",
		cloudprovider.REDIS_METRIC_TYPE_USED_CONN:      "UsedConnection",
		cloudprovider.REDIS_METRIC_TYPE_OPT_SES:        "UsedQPS",
		cloudprovider.REDIS_METRIC_TYPE_CACHE_KEYS:     "StandardKeys",
		cloudprovider.REDIS_METRIC_TYPE_DATA_MEM_USAGE: "UsedMemory",
	}
	metric, ok := metrics[opts.MetricType]
	if !ok {
		return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "%s", opts.MetricType)
	}
	ret := []cloudprovider.MetricValues{}
	result, err := self.ListMetrics("acs_kvstore", metric, opts.StartTime, opts.EndTime)
	if err != nil {
		return nil, errors.Wrapf(err, "ListMetric(%s)", metric)
	}
	for i := range result {
		tags := map[string]string{}
		if strings.Contains(result[i].InstanceId, "-db-") {
			tags[cloudprovider.METRIC_TAG_NODE] = result[i].InstanceId
			idx := strings.Index(result[i].InstanceId, "-db-")
			result[i].InstanceId = result[i].InstanceId[:idx]
		}
		value := cloudprovider.MetricValues{
			Id:         result[i].InstanceId,
			MetricType: opts.MetricType,
			Values: []cloudprovider.MetricValue{
				{
					Timestamp: time.UnixMilli(result[i].Timestamp),
					Value:     result[i].GetValue(),
					Tags:      tags,
				},
			},
		}
		ret = append(ret, value)
	}
	return ret, nil
}

func (self *SAliyunClient) GetRdsMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	metricTags := map[string]string{}
	switch opts.MetricType {
	case cloudprovider.RDS_METRIC_TYPE_CPU_USAGE:
		metricTags = map[string]string{
			"CpuUsage": "",
		}
	case cloudprovider.RDS_METRIC_TYPE_MEM_USAGE:
		metricTags = map[string]string{
			"MemoryUsage": "",
		}
	case cloudprovider.RDS_METRIC_TYPE_NET_BPS_RX:
		metricTags = map[string]string{
			"MySQL_NetworkInNew":     "",
			"SQLServer_NetworkInNew": "",
		}
	case cloudprovider.RDS_METRIC_TYPE_NET_BPS_TX:
		metricTags = map[string]string{
			"MySQL_NetworkOutNew":     "",
			"SQLServer_NetworkOutNew": "",
		}
	case cloudprovider.RDS_METRIC_TYPE_DISK_USAGE:
		metricTags = map[string]string{
			"DiskUsage": "",
		}
	case cloudprovider.RDS_METRIC_TYPE_CONN_USAGE:
		metricTags = map[string]string{
			"ConnectionUsage": "",
		}
	default:
		return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "%s", opts.MetricType)
	}
	ret := []cloudprovider.MetricValues{}
	for metric := range metricTags {
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

func (self *SAliyunClient) GetElbMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
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

func (self *SAliyunClient) GetK8sMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	metricName := ""
	switch opts.MetricType {
	case cloudprovider.K8S_NODE_METRIC_TYPE_CPU_USAGE:
		metricName = "node.cpu.utilization"
	case cloudprovider.K8S_NODE_METRIC_TYPE_MEM_USAGE:
		metricName = "node.memory.utilization"
	default:
		return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "%s", opts.MetricType)
	}
	result, err := self.ListMetrics("acs_k8s", metricName, opts.StartTime, opts.EndTime)
	if err != nil {
		return nil, errors.Wrapf(err, "ListMetrics(%s)", metricName)
	}
	ret := []cloudprovider.MetricValues{}
	for i := range result {
		tags := map[string]string{}
		if len(result[i].Node) > 0 {
			tags[cloudprovider.METRIC_TAG_NODE] = result[i].Node
		}
		ret = append(ret, cloudprovider.MetricValues{
			Id:         result[i].Cluster,
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
	return ret, nil
}
