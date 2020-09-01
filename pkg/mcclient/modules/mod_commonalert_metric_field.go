package modules

import "yunion.io/x/onecloud/pkg/mcclient/modulebase"

type SMetricFieldManager struct {
	*modulebase.ResourceManager
}

var (
	MetricFieldManager *SMetricFieldManager
)

func init() {
	MetricFieldManager = NewMetricFieldManager()
	register(MetricFieldManager)
}

func NewMetricFieldManager() *SMetricFieldManager {
	man := NewMonitorV2Manager("metricfield", "metricfields",
		[]string{"id", "name", "display_name", "Unit"},
		[]string{})
	return &SMetricFieldManager{
		ResourceManager: &man,
	}
}
