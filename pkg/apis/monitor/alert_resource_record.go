package monitor

import (
	"time"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis"
)

const (
	ALERT_RESOURCE_RECORD_SHIELD_KEY   = "send_state"
	ALERT_RESOURCE_RECORD_SHIELD_VALUE = "hide"
)

type AlertResourceRecordCreateInput struct {
	apis.Meta
	apis.SStandaloneResourceBase

	EvalData      EvalMatch
	AlertTime     time.Time
	ResName       string
	ResType       string
	Brand         string
	TriggerVal    string
	AlertRecordId string
	AlertId       string
	SendState     string
}

type AlertResourceRecordDetails struct {
	apis.StatusStandaloneResourceDetails
	apis.ScopedResourceBaseInfo

	AlertName string `json:"alert_name"`
	AlertRule jsonutils.JSONObject
	ResType   string
	Level     string
	State     string
}
