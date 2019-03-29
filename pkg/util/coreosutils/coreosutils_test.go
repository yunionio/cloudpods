// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
