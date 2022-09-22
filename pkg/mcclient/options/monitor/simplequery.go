package monitor

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type SimpleQueryTest struct {
	Id string
	NameSpace string
	MetricName string
	Starttime string
	Endtime string
	Tags map[string]string
}
func (o SimpleQueryTest) GetId() string {
	return o.Id
}

func (o SimpleQueryTest) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}