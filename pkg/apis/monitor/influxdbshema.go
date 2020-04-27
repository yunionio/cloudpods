package monitor

import "yunion.io/x/onecloud/pkg/apis"

const (
	METRIC_TAG   = "TAG"
	METRIC_FIELD = "FIELD"
)

var PROPERTY_TYPE = []string{"databases", "measurements", "metric-measurement"}

var METRIC_ATTRI = []string{METRIC_TAG, METRIC_FIELD}

type InfluxMeasurement struct {
	apis.Meta
	Database    string
	Measurement string
	TagKey      []string
	FieldKey    []string
	Unit        []string
}
