package system_service

import (
	"testing"
)

func TestParseSystemd(t *testing.T) {
	cases := []struct {
		in         string
		name       string
		wantLoaded bool
		wantActive bool
	}{
		{`openvswitch.service - LSB: Open vSwitch switch
   Loaded: loaded (/etc/rc.d/init.d/openvswitch; bad; vendor preset: disabled)
   Active: active (running) since Wed 2019-04-10 15:11:23 CST; 4 days ago
     Docs: man:systemd-sysv-generator(8)
  Process: 3686 ExecStart=/etc/rc.d/init.d/openvswitch start (code=exited, status=0/SUCCESS)
   CGroup: /system.slice/openvswitch.service
           ├─3722 ovsdb-server: monitoring pid 3723 (healthy)
           ├─3723 ovsdb-server /etc/openvswitch/conf.db -vconsole:emer -vsysl...
           ├─3774 ovs-vswitchd: monitoring pid 3775 (healthy)
           └─3775 ovs-vswitchd unix:/var/run/openvswitch/db.sock -vconsole:em...`, "openvswitch", true, true},
		{"Unit yunion-host.service could not be found.", "yunion-host", false, false},
	}
	for _, c := range cases {
		status := parseSystemdStatus(c.in, c.name)
		if status.Loaded != c.wantLoaded || status.Active != c.wantActive {
			t.Errorf("parseSystemdStatus %s Loaded: want %v got %v Active: want %v got %v", c.name, c.wantLoaded, status.Loaded, c.wantActive, status.Active)
		}
	}
}
