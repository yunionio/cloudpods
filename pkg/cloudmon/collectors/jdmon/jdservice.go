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

package jdmon

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/timeutils"

	"yunion.io/x/onecloud/pkg/cloudmon/collectors/common"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud/jdcloud"
	"yunion.io/x/onecloud/pkg/util/influxdb"
)

const (
	SERVICE_CODE_VM             = "vm"
	SERVICE_CODE_RDS_SQLSERVER  = "sqlserver"
	SERVICE_CODE_RDS_MYSQL      = "database"
	SERVICE_CODE_RDS_PERCONA    = "percona"
	SERVICE_CODE_RDS_MARIADB    = "mariadb"
	SERVICE_CODE_RDS_POSTGRESQL = "pg"
)

var (
	rdsEngineMap = map[string]string{
		"SQL Server": SERVICE_CODE_RDS_SQLSERVER,
		"MySQL":      SERVICE_CODE_RDS_MYSQL,
		"Percona":    SERVICE_CODE_RDS_PERCONA,
		"MariaDB":    SERVICE_CODE_RDS_MARIADB,
		"PostgreSQL": SERVICE_CODE_RDS_POSTGRESQL,
	}
)

func (self *SJdCloudReport) collectRegionMetricOfServer(region cloudprovider.ICloudRegion,
	servers []jsonutils.JSONObject) error {
	dataList := make([]influxdb.SMetricData, 0)
	jdReg := region.(*jdcloud.SRegion)
	since, until, err := common.TimeRangeFromArgs(self.Args)
	if err != nil {
		return err
	}
	sinceStr := since.Format(timeutils.FullIsoTimeFormat)
	untilStr := until.Format(timeutils.FullIsoTimeFormat)
	for _, server := range servers {
		metrics := self.GetInstanceMetric(server, jdReg, jdMetricSpecs, sinceStr, untilStr, SERVICE_CODE_VM)
		if len(metrics) != 0 {
			dataList = append(dataList, metrics...)
		}
	}
	return common.SendMetrics(self.Session, dataList, self.Args.Debug, "")
}

func (self *SJdCloudReport) GetInstanceMetric(instance jsonutils.JSONObject, jdReg *jdcloud.SRegion,
	metricMap map[string][]string,
	startTime string, endTime string, serviceCode string) []influxdb.SMetricData {
	dataList := make([]influxdb.SMetricData, 0)
	name, _ := instance.GetString("name")
	instanceId, _ := instance.GetString("external_id")
	for metricName, influxDbSpecs := range metricMap {
		request := jdcloud.NewDescribeMetricDataRequestWithAllParams(jdReg.GetId(), metricName, &startTime, &endTime,
			nil, &serviceCode, instanceId)
		response, err := jdReg.GetMetricsData(request)
		if err != nil {
			log.Errorf("get instance:%s metric err:%v", name, err)
			continue
		}
		metricData, err := self.collectMetricFromThisServer(instance, response, influxDbSpecs)
		if err != nil {
			log.Errorf("collectMetricFromThisServer:%s err:%v", name, err)
			continue
		}
		dataList = append(dataList, metricData...)
	}
	return dataList
}

func (self *SJdCloudReport) collectMetricFromThisServer(server jsonutils.JSONObject,
	metricRep *jdcloud.DescribeMetricDataResponse,
	influxDbSpecs []string) ([]influxdb.SMetricData, error) {
	datas := make([]influxdb.SMetricData, 0)
	for _, metricData := range metricRep.Result.MetricDatas {
		for _, datapoint := range metricData.Data {
			//metric, err := common.JsonToMetric(server.(*jsonutils.JSONDict), "", common.ServerTags, make([]string, 0))
			metric, err := self.NewMetricFromJson(server)
			if err != nil {
				return datas, err
			}
			metric.Timestamp = time.Unix(datapoint.Timestamp/1000, 0)
			fieldValue := self.parseDataValue(datapoint.Value)
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
			if influxDbSpecs[1] == UNIT_KBPS {
				fieldValue = fieldValue * 1000
			}
			tag := common.SubstringAfter(influxDbSpec, ",")
			if tag != "" && strings.Contains(influxDbSpec, "=") {
				metric.Tags = append(metric.Tags, influxdb.SKeyValue{
					Key:   common.SubstringBefore(tag, "="),
					Value: common.SubstringAfter(tag, "="),
				})
			}
			metric.Metrics = append(metric.Metrics, influxdb.SKeyValue{
				Key:   pairsKey,
				Value: strconv.FormatFloat(fieldValue, 'E', -1, 64),
			})
			metric.Name = measurement
			self.AddMetricTag(&metric, common.OtherVmTags)
			datas = append(datas, metric)
		}
	}
	return datas, nil
}

func (self *SJdCloudReport) parseDataValue(value interface{}) float64 {
	str, ok := value.(string)
	if !ok {
		log.Errorf("parseDataValue err:%v", value)
		return 0
	}
	number := json.Number(str)
	fvalue, err := number.Float64()
	if err == nil {
		return fvalue
	}

	ivalue, err := number.Int64()
	if err == nil {
		ret := float64(ivalue)
		return ret
	}
	log.Errorln("parseDataValue data type err")
	return 0
}

func (self *SJdCloudReport) collectRegionMetricOfRds(region cloudprovider.ICloudRegion,
	servers []jsonutils.JSONObject) error {
	dataList := make([]influxdb.SMetricData, 0)

	jdReg := region.(*jdcloud.SRegion)
	since, until, err := common.TimeRangeFromArgs(self.Args)
	if err != nil {
		return err
	}
	sinceStr := since.Format(timeutils.FullIsoTimeFormat)
	untilStr := until.Format(timeutils.FullIsoTimeFormat)

	for _, server := range servers {
		engine, _ := server.GetString("engine")
		metrics := make([]influxdb.SMetricData, 0)
		if serviceCode, ok := rdsEngineMap[engine]; ok {
			switch serviceCode {
			case SERVICE_CODE_RDS_SQLSERVER:
				metrics = self.GetInstanceMetric(server, jdReg, jdRdsSqlserverMetricSpecs, sinceStr, untilStr, serviceCode)
			case SERVICE_CODE_RDS_MYSQL:
				metrics = self.GetInstanceMetric(server, jdReg, jdRdsMysqlMetricSpecs, sinceStr, untilStr, serviceCode)
			case SERVICE_CODE_RDS_PERCONA:
				metrics = self.GetInstanceMetric(server, jdReg, jdRdsPerconaMetricSpecs, sinceStr, untilStr, serviceCode)
			case SERVICE_CODE_RDS_MARIADB:
				metrics = self.GetInstanceMetric(server, jdReg, jdRdsMariadbMetricSpecs, sinceStr, untilStr, serviceCode)
			case SERVICE_CODE_RDS_POSTGRESQL:
				metrics = self.GetInstanceMetric(server, jdReg, jdRdsPostgresqlMetricSpecs, sinceStr, untilStr, serviceCode)
			}
			if len(metrics) != 0 {
				dataList = append(dataList, metrics...)
			}
		}
	}
	return common.SendMetrics(self.Session, dataList, self.Args.Debug, "")
}
