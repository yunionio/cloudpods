package modules

import "yunion.io/x/onecloud/pkg/mcclient/modulebase"

type SAlertPanelManager struct {
	*modulebase.ResourceManager
}

var (
	AlertPanelManager *SAlertPanelManager
)

func init() {
	AlertPanelManager = NewAlertPanelManager()
}

func NewAlertPanelManager() *SAlertPanelManager {
	man := NewMonitorV2Manager("alertpanel", "alertpanels",
		[]string{"id", "name", "setting", "common_alert_metric_details"},
		[]string{})
	return &SAlertPanelManager{
		ResourceManager: &man,
	}
}
