package translator

import "github.com/influxdata/influxql"

type Translator interface {
	Translate(s influxql.Statement) (string, error)
	GetTimeRange() *influxql.TimeRange
}
