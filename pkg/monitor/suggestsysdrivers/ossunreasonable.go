package suggestsysdrivers

import (
	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/monitor/dbinit"
	"yunion.io/x/onecloud/pkg/monitor/models"
)

type OssUnreasonable struct {
	*InfluxdbBaseDriver
}

func NewOssUnreasonableDriver() models.ISuggestSysRuleDriver {
	return &OssUnreasonable{
		InfluxdbBaseDriver: NewInfluxdbBaseDriver(monitor.OSS_UNREASONABLE,
			monitor.OSS_UNREASONABLE_MONITOR_RES_TYPE,
			monitor.REDIS_UNREASONABLE_DRIVER_ACTION,
			monitor.SCALE_DOWN_MONITOR_SUGGEST,
			*dbinit.OssUnReasonableCreateInput,
		),
	}
}
