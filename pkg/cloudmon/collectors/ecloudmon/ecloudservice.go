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

package ecloudmon

import (
	"strconv"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudmon/collectors/common"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud/ecloud"
	"yunion.io/x/onecloud/pkg/util/influxdb"
)

const (
	SERVER_PRODUCT_TYPE = "vm"
)

func (self *SECloudReport) collectRegionMetricOfHost(region cloudprovider.ICloudRegion,
	servers []jsonutils.JSONObject) error {
	dataList := make([]influxdb.SMetricData, 0)
	ecloudReg := region.(*ecloud.SRegion)
	since, until, err := common.TimeRangeFromArgs(self.Args)
	if err != nil {
		return err
	}
	for _, server := range servers {
		external_id, err := server.GetString("external_id")
		if err != nil {
			continue
		}
		name, _ := server.GetString("name")
		external_id = strings.Split(external_id, "/")[1]
		metrics := make([]ecloud.Metric, 0)
		for metricName, _ := range ecloudMetricSpecs {
			metrics = append(metrics, ecloud.Metric{Name: metricName})
		}
		data, err := ecloudReg.DescribeMetricList(SERVER_PRODUCT_TYPE, metrics, external_id, since, until)
		if err != nil {
			return errors.Wrap(err, "region DescribeMetricList err:")
		}
		for _, entity := range data.Entitys {
			if influxDbSpecs, ok := ecloudMetricSpecs[entity.MetricName]; ok {
				metricData, err := self.collectMetricFromThisServer(server, entity, influxDbSpecs)
				if err != nil {
					log.Errorf("server:%s collectMetric:%s err", name, entity.MetricName)
					continue
				}
				dataList = append(dataList, metricData...)
			}

		}
	}
	return common.SendMetrics(self.Session, dataList, self.Args.Debug, "")
}

func (self *SECloudReport) collectMetricFromThisServer(server jsonutils.JSONObject,
	entity ecloud.Entity, influxDbSpecs []string) ([]influxdb.SMetricData, error) {
	datas := make([]influxdb.SMetricData, 0)
	for _, point := range entity.Datapoints {
		metric, err := common.JsonToMetric(server.(*jsonutils.JSONDict), "", common.ServerTags, make([]string, 0))
		if err != nil {
			return nil, err
		}
		if len(point) != 2 {
			log.Errorf("invalid point:%v", point)
			continue
		}
		influxDbSpec := influxDbSpecs[2]

		measurement := common.SubstringBefore(influxDbSpec, ".")
		metric.Name = measurement
		pointTime, err := strconv.ParseInt(point[1], 10, 64)
		if err != nil {
			return nil, errors.Wrap(err, "point parseInt err")
		}
		metric.Timestamp = time.Unix(pointTime, 0)
		pointVal, err := strconv.ParseFloat(point[0], 64)
		if err != nil {
			return nil, errors.Wrap(err, "point parseInt err")
		}
		if influxDbSpecs[1] == UNIT_BYTEPS {
			pointVal = pointVal * 8
		}
		if influxDbSpecs[1] == UNIT_KBYTEPS {
			pointVal = pointVal * 8 * 1024
		}
		var pairsKey string
		if strings.Contains(influxDbSpec, ",") {
			pairsKey = common.SubstringBetween(influxDbSpec, ".", ",")
		} else {
			pairsKey = common.SubstringAfter(influxDbSpec, ".")
		}

		tag := common.SubstringAfter(influxDbSpec, ",")
		if tag != "" && strings.Contains(influxDbSpec, "=") {
			metric.Tags = append(metric.Tags, influxdb.SKeyValue{
				Key:   common.SubstringBefore(tag, "="),
				Value: common.SubstringAfter(tag, "="),
			})
		}
		metric.Metrics = append(metric.Metrics, influxdb.SKeyValue{
			Key:   pairsKey,
			Value: strconv.FormatFloat(pointVal, 'f', -1, 64),
		})
		self.AddMetricTag(&metric, common.AddTags)
		datas = append(datas, metric)
	}
	return datas, nil
}
