package monitor

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis/monitor"
)

type SimpleQueryTest struct {
	Id         string            `json:"id"`
	Database   string            `json:"database"`
	MetricName string            `json:"metric_name"`
	StartTime  string            `json:"start_time"`
	EndTime    string            `json:"end_time"`
	Tags       map[string]string `json:"tags"`
}

func (o *SimpleQueryTest) GetId() string {
	return o.Id
}

func (o *SimpleQueryTest) Params() (jsonutils.JSONObject, error) {
	output := new(monitor.SimpleQueryTest)
	output.Id = o.Id
	output.Database = o.Database
	output.StartTime = o.StartTime
	output.EndTime = o.EndTime
	output.Tags = o.Tags
	return jsonutils.Marshal(output), nil
}
