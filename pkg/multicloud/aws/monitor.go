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

package aws

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
)

func (self *SRegion) GetMonitorData(name string, ns string, instanceId string, since time.Time,
	until time.Time) (*cloudwatch.GetMetricStatisticsOutput, error) {
	return self.GetMonitorDataByDimensionName(name, ns, "InstanceId", instanceId, since, until)
}

func (self *SRegion) GetMonitorDataByDimensionName(name, ns, dimensionName, instanceId string, since time.Time,
	until time.Time) (*cloudwatch.GetMetricStatisticsOutput, error) {
	params := cloudwatch.GetMetricStatisticsInput{}
	params.MetricName = &name
	params.Namespace = &ns
	params.Period = aws.Int64(int64(1))
	params.Statistics = []*string{aws.String("Average")}
	if len(dimensionName) != 0 {
		params.Dimensions = []*cloudwatch.Dimension{&cloudwatch.Dimension{
			Name:  aws.String(dimensionName),
			Value: aws.String(instanceId),
		}}
	}
	if !since.IsZero() {
		params.StartTime = aws.Time(since)
	}
	if !until.IsZero() {
		params.EndTime = aws.Time(until)
	}
	dataPoints := cloudwatch.GetMetricStatisticsOutput{}
	err := self.cloudWatchRequest("GetMetricStatistics", &params, &dataPoints)
	return &dataPoints, err
}
