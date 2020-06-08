package monitor

var (
	UNIFIED_MONITOR_FIELD_OPT_TYPE   = []string{"Aggregations", "Selectors"}
	UNIFIED_MONITOR_GROUPBY_OPT_TYPE = []string{"time", "tag", "fill"}
	UNIFIED_MONITOR_FIELD_OPT_VALUE  = map[string][]string{
		"Aggregations": {"COUNT", "DISTINCT", "INTEGRAL",
			"MEAN", "MEDIAN", "MODE", "STDDEV", "SUM"},
		"Selectors": {"BOTTOM", "FIRST", "LAST", "MAX", "MIN", "TOP"},
	}
	UNIFIED_MONITOR_GROUPBY_OPT_VALUE = map[string][]string{
		"fill": {"linear", "none", "previous", "0"},
	}

	MEASUREMENT_TAG_KEYWORD = map[string]string{
		"cpu":       "host",
		"disk":      "device",
		"mem":       "host",
		"net":       "host",
		"netstat":   "host",
		"vm_cpu":    "vm_name",
		"vm_diskio": "vm_name",
		"vm_mem":    "vm_name",
		"vm_netio":  "vm_name",
	}
)

type MetricFunc struct {
	FieldOptType  []string            `json:"field_opt_type"`
	FieldOptValue map[string][]string `json:"field_opt_value"`
	GroupOptType  []string            `json:"group_opt_type"`
	GroupOptValue map[string][]string `json:"group_opt_value"`
}

type MetricInputQuery struct {
	From        string        `json:"from"`
	To          string        `json:"to"`
	Interval    string        `json:"interval"`
	MetricQuery []*AlertQuery `json:"metric_query"`
}
