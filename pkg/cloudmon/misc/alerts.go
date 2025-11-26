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

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/cloudcommon/tsdb"
	"yunion.io/x/onecloud/pkg/cloudmon/options"
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
		resp, err := monitor.AlertRecordManager.Get(s, "history-alert", nil)
		if err != nil {
			return errors.Wrapf(err, "history-alert")
		}
		alerts := api.AlertRecordHistoryAlert{}
		err = resp.Unmarshal(&alerts)
		if err != nil {
			return errors.Wrapf(err, "Unmarshal AlertRecordHistoryAlert")
		}
		metrics := []influxdb.SMetricData{}
		for _, alert := range alerts.Data {
			if alert.ResType == "host" && len(alert.DomainId) == 0 {
				continue
			}
			if alert.ResType == "guest" && len(alert.ProjectId) == 0 {
				continue
			}
			metric := influxdb.SMetricData{}
			metric.Name = ALERT_RECORD_HISTORY_MEASUREMENT
			metric.Timestamp = time.Now()
			metric.Metrics = []influxdb.SKeyValue{
				{
					Key:   "res_num",
					Value: fmt.Sprintf("%d", alert.ResNum),
				},
			}
			for k, v := range alert.GetMetricTags() {
				metric.Tags = append(metric.Tags, influxdb.SKeyValue{
					Key:   k,
					Value: v,
				})
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
