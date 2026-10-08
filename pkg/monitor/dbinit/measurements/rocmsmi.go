package measurements

import "yunion.io/x/onecloud/pkg/apis/monitor"

var rocmSmi = SMeasurement{
	Context: newHostAgentMonitorContext("amd_rocm_smi", "Amd rocm-smi GPU metrics"),
	Metrics: []SMetric{
		{
			"utilization_gpu", "AMD GPU utilization", monitor.METRIC_UNIT_PERCENT,
		},
		{
			"utilization_memory", "AMD GPU memory utilization", monitor.METRIC_UNIT_PERCENT,
		},
		{
			"memory_free", "AMD GPU memory free size", "",
		},
		{
			"memory_used", "AMD GPU memory used size", "",
		},
		{
			"memory_total", "AMD GPU memory total size", "",
		},
		{
			"power_draw", "AMD GPU power draw", "",
		},
		{
			"temperature_sensor_edge", "AMD GPU temperature sensor edge", "",
		},
		{
			"fan_speed", "AMD GPU fan speed", "",
		},
		{
			"driver_version", "AMD GPU driver version", "",
		},
	},
}
