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

package alimon

import (
	"strconv"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudmon/collectors/common"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud/aliyun"
	"yunion.io/x/onecloud/pkg/util/influxdb"
)

func (self *SAliCloudReport) collectRegionMetricOfHost(region cloudprovider.ICloudRegion, servers []jsonutils.JSONObject) error {
	dataList := make([]influxdb.SMetricData, 0)
	aliReg := region.(*aliyun.SRegion)
	since, until, err := common.TimeRangeFromArgs(self.Args)
	if err != nil {
		return err
	}
	for metricName, influxDbSpecs := range aliMetricSpecs {
		rtnArray, _, err := aliReg.DescribeMetricList(metricName, "acs_ecs_dashboard", since, until, "")
		if err != nil {
			log.Errorln(err)
			continue
		}
		if len(rtnArray) > 0 {
			for _, rtnMetric := range rtnArray {
				for _, server := range servers {
					external_id, _ := server.GetString("external_id")
					if instanceId, _ := rtnMetric.GetString("instanceId"); instanceId == external_id {
						metric, err := common.FillVMCapacity(server.(*jsonutils.JSONDict))
						if err != nil {
							return err
						}
						dataList = append(dataList, metric)
						serverMetric, err := self.collectMetricFromThisServer(server, rtnMetric, influxDbSpecs)
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

func (self *SAliCloudReport) collectRegionMetricOfRedis(region cloudprovider.ICloudRegion,
	servers []jsonutils.JSONObject) error {
	dataList := make([]influxdb.SMetricData, 0)
	aliReg := region.(*aliyun.SRegion)
	since, until, err := common.TimeRangeFromArgs(self.Args)
	if err != nil {
		return err
	}
	redisDeployType := make(map[string]string)
	for _, server := range servers {
		local_category, err := server.GetString("local_category")
		if err != nil {
			return err
		}
		switch local_category {
		case "single":
			redisDeployType["single"] = "Standard"
		case "master":
			redisDeployType["master"] = "Standard"
		case "cluster":
			redisDeployType["cluster"] = "Sharding"
		case "rwsplit":
			redisDeployType["rwsplit"] = "Splitrw"
		}

	}
	for metricName, influxDbSpecs := range aliRedisMetricSpecs {
		for _, pre := range redisDeployType {
			rtnArray, _, err := aliReg.DescribeMetricList(pre+metricName, "acs_kvstore", since, until, "")
			if err != nil {
				log.Errorln(err)
				continue
			}
			if len(rtnArray) > 0 {
				for _, rtnMetric := range rtnArray {
					for _, server := range servers {
						external_id, _ := server.GetString("external_id")
						if instanceId, _ := rtnMetric.GetString("instanceId"); instanceId == external_id {
							serverMetric, err := self.collectMetricFromThisServer(server, rtnMetric, influxDbSpecs)
							node_id, _ := rtnMetric.GetString("nodeId")
							serverMetric.Tags = append(serverMetric.Tags, influxdb.SKeyValue{Key: "node_id", Value: node_id})
							if err != nil {
								return err
							}
							dataList = append(dataList, serverMetric)
						}
					}
				}
			}
		}
	}
	return common.SendMetrics(self.Session, dataList, self.Args.Debug, "")
}

func (self *SAliCloudReport) collectRegionMetricOfRds(region cloudprovider.ICloudRegion, servers []jsonutils.JSONObject) error {
	dataList := make([]influxdb.SMetricData, 0)
	aliReg := region.(*aliyun.SRegion)
	since, until, err := common.TimeRangeFromArgs(self.Args)
	if err != nil {
		return err
	}
	for metricName, influxDbSpecs := range aliRdsMetricSpecs {
		rtnArray, _, err := aliReg.DescribeMetricList(metricName, "acs_rds_dashboard", since, until, "")
		if err != nil {
			log.Errorln(err)
			continue
		}
		if len(rtnArray) > 0 {
			for _, rtnMetric := range rtnArray {
				for _, server := range servers {
					external_id, _ := server.GetString("external_id")
					if instanceId, _ := rtnMetric.GetString("instanceId"); instanceId == external_id {
						metric, err := common.FillVMCapacity(server.(*jsonutils.JSONDict))
						if err != nil {
							return err
						}
						dataList = append(dataList, metric)
						serverMetric, err := self.collectMetricFromThisServer(server, rtnMetric, influxDbSpecs)
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

func (self *SAliCloudReport) collectRegionMetricOfOss(region cloudprovider.ICloudRegion, servers []jsonutils.JSONObject) error {
	dataList := make([]influxdb.SMetricData, 0)
	aliReg := region.(*aliyun.SRegion)
	since, until, err := common.TimeRangeFromArgs(self.Args)
	if err != nil {
		return err
	}
	for metricName, influxDbSpecs := range aliOSSMetricSpecs {
		rtnArray, _, err := aliReg.DescribeMetricList(metricName, "acs_oss", since, until, "")
		if err != nil {
			log.Errorln(err)
			continue
		}
		if len(rtnArray) > 0 {
			for _, rtnMetric := range rtnArray {
				for _, server := range servers {
					name, _ := server.GetString("name")
					if bucketName, _ := rtnMetric.GetString("BucketName"); bucketName == name {
						serverMetric, err := self.collectOssMetricFromThisServer(server, rtnMetric, metricName,
							influxDbSpecs)
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

func (self *SAliCloudReport) collectRegionMetricOfElb(region cloudprovider.ICloudRegion, servers []jsonutils.JSONObject) error {
	dataList := make([]influxdb.SMetricData, 0)
	aliReg := region.(*aliyun.SRegion)
	since, until, err := common.TimeRangeFromArgs(self.Args)
	if err != nil {
		return err
	}
	for metricName, influxDbSpecs := range aliElbMetricSpecs {
		rtnArray, _, err := aliReg.DescribeMetricList(metricName, "acs_slb_dashboard", since, until, "")
		if err != nil {
			log.Errorln(err)
			continue
		}
		if len(rtnArray) > 0 {
			for _, rtnMetric := range rtnArray {
				for _, server := range servers {
					external_id, _ := server.GetString("external_id")
					if instanceId, _ := rtnMetric.GetString("instanceId"); instanceId == external_id {
						serverMetric, err := self.collectMetricFromThisServer(server, rtnMetric, influxDbSpecs)
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

func (self *SAliCloudReport) collectMetricFromThisServer(server jsonutils.JSONObject, rtnMetric jsonutils.JSONObject,
	influxDbSpecs []string) (influxdb.SMetricData, error) {
	metric, err := self.NewMetricFromJson(server)
	//metric, err := common.JsonToMetric(server.(*jsonutils.JSONDict), "", common.ServerTags, make([]string, 0))
	if err != nil {
		return influxdb.SMetricData{}, err
	}
	timestamp, _ := rtnMetric.Get("timestamp")
	metric.Timestamp = time.Unix(timestamp.(*jsonutils.JSONInt).Value()/1000, 0)
	fieldValue, err := rtnMetric.Float(UNIT_AVERAGE)
	if err != nil {
		return influxdb.SMetricData{}, err
	}
	//根据条件拼装metric的tag和metirc信息
	influxDbSpec := influxDbSpecs[2]
	measurement := common.SubstringBefore(influxDbSpec, ".")
	var pairsKey string
	if strings.Contains(influxDbSpec, ",") {
		pairsKey = common.SubstringBetween(influxDbSpec, ".", ",")
	} else {
		pairsKey = common.SubstringAfter(influxDbSpec, ".")
	}
	if influxDbSpecs[1] == UNIT_BYTEPS && strings.Contains(pairsKey, UNIT_BPS) {
		fieldValue = fieldValue * 8
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
	metric.Name = measurement
	return metric, nil
}

func (self *SAliCloudReport) collectOssMetricFromThisServer(server jsonutils.JSONObject, rtnMetric jsonutils.JSONObject,
	metricName string, influxDbSpecs []string) (influxdb.SMetricData, error) {
	metric, err := self.NewMetricFromJson(server)
	//metric, err := common.JsonToMetric(server.(*jsonutils.JSONDict), "", common.ServerTags, make([]string, 0))
	if err != nil {
		return influxdb.SMetricData{}, err
	}
	timestamp, _ := rtnMetric.Get("timestamp")
	metric.Timestamp = time.Unix(timestamp.(*jsonutils.JSONInt).Value()/1000, 0)
	fieldValue, err := rtnMetric.Float(metricName)
	if err != nil {
		return influxdb.SMetricData{}, err
	}
	//根据条件拼装metric的tag和metirc信息
	influxDbSpec := influxDbSpecs[2]
	measurement := common.SubstringBefore(influxDbSpec, ".")
	var pairsKey string
	if strings.Contains(influxDbSpec, ",") {
		pairsKey = common.SubstringBetween(influxDbSpec, ".", ",")
	} else {
		pairsKey = common.SubstringAfter(influxDbSpec, ".")
	}
	if influxDbSpecs[1] == UNIT_BYTEPS && strings.Contains(pairsKey, UNIT_BPS) {
		fieldValue = fieldValue * 8
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
	metric.Name = measurement
	return metric, nil
}
