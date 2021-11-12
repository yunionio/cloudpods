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

package alertrecordhistorymon

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"golang.org/x/sync/errgroup"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/timeutils"

	"yunion.io/x/onecloud/pkg/cloudmon/collectors/common"
	"yunion.io/x/onecloud/pkg/cloudmon/options"
	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/monitor"
)

func init() {
	factory := SAlertRecordHistoryFactory{}
	common.RegisterFactory(&factory)
}

type SAlertRecordHistoryFactory struct {
}

func (self *SAlertRecordHistoryFactory) MyRoutineInteval(monOptions options.CloudMonOptions) time.Duration {
	return time.Duration(monOptions.AlertRecordHistoryInterval)
}

func (self *SAlertRecordHistoryFactory) MyRoutineFunc() common.RoutineFunc {
	return common.MakePullMetricRoutineAtZeroPoint
}

func (self *SAlertRecordHistoryFactory) NewCloudReport(provider *common.SProvider, session *mcclient.ClientSession,
	args *options.ReportOptions,
	operatorType string) common.ICloudReport {
	return &SAlertRecordHistoryReport{
		common.CloudReportBase{
			SProvider: nil,
			Session:   session,
			Args:      args,
			Operator:  string(common.ALERT_RECORD),
		},
	}
}

func (S *SAlertRecordHistoryFactory) GetId() string {
	return string(common.ALERT_RECORD)
}

type SAlertRecordHistoryReport struct {
	common.CloudReportBase
}

func (self *SAlertRecordHistoryReport) Report() error {
	alerts, err := self.getMonitorCommonAlert()
	if err != nil {
		return err
	}
	errs := make([]error, 0)
	recordGroup, _ := errgroup.WithContext(context.Background())
	count := 0
	for i, _ := range alerts {
		tmp := alerts[i]
		count++
		recordGroup.Go(func() error {
			alert_id, _ := tmp.GetString("id")
			alertRecords, err := self.getAlertRecordsByAlertId(alert_id)
			if err != nil {
				return errors.Wrapf(err, "getAlertRecordsByAlertId:%s error", alert_id)
			}
			return self.collectMetric(alertRecords)
		})
		if count == 4 {
			err := recordGroup.Wait()
			if err != nil {
				errs = append(errs, errors.Wrap(err, "alertRecordHistoryReport collectMetric error"))
			}
			count = 0
		}
	}
	err = recordGroup.Wait()
	if err != nil {
		errs = append(errs, errors.Wrap(err, "alertRecordHistoryReport collectMetric error"))
	}
	return errors.NewAggregate(errs)
}

func (self *SAlertRecordHistoryReport) getMonitorCommonAlert() ([]jsonutils.JSONObject, error) {
	alerts := make([]jsonutils.JSONObject, 0)
	query := jsonutils.NewDict()
	query.Add(jsonutils.NewString("0"), common.KEY_LIMIT)
	//query.Add(jsonutils.NewBool(true), common.DETAILS)
	query.Add(jsonutils.NewString("system"), "scope")

	alerts, err := self.ListAllResource(modules.CommonAlertManager, query)
	if err != nil {
		return nil, errors.Wrap(err, "getMonitorCommonAlert error")
	}
	return alerts, nil
}

func (self *SAlertRecordHistoryReport) getAlertRecordsByAlertId(id string) ([]jsonutils.JSONObject, error) {
	now := time.Now().UTC()
	period64, err := strconv.ParseInt(self.Args.Interval, 10, 8)
	if err != nil {
		return nil, err
	}
	query := jsonutils.NewDict()
	query.Add(jsonutils.NewString("0"), common.KEY_LIMIT)
	query.Add(jsonutils.NewBool(true), common.DETAILS)
	query.Add(jsonutils.NewString("system"), "scope")
	query.Add(jsonutils.NewString(id), "alert_id")
	query.Add(jsonutils.NewString("alerting"), "state")
	query.Add(jsonutils.NewString(fmt.Sprintf(`created_at.between("%s","%s")`,
		now.Add(-time.Hour*24*time.Duration(period64)).Format(timeutils.MysqlTimeFormat),
		now.Format(timeutils.MysqlTimeFormat))),
		"filter")

	alertRecords, err := self.ListAllResource(modules.AlertRecordManager, query)
	if err != nil {
		return nil, err
	}
	return alertRecords, nil
}
