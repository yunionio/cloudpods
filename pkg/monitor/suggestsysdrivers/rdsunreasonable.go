package suggestsysdrivers

import (
	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/monitor/models"
)

type RdsUnreasonable struct {
	*InfluxdbBaseDriver
}

func NewRdsUnreasonableDriver() models.ISuggestSysRuleDriver {
	return &RdsUnreasonable{
		InfluxdbBaseDriver: NewInfluxdbBaseDriver(monitor.RDS_UNREASONABLE,
			monitor.RDS_UNREASONABLE_MONITOR_RES_TYPE,
			monitor.REDIS_UNREASONABLE_DRIVER_ACTION,
			monitor.SCALE_DOWN_MONITOR_SUGGEST),
	}
}
