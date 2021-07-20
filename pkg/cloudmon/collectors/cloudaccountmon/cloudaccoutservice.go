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
