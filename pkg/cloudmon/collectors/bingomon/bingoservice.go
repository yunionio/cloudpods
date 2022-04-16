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

package bingomon

import (
	"strconv"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudmon/collectors/common"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud/bingocloud"
	"yunion.io/x/onecloud/pkg/util/influxdb"
)

func (self *SBingoCloudReport) collectRegionMetricOfResource(region cloudprovider.ICloudRegion, servers []jsonutils.JSONObject) error {
	dataList := make([]influxdb.SMetricData, 0)
	bingoReg := region.(*bingocloud.SRegion)
	since, until, err := common.TimeRangeFromArgs(self.Args)
	if err != nil {
		return err
	}
	namespace, metrics := self.getMetricSpecs(servers)
	dimensionId := self.getDimensionId()
	if dimensionId == nil {
		return errors.Errorf("can not get BingoCloudReport getDimensionId")
	}

	for metricName, influxDbSpecs := range metrics {
		for _, server := range servers {
			external_id, _ := server.GetString(dimensionId.LocalId)
			if len(external_id) == 0 {
				continue
			}
			name, _ := server.GetString("name")
			dimension := bingocloud.Dimension{
				Name:  dimensionId.ExtId,
				Value: external_id,
			}
			rtnMetric, err := bingoReg.DescribeMetricList(dimension, namespace, metricName, since,
				until, "")
			if err != nil {
				log.Errorln(err)
				continue
			}
			if rtnMetric != nil {
				serverMetric, err := self.collectMetricFromThisServer(server, *rtnMetric, influxDbSpecs)
				if err != nil {
					log.Errorf("provider: %s,metric: %s collectMetricFromThisServer: %s, err: %#v", self.SProvider.Name,
						metricName, name, err)
					continue
				}
				dataList = append(dataList, serverMetric...)
			}
		}

	}
	return common.SendMetrics(self.Session, dataList, self.Args.Debug, "")
}

func (self *SBingoCloudReport) collectMetricFromThisServer(server jsonutils.JSONObject, rtnMetric bingocloud.GetMetricStatisticsOutput,
	influxDbSpecs []string) ([]influxdb.SMetricData, error) {
	datas := make([]influxdb.SMetricData, 0)
	for _, point := range rtnMetric.Datapoints.Member {
		metric, err := self.NewMetricFromJson(server)
		//metric, err := common.JsonToMetric(server.(*jsonutils.JSONDict), "", common.ServerTags, make([]string, 0))
		if err != nil {
			return nil, err
		}
		metric.Timestamp = *point.Timestamp
		//根据条件拼装metric的tag和metirc信息
		influxDbSpec := influxDbSpecs[2]
		var pairsKey string
		if strings.Contains(influxDbSpec, ",") {
			pairsKey = common.SubstringBetween(influxDbSpec, ".", ",")
		} else {
			pairsKey = common.SubstringAfter(influxDbSpec, ".")
		}
		fieldValue := *point.Average
		if influxDbSpecs[1] == common.UNIT_BYTEPS && strings.Contains(pairsKey, common.UNIT_BPS) {
			fieldValue = fieldValue * 8
		}

		measurement := self.getMeasurement(influxDbSpec)
		tag := common.SubstringAfter(influxDbSpec, ",")
		if tag != "" && strings.Contains(influxDbSpec, "=") {
			metric.Tags = append(metric.Tags, influxdb.SKeyValue{
				Key:   common.SubstringBefore(tag, "="),
				Value: common.SubstringAfter(tag, "="),
			})
		}
		metric.Metrics = append(metric.Metrics, influxdb.SKeyValue{
			Key:   pairsKey,
			Value: strconv.FormatFloat(fieldValue, 'E', -1, 64),
		})
		metric.Name = measurement
		datas = append(datas, metric)
	}

	return datas, nil
}

func (self *SBingoCloudReport) getMetricSpecs(res []jsonutils.JSONObject) (string, map[string][]string) {
	switch common.MonType(self.Operator) {
	case common.SERVER:
		return SERVER_METRIC_NAMESPACE, bingoMetricSpecs
	case common.HOST:
		return HOST_METRIC_NAMESPACE, bingoMetricSpecs
	}
	return "", map[string][]string{}
}

func (self *SBingoCloudReport) getMeasurement(influxDbSpec string) (measurement string) {
	switch common.MonType(self.Operator) {
	case common.HOST:
		measurement = common.SubstringBetween(influxDbSpec, "vm_", ".")
		if strings.Contains(influxDbSpec, "vm_netio") {
			measurement = "net"
		}
	default:
		measurement = common.SubstringBefore(influxDbSpec, ".")
	}
	return measurement
}

func (self *SBingoCloudReport) getDataPointValue(valueKey string, influxDbSpecs []string, rtnMetric jsonutils.JSONObject) (float64, error) {
	key := common.UNIT_AVERAGE
	switch common.MonType(self.Operator) {
	case common.OSS:
		key = valueKey
	case common.K8S:
		key = "Value"
	}
	fieldValue, err := rtnMetric.Float(key)
	if err != nil {
		return 0, err
	}
	return fieldValue, nil
}

func (self *SBingoCloudReport) getDimensionId() *common.DimensionId {
	switch common.MonType(self.Operator) {
	case common.SERVER:
		return &common.DimensionId{
			LocalId: "external_id",
			ExtId:   "InstanceId",
		}
	case common.HOST:
		return &common.DimensionId{
			LocalId: "access_ip",
			ExtId:   "HostId",
		}

	}
	return nil
}
