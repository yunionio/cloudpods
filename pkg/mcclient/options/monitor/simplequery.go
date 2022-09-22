package monitor

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type SimpleQueryTest struct {
	Database string
	Measurement string
	Filed string
}
func (o SimpleQueryTest) GetId() string {
	return ""
}

func (o SimpleQueryTest) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}