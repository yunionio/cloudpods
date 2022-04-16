package bingocloud

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/service/cloudwatch"

	"yunion.io/x/pkg/errors"
)

type Dimension struct {
	Name  string
	Value string
}

func (self *SRegion) DescribeMetricList(dimension Dimension, ns string, metricNm string, since time.Time,
	until time.Time,
	nextToken string,
) (*GetMetricStatisticsOutput, error) {
	params := map[string]string{}
	if len(ns) > 0 {
		params["Namespace"] = ns
	}
	if len(metricNm) > 0 {
		params["MetricName"] = metricNm
	}
	idx := 1
	if len(dimension.Value) > 0 {
		params[fmt.Sprintf("Dimensions.member.%d.Name", idx)] = dimension.Name
		params[fmt.Sprintf("Dimensions.member.%d.Value", idx)] = dimension.Value
		idx++
	}
	if !since.IsZero() {
		params["StartTime"] = since.Format(time.RFC3339)
	}
	if !until.IsZero() {
		params["EndTime"] = until.Format(time.RFC3339)
	}
	params["Statistics.member.1"] = "Average"
	params["Period"] = "60"
	jsonObject, err := self.invoke("GetMetricStatistics", params)
	if err != nil {
		return nil, errors.Wrap(err, "GetMetricStatistics err")
	}
	if jsonObject == nil {
		return nil, errors.Errorf("GetMetricStatistics return jsonObject is nil")
	}
	rtn, _ := jsonObject.Get("GetMetricStatisticsResult")
	output := new(GetMetricStatisticsOutput)
	err = rtn.Unmarshal(output)
	if err != nil {
		return nil, errors.Wrap(err, "Unmarshal GetMetricStatisticsOutput err")
	}
	return output, nil
}

type GetMetricStatisticsOutput struct {
	Datapoints GetMetricStatisticsDatapoints
	ObjName    string
	Period     int64
}
type GetMetricStatisticsDatapoints struct {
	Member []cloudwatch.Datapoint
}
