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
		METRIC_RES_TYPE_HOST:         "host",
		METRIC_RES_TYPE_GUEST:        "vm_name",
		METRIC_RES_TYPE_REDIS:        "redis_name",
		METRIC_RES_TYPE_RDS:          "rds_name",
		METRIC_RES_TYPE_OSS:          "oss_name",
		METRIC_RES_TYPE_CLOUDACCOUNT: "cloudaccount_name",
		METRIC_RES_TYPE_STORAGE:      "storage_name",
		METRIC_RES_TYPE_AGENT:        "vm_name",
	}
	MEASUREMENT_TAG_ID = map[string]string{
		METRIC_RES_TYPE_HOST:         "host_id",
		METRIC_RES_TYPE_GUEST:        "vm_id",
		METRIC_RES_TYPE_AGENT:        "vm_id",
		METRIC_RES_TYPE_REDIS:        "redis_id",
		METRIC_RES_TYPE_RDS:          "rds_id",
		METRIC_RES_TYPE_OSS:          "oss_id",
		METRIC_RES_TYPE_CLOUDACCOUNT: "cloudaccount_id",
		METRIC_RES_TYPE_TENANT:       "tenant_id",
		METRIC_RES_TYPE_DOMAIN:       "domain_id",
		METRIC_RES_TYPE_STORAGE:      "storage_id",
	}
	AlertReduceFunc = map[string]string{
		"avg":          "average value",
		"sum":          "Summation",
		"min":          "minimum value",
		"max":          "Maximum",
		"count":        "count value",
		"last":         "Latest value",
		"median":       "median",
		"diff":         "The difference between the latest value and the oldest value. The judgment basis value must be legal",
		"percent_diff": "The difference between the new value and the old value,based on the percentage of the old value",
	}
)

type MetricFunc struct {
	FieldOptType  []string            `json:"field_opt_type"`
	FieldOptValue map[string][]string `json:"field_opt_value"`
	GroupOptType  []string            `json:"group_opt_type"`
	GroupOptValue map[string][]string `json:"group_opt_value"`
}

type MetricInputQuery struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Scope   string `json:"scope"`
	Slimit  string `json:"slimit"`
	Soffset string `json:"soffset"`
	//default group by
	Unit        bool          `json:"unit"`
	Interval    string        `json:"interval"`
	DomainId    string        `json:"domain_id"`
	ProjectId   string        `json:"project_id"`
	MetricQuery []*AlertQuery `json:"metric_query"`
	Signature   string        `json:"signature"`
	ShowMeta    bool          `json:"show_meta"`
}
