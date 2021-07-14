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

package storagemon

import (
	"strconv"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudmon/collectors/common"
	"yunion.io/x/onecloud/pkg/util/influxdb"
)

func (self *SStorageReport) collectMetric(storages []jsonutils.JSONObject) error {
	dataList := make([]influxdb.SMetricData, 0)
	for _, storage := range storages {
		metric, err := self.collectMetricFromStorage(storage)
		if err != nil {
			return err
		}
		metric.Timestamp = time.Now()
		metric.Name = STORAGE_MEASUREMENT
		dataList = append(dataList, metric)
	}
	return common.SendMetrics(self.Session, dataList, self.Args.Debug, "")
}

func (self *SStorageReport) collectMetricFromStorage(storage jsonutils.JSONObject) (influxdb.SMetricData, error) {
	metric, err := self.NewMetricFromJson(storage)
	if err != nil {
		return metric, errors.Wrap(err, "collectMetricFromStorage NewMetricFromJson err")
	}
	capacity, _ := storage.Float("capacity")
	actUsedCapacity, _ := storage.Float("actual_capacity_used")
	var actFreeCapacity = float64(0)
	var capacityUsage = float64(0)
	if capacity != 0 {
		actFreeCapacity = capacity - actUsedCapacity
		capacityUsage = actUsedCapacity / capacity * 100
	}
	metric.Metrics = append(metric.Metrics, influxdb.SKeyValue{
		Key:   STORAGE_FIELD_USAGE,
		Value: strconv.FormatFloat(capacityUsage, 'f', 2, 64),
	}, influxdb.SKeyValue{
		Key:   STORAGE_FIELD_FREE,
		Value: strconv.FormatFloat(actFreeCapacity, 'f', -1, 64),
	})
	return metric, nil

}
