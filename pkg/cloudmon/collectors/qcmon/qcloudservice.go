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

package qcmon

import (
	"strconv"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudmon/collectors/common"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud/qcloud"
	"yunion.io/x/onecloud/pkg/util/influxdb"
)

func (self *SQCloudReport) CollectRegionMetric(region cloudprovider.ICloudRegion,
	servers []jsonutils.JSONObject) error {
	var err error
	switch self.Operator {
	case string(common.SERVER):
		err = self.collectRegionMetricOfHost(region, servers)
	}
	return err
}

func (self *SQCloudReport) collectRegionMetricOfHost(region cloudprovider.ICloudRegion, servers []jsonutils.JSONObject) error {
	dataList := make([]influxdb.SMetricData, 0)
	tecentReg := region.(*qcloud.SRegion)
	since, until, err := common.TimeRangeFromArgs(self.Args)
	if err != nil {
		return err
	}
	dimensions := make([]qcloud.SQcMetricDimension, 0)
	for _, server := range servers {
		external_id, _ := server.GetString("external_id")
		dimensions = append(dimensions, qcloud.SQcMetricDimension{
			Name:  "InstanceId",
			Value: external_id,
		})
	}
	//dimensions := []qcloud.SQcMetricDimension{qcloud.SQcMetricDimension{Name: "InstanceId", Value: external_id}}
	for metricName, influxDbSpecs := range tecentMetricSpecs {
		for index, tmp := 0, 0; index < len(dimensions); index += 10 {
			tmp = index + 10
			if tmp > len(dimensions) {
				tmp = len(dimensions)
			}
			rtnArray, err := tecentReg.GetMonitorData(metricName, "QCE/CVM", since, until,
				dimensions[index:tmp])
			if err != nil {
				log.Errorln(err)
				continue
			}
			for _, rtnMetric := range rtnArray {
				for _, server := range servers {
					external_id, _ := server.GetString("external_id")
					if external_id == rtnMetric.Dimensions[0].Value {
						metric, err := common.FillVMCapacity(server.(*jsonutils.JSONDict))
						if err != nil {
							return err
						}
						dataList = append(dataList, metric)
						if len(rtnMetric.Timestamps) == 0 {
							break
						}
						serverMetric, err := self.collectMetricFromThisServer(server, rtnMetric, influxDbSpecs)
						if err != nil {
							return err
						}
						dataList = append(dataList, serverMetric...)
					}
				}
			}
		}
	}
	return common.SendMetrics(self.Session, dataList, self.Args.Debug, "")
}

func (self *SQCloudReport) collectRegionMetricOfRedis(region cloudprovider.ICloudRegion, servers []jsonutils.JSONObject) error {
	dataList := make([]influxdb.SMetricData, 0)
	tecentReg := region.(*qcloud.SRegion)
	since, until, err := common.TimeRangeFromArgs(self.Args)
	if err != nil {
		return err
	}
	dimensions := make([]qcloud.SQcMetricDimension, 0)
	for _, server := range servers {
		external_id, _ := server.GetString("external_id")
		dimensions = append(dimensions, qcloud.SQcMetricDimension{
			Name:  "instanceid",
			Value: external_id,
		})
	}
	//dimensions := []qcloud.SQcMetricDimension{qcloud.SQcMetricDimension{Name: "InstanceId", Value: external_id}}
	for metricName, influxDbSpecs := range tecentRedisMetricSpecs {
		for index, tmp := 0, 0; index < len(dimensions); index += 10 {
			tmp = index + 10
			if tmp > len(dimensions) {
				tmp = len(dimensions)
			}
			rtnArray, err := tecentReg.GetMonitorData(metricName, "QCE/REDIS", since, until,
				dimensions[index:tmp])
			if err != nil {
				log.Errorln(err)
				continue
			}
			for _, rtnMetric := range rtnArray {
				for _, server := range servers {
					external_id, _ := server.GetString("external_id")
					if external_id == rtnMetric.Dimensions[0].Value {
						if len(rtnMetric.Timestamps) == 0 {
							break
						}
						serverMetric, err := self.collectMetricFromThisServer(server, rtnMetric, influxDbSpecs)
						if err != nil {
							serverName, _ := server.Get("name")
							log.Errorf("redis:%s,collectMetricFromThisServer err:%v", serverName, err)
							continue
						}
						dataList = append(dataList, serverMetric...)
					}
				}
			}
		}
	}
	return common.SendMetrics(self.Session, dataList, self.Args.Debug, "")
}

func (self *SQCloudReport) collectRegionMetricOfRds(region cloudprovider.ICloudRegion, servers []jsonutils.JSONObject) error {
	dataList := make([]influxdb.SMetricData, 0)
	tecentReg := region.(*qcloud.SRegion)
	since, until, err := common.TimeRangeFromArgs(self.Args)
	if err != nil {
		return err
	}
	dimensions := make([]qcloud.SQcMetricDimension, 0)
	for _, server := range servers {
		external_id, _ := server.GetString("external_id")
		dimensions = append(dimensions, qcloud.SQcMetricDimension{
			Name:  "InstanceId",
			Value: external_id,
		})
	}
	//dimensions := []qcloud.SQcMetricDimension{qcloud.SQcMetricDimension{Name: "InstanceId", Value: external_id}}
	for metricName, influxDbSpecs := range tecentRdsMetricSpecs {
		for index, tmp := 0, 0; index < len(dimensions); index += 10 {
			tmp = index + 10
			if tmp > len(dimensions) {
				tmp = len(dimensions)
			}
			rtnArray, err := tecentReg.GetMonitorData(metricName, "QCE/CDB", since, until,
				dimensions[index:tmp])
			if err != nil {
				log.Errorln(err)
				continue
			}
			for _, rtnMetric := range rtnArray {
				for _, server := range servers {
					external_id, _ := server.GetString("external_id")
					if external_id == rtnMetric.Dimensions[0].Value {
						if len(rtnMetric.Timestamps) == 0 {
							break
						}
						serverMetric, err := self.collectMetricFromThisServer(server, rtnMetric, influxDbSpecs)
						if err != nil {
							serverName, _ := server.Get("name")
							log.Errorf("redis:%s,collectMetricFromThisServer err:%v", serverName, err)
							continue
						}
						dataList = append(dataList, serverMetric...)
					}
				}
			}
		}
	}
	return common.SendMetrics(self.Session, dataList, self.Args.Debug, "")
}

func (self *SQCloudReport) collectMetricFromThisServer(server jsonutils.JSONObject, rtnMetric qcloud.SDataPoint, influxDbSpecs []string) ([]influxdb.SMetricData, error) {
	datas := make([]influxdb.SMetricData, 0)
	for index, timestamp := range rtnMetric.Timestamps {
		metric, err := self.NewMetricFromJson(server)
		if err != nil {
			return nil, err
		}
		//根据条件拼装metric的tag和metirc信息
		influxDbSpec := influxDbSpecs[2]
		measurement := common.SubstringBefore(influxDbSpec, ".")
		metric.Name = measurement
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
		metric.Timestamp = time.Unix(int64(timestamp), 0)
		fieldValue := rtnMetric.Values[index]
		if influxDbSpecs[1] == UNIT_MBPS {
			fieldValue = fieldValue * 1000 * 1000
		}
		metric.Metrics = append(metric.Metrics, influxdb.SKeyValue{
			Key:   pairsKey,
			Value: strconv.FormatFloat(fieldValue, 'E', -1, 64),
		})
		self.AddMetricTag(&metric, common.OtherVmTags)
		datas = append(datas, metric)
	}
	return datas, nil
}
