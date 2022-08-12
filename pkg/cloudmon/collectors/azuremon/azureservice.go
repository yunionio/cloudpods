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
	"fmt"
	"strconv"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	com_api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudmon/collectors/common"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/multicloud/azure"
	"yunion.io/x/onecloud/pkg/util/influxdb"
)

func (self *SAzureCloudReport) CollectRegionMetric(region cloudprovider.ICloudRegion,
	servers []jsonutils.JSONObject) error {
	return self.collectRegionMetricOfHost(region, servers)
}

func (self *SAzureCloudReport) collectRegionMetricOfHost(region cloudprovider.ICloudRegion,
	servers []jsonutils.JSONObject) error {
	azureReg := region.(*azure.SRegion)
	since, until, err := common.TimeRangeFromArgs(self.Args)
	if err != nil {
		return err
	}
	for _, server := range servers {
		dataList := make([]influxdb.SMetricData, 0)
		srvId, _ := server.GetString("id")
		srvName, _ := server.GetString("name")
		srvPrefix := srvId + "/" + "srvName"
		externalId, err := server.GetString("external_id")
		if err != nil {
			continue
		}
		classicKey := "microsoft.classiccompute/virtualmachines"
		ns, metricSpecs := self.getMetricSpecs(server)
		if strings.Contains(strings.ToLower(externalId), classicKey) {
			ns = classicKey
			metricSpecs = azureClassicMetricsSpec
		}
		// SQLServer with databases/master
		if strings.Contains(strings.ToLower(externalId), "microsoft.sql/servers") {
			externalId = fmt.Sprintf("%s/databases/master", externalId)
		}

		err = func() error {
			metricNameArr := make([]string, 0)
			for metricName := range metricSpecs {
				metricNameArr = append(metricNameArr, metricName)
			}
			metricNames := strings.Join(metricNameArr, ",")
			rtnMetrics, err := azureReg.GetMonitorData(metricNames, ns, externalId, since, until, self.Args.MetricInterval, "")
			if err != nil {
				return errors.Wrapf(err, "GetMonitorData")
			}
			if rtnMetrics == nil || rtnMetrics.Value == nil {
				return fmt.Errorf("server %s metic is nil", srvPrefix)
			}

			for metricName, influxDbSpecs := range metricSpecs {
				for _, value := range rtnMetrics.Value {
					if metricName == value.Name.LocalizedValue || (value.Name.Value == metricName) {
						if self.Operator == string(common.SERVER) {
							metric, err := common.FillVMCapacity(server.(*jsonutils.JSONDict))
							if err != nil {
								return errors.Wrapf(err, "fill vm %q capacity", srvPrefix)
							}
							dataList = append(dataList, metric)
						}
						if value.Timeseries != nil {
							for _, timeserie := range value.Timeseries {
								serverMetric, err := self.collectMetricFromThisServer(server, timeserie, influxDbSpecs)
								if err != nil {
									return errors.Wrapf(err, "collect metrics from server %q", srvPrefix)
								}
								dataList = append(dataList, serverMetric...)
							}
						}
					}
				}
			}
			return nil
		}()
		if err != nil {
			log.Errorf("collect azure %s %s %s metric error: %v", externalId, ns, srvName, err)
			continue
		}
		log.Infof("send %s %s %d metrics", ns, externalId, len(dataList))
		err = common.SendMetrics(self.Session, dataList, self.Args.Debug, "")
		if err != nil {
			log.Errorf("send %q metrics error: %v", srvName, err)
		}
	}
	return nil
}

func (self *SAzureCloudReport) collectMetricFromThisServer(server jsonutils.JSONObject,
	rtnMetric azure.TimeSeriesElement, influxDbSpecs []string) ([]influxdb.SMetricData, error) {
	datas := make([]influxdb.SMetricData, 0)
	for _, data := range rtnMetric.Data {
		metric, err := self.NewMetricFromJson(server)
		if err != nil {
			return nil, err
		}
		metric.Timestamp = data.TimeStamp
		//根据条件拼装metric的tag和metirc信息
		fieldValue := data.Average
		influxDbSpec := influxDbSpecs[2]
		measurement := common.SubstringBefore(influxDbSpec, ".")
		metric.Name = measurement
		var pairsKey string
		if strings.Contains(influxDbSpec, ",") {
			pairsKey = common.SubstringBetween(influxDbSpec, ".", ",")
		} else {
			pairsKey = common.SubstringAfter(influxDbSpec, ".")
		}
		if influxDbSpecs[1] == common.UNIT_MEM {
			//fieldValue = fieldValue * 8 / common.PERIOD
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
		datas = append(datas, metric)
	}

	return datas, nil
}

func (self *SAzureCloudReport) getMetricSpecs(res jsonutils.JSONObject) (string, map[string][]string) {
	switch common.MonType(self.Operator) {
	case common.SERVER:
		return SERVER_METRIC_NAMESPACE, azureMetricSpecs
	case common.REDIS:
		return REDIS_METRIC_NAMESPACE, azureRedisMetricsSpec
	case common.RDS:
		return self.getRdsMetricSpecsByEngine(res)
	case common.ELB:
		return ELB_METRIC_NAMESPACE, azureElbMetricSpecs
	default:
		return SERVER_METRIC_NAMESPACE, azureMetricSpecs
	}
}

func (self *SAzureCloudReport) getRdsMetricSpecsByEngine(res jsonutils.JSONObject) (string, map[string][]string) {
	engine, _ := res.GetString("engine")
	externalId, _ := res.GetString("external_id")
	suffix := "servers"
	if strings.Contains(externalId, "flexible") {
		suffix = "flexibleServers"
	}
	switch engine {
	case com_api.DBINSTANCE_TYPE_SQLSERVER:
		return "Microsoft.Sql/servers/databases", azureRdsMetricsSpecSqlserver
	case com_api.DBINSTANCE_TYPE_MYSQL:
		return fmt.Sprintf("Microsoft.DBforMySQL/%s", suffix), azureRdsMetricsSpec
	case com_api.DBINSTANCE_TYPE_POSTGRESQL:
		return fmt.Sprintf("Microsoft.DBforPostgreSQL/%s", suffix), azureRdsMetricsSpec
	case com_api.DBINSTANCE_TYPE_MARIADB:
		return "Microsoft.DBforMariaDB/servers", azureRdsMetricsSpec
	default:
		return fmt.Sprintf("Microsoft.DBforMySQL/%s", suffix), azureRdsMetricsSpec
	}
}

func (self *SAzureCloudReport) CollectK8sModuleMetric(region cloudprovider.ICloudRegion, cluster jsonutils.JSONObject,
	helper common.IK8sClusterModuleHelper) error {
	azureReg := region.(*azure.SRegion)
	since, until, err := common.TimeRangeFromArgs(self.Args)
	id, _ := cluster.GetString("id")
	resources, err := self.getClusterModuleResourceByType(helper.MyModuleType(), id, self.Session, nil)
	if err != nil {
		return errors.Wrapf(err, "getClusterModuleResourceByType")
	}
	externalId, _ := cluster.GetString("external_cloud_cluster_id")
	namespace, metricSpecs := helper.MyNamespaceAndMetrics()

	metricNameArr := make([]string, 0)
	for metricName := range metricSpecs {
		metricNameArr = append(metricNameArr, metricName)
	}
	metricNames := strings.Join(metricNameArr, ",")
	azureReg.GetClient().Debug(true)
	for _, resource := range resources {
		parentName, _ := resource.GetString("name")
		filter := helper.(ik8sModuleFilterHelper).filter(resource)
		rtnMetrics, err := azureReg.GetMonitorData(metricNames, namespace, externalId, since, until,
			self.Args.MetricInterval, filter)
		if err != nil {
			log.Errorf("get deploy/daemonset: %s metrics err %v", parentName, err)
			continue
		}
		if rtnMetrics == nil || rtnMetrics.Value == nil {
			log.Warningf("get deploy/daemonset: %s metrics is nil", parentName)
			continue
		}

		dataList := make([]influxdb.SMetricData, 0)
		for metricName, influxDbSpecs := range metricSpecs {
			for _, value := range rtnMetrics.Value {
				if metricName == value.Name.LocalizedValue || (value.Name.Value == metricName) {
					if value.Timeseries != nil {
						for _, timeserie := range value.Timeseries {
							serverMetric, err := self.collectMetricFromThisServer(resource, timeserie, influxDbSpecs)
							if err != nil {
								log.Errorf("collect pod: %s metric err: %v", parentName, err)
							}
							dataList = append(dataList, serverMetric...)
						}
					}
				}
			}
		}
		err = common.SendMetrics(self.Session, dataList, self.Args.Debug, "")
		if err != nil {
			log.Errorf("send resource: %s metrics error: %v", parentName, err)
		}
	}
	return nil
}

func (self *SAzureCloudReport) getClusterModuleResourceByType(typ common.K8sClusterModuleType, clusterId string,
	session *mcclient.ClientSession, query *jsonutils.JSONDict) ([]jsonutils.JSONObject, error) {
	switch typ {
	case common.K8S_MODULE_POD:
		return self.getK8sClusterPods(clusterId, session, nil)
	case common.K8S_MODULE_NODE:
		return common.ListK8sClusterModuleResources(typ, clusterId, session, nil)
	default:
		return nil, errors.Errorf("unsupport the clusterModuleType: %s", string(typ))
	}
}

func (self *SAzureCloudReport) getK8sClusterPods(clusterId string, session *mcclient.ClientSession, query *jsonutils.JSONDict) ([]jsonutils.JSONObject, error) {
	deployRes, err := common.ListK8sClusterModuleResources(common.K8S_MODULE_DEPLOY, clusterId, session, query)
	if err != nil {
		return nil, errors.Wrapf(err, "List cluster: %s deploy err", clusterId)
	}
	daemonsetRes, err := common.ListK8sClusterModuleResources(common.K8S_MODULE_DAEMONSET, clusterId, session, query)
	if err != nil {
		return nil, errors.Wrapf(err, "List cluster: %s daemonset err", clusterId)
	}
	return append(daemonsetRes, deployRes...), nil
}
