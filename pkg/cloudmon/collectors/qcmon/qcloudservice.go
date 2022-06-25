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
	dimensions := self.getDimensions(servers)
	//dimensions := []qcloud.SQcMetricDimension{qcloud.SQcMetricDimension{Name: "InstanceId", Value: external_id}}
	namespace, metricSpecs := self.getMetricSpecs(servers)
	for metricName, influxDbSpecs := range metricSpecs {
		for index, tmp := 0, 0; index < len(dimensions); index += 10 {
			tmp = index + 10
			if tmp > len(dimensions) {
				tmp = len(dimensions)
			}
			rtnArray, err := tecentReg.GetMonitorData(metricName, namespace, since, until,
				dimensions[index:tmp])
			if err != nil {
				log.Errorln(err)
				continue
			}
			for _, rtnMetric := range rtnArray {
				if len(rtnMetric.Timestamps) == 0 {
					break
				}
				for _, server := range servers {
					name, _ := server.GetString("name")
					external_id, _ := server.GetString("external_id")
					if external_id == rtnMetric.Dimensions[0].Value {
						if self.Operator == string(common.SERVER) {
							metric, err := common.FillVMCapacity(server.(*jsonutils.JSONDict))
							if err != nil {
								log.Errorf("provider: %s FillVMCapacity: %s, err: %#v", self.SProvider.Name, name, err)
							} else {
								dataList = append(dataList, metric)
							}
						}
						if len(rtnMetric.Timestamps) == 0 {
							break
						}
						serverMetric, err := self.collectMetricFromThisServer(server, rtnMetric, influxDbSpecs)
						if err != nil {
							log.Errorf("provider: %s collectMetricFromThisServer: %s, err: %#v", self.SProvider.Name, name, err)
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

func (self *SQCloudReport) collectRegionMetricOfK8S(region cloudprovider.ICloudRegion, servers []jsonutils.JSONObject) error {
	tecentReg := region.(*qcloud.SRegion)
	since, until, err := common.TimeRangeFromArgs(self.Args)
	if err != nil {
		return err
	}
	namespace, metricSpecs := self.getMetricSpecs(servers)
	metricNames := make([]string, 0)
	for metricName, _ := range metricSpecs {
		metricNames = append(metricNames, metricName)
	}
	for i, _ := range servers {
		dataList := make([]influxdb.SMetricData, 0)
		server := servers[i]
		name, _ := server.GetString("name")
		dimensionId := self.getDimensionId()
		id, _ := server.GetString(dimensionId.LocalId)
		dimensions := []qcloud.SQcMetricDimension{
			qcloud.SQcMetricDimension{
				Name:  dimensionId.ExtId,
				Value: id,
			},
		}
		rtnArray, err := tecentReg.GetK8sMonitorData(metricNames, namespace, since, until, dimensions)
		if err != nil {
			log.Errorln(err)
			continue
		}
		if len(rtnArray) == 0 {
			continue
		}
		for _, data := range rtnArray {
			for metricName, influxDbSpecs := range metricSpecs {
				if data.MetricName == metricName {
					serverMetric, err := self.collectMetricFromThisK8s(server, data, influxDbSpecs)
					if err != nil {
						log.Errorf("provider: %s collectK8s:%s metric err: %#v", self.SProvider.Name, name, err)
						continue
					}
					dataList = append(dataList, serverMetric...)
				}
			}
		}
		err = common.SendMetrics(self.Session, dataList, self.Args.Debug, "")
		if err != nil {
			log.Errorf("K8s: %s SendMetrics err: %#v", name, err)
			continue
		}
	}
	return nil
}

func (self *SQCloudReport) collectMetricFromThisServer(server jsonutils.JSONObject, rtnMetric qcloud.SDataPoint,
	influxDbSpecs []string) ([]influxdb.SMetricData, error) {
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
		if influxDbSpecs[1] == common.UNIT_MBPS {
			fieldValue = fieldValue * 1000 * 1000
		}
		metric.Metrics = append(metric.Metrics, influxdb.SKeyValue{
			Key:   pairsKey,
			Value: strconv.FormatFloat(fieldValue, 'E', -1, 64),
		})
		datas = append(datas, metric)
	}
	return datas, nil
}

func (self *SQCloudReport) collectMetricFromThisK8s(server jsonutils.JSONObject, rtnMetric qcloud.SK8SDataPoint,
	influxDbSpecs []string) ([]influxdb.SMetricData, error) {
	datas := make([]influxdb.SMetricData, 0)
	for index, _ := range rtnMetric.Points {
		for _, pointVal := range rtnMetric.Points[index].Values {

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
			metric.Timestamp = time.Unix(int64(pointVal.Timestamp), 0)
			fieldValue := pointVal.Value
			if influxDbSpecs[1] == common.UNIT_MBPS {
				fieldValue = fieldValue * 1000 * 1000
			}
			metric.Metrics = append(metric.Metrics, influxdb.SKeyValue{
				Key:   pairsKey,
				Value: strconv.FormatFloat(fieldValue, 'E', -1, 64),
			})
			self.AddMetricTag(&metric, map[string]string{
				"source": "cloudmon",
			})
			datas = append(datas, metric)
		}
	}
	return datas, nil
}

func (self *SQCloudReport) CollectK8sModuleMetric(region cloudprovider.ICloudRegion, cluster jsonutils.JSONObject,
	helper common.IK8sClusterModuleHelper) error {
	tecentReg := region.(*qcloud.SRegion)
	since, until, err := common.TimeRangeFromArgs(self.Args)
	id, _ := cluster.GetString("id")
	resources, err := common.ListK8sClusterModuleResources(helper.MyModuleType(), id, self.Session, nil)
	if err != nil {
		log.Errorf("ListK8sClusterModuleResources err: %v", err)
		return err
	}
	ext_id, _ := cluster.GetString("external_cloud_cluster_id")
	namespace, metricSpecs := helper.MyNamespaceAndMetrics()
	metricNames := make([]string, 0)
	for metricName, _ := range metricSpecs {
		metricNames = append(metricNames, metricName)
	}
	for i, _ := range resources {
		dataList := make([]influxdb.SMetricData, 0)
		resource := resources[i]
		name, _ := resource.GetString("name")
		dimensions := self.getK8sModuleDimensions(helper, resource, ext_id)
		rtnArray, err := tecentReg.GetK8sMonitorData(metricNames, namespace, since, until, dimensions)
		if err != nil {
			log.Errorln(err)
			continue
		}
		if len(rtnArray) == 0 {
			continue
		}
		for _, data := range rtnArray {
			for metricName, influxDbSpecs := range metricSpecs {
				if data.MetricName == metricName {
					serverMetric, err := self.collectMetricFromThisK8s(resource, data, influxDbSpecs)
					if err != nil {
						log.Errorf("provider: %s collectK8s:%s metric err: %#v", self.SProvider.Name, name, err)
						continue
					}
					dataList = append(dataList, serverMetric...)
				}
			}
		}
		err = common.SendMetrics(self.Session, dataList, self.Args.Debug, "")
		if err != nil {
			log.Errorf("K8s: %s SendMetrics err: %#v", name, err)
			continue
		}
	}
	return nil
}

func (self *SQCloudReport) getDimensions(res []jsonutils.JSONObject) []qcloud.SQcInstanceMetricDimension {
	dimensions := make([]qcloud.SQcInstanceMetricDimension, 0)
	dimensionId := self.getDimensionId()
	if dimensionId == nil {
		return dimensions
	}
	for _, server := range res {
		external_id, _ := server.GetString(dimensionId.LocalId)
		dimensions = append(dimensions, qcloud.SQcInstanceMetricDimension{
			Dimensions: []qcloud.SQcMetricDimension{qcloud.SQcMetricDimension{
				Name:  dimensionId.ExtId,
				Value: external_id,
			}},
		})
	}
	return dimensions
}

func (self *SQCloudReport) getK8sModuleDimensions(helper common.IK8sClusterModuleHelper, module jsonutils.JSONObject,
	clusterId string) []qcloud.SQcMetricDimension {
	dimensions := make([]qcloud.SQcMetricDimension, 0)
	dimensionId := helper.MyResDimensionId()

	localIds := strings.Split(dimensionId.LocalId, ",")
	extIds := strings.Split(dimensionId.ExtId, ",")
	if len(localIds) != len(extIds) {
		return dimensions
	}
	dimensions = append(dimensions, qcloud.SQcMetricDimension{
		Name:  "tke_cluster_instance_id",
		Value: clusterId,
	})
	for index, localId := range localIds {
		localIdKey := strings.Split(localId, ".")
		val, _ := module.GetString(localIdKey...)
		dimensions = append(dimensions, qcloud.SQcMetricDimension{
			Name:  extIds[index],
			Value: val,
		})

	}
	return dimensions
}

func (self *SQCloudReport) getDimensionId() *common.DimensionId {
	switch common.MonType(self.Operator) {
	case common.SERVER:
		return &common.DimensionId{
			LocalId: "external_id",
			ExtId:   "InstanceId",
		}
	case common.REDIS:
		return &common.DimensionId{
			LocalId: "external_id",
			ExtId:   "instanceid",
		}
	case common.RDS:
		return &common.DimensionId{
			LocalId: "external_id",
			ExtId:   "InstanceId",
		}
	case common.K8S:
		return &common.DimensionId{
			LocalId: "external_cloud_cluster_id",
			ExtId:   "tke_cluster_instance_id",
		}
	}
	return nil
}

func (self *SQCloudReport) getMetricSpecs(res []jsonutils.JSONObject) (string, map[string][]string) {
	switch common.MonType(self.Operator) {
	case common.SERVER:
		return SERVER_METRIC_NAMESPACE, tecentMetricSpecs
	case common.REDIS:
		return REDIS_METRIC_NAMESPACE, tecentRedisMetricSpecs
	case common.RDS:
		return RDS_METRIC_NAMESPACE, tecentRdsMetricSpecs
	case common.K8S:
		return K8S_METRIC_NAMESPACE, tecentK8SClusterMetricSpecs
	default:
		return SERVER_METRIC_NAMESPACE, tecentMetricSpecs
	}
}
