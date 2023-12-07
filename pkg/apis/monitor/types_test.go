package monitor

import (
	"testing"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
)

func TestMetricQueryInput_AddMetric(t *testing.T) {
	q := NewMetricQuery("cpu")
	q.Select("usage_active").MEAN()
	q.Select("usage_active_per_core")
	log.Infof("%s", jsonutils.Marshal(q))
}
