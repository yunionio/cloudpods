package monitor

import "yunion.io/x/onecloud/pkg/apis"

const (
	METRIC_RES_TYPE_GUEST = "guest"
	METRIC_RES_TYPE_HOST  = "host"
	METRIC_RES_TYPE_REDIS = "redis"
	METRIC_RES_TYPE_OSS   = "oss"
	METRIC_RES_TYPE_RDS   = "rds"

	METRIC_UNIT_PERCENT = "%"
	METRIC_UNIT_BPS     = "bps"
	METRIC_UNIT_MBPS    = "Mbps"
	METRIC_UNIT_BYTEPS  = "Bps"
	METRIC_UNIT_CPS     = "cps"
	METRIC_UNIT_COUNT   = "count"
	METRIC_UNIT_MS      = "ms"
	METRIC_UNIT_BYTE    = "byte"
)

var MetricResType = []string{METRIC_RES_TYPE_GUEST, METRIC_RES_TYPE_HOST, METRIC_RES_TYPE_REDIS, METRIC_RES_TYPE_OSS, METRIC_RES_TYPE_RDS}
var MetricUnit = []string{"%", "bps", "Mbps", "Bps", "count/s", "count", "ms", "byte"}

type MetricMeasurementCreateInput struct {
	apis.StandaloneResourceCreateInput
	apis.EnabledBaseResourceCreateInput

	ResType     string `json:"res_type"`
	DisplayName string `json:"display_name"`
	Database    string `json:"database"`
}

type MetricMeasurementUpdateInput struct {
	apis.StandaloneResourceBaseUpdateInput

	ResType     string `json:"res_type"`
	DisplayName string `json:"display_name"`
}

type MetricMeasurementListInput struct {
	apis.StatusStandaloneResourceListInput
	apis.EnabledResourceBaseListInput
	apis.ScopedResourceBaseListInput

	ResType     string `json:"res_type"`
	DisplayName string `json:"display_name"`
}

type MetricFieldCreateInput struct {
	apis.StandaloneResourceCreateInput

	DisplayName string `json:"display_name"`
	Unit        string `json:"unit"`
	ValueType   string `json:"value_type"`
}

type MetricFieldUpdateInput struct {
	apis.StandaloneResourceBaseUpdateInput

	Id          string
	DisplayName string `json:"display_name"`
	Unit        string `json:"unit"`
	ValueType   string `json:"value_type"`
}

type MetricFieldListInput struct {
	apis.StatusStandaloneResourceListInput
	apis.EnabledResourceBaseListInput
	apis.ScopedResourceBaseListInput

	DisplayName string `json:"display_name"`
	Unit        string `json:"unit"`
	Scope       string `json:"scope"`
}

type MetricCreateInput struct {
	Measurement  MetricMeasurementCreateInput `json:"measurement"`
	MetricFields []MetricFieldCreateInput     `json:"metric_fields"`
	Scope        string                       `json:"scope"`
}

type MetricUpdateInput struct {
	apis.Meta

	Measurement  MetricMeasurementUpdateInput `json:"measurement"`
	MetricFields []MetricFieldUpdateInput     `json:"metric_fields"`
	Scope        string                       `json:"scope"`
}

type MetricListInput struct {
	apis.Meta

	Measurement  MetricMeasurementListInput `json:"measurement"`
	MetricFields MetricFieldListInput       `json:"metric_fields"`
	Scope        string                     `json:"scope"`
}

type MetricDetails struct {
	apis.StatusStandaloneResourceDetails
	apis.ScopedResourceBaseInfo

	MetricFields []MetricFieldDetail `json:"metric_fields"`
}

type MetricFieldDetail struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Unit        string `json:"unit"`
	Id          string `json:"id"`
}
