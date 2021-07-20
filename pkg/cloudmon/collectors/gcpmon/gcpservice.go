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

package gcpmon

import (
	"strconv"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/timeutils"

	"yunion.io/x/onecloud/pkg/cloudmon/collectors/common"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud/google"
	"yunion.io/x/onecloud/pkg/util/influxdb"
)

func (self *SGoogleCloudReport) collectRegionMetricOfHost(region cloudprovider.ICloudRegion,
	servers []jsonutils.JSONObject) error {
	dataList := make([]influxdb.SMetricData, 0)
	googleReg := region.(*google.SRegion)
	since, until, err := common.TimeRangeFromArgs(self.Args)
	if err != nil {
		return err
	}
	for _, server := range servers {
		external_id, err := server.GetString("external_id")
		if err != nil {
			continue
		}
		serverName, _ := server.GetString("name")
		external_id = strings.Split(external_id, "/")[1]
		for metricName, influxDbSpecs := range gcpMetricSpecs {
			rtn, err := googleReg.GetMonitorData(external_id, serverName, metricName, since, until)
			if err != nil {
				log.Errorln(err)
				continue
			}
			metric, err := common.FillVMCapacity(server.(*jsonutils.JSONDict))
			if err != nil {
				return err
			}
			dataList = append(dataList, metric)
			for _, resp := range rtn.Value() {
				if !resp.(*jsonutils.JSONDict).Contains("points") {
					continue
				}
				serverMetric, err := self.collectMetricFromThisServer(server, resp, influxDbSpecs)
				if err != nil {
					return err
				}
				dataList = append(dataList, serverMetric...)
			}
		}
	}
	return common.SendMetrics(self.Session, dataList, self.Args.Debug, "")
}

func (self *SGoogleCloudReport) collectMetricFromThisServer(server jsonutils.JSONObject,
	rtnMetric jsonutils.JSONObject, influxDbSpecs []string) ([]influxdb.SMetricData, error) {
	datas := make([]influxdb.SMetricData, 0)
	points_, _ := rtnMetric.Get("points")
	for _, point := range points_.(*jsonutils.JSONArray).Value() {
		metric, err := common.JsonToMetric(server.(*jsonutils.JSONDict), "", common.ServerTags, make([]string, 0))
		if err != nil {
			return nil, err
		}
		if time, err := point.Get("interval", "startTime"); err == nil {
			influxDbSpec := influxDbSpecs[2]

			measurement := common.SubstringBefore(influxDbSpec, ".")
			metric.Name = measurement

			timestamp, err := timeutils.ParseTimeStr(time.(*jsonutils.JSONString).Value())
			if err != nil {
				log.Errorln(err)
				continue
			}
			metric.Timestamp = timestamp

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

			cpu_cout, err := server.Get("vcpu_count")
			if err == nil {
				metric.Metrics = append(metric.Metrics, influxdb.SKeyValue{
					Key:   "cpu_count",
					Value: strconv.FormatInt(cpu_cout.(*jsonutils.JSONInt).Value(), 10),
				})
			}

			valueTypeObj, _ := rtnMetric.Get("valueType")
			valueType := strings.ToLower(valueTypeObj.(*jsonutils.JSONString).Value())
			value := getMetricValue(point, valueType, influxDbSpecs[1])
			metric.Metrics = append(metric.Metrics, influxdb.SKeyValue{
				Key:   pairsKey,
				Value: value,
			})
			self.AddMetricTag(&metric, common.AddTags)
			datas = append(datas, metric)
		}
	}
	return datas, nil
}

func getMetricValue(point jsonutils.JSONObject, valueType string,
	influxDbSpec string) string {
	value, _ := point.Get("value", valueType+"Value")

	switch valueType {
	case "int64":
		if val, ok := value.(*jsonutils.JSONString); ok {
			fieldValue, _ := strconv.ParseInt(val.Value(), 10, 64)
			if influxDbSpec == UNIT_MEM {
				fieldValue = fieldValue * 8 / PERIOD
			}
			return strconv.FormatInt(fieldValue, 10)
		}
		fieldValue := value.(*jsonutils.JSONInt).Value()
		if influxDbSpec == UNIT_MEM {
			fieldValue = fieldValue * 8 / PERIOD
		}
		return strconv.FormatInt(fieldValue, 10)
	case "double":
		fieldValue := value.(*jsonutils.JSONFloat).Value()
		if influxDbSpec == UNIT_MEM {
			fieldValue = fieldValue * 8 / PERIOD
		}
		return strconv.FormatFloat(fieldValue, 'f', 3, 64)
	}
	return ""
}
