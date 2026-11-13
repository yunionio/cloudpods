package models

import (
	"testing"

	"github.com/stretchr/testify/require"

	"yunion.io/x/onecloud/pkg/apis/monitor"
)

func TestGuestMetricDriverUsesAgentMetrics(t *testing.T) {
	drv := GetResourceMetricDriver(monitor.METRIC_RES_TYPE_GUEST)
	require.NotNil(t, drv)

	specs := drv.GetMetricSpecs()
	require.Len(t, specs, 5)

	expected := []ResourceMetricSpec{
		{"agent_cpu", []string{"usage_active"}, []string{"cpu_usage"}},
		{"agent_mem", []string{"used_percent"}, []string{"mem_usage"}},
		{"agent_disk", []string{"used_percent"}, []string{"disk_usage"}},
		{"agent_diskio", []string{"read_bps", "write_bps"}, []string{"disk_read_rate", "disk_write_rate"}},
		{"agent_net", []string{"bps_recv", "bps_sent"}, []string{"net_in_rate", "net_out_rate"}},
	}
	require.Equal(t, expected, specs)
	require.Equal(t, "vm_id", drv.GetTagKey())
}
