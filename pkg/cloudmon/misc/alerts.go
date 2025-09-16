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

	api "yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/cloudcommon/tsdb"
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
		s := auth.GetAdminSession(ctx, options.Options.Region)
		params := map[string]interface{}{
			"field":    "res_type",
			"scope":    "system",
			"filter.0": fmt.Sprintf("created_at.ge('%s')", time.Now().Add(time.Hour*-24)),
		}
		ret, err := monitor.AlertRecordManager.Get(s, "distinct-field", jsonutils.Marshal(params))
		if err != nil {
			return errors.Wrapf(err, "distinct-filed")
		}
		resTypes := []string{}
		ret.Unmarshal(&resTypes, "res_type")
		metrics := []influxdb.SMetricData{}
		for _, resType := range resTypes {
			records := []api.AlertRecordDetails{}
			for {
				query := map[string]interface{}{
					"limit":    40,
					"offset":   len(records),
					"@state":   "alerting",
					"scope":    "system",
					"filter.0": fmt.Sprintf(`res_type.equals('%s')`, resType),
					"filter.1": fmt.Sprintf("created_at.ge('%s')", time.Now().Add(time.Hour*-24)),
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
				if len(records) >= ret.Total {
					break
				}
			}
			metric := influxdb.SMetricData{}
			metric.Name = ALERT_RECORD_HISTORY_MEASUREMENT
			cnt := 0
			for i, record := range records {
				cnt += int(record.ResNum)
				if i == 0 {
					for k, v := range record.GetMetricTags() {
						metric.Tags = append(metric.Tags, influxdb.SKeyValue{
							Key:   k,
							Value: v,
						})
					}
				}
			}
			metric.Timestamp = time.Now()
			metric.Metrics = []influxdb.SKeyValue{
				{
					Key:   "res_num",
					Value: fmt.Sprintf("%d", cnt),
				},
			}
			metrics = append(metrics, metric)
		}
		urls, err := tsdb.GetDefaultServiceSourceURLs(s, options.Options.SessionEndpointType)
		if err != nil {
			return errors.Wrap(err, "GetServiceURLs")
		}
		return influxdb.BatchSendMetrics(urls, ALERT_METRIC_DATABASE, metrics, false)
	}()
	if err != nil {
		log.Errorf("AlertHistoryReport error: %v", err)
	}
}
