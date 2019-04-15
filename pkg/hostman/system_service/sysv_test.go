package system_service

import (
	"testing"
)

func TestParseSysv(t *testing.T) {
	cases := []struct {
		in1        string
		in2        string
		name       string
		wantLoaded bool
		wantActive bool
	}{
		{`openvswitch    	0:off	1:off	2:off	3:off	4:off	5:off	6:off`, `ovsdb-server is running with pid 114553
ovs-vswitchd is running with pid 114563`, "openvswitch", true, true},
		{`yunion-host    	0:off	1:off	2:on	3:on	4:on	5:on	6:off`, "", "yunion-host", true, false},
	}
	for _, c := range cases {
		status := parseSysvStatus(c.in1, c.in2, c.name)
		if status.Loaded != c.wantLoaded || status.Active != c.wantActive {
			t.Errorf("parseSysVStatus %s Loaded want %v got %v Active want %v got %v", c.name, c.wantLoaded, status.Loaded, c.wantActive, status.Active)
		}
	}
}
