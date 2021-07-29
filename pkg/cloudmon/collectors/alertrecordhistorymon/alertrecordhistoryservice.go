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
