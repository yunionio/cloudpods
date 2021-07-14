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
	"time"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudmon/collectors/common"
	"yunion.io/x/onecloud/pkg/util/influxdb"
)

func (self *SAlertRecordHistoryReport) collectMetric(alertRecords []jsonutils.JSONObject) error {
	dataList := make([]influxdb.SMetricData, 0)
	matchRecord := self.getMaxEvalMatchOfAlertRecord(alertRecords)
	if matchRecord == nil {
		return nil
	}

	metric, err := self.NewMetricFromJson(matchRecord)
	if err != nil {
		return err
	}
	metric.Timestamp = time.Now()
	metric.Name = ALERT_RECORD_HISTORY_MEASUREMENT
	dataList = append(dataList, metric)
	return common.SendMetrics(self.Session, dataList, self.Args.Debug, ALERT_RECORD_HISTORY_DATABASE)
}

func (self *SAlertRecordHistoryReport) getMaxEvalMatchOfAlertRecord(alertRecords []jsonutils.JSONObject) jsonutils.JSONObject {
	var matchRecord jsonutils.JSONObject
	maxCount := int64(0)
	for _, record := range alertRecords {
		resNum, _ := record.Int("res_num")
		if resNum > maxCount {
			maxCount = resNum
			matchRecord = record
		}
	}
	return matchRecord
}
