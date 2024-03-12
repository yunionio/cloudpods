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

package guestman

import (
	"fmt"
	"testing"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/hostman/guestman/arch"
	"yunion.io/x/onecloud/pkg/hostman/guestman/desc"
)

type sKVMGuestInstance struct {
	SKVMGuestInstance
}

func TestSKVMGuestInstance_initGuestDesc(t *testing.T) {
	descStr := `{
  "admin_security_rules": "in:allow any",
  "bios": "BIOS",
  "boot_order": "cdn",
  "cpu": 8,
  "disks": [
    {
      "aio_mode": "threads",
      "bps": 0,
      "cache_mode": "writeback",
      "disk_id": "44566ebf-b9c5-4e8a-8b95-d1a347e83cea",
      "driver": "scsi",
      "format": "qcow2",
      "image_path": "/opt/cloud/workspace/disks/image_cache/9e050dd4-4e6c-4b71-825f-10613a59326d",
      "index": 0,
      "iops": 0,
      "is_ssd": false,
      "merge_snapshot": false,
      "migrating": true,
      "num_queues": 0,
      "path": "/opt/cloud/workspace/disks/44566ebf-b9c5-4e8a-8b95-d1a347e83cea",
      "size": 30720,
      "storage_id": "003f26ee-ba71-4d62-8bd3-057d6249df64",
      "storage_type": "local",
      "target_storage_id": "1b5895d9-3517-4e45-84f0-59db7660d840",
      "template_id": "9e050dd4-4e6c-4b71-825f-10613a59326d"
    }
  ],
  "domain": "test.yunion.io",
  "domain_id": "default",
  "host_id": "6fc10297-eb20-4a96-86a8-4b65260d6016",
  "machine": "q35",
  "mem": 8192,
  "metadata": {
    "__cpu_mode": "host",
    "__os_profile__": "{\"disk_driver\":\"scsi\",\"fs_format\":\"ext4\",\"hypervisor\":\"kvm\",\"net_driver\":\"virtio\",\"os_type\":\"Linux\"}",
    "__qemu_version": "2.12.1",
    "__ssh_port": "0",
    "__vnc_password": "Q8TyN4Td",
    "__vnc_port": "3",
    "generate_name": "test-ubuntu18",
    "hot_remove_nic": "enable",
    "hotplug_cpu_mem": "enable",
    "os_arch": "x86_32",
    "os_distribution": "Ubuntu",
    "os_name": "Linux",
    "os_version": "18.04"
  },
  "name": "test-ubuntu18",
  "nics": [
    {
      "bridge": "br1",
      "bw": 1000,
      "dns": "114.114.114.114",
      "domain": "test.yunion.io",
      "driver": "virtio",
      "gateway": "10.127.190.1",
      "ifname": "GUESTNET1-194",
      "index": 0,
      "interface": "em2",
      "ip": "10.127.190.194",
      "link_up": false,
      "mac": "00:22:a8:be:a3:13",
      "masklen": 24,
      "mtu": 1500,
      "net": "GUEST-NET190",
      "net_id": "21fbe474-77aa-472d-82e7-6f0098e15b57",
      "num_queues": 2,
      "rate": 0,
      "virtual": false,
      "vlan": 1,
      "wire_id": "b0aaa839-2f33-464a-8454-1736d0707fe3"
    }
  ],
  "os_name": "Linux",
  "pending_deleted": false,
  "project_domain": "Default",
  "secgroups": [
    {
      "id": "default",
      "name": "Default"
    }
  ],
  "security_rules": "in:allow any",
  "src_ip_check": true,
  "src_mac_check": true,
  "tenant": "system",
  "tenant_id": "55bb511b62bf47dc86e82c731005ba10",
  "uuid": "79a14b71-e752-4fb1-868e-1362ff9ed5e5",
  "vdi": "vnc",
  "vga": "qxl",
  "zone": "Wangjing",
  "zone_id": "eb40b924-7b44-4490-8197-9d695c0bae4b"
}
`

	s := &sKVMGuestInstance{
		SKVMGuestInstance: SKVMGuestInstance{
			archMan:            arch.NewArch(arch.Arch_x86_64),
			sBaseGuestInstance: newBaseGuestInstance("", nil, api.HYPERVISOR_KVM),
		},
		//manager:
	}
	Desc := new(desc.SGuestDesc)
	descJson, err := jsonutils.ParseString(descStr)
	if err != nil {
		t.Errorf("parse desc %s", err)
	}
	err = descJson.Unmarshal(Desc)
	if err != nil {
		t.Errorf("unmarshal desc %s", err)
	}
	s.Desc = Desc

	s.setPcieExtendBus()
	s.Desc.CpuDesc = &desc.SGuestCpu{
		Accel: "kvm",
	}
	//s.initCpuDesc()
	// s.initMemDesc()
	s.Desc.MemDesc = new(desc.SGuestMem)
	s.Desc.MemDesc.SizeMB = s.Desc.Mem
	memDesc := desc.NewMemDesc("memory-backend-memfd", "mem", nil, nil)
	s.Desc.MemDesc.Mem = desc.NewMemsDesc(*memDesc, nil)
	s.Desc.MemDesc.Mem.Options = map[string]string{
		"size":  fmt.Sprintf("%dM", s.Desc.Mem),
		"share": "on", "prealloc": "on",
	}
	// s.initMachineDesc()

	pciRoot, _ := s.initGuestPciControllers(true)
	err = s.initGuestPciAddresses()
	if err != nil {
		t.Error(err)
	}
	if err := s.initMachineDefaultAddresses(); err != nil {
		t.Error(err)
	}

	// vdi device for spice
	s.Desc.VdiDevice = new(desc.SGuestVdi)
	if s.IsVdiSpice() {
		s.initSpiceDevices(pciRoot)
	}

	s.initVirtioSerial(pciRoot)
	//s.initGuestVga(pciRoot)
	s.initCdromDesc()

	log.Infof("guest desc %s", jsonutils.Marshal(s.Desc).PrettyString())
	//s.initGuestDisks(pciRoot, pciBridge)
	//if err = s.initGuestNetworks(pciRoot, pciBridge); err != nil {
	//	t.Error(err)
	//}

	//s.initIsolatedDevices(pciRoot, pciBridge)
	s.initUsbController(pciRoot)
	s.initRandomDevice(pciRoot, true)
	//s.initQgaDesc()
	s.initPvpanicDesc()
	s.initIsaSerialDesc()

	if err = s.ensurePciAddresses(); err != nil {
		t.Error(err)
	}

	log.Infof("guest desc %s", jsonutils.Marshal(s.Desc).PrettyString())
}
