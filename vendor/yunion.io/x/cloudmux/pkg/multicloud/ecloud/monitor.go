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

package ecloud

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/timeutils"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

func getPoolIdByRegionId(regionId string) (string, bool) {
	if regionId == "" {
		return "", false
	}
	// regionIdToPoolId 使用 cn-* 作为 key，这里兼容传入不带 cn- 的情况
	if poolID := regionIdToPoolId[regionId]; poolID != "" {
		return poolID, true
	}
	if poolID := regionIdToPoolId["cn-"+regionId]; poolID != "" {
		return poolID, true
	}
	return "", false
}

type Metric struct {
	Name string `json:"metricName"`
}

type MetricData struct {
	Entitys []Entity `json:"entity"`
}

type Entity struct {
	ResourceID         string      `json:"resourceId"`
	MetricName         string      `json:"metricName"`
	MetricNameCN       string      `json:"metricNameCn"`
	Unit               string      `json:"unit"`
	MaxValue           int64       `json:"maxValue"`
	AvgValue           int64       `json:"avgValue"`
	MinValue           int64       `json:"minValue"`
	Granularity        string      `json:"granularity"`
	PolymerizeType     string      `json:"polymerizeType"`
	SelectedMetricItem interface{} `json:"selectedMetricItem"`
	MetricItems        interface{} `json:"metricItems"`
	IsChildnode        bool        `json:"isChildnode"`
	Datapoints         []Datapoint `json:"datapoints"`
}

type Datapoint []string

type SMonitorRequest struct {
	SJSONRequest
}

// NewMonitorRequest 监控 OpenAPI 统一走 ecloud.10086.cn（443）。
func NewMonitorRequest(regionId string, serverPath string, query map[string]string, data jsonutils.JSONObject) *SMonitorRequest {
	r := SMonitorRequest{SJSONRequest: newBaseJSONRequest(regionId, "ecloud.10086.cn", "", serverPath, query, data)}
	return &r
}

func (r *SMonitorRequest) Base() *SBaseRequest {
	return &r.SJSONRequest.SBaseRequest
}

func (br *SMonitorRequest) ForMateResponseBody(jrbody jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	code, _ := jrbody.GetString("code")
	if code != "000000" {
		message, _ := jrbody.(*jsonutils.JSONDict).GetString("message")
		return nil, errors.Errorf("rep body code is :%s, message:%s,body:%v", code, message, jrbody)
	}
	if jrbody == nil || !jrbody.Contains("entity") {
		return nil, ErrMissKey{
			Key: "entity",
			Jo:  jrbody,
		}
	}
	return jrbody, nil
}

func (self *SEcloudClient) DescribeMetricList(regionId, productType string, metrics []Metric, resourceId string,
	since time.Time, until time.Time) (MetricData, error) {
	metricData := MetricData{
		Entitys: make([]Entity, 0),
	}
	// 新接口要求 query.poolId（资源池 ID）
	poolID, ok := getPoolIdByRegionId(regionId)
	if !ok {
		return metricData, fmt.Errorf("missing poolId mapping for regionId %q", regionId)
	}
	params := map[string]string{
		"poolId": poolID,
	}
	getBody := jsonutils.NewDict()
	sh, _ := time.LoadLocation("Asia/Shanghai")
	getBody.Set("startTime", jsonutils.NewString(since.In(sh).Format(timeutils.MysqlTimeFormat)))
	getBody.Set("endTime", jsonutils.NewString(until.In(sh).Format(timeutils.MysqlTimeFormat)))
	getBody.Set("productType", jsonutils.NewString(productType))
	getBody.Set("resourceId", jsonutils.NewString(resourceId))
	getBody.Set("metrics", jsonutils.Marshal(&metrics))
	request := NewMonitorRequest(regionId, "/api/edw/openapi/version2/v1/dawn/monitor/distribute/fetch", params, getBody)
	// 新接口为 POST，且返回不遵循通用 state/body，需用 SMonitorRequest.ForMateResponseBody 校验/取数
	base := request.Base()
	base.SetMethod("POST")
	jrbody, err := self.doRequest(context.Background(), base)
	if err != nil {
		return metricData, errors.Wrap(err, "client doRequest error")
	}
	body, err := request.ForMateResponseBody(jrbody)
	if err != nil {
		return metricData, err
	}
	if err := body.Unmarshal(&metricData); err != nil {
		return metricData, errors.Wrap(err, "unmarshal metric data")
	}
	return metricData, nil
}

func (r *SRegion) GetMetricTypes() (jsonutils.JSONObject, error) {
	poolID, ok := getPoolIdByRegionId(r.RegionId)
	if !ok {
		return nil, fmt.Errorf("missing poolId mapping for regionId %q", r.RegionId)
	}
	params := map[string]string{
		"poolId":      poolID,
		"productType": "vm",
	}
	request := NewMonitorRequest(r.RegionId, "/api/edw/openapi/version2/v1/dawn/monitor/distribute/metricindicators", params, nil)
	base := request.Base()
	base.SetMethod("GET")
	jrbody, err := r.client.doRequest(context.Background(), base)
	if err != nil {
		return nil, errors.Wrap(err, "client doRequest error")
	}
	body, err := request.ForMateResponseBody(jrbody)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func (self *SEcloudClient) GetMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	switch opts.ResourceType {
	case cloudprovider.METRIC_RESOURCE_TYPE_SERVER:
		return self.GetEcsMetrics(opts)
	default:
		return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "%s", opts.ResourceType)
	}
}

func (self *SEcloudClient) GetEcsMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	metrics := map[string]cloudprovider.TMetricType{
		"vm_realtime_cpu_avg_util_percent":     cloudprovider.VM_METRIC_TYPE_CPU_USAGE,
		"vm_realtime_mem_avg_util_percent":     cloudprovider.VM_METRIC_TYPE_MEM_USAGE,
		"vm_disk_read_bytes_rate":              cloudprovider.VM_METRIC_TYPE_DISK_IO_READ_IOPS,
		"vm_disk_write_bytes_rate":             cloudprovider.VM_METRIC_TYPE_DISK_IO_WRITE_IOPS,
		"vm_realtime_allnetwork_incoming_rate": cloudprovider.VM_METRIC_TYPE_NET_BPS_RX,
		"vm_realtime_allnetwork_outgoing_rate": cloudprovider.VM_METRIC_TYPE_NET_BPS_TX,
	}

	metricNames := []Metric{}
	for metric := range metrics {
		metricNames = append(metricNames, Metric{
			Name: metric,
		})
	}
	data, err := self.DescribeMetricList(opts.RegionExtId, "vm", metricNames, opts.ResourceId, opts.StartTime, opts.EndTime)
	if err != nil {
		return nil, errors.Wrapf(err, "DescribeMetricList")
	}
	ret := []cloudprovider.MetricValues{}
	for _, value := range data.Entitys {
		metric := cloudprovider.MetricValues{}
		metric.Id = opts.ResourceId
		metricType, ok := metrics[value.MetricName]
		if !ok {
			continue
		}
		metric.MetricType = metricType
		for _, points := range value.Datapoints {
			if len(points) != 2 {
				continue
			}
			metricValue := cloudprovider.MetricValue{}
			pointTime, err := strconv.ParseInt(points[1], 10, 64)
			if err != nil {
				continue
			}
			metricValue.Timestamp = time.Unix(pointTime, 0)
			metricValue.Value, err = strconv.ParseFloat(points[0], 64)
			if err != nil {
				continue
			}
			metric.Values = append(metric.Values, metricValue)
		}
		ret = append(ret, metric)
	}
	return ret, nil
}
