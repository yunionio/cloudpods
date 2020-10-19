package modules

import (
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
)

var (
	CommonAlertManager *SCommonAlertManager
)

type SCommonAlertManager struct {
	*modulebase.ResourceManager
}

func init() {
	CommonAlertManager = NewCommonAlertManager()
	for _, m := range []modulebase.IBaseManager{
		CommonAlertManager,
	} {
		registerCompute(m)
	}
}

func NewCommonAlertManager() *SCommonAlertManager {
	man := NewMonitorV2Manager("commonalert", "commonalerts",
		[]string{"id", "name", "enabled", "level", "alert_type", "period", "recipients", "channel"},
		[]string{})
	return &SCommonAlertManager{
		ResourceManager: &man,
	}
}
