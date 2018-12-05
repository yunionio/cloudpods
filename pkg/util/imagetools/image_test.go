package imagetools

import "testing"

func TestNormalizeImageInfo(t *testing.T) {
	info := NormalizeImageInfo("rhel67_20180816.qcow2", "", "", "", "")
	t.Logf("%#v", info)

	info = NormalizeImageInfo("Ubuntu_16.04.3_amd64_qingcloud_20180817.qcow2", "", "", "", "")
	t.Logf("%#v", info)

	info = NormalizeImageInfo("windows-server-2008-dc-cn-20180717", "", "", "", "")
	t.Logf("%#v", info)
}
