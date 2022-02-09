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

package apsaramon

import (
	"strconv"
	"strings"
	"sync"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudmon/collectors/common"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	iden_modules "yunion.io/x/onecloud/pkg/mcclient/modules/identity"
	"yunion.io/x/onecloud/pkg/multicloud/apsara"
	"yunion.io/x/onecloud/pkg/util/influxdb"
)

type SApsaraMetric struct {
	Timestamp  int64
	UserId     string
	InstanceId string
	Maximum    float64
	Minimum    float64
	Average    float64
	NodeId     string
}

func (self *SApsaraCloudReport) collectRegionMetricOfHost(region cloudprovider.ICloudRegion, servers []jsonutils.JSONObject) error {
	dataList := make([]influxdb.SMetricData, 0)
	aliReg := region.(*apsara.SRegion)
	since, until, err := common.TimeRangeFromArgs(self.Args)
	if err != nil {
		return errors.Wrapf(err, "common.TimeRangeFromArgs")
	}

	instances := []api.ServerDetails{}
	instanceMaps := map[string]api.ServerDetails{}
	jsonutils.Update(&instances, servers)
	for i := range instances {
		if len(instances[i].ExternalId) == 0 {
			continue
		}
		instanceMaps[instances[i].ExternalId] = instances[i]
		metric, err := common.FillVMCapacity(jsonutils.Marshal(instances[i]).(*jsonutils.JSONDict))
		if err != nil {
			return errors.Wrapf(err, "common.FillVMCapacity")
		}
		dataList = append(dataList, metric)
	}

	err = common.SendMetrics(self.Session, dataList, self.Args.Debug, "")
	if err != nil {
		log.Errorf("send server base metric error: %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(len(apsaraMetricSpecs))
	for _metricName, _influxDbSpecs := range apsaraMetricSpecs {
		go func(metricName string, influxDbSpecs []string) {
			defer wg.Done()
			dataList = []influxdb.SMetricData{}
			metricArray, err := aliReg.FetchMetricData(metricName, "acs_ecs_dashboard", since, until)
			if err != nil {
				log.Errorln(err)
				return
			}

			metrics := []SApsaraMetric{}
			jsonutils.Update(&metrics, metricArray)

			for _, rtnMetric := range metrics {
				server, ok := instanceMaps[rtnMetric.InstanceId]
				if ok {
					serverMetric, err := self.collectMetricFromThisServer(jsonutils.Marshal(server), rtnMetric, influxDbSpecs)
					if err != nil {
						log.Errorf("collect %s error: %v", metricName, err)
						continue
					}

					project, err := self.GetResourceById(server.ProjectId, &iden_modules.Projects)
					if err != nil {
						log.Errorf("server: %s getProject: %s err: %v", server.Name, server.ProjectId, err)
						continue
					}
					metaMap, _ := project.GetMap("metadata")
					if len(metaMap) > 0 {
						for key, valObj := range metaMap {
							if strings.Contains(key, "user:") {
								val, _ := valObj.GetString()
								serverMetric.Tags = append(serverMetric.Tags, influxdb.SKeyValue{
									Key:   key,
									Value: val,
								})
							}
						}
					}
					dataList = append(dataList, serverMetric)
				}
			}
			log.Infof("SendMetrics %s length: %d", metricName, len(dataList))
			err = common.SendMetrics(self.Session, dataList, self.Args.Debug, "")
			if err != nil {
				log.Errorf("SendMetrics %s length: %d error: %v", metricName, len(dataList), err)
			}

		}(_metricName, _influxDbSpecs)
	}
	wg.Wait()
	return nil
}

func (self *SApsaraCloudReport) collectRegionMetricOfRedis(region cloudprovider.ICloudRegion,
	servers []jsonutils.JSONObject) error {
	dataList := make([]influxdb.SMetricData, 0)
	aliReg := region.(*apsara.SRegion)
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
			rtnArray, err := aliReg.FetchMetricData(pre+metricName, "acs_kvstore", since, until)
			if err != nil {
				log.Errorln(err)
				continue
			}

			metrics := []SApsaraMetric{}
			jsonutils.Update(&metrics, rtnArray)

			for _, rtnMetric := range metrics {
				for _, server := range servers {
					external_id, _ := server.GetString("external_id")
					if rtnMetric.InstanceId == external_id {
						serverMetric, err := self.collectMetricFromThisServer(server, rtnMetric, influxDbSpecs)
						serverMetric.Tags = append(serverMetric.Tags, influxdb.SKeyValue{Key: "node_id", Value: rtnMetric.NodeId})
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

func (self *SApsaraCloudReport) collectRegionMetricOfRds(region cloudprovider.ICloudRegion, servers []jsonutils.JSONObject) error {
	dataList := make([]influxdb.SMetricData, 0)
	aliReg := region.(*apsara.SRegion)
	since, until, err := common.TimeRangeFromArgs(self.Args)
	if err != nil {
		return err
	}
	for metricName, influxDbSpecs := range aliRdsMetricSpecs {
		rtnArray, err := aliReg.FetchMetricData(metricName, "acs_rds_dashboard", since, until)
		if err != nil {
			log.Errorln(err)
			continue
		}
		metrics := []SApsaraMetric{}
		jsonutils.Update(&metrics, rtnArray)

		for _, rtnMetric := range metrics {
			for _, server := range servers {
				external_id, _ := server.GetString("external_id")
				if rtnMetric.InstanceId == external_id {
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
	return common.SendMetrics(self.Session, dataList, self.Args.Debug, "")
}

func (self *SApsaraCloudReport) collectRegionMetricOfOss(region cloudprovider.ICloudRegion, servers []jsonutils.JSONObject) error {
	dataList := make([]influxdb.SMetricData, 0)
	aliReg := region.(*apsara.SRegion)
	since, until, err := common.TimeRangeFromArgs(self.Args)
	if err != nil {
		return err
	}
	for metricName, influxDbSpecs := range aliOSSMetricSpecs {
		rtnArray, err := aliReg.FetchMetricData(metricName, "acs_oss", since, until)
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

func (self *SApsaraCloudReport) collectRegionMetricOfElb(region cloudprovider.ICloudRegion, servers []jsonutils.JSONObject) error {
	dataList := make([]influxdb.SMetricData, 0)
	aliReg := region.(*apsara.SRegion)
	since, until, err := common.TimeRangeFromArgs(self.Args)
	if err != nil {
		return err
	}
	for metricName, influxDbSpecs := range aliElbMetricSpecs {
		rtnArray, err := aliReg.FetchMetricData(metricName, "acs_slb_dashboard", since, until)
		if err != nil {
			log.Errorln(err)
			continue
		}

		metrics := []SApsaraMetric{}
		jsonutils.Update(&metrics, rtnArray)

		for _, rtnMetric := range metrics {
			for _, server := range servers {
				external_id, _ := server.GetString("external_id")
				if rtnMetric.InstanceId == external_id {
					serverMetric, err := self.collectMetricFromThisServer(server, rtnMetric, influxDbSpecs)
					if err != nil {
						return err
					}
					dataList = append(dataList, serverMetric)
				}
			}
		}
		log.Infof("SendMetrics %s length: %d", metricName, len(dataList))
	}
	return common.SendMetrics(self.Session, dataList, self.Args.Debug, "")
}

func (self *SApsaraCloudReport) collectMetricFromThisServer(server jsonutils.JSONObject, rtnMetric SApsaraMetric, influxDbSpecs []string) (influxdb.SMetricData, error) {
	metric, err := self.NewMetricFromJson(server)
	if err != nil {
		return influxdb.SMetricData{}, err
	}
	metric.Timestamp = time.Unix(rtnMetric.Timestamp/1000, 0)
	fieldValue := rtnMetric.Average
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

func (self *SApsaraCloudReport) collectOssMetricFromThisServer(server jsonutils.JSONObject, rtnMetric jsonutils.JSONObject,
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
