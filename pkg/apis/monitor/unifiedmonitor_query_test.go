package monitor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetMeasurementTagNameIdMapByResType(t *testing.T) {
	cases := []struct {
		resType string
		want    map[string]string
	}{
		{
			resType: METRIC_RES_TYPE_GUEST,
			want:    map[string]string{"vm_name": "vm_id"},
		},
		{
			resType: METRIC_RES_TYPE_AGENT,
			want:    map[string]string{"vm_name": "vm_id"},
		},
		{
			resType: METRIC_RES_TYPE_HOST,
			want:    map[string]string{"host": "host_id"},
		},
		{
			resType: METRIC_RES_TYPE_REDIS,
			want:    map[string]string{"redis_name": "redis_id"},
		},
		{
			resType: "unknown",
			want:    nil,
		},
	}
	for _, tc := range cases {
		got := GetMeasurementTagNameIdMapByResType(tc.resType)
		assert.Equal(t, tc.want, got, "resType=%s", tc.resType)
	}
}
