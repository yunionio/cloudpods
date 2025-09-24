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

package cloudpods

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"time"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	monitorapi "yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/mcclient/modules/monitor"
)

func (cli *SCloudpodsClient) GetMetrics(opts *cloudprovider.MetricListOptions) ([]cloudprovider.MetricValues, error) {
	usefulResourceType := []cloudprovider.TResourceType{cloudprovider.METRIC_RESOURCE_TYPE_HOST, cloudprovider.METRIC_RESOURCE_TYPE_SERVER}
	isUse, _ := utils.InArray(opts.ResourceType, usefulResourceType)
	if !isUse {
		return nil, nil
	}

	info := strings.Split(string(opts.MetricType), ".")
	if len(info) != 2 {
		return nil, errors.Errorf("invalid metric type: %s", opts.MetricType)
	}
	measurement := info[0]
	field := info[1]
	from := fmt.Sprintf("now-%dm", int(time.Now().Sub(opts.StartTime).Minutes()))
	to := fmt.Sprintf("now-%dm", int(time.Now().Sub(opts.EndTime).Minutes()))

	query := &monitorapi.AlertQuery{
		Model: monitorapi.MetricQuery{
			Database:    "telegraf",
			Measurement: measurement,
			Selects: []monitorapi.MetricQuerySelect{
				{
					{
						Type:   "field",
						Params: []string{field},
					},
				},
			},
			Tags: []monitorapi.MetricQueryTag{
				{
					Key:      "brand",
					Operator: "=",
					Value:    "OneCloud",
				},
			},
		},
		From: from,
		To:   to,
	}

	input := monitorapi.MetricQueryInput{
		From:     from,
		To:       to,
		Scope:    "system",
		Interval: "1m",
		MetricQuery: []*monitorapi.AlertQuery{
			query,
		},
		SkipCheckSeries: true,
	}
	if cli.debug {
		input.ShowMeta = true
	}

	data := jsonutils.Marshal(input).(*jsonutils.JSONDict)
	sum256 := sha256.Sum256([]byte(data.String()))
	data.Set("signature", jsonutils.NewString(fmt.Sprintf("%x", sum256)))
	resp, err := monitor.UnifiedMonitorManager.PerformClassAction(cli.s, "query", data)
	if err != nil {
		return nil, errors.Wrapf(err, "query metric")
	}

	metrics := &monitorapi.MetricsQueryResult{}
	err = resp.Unmarshal(metrics)
	if err != nil {
		return nil, errors.Wrapf(err, "query metric Unmarshal")
	}

	res := []cloudprovider.MetricValues{}
	for _, serie := range metrics.Series {
		if len(serie.Tags) == 0 || (len(serie.Tags["vm_id"]) == 0 && len(serie.Tags["host_id"]) == 0) {
			continue
		}
		id := serie.Tags["vm_id"]
		if len(id) == 0 {
			id = serie.Tags["host_id"]
		}
		metric := cloudprovider.MetricValues{
			Id:         id,
			MetricType: opts.MetricType,
			Values:     []cloudprovider.MetricValue{},
		}
		values := []cloudprovider.MetricValue{}
		for _, point := range serie.Points {
			if len(point) != 2 {
				continue
			}
			values = append(values, cloudprovider.MetricValue{
				Timestamp: point.Time(),
				Value:     point.Value(),
			})
		}
		if len(values) == 0 {
			continue
		}
		metric.Values = values
		res = append(res, metric)
	}
	return res, nil
}
