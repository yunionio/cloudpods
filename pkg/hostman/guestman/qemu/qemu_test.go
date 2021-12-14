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

package qemu

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_baseOptions(t *testing.T) {
	opt := newBaseOptions_x86_64()
	assert := assert.New(t)

	// test object
	assert.Equal("-object iothread,id=iothread0", opt.Object("iothread", map[string]string{"id": "iothread0"}))
	// test chardev
	assert.Equal("-chardev socket,id=test", opt.Chardev("socket", "test", ""))
	assert.Equal("-chardev socket,id=test,name=tname", opt.Chardev("socket", "test", "tname"))
	assert.Equal("-chardev socket,id=test,port=1234,host=127.0.0.1,nodelay,server,nowait", opt.MonitorChardev("test", 1234, "127.0.0.1"))
	assert.Equal([]string{
		"-chardev socket,id=testdev,port=1234,host=127.0.0.1,nodelay,server,nowait",
		"-mon chardev=testdev,id=test,mode=readline",
	}, getMonitorOptions(opt, &Monitor{
		Id:   "test",
		Port: 1234,
		Mode: "readline",
	}))
	// test memory
	assert.Equal("-m 1024M,slots=4,maxmem=524288M", opt.Memory(1024))
	// test device
	assert.Equal("-device isa-applesmc,osk=ourhardworkbythesewordsguardedpleasedontsteal(c)AppleComputerInc", opt.Device("isa-applesmc,osk=ourhardworkbythesewordsguardedpleasedontsteal(c)AppleComputerInc"))
	// test vdi spice
	assert.Equal([]string{
		"-device qxl-vga,id=video0,ram_size=141557760,vram_size=141557760",
		"-device intel-hda,id=sound0",
		"-device hda-duplex,id=sound0-codec0,bus=sound0.0,cad=0",
		"-spice port=5910,password=87654312,seamless-migration=on",
		"-device virtio-serial-pci,id=virtio-serial0,max_ports=16,bus=pcie.0",
		"-chardev spicevmc,id=vdagent,name=vdagent",
		"-device virtserialport,nr=1,bus=virtio-serial0.0,chardev=vdagent,name=com.redhat.spice.0",
		"-device ich9-usb-ehci1,id=usbspice",
		"-device ich9-usb-uhci1,masterbus=usbspice.0,firstport=0,multifunction=on",
		"-device ich9-usb-uhci2,masterbus=usbspice.0,firstport=2",
		"-device ich9-usb-uhci3,masterbus=usbspice.0,firstport=4",
		"-chardev spicevmc,id=usbredirchardev1,name=usbredir",
		"-device usb-redir,chardev=usbredirchardev1,id=usbredirdev1",
		"-chardev spicevmc,id=usbredirchardev2,name=usbredir",
		"-device usb-redir,chardev=usbredirchardev2,id=usbredirdev2",
	},
		opt.VdiSpice(5910, "pcie.0"))
	// test vnc
	assert.Equal("-vnc :5900,password", opt.VNC(5900, true))
	assert.Equal("-vnc :5900", opt.VNC(5900, false))
	// test vga
	assert.Equal("-vga std", opt.VGA("std", ""))
	assert.Equal("-vga x", opt.VGA("std", "-vga x"))
}
