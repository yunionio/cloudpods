package suggestsysdrivers

import (
	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/monitor/models"
)

type RedisUnreasonable struct {
	*InfluxdbBaseDriver
}

func NewRedisUnreasonableDriver() models.ISuggestSysRuleDriver {
	return &RedisUnreasonable{
		InfluxdbBaseDriver: NewInfluxdbBaseDriver(monitor.REDIS_UNREASONABLE,
			monitor.REDIS_UNREASONABLE_MONITOR_RES_TYPE, monitor.REDIS_UNREASONABLE_DRIVER_ACTION,
			monitor.SCALE_DOWN_MONITOR_SUGGEST),
	}
}
