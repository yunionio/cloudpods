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
	"strconv"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/timeutils"
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

func (r *SRegion) metricsRequest(action string, params map[string]string) (jsonutils.JSONObject, error) {
	client := r.GetClient()
	cli, err := client.getDefaultClient(params)
	if err != nil {
		return nil, err
	}
	return monitorRequest(cli, action, params, client.cpcfg.UpdatePermission, client.debug)
}

func (r *SRegion) GetMonitorData(name string, ns string, since time.Time, until time.Time, demensions []SQcInstanceMetricDimension) ([]SDataPoint, error) {
	params := make(map[string]string)
	params["Region"] = r.Region
	params["MetricName"] = name
	params["Namespace"] = ns
	if !since.IsZero() {
		params["StartTime"] = since.Format(timeutils.IsoTimeFormat)

	}
	if !until.IsZero() {
		params["EndTime"] = until.Format(timeutils.IsoTimeFormat)
	}
	for index, metricDimension := range demensions {
		i := strconv.FormatInt(int64(index), 10)
		for internalIndex, interDimension := range metricDimension.Dimensions {
			j := strconv.FormatInt(int64(internalIndex), 10)
			params["Instances."+i+".Dimensions."+j+".Name"] = interDimension.Name
			params["Instances."+i+".Dimensions."+j+".Value"] = interDimension.Value
		}
	}
	body, err := r.metricsRequest("GetMonitorData", params)
	if err != nil {
		return nil, errors.Wrap(err, "region.MetricRequest")
	}
	dataArray := make([]SDataPoint, 0)
	err = body.Unmarshal(&dataArray, "DataPoints")
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return dataArray, nil
}

func (r *SRegion) GetK8sMonitorData(metricNames []string, ns string, since time.Time, until time.Time,
	demensions []SQcMetricDimension) ([]SK8SDataPoint, error) {
	params := make(map[string]string)
	params["Module"] = "monitor"
	params["Region"] = r.Region
	for index, name := range metricNames {
		i := strconv.FormatInt(int64(index), 10)
		params["MetricNames."+i] = name

	}
	params["Namespace"] = ns
	if !since.IsZero() {
		params["StartTime"] = since.Format(timeutils.IsoTimeFormat)

	}
	if !until.IsZero() {
		params["EndTime"] = until.Format(timeutils.IsoTimeFormat)
	}
	for index, metricDimension := range demensions {
		i := strconv.FormatInt(int64(index), 10)
		params["Conditions."+i+".Key"] = metricDimension.Name
		params["Conditions."+i+".Operator"] = "="
		params["Conditions."+i+".Value.0"] = metricDimension.Value
	}
	body, err := r.metricsRequest("DescribeStatisticData", params)
	if err != nil {
		return nil, errors.Wrap(err, "region.MetricRequest")
	}
	dataArray := make([]SK8SDataPoint, 0)
	err = body.Unmarshal(&dataArray, "Data")
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return dataArray, nil
}
