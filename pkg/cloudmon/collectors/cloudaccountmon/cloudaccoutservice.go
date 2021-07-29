package cloudaccountmon

import (
	"time"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudmon/collectors/common"
	"yunion.io/x/onecloud/pkg/util/influxdb"
)

func (self *SCloudAccountReport) collectMetric(accounts []jsonutils.JSONObject) error {
	dataList := make([]influxdb.SMetricData, 0)
	for _, account := range accounts {
		metric, err := self.NewMetricFromJson(account)
		if err != nil {
			return err
		}
		metric.Timestamp = time.Now()
		metric.Name = CLOUDACCOUNT_MEASUREMENT
		dataList = append(dataList, metric)
	}
	return common.SendMetrics(self.Session, dataList, self.Args.Debug, CLOUDACCOUNT_DATABASE)
}
