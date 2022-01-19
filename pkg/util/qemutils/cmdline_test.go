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

package qemutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	cmd1 = `$QEMU_CMD $QEMU_CMD_KVM_ARG -cpu host,+kvm_pv_eoi -chardev socket,id=hmqmondev,port=55902,host=127.0.0.1,nodelay,server,nowait -mon chardev=hmqmondev,id=hmqmon,mode=readline -chardev socket,id=qmqmondev,port=56102,host=127.0.0.1,nodelay,server,nowait -mon chardev=qmqmondev,id=qmqmon,mode=control -rtc base=utc,clock=host,driftfix=none -daemonize -nodefaults -nodefconfig -no-kvm-pit-reinjection -global kvm-pit.lost_tick_policy=discard -machine pc,accel=kvm -k en-us -smp cpus=1,sockets=2,cores=64,maxcpus=128 -name tvm -uuid 3aad3b6b-ddf8-474a-8906-06d83ef318cb -m 1024M,slots=4,maxmem=524288M -boot order=cdn -device virtio-serial -usb -device usb-kbd -device usb-tablet -vga std -vnc :2,password -object iothread,id=iothread0 -device virtio-scsi-pci,id=scsi,num_queues=4,vectors=5 -drive file=$DISK_0,if=none,id=drive_0,cache=writeback,aio=threads,file.locking=off -device scsi-hd,drive=drive_0,bus=scsi.0,id=drive_0 -device ide-cd,drive=ide0-cd0,bus=ide.1,unit=1 -drive id=ide0-cd0,media=cdrom,if=none -netdev type=tap,id=vnet-100,ifname=vnet-100,vhost=on,vhostforce=off,script=/opt/cloud/workspace/servers/3aad3b6b-ddf8-474a-8906-06d83ef318cb/if-up-br0-vnet-100.sh,downscript=/opt/cloud/workspace/servers/3aad3b6b-ddf8-474a-8906-06d83ef318cb/if-down-br0-vnet-100.sh -device virtio-net-pci,id=netdev-vnet-100,netdev=vnet-100,mac=00:22:6a:9a:ef:8d,addr=0xf$(nic_speed 1000) -device qemu-xhci,id=usb -pidfile /opt/cloud/workspace/servers/3aad3b6b-ddf8-474a-8906-06d83ef318cb/pid -chardev socket,path=/opt/cloud/workspace/servers/3aad3b6b-ddf8-474a-8906-06d83ef318cb/qga.sock,server,nowait,id=qga0 -device virtserialport,chardev=qga0,name=org.qemu.guest_agent.0 -chardev pty,id=charserial0 -device isa-serial,chardev=charserial0,id=serial0 -incoming tcp:0:4397 -device pvpanic`
)

func TestNewCmdline(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    *Cmdline
		wantErr bool
	}{
		{
			name:    "startvm CMD",
			content: cmd1,
			want: &Cmdline{
				options: []Option{
					{"$QEMU_CMD", "$QEMU_CMD_KVM_ARG"},
					{"cpu", "host,+kvm_pv_eoi"},
					{"chardev", "socket,id=hmqmondev,port=55902,host=127.0.0.1,nodelay,server,nowait"},
					{"mon", "chardev=hmqmondev,id=hmqmon,mode=readline"},
					{"chardev", "socket,id=qmqmondev,port=56102,host=127.0.0.1,nodelay,server,nowait"},
					{"mon", "chardev=qmqmondev,id=qmqmon,mode=control"},
					{"rtc", "base=utc,clock=host,driftfix=none"},
					{"daemonize", ""},
					{"nodefaults", ""},
					{"nodefconfig", ""},
					{"no-kvm-pit-reinjection", ""},
					{"global", "kvm-pit.lost_tick_policy=discard"},
					{"machine", "pc,accel=kvm"},
					{"k", "en-us"},
					{"smp", "cpus=1,sockets=2,cores=64,maxcpus=128"},
					{"name", "tvm"},
					{"uuid", "3aad3b6b-ddf8-474a-8906-06d83ef318cb"},
					{"m", "1024M,slots=4,maxmem=524288M"},
					{"boot", "order=cdn"},
					{"device", "virtio-serial"},
					{"usb", ""},
					{"device", "usb-kbd"},
					{"device", "usb-tablet"},
					{"vga", "std"},
					{"vnc", ":2,password"},
					{"object", "iothread,id=iothread0"},
					{"device", "virtio-scsi-pci,id=scsi,num_queues=4,vectors=5"},
					{"drive", "file=$DISK_0,if=none,id=drive_0,cache=writeback,aio=threads,file.locking=off"},
					{"device", "scsi-hd,drive=drive_0,bus=scsi.0,id=drive_0"},
					{"device", "ide-cd,drive=ide0-cd0,bus=ide.1,unit=1"},
					{"drive", "id=ide0-cd0,media=cdrom,if=none"},
					{"netdev", "type=tap,id=vnet-100,ifname=vnet-100,vhost=on,vhostforce=off,script=/opt/cloud/workspace/servers/3aad3b6b-ddf8-474a-8906-06d83ef318cb/if-up-br0-vnet-100.sh,downscript=/opt/cloud/workspace/servers/3aad3b6b-ddf8-474a-8906-06d83ef318cb/if-down-br0-vnet-100.sh"},
					{"device", "virtio-net-pci,id=netdev-vnet-100,netdev=vnet-100,mac=00:22:6a:9a:ef:8d,addr=0xf$(nic_speed 1000)"},
					{"device", "qemu-xhci,id=usb"},
					{"pidfile", "/opt/cloud/workspace/servers/3aad3b6b-ddf8-474a-8906-06d83ef318cb/pid"},
					{"chardev", "socket,path=/opt/cloud/workspace/servers/3aad3b6b-ddf8-474a-8906-06d83ef318cb/qga.sock,server,nowait,id=qga0"},
					{"device", "virtserialport,chardev=qga0,name=org.qemu.guest_agent.0"},
					{"chardev", "pty,id=charserial0"},
					{"device", "isa-serial,chardev=charserial0,id=serial0"},
					{"incoming", "tcp:0:4397"},
					{"device", "pvpanic"},
				},
			},
			wantErr: false,
		},
	}
	assert := assert.New(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewCmdline(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewCmdline() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !assert.Equal(tt.want, got) {
				t.Errorf("NewCmdline() = %v, want %v", got, tt.want)
			}
			assert.Equal(got.ToString(), tt.content)
		})
	}
}
