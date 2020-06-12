package modules

import "yunion.io/x/onecloud/pkg/mcclient/modulebase"

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
		[]string{},
		[]string{})
	return &SCommonAlertManager{
		ResourceManager: &man,
	}
}
