package conditions

import (
	"time"

	"yunion.io/x/onecloud/pkg/monitor/tsdb"
)

type suggestRuleReducer struct {
	*queryReducer
	duration time.Duration
}

func NewSuggestRuleReducer(t string, duration time.Duration) Reducer {
	return &suggestRuleReducer{
		queryReducer: &queryReducer{Type: t},
		duration:     duration,
	}
}

func (s *suggestRuleReducer) Reduce(series *tsdb.TimeSeries) *float64 {
	if int(s.duration.Seconds()) > len(series.Points) {
		return nil
	}
	return s.queryReducer.Reduce(series)
}
