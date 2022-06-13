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

func (self *SAliCloudReport) collectRegionMetricOfResource(region cloudprovider.ICloudRegion, servers []jsonutils.JSONObject) error {
	dataList := make([]influxdb.SMetricData, 0)
	aliReg := region.(*aliyun.SRegion)
	since, until, err := common.TimeRangeFromArgs(self.Args)
	if err != nil {
		return err
	}
	namespace, metrics := self.getMetricSpecs(servers)
	dimensionId := self.getDimensionId()
	dataCnt := 0
	for metricName, influxDbSpecs := range metrics {
		rtnArray, _, err := aliReg.DescribeMetricList(metricName, namespace, since, until, "", nil)
		if err != nil {
			log.Errorln(err)
			continue
		}
		dataCnt = len(dataList)
		if len(rtnArray) > 0 {
			for _, rtnMetric := range rtnArray {
				for _, server := range servers {
					external_id, _ := server.GetString(dimensionId.LocalId)
					if instanceId, _ := rtnMetric.GetString(dimensionId.ExtId); instanceId == external_id {
						if self.Operator == string(common.SERVER) {
							metric, err := common.FillVMCapacity(server.(*jsonutils.JSONDict))
							if err != nil {
								return err
							}
							dataList = append(dataList, metric)
						}
						serverMetric, err := self.collectMetricFromThisServer(server, rtnMetric, influxDbSpecs, metricName)
						if err != nil {
							return err
						}
						dataList = append(dataList, serverMetric)
					}
				}
			}
		}
		log.Debugf("%s %s %s report %d metric", self.SProvider.Name, self.Operator, metricName, len(dataList)-dataCnt)
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
	dataCnt := 0
	for metricName, influxDbSpecs := range aliRedisMetricSpecs {
		for _, pre := range redisDeployType {
			rtnArray, _, err := aliReg.DescribeMetricList(pre+metricName, "acs_kvstore", since, until, "", nil)
			if err != nil {
				log.Errorln(err)
				continue
			}
			dataCnt = len(dataList)
			if len(rtnArray) > 0 {
				for _, rtnMetric := range rtnArray {
					for _, server := range servers {
						external_id, _ := server.GetString("external_id")
						if instanceId, _ := rtnMetric.GetString("instanceId"); instanceId == external_id {
							serverMetric, err := self.collectMetricFromThisServer(server, rtnMetric, influxDbSpecs, "")
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
		log.Debugf("%s %s %s report %d metric", self.SProvider.Name, self.Operator, metricName, len(dataList)-dataCnt)
	}
	return common.SendMetrics(self.Session, dataList, self.Args.Debug, "")
}

func (self *SAliCloudReport) collectMetricFromThisServer(server jsonutils.JSONObject, rtnMetric jsonutils.JSONObject, influxDbSpecs []string, valueKey string) (influxdb.SMetricData, error) {
	metric, err := self.NewMetricFromJson(server)
	//metric, err := common.JsonToMetric(server.(*jsonutils.JSONDict), "", common.ServerTags, make([]string, 0))
	if err != nil {
		return influxdb.SMetricData{}, err
	}
	timestamp, _ := rtnMetric.Get("timestamp")
	metric.Timestamp = time.Unix(timestamp.(*jsonutils.JSONInt).Value()/1000, 0)
	fieldValue, err := self.getDataPointValue(valueKey, influxDbSpecs, rtnMetric)
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
	if influxDbSpecs[1] == common.UNIT_BYTEPS && strings.Contains(pairsKey, common.UNIT_BPS) {
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

func (self *SAliCloudReport) getDataPointValue(valueKey string, influxDbSpecs []string, rtnMetric jsonutils.JSONObject) (float64, error) {
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

func (self *SAliCloudReport) getMetricSpecs(res []jsonutils.JSONObject) (string, map[string][]string) {
	switch common.MonType(self.Operator) {
	case common.SERVER:
		return SERVER_METRIC_NAMESPACE, aliMetricSpecs
	case common.REDIS:
		return REDIS_METRIC_NAMESPACE, aliRedisMetricSpecs
	case common.RDS:
		return RDS_METRIC_NAMESPACE, aliRdsMetricSpecs
	case common.OSS:
		return OSS_METRIC_NAMESPACE, aliOSSMetricSpecs
	case common.ELB:
		return ELB_METRIC_NAMESPACE, aliElbMetricSpecs
	case common.K8S:
		return K8S_METRIC_NAMESPACE, aliK8SClusterMetricSpecs
	default:
		return SERVER_METRIC_NAMESPACE, aliMetricSpecs
	}
}

func (self *SAliCloudReport) getDimensionId() *common.DimensionId {
	switch common.MonType(self.Operator) {
	case common.SERVER:
		return &common.DimensionId{
			LocalId: "external_id",
			ExtId:   "instanceId",
		}
	case common.REDIS:
		return &common.DimensionId{
			LocalId: "external_id",
			ExtId:   "instanceId",
		}
	case common.RDS:
		return &common.DimensionId{
			LocalId: "external_id",
			ExtId:   "instanceId",
		}
	case common.OSS:
		return &common.DimensionId{
			LocalId: "name",
			ExtId:   "BucketName",
		}
	case common.ELB:
		return &common.DimensionId{
			LocalId: "external_id",
			ExtId:   "InstanceId",
		}
	case common.K8S:
		return &common.DimensionId{
			LocalId: "external_cloud_cluster_id",
			ExtId:   "cluster",
		}
	}
	return nil
}

func (self *SAliCloudReport) CollectK8sModuleMetric(region cloudprovider.ICloudRegion, cluster jsonutils.JSONObject,
	helper common.IK8sClusterModuleHelper) error {
	aliReg := region.(*aliyun.SRegion)
	since, until, err := common.TimeRangeFromArgs(self.Args)
	id, _ := cluster.GetString("id")
	resources, err := common.ListK8sClusterModuleResources(helper.MyModuleType(), id, self.Session, nil)
	if err != nil {
		log.Errorf("ListK8sClusterModuleResources err: %v", err)
		return err
	}
	namespace, metricSpecs := helper.MyNamespaceAndMetrics()
	metricNames := make([]string, 0)
	for metricName := range metricSpecs {
		metricNames = append(metricNames, metricName)
	}
	dimensionId := helper.MyResDimensionId()
	dataList := make([]influxdb.SMetricData, 0)
	dataCnt := 0
	for metricName, influxDbSpecs := range metricSpecs {
		rtnArray, _, err := aliReg.DescribeMetricList(metricName, namespace, since, until, "", nil)
		if err != nil {
			log.Errorln(err)
			continue
		}
		dataCnt = len(dataList)
		if len(rtnArray) > 0 {
			for _, rtnMetric := range rtnArray {
				for _, resource := range resources {
					external_id, _ := resource.GetString(dimensionId.LocalId)
					if instanceId, _ := rtnMetric.GetString(dimensionId.ExtId); instanceId == external_id {
						serverMetric, err := self.collectMetricFromThisServer(resource, rtnMetric, influxDbSpecs, metricName)
						if err != nil {
							return err
						}
						dataList = append(dataList, serverMetric)
					}
				}
			}
		}
		log.Debugf("%s %s %s report %d metric", self.SProvider.Name, self.Operator, metricName, len(dataList)-dataCnt)
	}
	err = common.SendMetrics(self.Session, dataList, self.Args.Debug, "")
	if err != nil {
		log.Errorf("K8s metricName: %s SendMetrics err: %#v", "", err)
	}
	return nil
}

func (self *SAliCloudReport) getK8sModuleDimensions(helper common.IK8sClusterModuleHelper, module jsonutils.JSONObject,
	clusterId string) []aliyun.SResourceLabel {
	dimensions := make([]aliyun.SResourceLabel, 0)
	dimensionId := helper.MyResDimensionId()

	localIds := strings.Split(dimensionId.LocalId, ",")
	extIds := strings.Split(dimensionId.ExtId, ",")
	if len(localIds) != len(extIds) {
		return dimensions
	}
	for index, localId := range localIds {
		localIdKey := strings.Split(localId, ".")
		val, _ := module.GetString(localIdKey...)
		dimensions = append(dimensions, aliyun.SResourceLabel{
			Name:  extIds[index],
			Value: val,
		})

	}
	return dimensions
}
