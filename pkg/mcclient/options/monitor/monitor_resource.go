package monitor

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type MonitorResourceJointAlertOptions struct {
}

func (o *MonitorResourceJointAlertOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(o)
}

func (o *MonitorResourceJointAlertOptions) GetId() string {
	return "alert"
}
