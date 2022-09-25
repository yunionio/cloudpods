package monitor

import (
	"yunion.io/x/jsonutils"
)

type SimpleInputQuery struct {
	Id         string            `json:"id"`
	Database   string            `json:"database"`
	MetricName string            `json:"metric_name"`
	StartTime  string            `json:"start_time"`
	EndTime    string            `json:"end_time"`
	Tags       map[string]string `json:"tags"`
}

func (o *SimpleInputQuery) GetId() string {
	return "simple-query"
}

func (o *SimpleInputQuery) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(o), nil
}
