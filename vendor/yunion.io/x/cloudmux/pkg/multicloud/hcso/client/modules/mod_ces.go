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

package modules

import (
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud/hcso/client/manager"
	"yunion.io/x/cloudmux/pkg/multicloud/hcso/client/requests"
)

type SCloudEyeManager struct {
	SResourceManager
}

type SMetricDimension struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type SDatapoint struct {
	Timestamp int64   `json:"timestamp"`
	Max       float64 `json:"max,omitzero"`
	Min       float64 `json:"min,omitzero"`
	Average   float64 `json:"average,omitzero"`
	Sum       float64 `json:"sum,omitzero"`
	Variance  float64 `json:"variance,omitzero"`
}

type SMetricData struct {
	SMetricMeta

	Datapoints []SDatapoint
}

type SMetricMeta struct {
	SMetric

	Unit string `json:"unit"`
}

type SMetric struct {
	MetricName string `json:"metric_name"`
	Namespace  string `json:"namespace"`

	Dimensions []SMetricDimension `json:"dimensions"`
}

func NewCloudEyeManager(cfg manager.IManagerConfig) *SCloudEyeManager {
	return &SCloudEyeManager{SResourceManager: SResourceManager{
		SBaseManager:    NewBaseManager(cfg),
		ServiceName:     ServiceNameCES,
		Region:          cfg.GetRegionId(),
		ProjectId:       cfg.GetProjectId(),
		version:         "V1.0",
		Keyword:         "",
		KeywordPlural:   "metrics",
		ResourceKeyword: "metrics",
	}}
}

func (ces *SCloudEyeManager) ListMetrics() ([]SMetricMeta, error) {
	metrics := make([]SMetricMeta, 0)
	next := ""
	for {
		marker, data, err := ces.listMetricsInternal(next)
		if err != nil {
			return nil, errors.Wrap(err, "ces.listMetricsInternal")
		}
		if len(data) == 0 {
			break
		}
		metrics = append(metrics, data...)
		next = marker
	}
	return metrics, nil
}

func (ces *SCloudEyeManager) listMetricsInternal(start string) (string, []SMetricMeta, error) {
	request := requests.NewResourceRequest(ces.GetEndpoint(), "GET", string(ces.ServiceName), ces.version, ces.Region, ces.ProjectId, ces.ResourceKeyword)
	request.AddQueryParam("limit", "1000")
	if len(start) > 0 {
		request.AddQueryParam("start", start)
	}
	_, resp, err := ces.jsonRequest(request)
	if err != nil {
		return "", nil, errors.Wrap(err, "ces.jsonRequest")
	}
	marker, _ := resp.GetString("meta_data", "marker")
	metrics := make([]SMetricMeta, 0)
	err = resp.Unmarshal(&metrics, "metrics")
	if err != nil {
		return "", nil, errors.Wrap(err, "resp.Unmarshal metrics")
	}
	return marker, metrics, nil
}

type SBatchQueryMetricDataInput struct {
	Metrics []SMetric `json:"metrics"`

	From   int64  `json:"from"`
	To     int64  `json:"to"`
	Period string `json:"period"`
	Filter string `json:"filter"`
}

func (ces *SCloudEyeManager) GetMetricsData(metrics []SMetricMeta, since time.Time, until time.Time) ([]SMetricData, error) {
	if len(metrics) > 10 {
		return nil, errors.Wrap(cloudprovider.ErrTooLarge, "request more than 10 metrics")
	}
	metricReq := make([]SMetric, len(metrics))
	for i := range metrics {
		metricReq[i] = metrics[i].SMetric
	}
	request := requests.NewResourceRequest(ces.GetEndpoint(), "POST", string(ces.ServiceName), ces.version, ces.Region, ces.ProjectId, "batch-query-metric-data")
	input := SBatchQueryMetricDataInput{
		Metrics: metricReq,
		From:    since.Unix() * 1000,
		To:      until.Unix() * 1000,
		Period:  "1",
		Filter:  "average",
	}
	body := jsonutils.Marshal(&input).String()
	request.SetContent([]byte(body))
	_, resp, err := ces.jsonRequest(request)
	if err != nil {
		return nil, errors.Wrap(err, "ces.jsonRequest")
	}
	//log.Debugf("%s", resp)
	result := make([]SMetricData, 0)
	err = resp.Unmarshal(&result, "metrics")
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return result, nil
}
