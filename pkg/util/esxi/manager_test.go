package esxi

import "testing"

func TestNewClient(t *testing.T) {
	cli, err := NewESXiClient("", "", "10.168.222.104", 443, "root", "123@Vmware")

	if err != nil {
		t.Errorf("%s", err)
	} else {
		t.Logf("%s", cli.About())

		cli.fetchDatacenters()

		host, err := cli.FindHostByIp("10.168.222.104")
		if err != nil {
			t.Errorf("find_host_by_ip %s", err)
		} else {
			t.Logf("host %s %s", host.GetAccessIp(), host.GetName())
		}
	}
}
