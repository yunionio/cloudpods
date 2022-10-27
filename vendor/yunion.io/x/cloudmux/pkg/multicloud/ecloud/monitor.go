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
	"strconv"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/timeutils"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
)

const (
	MONITOR_FETCH_REQUEST_ACTION   = "v1.dawn.monitor.fetch"
	MONITOR_PRODUCT_REQUEST_ACTION = "v1.dawn.monitor.products"

	MONITOR_FETCH_SERVER_PATH = "/api/edw/edw/api"

	REQUEST_SUCCESS_CODE = "000000"
)

var (
	noMetricRegion = []string{"guangzhou-2", "beijing-1", "hunan-1"}

	portRegionMap = map[string][]string{
		"8443": {"wuxi-1", "dongguan-1", "yaan-1", "zhengzhou-1", "beijing-2", "zhuzhou-1", "jinan-1",
			"xian-1", "shanghai-1", "chongqing-1", "ningbo-1"},
		"18080": {"tianjin-1", "jilin-1", "hubei-1", "jiangxi-1", "gansu-1", "shanxi-1", "liaoning-1",
			"yunnan-2", "hebei-1", "fujian-1", "guangxi-1", "anhui-1", "huhehaote-1", "guiyang-1"},
	}
)

type Metric struct {
	Name string `json:"MetricName"`
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
	SApiRequest
}

func NewMonitorRequest(regionId string, serverPath string, query map[string]string, data jsonutils.JSONObject) *SMonitorRequest {
	apiRequest := NewApiRequest(regionId, serverPath, query, data)
	return &SMonitorRequest{*apiRequest}
}

func (rr *SMonitorRequest) GetPort() string {
	for port, regions := range portRegionMap {
		if utils.IsInStringArray(rr.GetRegionId(), regions) {
			return port
		}
	}
	return "8443"
}

func (br *SMonitorRequest) ForMateResponseBody(jrbody jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	code, _ := jrbody.GetString("code")
	if code != REQUEST_SUCCESS_CODE {
		message, _ := jrbody.(*jsonutils.JSONDict).GetString("message")
		return nil, httperrors.NewBadRequestError("rep body code is :%s, message:%s,body:%v", code, message, jrbody)
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
	params := map[string]string{
		"eAction": MONITOR_FETCH_REQUEST_ACTION,
	}
	getBody := jsonutils.NewDict()
	getBody.Set("startTime", jsonutils.NewString(since.Format(timeutils.MysqlTimeFormat)))
	getBody.Set("endTime", jsonutils.NewString(until.Format(timeutils.MysqlTimeFormat)))
	getBody.Set("productType", jsonutils.NewString(productType))
	getBody.Set("resourceId", jsonutils.NewString(resourceId))
	getBody.Set("metrics", jsonutils.Marshal(&metrics))
	request := NewMonitorRequest(regionId, MONITOR_FETCH_SERVER_PATH, params, getBody)
	err := self.doGet(context.Background(), request, &metricData)
	if err != nil {
		return metricData, errors.Wrap(err, "client doGet error")
	}
	return metricData, nil
}

func (r *SRegion) GetProductTypes() (jsonutils.JSONObject, error) {
	params := map[string]string{
		"eAction": MONITOR_PRODUCT_REQUEST_ACTION,
	}
	request := NewMonitorRequest(r.ID, MONITOR_FETCH_SERVER_PATH, params, nil)
	rtn := jsonutils.NewDict()
	err := r.client.doGet(context.Background(), request, rtn)
	if err != nil {
		return nil, errors.Wrap(err, "client doGet error")
	}
	return rtn, nil
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
		"cpu_util":                        cloudprovider.VM_METRIC_TYPE_CPU_USAGE,
		"memory.util":                     cloudprovider.VM_METRIC_TYPE_MEM_USAGE,
		"disk.device.read.requests.rate":  cloudprovider.VM_METRIC_TYPE_DISK_IO_READ_IOPS,
		"disk.device.write.requests.rate": cloudprovider.VM_METRIC_TYPE_DISK_IO_WRITE_IOPS,
		"disk.device.read.bytes.rate":     cloudprovider.VM_METRIC_TYPE_DISK_IO_READ_BPS,
		"disk.device.write.bytes.rate":    cloudprovider.VM_METRIC_TYPE_DISK_IO_WRITE_BPS,
		"network.incoming.bytes":          cloudprovider.VM_METRIC_TYPE_NET_BPS_RX,
		"network.outgoing.bytes":          cloudprovider.VM_METRIC_TYPE_NET_BPS_TX,
	}
	metricNames := []Metric{}
	for metric := range metrics {
		metricNames = append(metricNames, Metric{
			Name: metric,
		})
	}
	if utils.IsInStringArray(opts.RegionExtId, noMetricRegion) {
		return []cloudprovider.MetricValues{}, nil
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
