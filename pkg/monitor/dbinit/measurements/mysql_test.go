package measurements

import (
	"testing"
)

func TestMysqlMetrics(t *testing.T) {
	metrics := map[string]bool{
		"wsrep_cluster_size":    false,
		"wsrep_cluster_status":  false,
		"wsrep_ready":           false,
		"wsrep_connected":       false,
	}

	for _, m := range mysql.Metrics {
		if _, ok := metrics[m.Name]; ok {
			metrics[m.Name] = true
		}
	}

	for name, found := range metrics {
		if !found {
			t.Errorf("metric %s not found in mysql definitions", name)
		}
	}
}
