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

package awsmon

import (
	"strconv"
	"strings"

	aws_sdk "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudmon/collectors/common"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud/aws"
	"yunion.io/x/onecloud/pkg/util/influxdb"
)

func (self *SAwsCloudReport) CollectRegionMetric(region cloudprovider.ICloudRegion,
	servers []jsonutils.JSONObject) error {
	var err error
	switch self.Operator {
	case string(common.SERVER):
		err = self.collectRegionMetricOfHost(region, servers)
	}
	return err
}

func (self *SAwsCloudReport) collectRegionMetricOfHost(region cloudprovider.ICloudRegion,
	servers []jsonutils.JSONObject) error {
	dataList := make([]influxdb.SMetricData, 0)
	awsRegion := region.(*aws.SRegion)
	since, until, err := common.TimeRangeFromArgs(self.Args)
	if err != nil {
		return err
	}
	for metricName, influxDbSpecs := range awsMetricSpecs {
		for i, server := range servers {
			external_id, _ := servers[i].GetString("external_id")
			rtnArray, err := awsRegion.GetMonitorData(metricName, "AWS/EC2", external_id, since, until)
			if err != nil {
				log.Errorln(err)
				continue
			}
			if len(rtnArray.Datapoints) > 0 {
				for _, dataPoint := range rtnArray.Datapoints {
					metric, err := common.FillVMCapacity(server.(*jsonutils.JSONDict))
					if err != nil {
						return err
					}
					dataList = append(dataList, metric)
					serverMetric, err := self.collectMetricFromThisServerForAws(server, dataPoint, influxDbSpecs)
					if err != nil {
						return err
					}
					dataList = append(dataList, serverMetric)
				}

			}
		}
		err = common.SendMetrics(self.Session, dataList, self.Args.Debug, "")
		if err != nil {
			log.Errorln(err)
		}
	}
	return nil
}

func (self *SAwsCloudReport) collectMetricFromThisServerForAws(server jsonutils.JSONObject, datapoint *cloudwatch.Datapoint,
	influxDbSpecs []string) (metric influxdb.SMetricData, err error) {
	metric, err = common.JsonToMetric(server.(*jsonutils.JSONDict), "", common.ServerTags, make([]string, 0))
	if err != nil {
		return metric, err
	}
	metric.Timestamp = aws_sdk.TimeValue(datapoint.Timestamp)
	fieldValue := aws_sdk.Float64Value(datapoint.Average)
	//根据条件拼装metric的tag和metirc信息
	influxDbSpec := influxDbSpecs[2]
	measurement := common.SubstringBefore(influxDbSpec, ".")
	var pairsKey string
	if strings.Contains(influxDbSpec, ",") {
		pairsKey = common.SubstringBetween(influxDbSpec, ".", ",")
	} else {
		pairsKey = common.SubstringAfter(influxDbSpec, ".")
	}
	// 300:AWS metric collection period is 5m
	if influxDbSpecs[1] == UNIT_COUNT {
		fieldValue = (fieldValue / float64(300))
	}
	if influxDbSpecs[1] == UNIT_MEM {
		fieldValue = (fieldValue / float64(300) * 8)
	}
	tag := common.SubstringAfter(influxDbSpec, ",")
	if tag != "" && strings.Contains(influxDbSpec, "=") {
		metric.Tags = append(metric.Tags, influxdb.SKeyValue{Key: common.SubstringBefore(tag, "="),
			Value: common.SubstringAfter(tag, "=")})
	}
	cpu_cout, err := server.Get("vcpu_count")
	if err != nil {
		metric.Metrics = append(metric.Metrics, influxdb.SKeyValue{Key: pairsKey,
			Value: strconv.FormatFloat(fieldValue, 'E', -1, 64)})
	} else {
		metric.Metrics = append(metric.Metrics, influxdb.SKeyValue{Key: "cpu_count",
			Value: strconv.FormatInt(cpu_cout.(*jsonutils.JSONInt).Value(), 10)},
			influxdb.SKeyValue{Key: pairsKey,
				Value: strconv.FormatFloat(fieldValue, 'E', -1, 64)})
	}
	self.AddMetricTag(&metric, common.OtherVmTags)
	metric.Name = measurement
	return metric, nil
}
