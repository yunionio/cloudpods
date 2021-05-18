package monitor

import (
	"time"

	"yunion.io/x/onecloud/pkg/apis"
)

type AlertRecordShieldCreateInput struct {
	apis.StandaloneResourceCreateInput

	AlertId string `json:"alert_id"`
	ResType string `json:"res_type"`
	ResName string `json:"res_name"`

	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
}

type AlertRecordShieldDetails struct {
	apis.StatusStandaloneResourceDetails
	apis.ScopedResourceBaseInfo

	CommonAlertDetails
	AlertName string
}

type AlertRecordShieldListInput struct {
	apis.Meta

	apis.ScopedResourceBaseListInput
	apis.EnabledResourceBaseListInput
	apis.StatusStandaloneResourceListInput

	AlertName string `json:"alerting"`
	ResType   string `json:"res_type"`
	ResName   string `json:"res_name"`
	AlertId   string `json:"alert_id"`
}
