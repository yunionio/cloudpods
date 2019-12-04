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
	"strconv"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
)

const (
	ALIYUN_API_VERSION_METRICS = "2019-01-01"
)

func (r *SRegion) metricsRequest(action string, params map[string]string) (jsonutils.JSONObject, error) {
	client, err := r.getSdkClient()
	if err != nil {
		return nil, errors.Wrap(err, "r.getSdkClient")
	}
	return jsonRequest(client, "metrics.aliyuncs.com", ALIYUN_API_VERSION_METRICS, action, params, r.client.Debug)
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

func (r *SRegion) DescribeMetricList(name string, ns string, since time.Time, until time.Time, nextToken string) ([]jsonutils.JSONObject, string, error) {
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

func (r *SRegion) FetchMetricData(name string, ns string, since time.Time, until time.Time) ([]jsonutils.JSONObject, error) {
	data := make([]jsonutils.JSONObject, 0)
	nextToken := ""
	for {
		datArray, next, err := r.DescribeMetricList(name, ns, since, until, nextToken)
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
