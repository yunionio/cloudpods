package ansiblev2

import (
	"testing"
)

func TestInventory(t *testing.T) {
	inv := NewInventory(
		"ansible_user", "root",
		"ansible_become", "yes",
		"redis_bind_address", "0.0.0.0",
		"redis_bind_port", 9736,
	)
	inv.SetHost("redis", NewHost(
		"ansible_host", "192.168.0.248",
		"ansible_port", 2222,
	))
	inv.SetHost("nameonly", NewHost())
	child := NewHostGroup()
	child.SetHost("aws-bj01", NewHost(
		"ansible_user", "ec2-user",
		"ansible_host", "1.2.2.3",
	))
	child.SetHost("ali-hk01", NewHost(
		"ansible_user", "cloudroot",
		"ansible_host", "1.2.2.4",
	))
	inv.SetChild("netmon-agents", child)
	t.Logf("\n%s", inv.String())
}
