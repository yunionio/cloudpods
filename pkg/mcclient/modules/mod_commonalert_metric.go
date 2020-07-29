package modules

import "yunion.io/x/onecloud/pkg/mcclient/modulebase"

type SMetricManager struct {
	*modulebase.ResourceManager
}

var (
	MetricManager *SMetricManager
)

func init() {
	MetricManager = NewMetricManager()
	register(MetricManager)
}

func NewMetricManager() *SMetricManager {
	man := NewMonitorV2Manager("metricmeasurement", "metricmeasurements",
		[]string{"id", "name", "display_name", "res_type", "metric_fields"},
		[]string{})
	return &SMetricManager{
		ResourceManager: &man,
	}
}
