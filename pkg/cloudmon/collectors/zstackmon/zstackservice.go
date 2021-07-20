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

package zstackmon

import (
	"strconv"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudmon/collectors/common"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud/zstack"
	"yunion.io/x/onecloud/pkg/util/influxdb"
)

func (self *SZStackCloudReport) CollectRegionMetric(region cloudprovider.ICloudRegion,
	servers []jsonutils.JSONObject) error {
	var err error
	switch self.Operator {
	case string(common.SERVER):
		err = self.collectRegionMetricOfServer(region, servers)
	case string(common.HOST):
		err = self.collectRegionMetricOfHost(region, servers)
	}
	return err
}

func (self *SZStackCloudReport) collectRegionMetricOfServer(region cloudprovider.ICloudRegion,
	servers []jsonutils.JSONObject) error {
	dataList := make([]influxdb.SMetricData, 0)
	zstackReg := region.(*zstack.SRegion)
	since, until, err := common.TimeRangeFromArgs(self.Args)
	if err != nil {
		return err
	}
	for metricName, influxDbSpecs := range zstackMetricSpecs {
		rtn, err := zstackReg.GetMonitorData(metricName, NAMESPACE_VM, since, until)
		if err != nil {
			log.Errorln(err)
			continue
		}
		if len(rtn.DataPoints) > 0 {
			for _, dataPoint := range rtn.DataPoints {
				for _, server := range servers {
					external_id, _ := server.GetString("external_id")
					if dataPoint.Labels.VMUuid == external_id {
						metric, err := common.FillVMCapacity(server.(*jsonutils.JSONDict))
						if err != nil {
							return err
						}
						dataList = append(dataList, metric)
						serverMetric, err := self.collectMetricFromThisServer(server, common.TYPE_VIRTUALMACHINE, dataPoint, influxDbSpecs)
						if err != nil {
							return err
						}
						dataList = append(dataList, serverMetric)
					}
				}
			}
		}
	}
	return common.SendMetrics(self.Session, dataList, self.Args.Debug, "")
}

func (self *SZStackCloudReport) collectRegionMetricOfHost(region cloudprovider.ICloudRegion,
	hosts []jsonutils.JSONObject) error {
	dataList := make([]influxdb.SMetricData, 0)
	zstackReg := region.(*zstack.SRegion)
	since, until, err := common.TimeRangeFromArgs(self.Args)
	if err != nil {
		return err
	}
	for metricName, influxDbSpecs := range zstackMetricSpecs {
		rtn, err := zstackReg.GetMonitorData(metricName, NAMESPACE_HOST, since, until)
		if err != nil {
			return err
		}
		if len(rtn.DataPoints) > 0 {
			for _, dataPoint := range rtn.DataPoints {
				for _, host := range hosts {
					external_id, _ := host.GetString("external_id")
					if dataPoint.Labels.HostUuid == external_id {
						serverMetric, err := self.collectMetricFromThisServer(host, common.TYPE_HOSTSYSTEM, dataPoint, influxDbSpecs)
						if err != nil {
							return err
						}
						dataList = append(dataList, serverMetric)
					}
				}
			}
		}
	}
	return common.SendMetrics(self.Session, dataList, self.Args.Debug, "")
}

func (self *SZStackCloudReport) collectMetricFromThisServer(server jsonutils.JSONObject,
	monType string, dataPoint zstack.DataPoint, influxDbSpecs []string) (influxdb.SMetricData, error) {
	metric := influxdb.SMetricData{}
	if monType == common.TYPE_HOSTSYSTEM {
		metric, _ = common.JsonToMetric(server.(*jsonutils.JSONDict), "", common.HostTags, make([]string, 0))
	} else {
		metric, _ = common.JsonToMetric(server.(*jsonutils.JSONDict), "", common.ServerTags, make([]string, 0))
	}
	fieldValue := dataPoint.Value
	metric.Timestamp = time.Unix(dataPoint.TimeStemp, 0)
	//根据条件拼装metric的tag和metirc信息
	metric.Name = common.GetMeasurement(monType, influxDbSpecs[2])
	var pairsKey string
	if strings.Contains(influxDbSpecs[2], ",") {
		pairsKey = common.SubstringBetween(influxDbSpecs[2], ".", ",")
	} else {
		pairsKey = common.SubstringAfter(influxDbSpecs[2], ".")
	}
	if influxDbSpecs[1] == UNIT_MEM {
		fieldValue = fieldValue * 8
	}
	tag := common.SubstringAfter(influxDbSpecs[2], ",")
	if tag != "" && strings.Contains(influxDbSpecs[2], "=") {
		metric.Tags = append(metric.Tags, influxdb.SKeyValue{
			Key:   common.SubstringBefore(tag, "="),
			Value: common.SubstringAfter(tag, "="),
		})
	}
	metric.Metrics = append(metric.Metrics, influxdb.SKeyValue{
		Key:   pairsKey,
		Value: strconv.FormatFloat(fieldValue, 'E', -1, 64),
	})
	if monType == common.TYPE_HOSTSYSTEM {
		self.AddMetricTag(&metric, common.OtherHostTag)
	} else {
		self.AddMetricTag(&metric, common.OtherVmTags)
	}
	return metric, nil
}
