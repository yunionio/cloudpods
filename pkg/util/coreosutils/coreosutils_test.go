package coreosutils

import (
	"fmt"
	"testing"
)

func TestSCloudConfig_String(t *testing.T) {
	cc := NewCloudConfig()
	cc.YunionInit()
	cc.SetHostname("coreos1")
	cont := "[Match]\n"
	cont += "Name=eth*\n\n"
	cont += "[Network]\n"
	cont += "DHCP=yes\n"
	runtime := true
	cc.AddUnits("00-dhcp.network", nil, nil, &runtime, cont, "", nil)
	cont = "/dev/vdb1 swap defaults 0 0\n"
	cont += "/dev/vdc1 /data ext4 defaults 2 2\n"
	cc.AddWriteFile("/etc/fstab", cont, "", "", false)
	cc.SetEtcHosts("localhost")
	cc.AddUser("core", "123456", nil, false)
	cc.AddUser("core1", "123456", []string{"ssh-rsa xxxxx", "ssh-dsa xxxxxxxxx"}, false)
	cc.AddSwap("/dev/vdb1")
	cc.AddPartition("/dev/vdc1", "/data", "ext4")
	cc.SetTimezone("Asia/Shanghai")
	fmt.Println(cc)
}
