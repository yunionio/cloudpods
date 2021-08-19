package metricquery

import (
	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/monitor/tsdb"
)

type Metrics struct {
	SeriesTotal int64
	Series      tsdb.TimeSeriesSlice
	Metas       []tsdb.QueryResultMeta
}

type MetricQuery interface {
	ExecuteQuery() (*Metrics, error)
}

type QueryFactory func(model []*monitor.AlertCondition) (MetricQuery, error)

var queryFactories = make(map[string]QueryFactory)

func RegisterMetricQuery(typeName string, factory QueryFactory) {
	queryFactories[typeName] = factory
}

func GetQueryFactories() map[string]QueryFactory {
	return queryFactories
}
