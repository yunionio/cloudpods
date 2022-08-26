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

package misc

import (
	"context"
	"fmt"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/cloutpost/options"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/monitor"
	"yunion.io/x/onecloud/pkg/util/influxdb"
)

const (
	ALERT_METRIC_DATABASE            = "monitor"
	ALERT_RECORD_HISTORY_MEASUREMENT = "alert_record_history"
)

func AlertHistoryReport(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	err := func() error {
		alerts := []api.CommonAlertDetails{}
		s := auth.GetAdminSession(ctx, options.Options.Region)
		for {
			params := map[string]interface{}{
				"limit":  1024,
				"offset": len(alerts),
				"scope":  "system",
			}
			resp, err := monitor.CommonAlertManager.List(s, jsonutils.Marshal(params))
			if err != nil {
				return errors.Wrapf(err, "CommonAlertManager.List")
			}
			part := []api.CommonAlertDetails{}
			err = jsonutils.Update(&part, resp.Data)
			if err != nil {
				return errors.Wrapf(err, "jsonutils.Update")
			}
			alerts = append(alerts, part...)
			if len(alerts) >= resp.Total {
				break
			}
		}
		metrics := []influxdb.SMetricData{}
		for i := range alerts {
			records := []api.AlertRecordDetails{}
			for {
				query := map[string]interface{}{
					"limit":    40,
					"offset":   len(records),
					"alert_id": alerts[i].Id,
					"state":    "alerting",
					"order_by": "res_num",
					"order":    "desc",
					"filter":   fmt.Sprintf("created_at.ge('%s')", time.Now().Add(time.Hour*-24)),
				}
				ret, err := monitor.AlertRecordManager.List(s, jsonutils.Marshal(query))
				if err != nil {
					log.Errorf("AlertRecordManager.List error: %v", err)
					break
				}
				part := []api.AlertRecordDetails{}
				err = jsonutils.Update(&part, ret.Data)
				if err != nil {
					break
				}
				records = append(records, part...)
			}
			metric := influxdb.SMetricData{}
			metric.Name = ALERT_RECORD_HISTORY_MEASUREMENT
			maxIdx, maxValue := -1, int64(0)
			for j := range records {
				if records[j].ResNum > maxValue {
					maxValue = records[j].ResNum
					maxIdx = j
				}
			}
			if maxIdx == -1 {
				break
			}
			for k, v := range records[maxIdx].GetMetricTags() {
				metric.Tags = append(metric.Tags, influxdb.SKeyValue{
					Key:   k,
					Value: v,
				})
			}
			metric.Timestamp = records[maxIdx].CreatedAt
			metric.Metrics = []influxdb.SKeyValue{
				{
					Key:   "res_num",
					Value: fmt.Sprintf("%d", records[maxIdx].ResNum),
				},
			}
			metrics = append(metrics, metric)
		}
		urls, err := s.GetServiceURLs(apis.SERVICE_TYPE_INFLUXDB, options.Options.SessionEndpointType, "")
		if err != nil {
			return errors.Wrap(err, "GetServiceURLs")
		}
		return influxdb.SendMetrics(urls, ALERT_METRIC_DATABASE, metrics, false)
	}()
	if err != nil {
		log.Errorf("AlertHistoryReport error: %v", err)
	}
}
