**What this PR does / why we need it**:

Feature: MariaDB Galera Cluster Status Monitoring

This PR addresses issue #20755 by adding support for monitoring MariaDB Galera cluster status. It registers the standard Galera status variables as metrics in the Cloudpods monitoring system, enabling alerting and dashboards for cluster health.

### Implementation Details
**1. `pkg/monitor/dbinit/measurements/mysql.go`**:
Registered standard Galera status variables to `mysql` measurement definition:
```go
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
```

**2. `pkg/monitor/dbinit/measurements/metrics.csv`**:
- Updated to include the new wsrep metrics keys.

<!--
- [ ] Smoke testing completed
- [x] Unit test written
-->

**Does this PR need to be backport to the previous release branch?**:

NONE

<!--
If no, just write "NONE".
-->

### Verification
**1. `pkg/monitor/dbinit/measurements/mysql_test.go`**:
Added unit test to verify metric presence:
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

**Automated Tests**:
Run the verification test:
```bash
go test -v -run TestMysqlMetrics ./pkg/monitor/dbinit/measurements/
```

**Output**:
```
=== RUN   TestMysqlMetrics
--- PASS: TestMysqlMetrics (0.00s)
PASS
ok  	yunion.io/x/onecloud/pkg/monitor/dbinit/measurements	0.002s
```
