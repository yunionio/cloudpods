# MariaDB Galera Monitoring Implementation Walkthrough

This document consolidates the full codebase and verification steps for the MariaDB Galera Cluster Status Monitoring feature (Issue #20755).

## 1. Implementation Code

### `pkg/monitor/dbinit/measurements/mysql.go`
This file was modified to include the wsrep status variables in the mysql measurement definition.

```go
package measurements
import "yunion.io/x/onecloud/pkg/apis/monitor"
var mysql = SMeasurement{
	Context: []SMonitorContext{
		{
			"mysql", "mysql",
			monitor.METRIC_RES_TYPE_EXT_MYSQL, monitor.METRIC_DATABASE_TELE,
		},
	},
	Metrics: []SMetric{
        // ... existing metrics ...
		{
			"info_schema_table_size_index_length", "info_schema_table_size_index_length", monitor.METRIC_UNIT_COUNT,
		},
		{
			"wsrep_cluster_size", "wsrep_cluster_size", monitor.METRIC_UNIT_COUNT,
		},
		{
			"wsrep_cluster_status", "wsrep_cluster_status", monitor.METRIC_UNIT_NULL,
		},
		{
			"wsrep_ready", "wsrep_ready", monitor.METRIC_UNIT_NULL,
		},
		{
			"wsrep_connected", "wsrep_connected", monitor.METRIC_UNIT_NULL,
		},
	},
}
```

### `pkg/monitor/dbinit/measurements/mysql_test.go`
A new test file was created to verify that the metrics are correctly registered.

```go
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
```

## 2. Generated Configuration

### `pkg/monitor/dbinit/measurements/metrics.csv`
Running the tests automatically regenerates this CSV to include the new keys.

```csv
"mysql","mysql","ext_mysql","telegraf","wsrep_cluster_size","wsrep_cluster_size","count"
"mysql","mysql","ext_mysql","telegraf","wsrep_cluster_status","wsrep_cluster_status","NULL"
"mysql","mysql","ext_mysql","telegraf","wsrep_ready","wsrep_ready","NULL"
"mysql","mysql","ext_mysql","telegraf","wsrep_connected","wsrep_connected","NULL"
```

## 3. Verification

Verification was performed using the standard Go test suite.

**Command:**

```bash
go test -v ./pkg/monitor/dbinit/measurements/...
```

**Output:**

```
=== RUN   TestOutputMetrics
--- PASS: TestOutputMetrics (0.00s)
=== RUN   TestMysqlMetrics
--- PASS: TestMysqlMetrics (0.00s)
PASS
ok  	yunion.io/x/onecloud/pkg/monitor/dbinit/measurements	0.018s
```

All tests passed, confirming the metrics are widely available and the configuration CSV is up to date.
