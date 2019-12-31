package aws

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
)

func (self *SRegion) GetMonitorData(name string, ns string, instanceId string, since time.Time,
	until time.Time) (*cloudwatch.GetMetricStatisticsOutput, error) {
	params := cloudwatch.GetMetricStatisticsInput{}
	params.MetricName = &name
	params.Namespace = &ns
	params.Period = aws.Int64(int64(1))
	params.Statistics = []*string{aws.String("Average")}
	params.Dimensions = []*cloudwatch.Dimension{&cloudwatch.Dimension{
		Name:  aws.String("InstanceId"),
		Value: aws.String(instanceId),
	}}
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
