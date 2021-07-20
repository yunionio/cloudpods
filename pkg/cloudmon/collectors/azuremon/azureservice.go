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

package azuremon

import (
	"strconv"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudmon/collectors/common"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud/azure"
	"yunion.io/x/onecloud/pkg/util/influxdb"
)

func (self *SAzureCloudReport) CollectRegionMetric(region cloudprovider.ICloudRegion,
	servers []jsonutils.JSONObject) error {
	return self.collectRegionMetricOfHost(region, servers)
}

func (self *SAzureCloudReport) collectRegionMetricOfHost(region cloudprovider.ICloudRegion,
	servers []jsonutils.JSONObject) error {
	dataList := make([]influxdb.SMetricData, 0)
	azureReg := region.(*azure.SRegion)
	since, until, err := common.TimeRangeFromArgs(self.Args)
	if err != nil {
		return err
	}
	for _, server := range servers {
		external_id, err := server.GetString("external_id")
		if err != nil {
			continue
		}
		metricNameArr := make([]string, 0)
		for metricName, _ := range azureMetricSpecs {
			metricNameArr = append(metricNameArr, metricName)
		}
		metricNames := strings.Join(metricNameArr, ",")
		rtnMetrics, err := azureReg.GetMonitorData(metricNames, "Microsoft.Compute/virtualMachines", external_id, since, until)
		if err != nil {
			log.Errorln(err)
			continue
		}
		if rtnMetrics == nil || rtnMetrics.Value == nil {
			continue
		}
		for metricName, influxDbSpecs := range azureMetricSpecs {
			for _, value := range *rtnMetrics.Value {
				if value.Name.LocalizedValue != nil {
					if metricName == *(value.Name.LocalizedValue) {
						metric, err := common.FillVMCapacity(server.(*jsonutils.JSONDict))
						if err != nil {
							return err
						}
						dataList = append(dataList, metric)
						if value.Timeseries != nil {
							for _, timeserie := range *value.Timeseries {
								serverMetric, err := self.collectMetricFromThisServer(server, timeserie, influxDbSpecs)
								if err != nil {
									return err
								}
								dataList = append(dataList, serverMetric...)
							}
						}
					}
				}
			}
		}
		err = common.SendMetrics(self.Session, dataList, self.Args.Debug, "")
		if err != nil {
			log.Errorln(err)
		}
		dataList = dataList[:0]
	}
	return nil
}

func (self *SAzureCloudReport) collectMetricFromThisServer(server jsonutils.JSONObject,
	rtnMetric azure.TimeSeriesElement, influxDbSpecs []string) ([]influxdb.SMetricData, error) {
	datas := make([]influxdb.SMetricData, 0)
	for _, data := range *rtnMetric.Data {
		metric, err := common.JsonToMetric(server.(*jsonutils.JSONDict), "", common.ServerTags, make([]string, 0))
		if err != nil {
			return nil, err
		}
		if data.TimeStamp != nil {
			metric.Timestamp = *data.TimeStamp
			if data.Average != nil {
				//根据条件拼装metric的tag和metirc信息
				fieldValue := *data.Average
				influxDbSpec := influxDbSpecs[2]
				measurement := common.SubstringBefore(influxDbSpec, ".")
				metric.Name = measurement
				var pairsKey string
				if strings.Contains(influxDbSpec, ",") {
					pairsKey = common.SubstringBetween(influxDbSpec, ".", ",")
				} else {
					pairsKey = common.SubstringAfter(influxDbSpec, ".")
				}
				if influxDbSpecs[1] == UNIT_MEM {
					fieldValue = fieldValue * 8 / PERIOD
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
				metric.Metrics = append(metric.Metrics, influxdb.SKeyValue{
					Key:   pairsKey,
					Value: strconv.FormatFloat(fieldValue, 'E', -1, 64),
				})
				self.AddMetricTag(&metric, common.OtherVmTags)
				datas = append(datas, metric)
			}
		}
	}

	return datas, nil
}
