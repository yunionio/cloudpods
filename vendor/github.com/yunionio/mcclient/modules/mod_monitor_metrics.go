package modules

type MonitorMetricManager struct {
	ResourceManager
}

var (
	MonitorMetrics MonitorMetricManager
)

func init() {
	MonitorMetrics = MonitorMetricManager{NewMonitorManager("monitor_metric", "monitor_metrics",
		[]string{"id", "node_labels", "monitor_metrics", "status", "create_by", "update_by", "delete_by", "gmt_create", "gmt_modified", "gmt_delete", "project_id", "remark"},
		[]string{})}

	register(&MonitorMetrics)
}
