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
	ResId   string `json:"res_id"`

	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
}

type AlertRecordShieldDetails struct {
	apis.StatusStandaloneResourceDetails
	apis.ScopedResourceBaseInfo

	CommonAlertDetails
	AlertName string `json:"alert_name"`
	ResName   string `json:"res_name"`
}

type AlertRecordShieldListInput struct {
	apis.Meta

	apis.ScopedResourceBaseListInput
	apis.EnabledResourceBaseListInput
	apis.StatusStandaloneResourceListInput

	AlertName string `json:"alert_name"`
	ResType   string `json:"res_type"`
	ResName   string `json:"res_name"`
	ResId     string `json:"res_id"`
	AlertId   string `json:"alert_id"`
}
