package monitor

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/apis/monitor"
)

type SimpleQueryTest struct {
	Id         string            `json:"id"`
	NameSpace  string            `json:"name_space"`
	MetricName string            `json:"metric_name"`
	Starttime  string            `json:"start_time"`
	Endtime    string            `json:"end_time"`
	Tags       map[string]string `json:"tags"`
}

func (o *SimpleQueryTest) GetId() string {
	return o.Id
}

func (o *SimpleQueryTest) Params() (jsonutils.JSONObject, error) {
	output := new(monitor.SimpleQueryTest)
	output.Id = o.Id
	output.NameSpace = o.NameSpace
	output.Starttime = o.Starttime
	output.Endtime = o.Endtime
	output.Tags = o.Tags
	return jsonutils.Marshal(output), nil
}
