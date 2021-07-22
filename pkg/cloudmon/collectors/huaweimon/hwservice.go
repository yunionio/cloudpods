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

package huaweimon

import (
	"strconv"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudmon/collectors/common"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud/huawei"
	hw_moudules "yunion.io/x/onecloud/pkg/multicloud/huawei/client/modules"
	"yunion.io/x/onecloud/pkg/util/influxdb"
)

func (self *SHwCloudReport) collectRegionMetricOfHost(region cloudprovider.ICloudRegion, servers []jsonutils.JSONObject) error {
	dataList := make([]influxdb.SMetricData, 0)
	hwReg := region.(*huawei.SRegion)
	since, until, err := common.TimeRangeFromArgs(self.Args)
	if err != nil {
		return err
	}
	for _, server := range servers {
		instanceId, _ := server.GetString("external_id")
		metric, err := common.FillVMCapacity(server.(*jsonutils.JSONDict))
		if err != nil {
			return err
		}
		dataList = append(dataList, metric)
		metas := make([]hw_moudules.SMetricMeta, 0)
		for metricName := range huaweiMetricSpecs {
			hwMeta := hw_moudules.SMetricMeta{}
			hwMeta.MetricName = metricName
			hwMeta.Namespace = "SYS.ECS"
			hwMeta.Dimensions = make([]hw_moudules.SMetricDimension, 0)
			hwMeta.Dimensions = append(hwMeta.Dimensions, hw_moudules.SMetricDimension{Name: "instance_id", Value: instanceId})
			metas = append(metas, hwMeta)
		}

		metricDatas, err := hwReg.GetMetricsData(metas, since, until)
		if err != nil {
			log.Errorln(err)
			continue
		}
		if len(metricDatas) > 0 {
			for _, metricData := range metricDatas {
				for metricName, influxDbSpecs := range huaweiMetricSpecs {
					if metricData.MetricName == metricName {
						if len(metricData.Datapoints) > 0 {
							for _, datapoint := range metricData.Datapoints {
								serverMetric, err := self.collectMetricFromThisServer(server, datapoint,
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
		}

	}
	return common.SendMetrics(self.Session, dataList, self.Args.Debug, "")
}

func (self *SHwCloudReport) collectRegionMetricOfRedis(region cloudprovider.ICloudRegion, servers []jsonutils.JSONObject) error {
	dataList := make([]influxdb.SMetricData, 0)

	hwReg := region.(*huawei.SRegion)
	since, until, err := common.TimeRangeFromArgs(self.Args)
	if err != nil {
		return err
	}
	for _, server := range servers {
		instanceId, _ := server.GetString("external_id")
		metas := make([]hw_moudules.SMetricMeta, 0)
		for metricName := range huaweiRedisMetricSpecs {
			hwMeta := hw_moudules.SMetricMeta{}
			hwMeta.MetricName = metricName
			hwMeta.Namespace = "SYS.DCS"
			hwMeta.Dimensions = make([]hw_moudules.SMetricDimension, 0)
			hwMeta.Dimensions = append(hwMeta.Dimensions, hw_moudules.SMetricDimension{Name: "dcs_instance_id", Value: instanceId})
			metas = append(metas, hwMeta)
		}

		metricDatas, err := hwReg.GetMetricsData(metas, since, until)
		if err != nil {
			return err
		}
		if len(metricDatas) > 0 {
			for _, metricData := range metricDatas {
				for metricName, influxDbSpecs := range huaweiRedisMetricSpecs {
					if metricData.MetricName == metricName {
						if len(metricData.Datapoints) > 0 {
							for _, datapoint := range metricData.Datapoints {
								serverMetric, err := self.collectMetricFromThisServer(server, datapoint,
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
		}

	}
	return common.SendMetrics(self.Session, dataList, self.Args.Debug, "")
}

func (self *SHwCloudReport) collectRegionMetricOfRds(region cloudprovider.ICloudRegion,
	servers []jsonutils.JSONObject) error {
	dataList := make([]influxdb.SMetricData, 0)

	hwReg := region.(*huawei.SRegion)
	since, until, err := common.TimeRangeFromArgs(self.Args)
	if err != nil {
		return err
	}
	for _, server := range servers {
		instanceId, _ := server.GetString("external_id")
		engine, _ := server.GetString("engine")
		metric, err := common.FillVMCapacity(server.(*jsonutils.JSONDict))
		if err != nil {
			return err
		}
		dataList = append(dataList, metric)
		metas := make([]hw_moudules.SMetricMeta, 0)
		for metricName := range huaweiRdsMetricSpecs {
			hwMeta := hw_moudules.SMetricMeta{}
			hwMeta.MetricName = metricName
			hwMeta.Namespace = "SYS.RDS"
			hwMeta.Dimensions = make([]hw_moudules.SMetricDimension, 0)
			switch engine {
			case "MySQL":
				hwMeta.Dimensions = append(hwMeta.Dimensions, hw_moudules.SMetricDimension{Name: "rds_cluster_id", Value: instanceId})
			case "PostgreSQL":
				hwMeta.Dimensions = append(hwMeta.Dimensions, hw_moudules.SMetricDimension{Name: "postgresql_cluster_id", Value: instanceId})
			case "SQLServer":
				hwMeta.Dimensions = append(hwMeta.Dimensions, hw_moudules.SMetricDimension{Name: "rds_cluster_sqlserver_id", Value: instanceId})
			}
			metas = append(metas, hwMeta)
		}
		index := 0
		tmp := 0
		for {
			if index > len(metas) {
				break
			}
			tmp = index + 10
			if tmp > len(metas) {
				tmp = len(metas)
			}
			metricDatas, err := hwReg.GetMetricsData(metas[index:tmp], since, until)
			if err != nil {
				return err
			}
			if len(metricDatas) > 0 {
				for _, metricData := range metricDatas {
					for metricName, influxDbSpecs := range huaweiRdsMetricSpecs {
						if metricData.MetricName == metricName {
							if len(metricData.Datapoints) > 0 {
								for _, datapoint := range metricData.Datapoints {
									serverMetric, err := self.collectMetricFromThisServer(server, datapoint, influxDbSpecs)
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
			index += 10
		}
	}
	return common.SendMetrics(self.Session, dataList, self.Args.Debug, "")
}

func (self *SHwCloudReport) collectRegionMetricOfOss(region cloudprovider.ICloudRegion,
	servers []jsonutils.JSONObject) error {
	dataList := make([]influxdb.SMetricData, 0)
	hwReg := region.(*huawei.SRegion)
	since, until, err := common.TimeRangeFromArgs(self.Args)
	if err != nil {
		return err
	}
	for _, server := range servers {
		name, _ := server.GetString("name")
		metas := make([]hw_moudules.SMetricMeta, 0)
		for metricName := range huaweiOSSMetricSpecs {
			hwMeta := hw_moudules.SMetricMeta{}
			hwMeta.MetricName = metricName
			hwMeta.Namespace = "SYS.OBS"
			hwMeta.Dimensions = make([]hw_moudules.SMetricDimension, 0)
			hwMeta.Dimensions = append(hwMeta.Dimensions, hw_moudules.SMetricDimension{Name: "bucket_name", Value: name})
			metas = append(metas, hwMeta)
		}

		metricDatas, err := hwReg.GetMetricsData(metas, since, until)
		if err != nil {
			return err
		}
		if len(metricDatas) > 0 {
			for _, metricData := range metricDatas {
				for metricName, influxDbSpecs := range huaweiOSSMetricSpecs {
					if metricData.MetricName == metricName {
						if len(metricData.Datapoints) > 0 {
							for _, datapoint := range metricData.Datapoints {
								serverMetric, err := self.collectMetricFromThisServer(server, datapoint,
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
		}

	}
	return common.SendMetrics(self.Session, dataList, self.Args.Debug, "")
}

func (self *SHwCloudReport) collectRegionMetricOfElb(region cloudprovider.ICloudRegion, servers []jsonutils.JSONObject) error {
	dataList := make([]influxdb.SMetricData, 0)
	hwReg := region.(*huawei.SRegion)
	since, until, err := common.TimeRangeFromArgs(self.Args)
	if err != nil {
		return err
	}
	for _, server := range servers {
		external_id, _ := server.GetString("external_id")
		metas := make([]hw_moudules.SMetricMeta, 0)
		for metricName := range huaweiOSSMetricSpecs {
			hwMeta := hw_moudules.SMetricMeta{}
			hwMeta.MetricName = metricName
			hwMeta.Namespace = "SYS.ELB"
			hwMeta.Dimensions = make([]hw_moudules.SMetricDimension, 0)
			hwMeta.Dimensions = append(hwMeta.Dimensions, hw_moudules.SMetricDimension{Name: "lb_instance_id", Value: external_id})
			metas = append(metas, hwMeta)
		}

		metricDatas, err := hwReg.GetMetricsData(metas, since, until)
		if err != nil {
			return err
		}
		if len(metricDatas) > 0 {
			for _, metricData := range metricDatas {
				for metricName, influxDbSpecs := range huaweiOSSMetricSpecs {
					if metricData.MetricName == metricName {
						if len(metricData.Datapoints) > 0 {
							for _, datapoint := range metricData.Datapoints {
								serverMetric, err := self.collectMetricFromThisServer(server, datapoint,
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
		}

	}
	return common.SendMetrics(self.Session, dataList, self.Args.Debug, "")
}

func (self *SHwCloudReport) collectMetricFromThisServer(server jsonutils.JSONObject, datapoint hw_moudules.SDatapoint,
	influxDbSpecs []string) (influxdb.SMetricData, error) {
	metric, err := self.NewMetricFromJson(server)
	//metric, err := common.JsonToMetric(server.(*jsonutils.JSONDict), "", common.ServerTags, make([]string, 0))
	if err != nil {
		return influxdb.SMetricData{}, err
	}
	metric.Timestamp = time.Unix(datapoint.Timestamp/1000, 0)
	fieldValue := datapoint.Average
	//根据条件拼装metric的tag和metirc信息
	influxDbSpec := influxDbSpecs[2]
	measurement := common.SubstringBefore(influxDbSpec, ".")
	var pairsKey string
	if strings.Contains(influxDbSpec, ",") {
		pairsKey = common.SubstringBetween(influxDbSpec, ".", ",")
	} else {
		pairsKey = common.SubstringAfter(influxDbSpec, ".")
	}
	if influxDbSpecs[1] == UNIT_BYTEPS {
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
	self.AddMetricTag(&metric, common.OtherVmTags)
	return metric, nil
}
