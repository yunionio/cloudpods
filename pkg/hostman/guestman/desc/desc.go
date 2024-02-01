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

package desc

import (
	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
)

type SGuestCpu struct {
	Cpus    uint
	Sockets uint
	Cores   uint
	Threads uint
	MaxCpus uint

	Model    string
	Vendor   string
	Level    string
	Features map[string]bool `json:",omitempty"`

	Accel string
	// emulate, passthrough, disable
	// CpuCacheMode string
}

type CpuPin struct {
	Vcpus string
	Pcpus string
}

type SMemObject struct {
	*Object
	SizeMB int64
}

type SMemDevice struct {
	Type string
	Id   string
}

type SMemSlot struct {
	SizeMB int64

	MemObj *Object
	MemDev *SMemDevice
}

type SGuestMem struct {
	Slots  uint
	MaxMem uint

	SizeMB int64
	Mem    *Object `json:",omitempty"`

	MemSlots []*SMemSlot `json:",omitempty"`
}

type SGuestHardwareDesc struct {
	Cpu     int64
	CpuDesc *SGuestCpu `json:",omitempty"`
	VcpuPin []CpuPin   `json:",omitempty"`
	// Clock   *SGuestClock `json:",omitempty"`

	Mem     int64
	MemDesc *SGuestMem `json:",omitempty"`

	Bios      string
	BootOrder string

	// supported machine type: pc, q35, virt
	Machine     string
	MachineDesc *SGuestMachine `json:",omitempty"`
	NoHpet      *bool          `json:",omitempty"` // i386 target only

	VirtioSerial *SGuestVirtioSerial

	// std virtio cirrus vmware qlx none
	Vga       string
	VgaDevice *SGuestVga `json:",omitempty"`

	// vnc or spice
	Vdi       string
	VdiDevice *SGuestVdi `json:",omitempty"`

	VirtioScsi      *SGuestVirtioScsi       `json:",omitempty"`
	PvScsi          *SGuestPvScsi           `json:",omitempty"`
	Cdroms          []*SGuestCdrom          `json:"cdroms,omitempty"`
	Floppys         []*SGuestFloppy         `json:",omitempty"`
	Disks           []*SGuestDisk           `json:",omitempty"`
	Nics            []*SGuestNetwork        `json:",omitempty"`
	IsolatedDevices []*SGuestIsolatedDevice `json:",omitempty"`

	// Random Number Generator Device
	Rng       *SGuestRng       `json:",omitempty"`
	Qga       *SGuestQga       `json:",omitempty"`
	Pvpanic   *SGuestPvpanic   `json:",omitempty"`
	IsaSerial *SGuestIsaSerial `json:",omitempty"`

	Usb            *UsbController   `json:",omitempty"`
	PCIControllers []*PCIController `json:",omitempty"`

	AnonymousPCIDevs []*PCIDevice `json:",omitempty"`

	RescueInitdPath      string `json:",omitempty"` // rescue initramfs path
	RescueKernelPath     string `json:",omitempty"` // rescue kernel path
	RescueDiskPath       string `json:",omitempty"` // rescue disk path
	RescueDiskDeviceBus  uint   `json:",omitempty"` // rescue disk device bus
	RescueDiskDeviceSlot uint   `json:",omitempty"` // rescue disk device slot
}

type SGuestIsaSerial struct {
	Pty *CharDev
	Id  string
}

type SGuestQga struct {
	Socket     *CharDev
	SerialPort *VirtSerialPort
}

type VirtSerialPort struct {
	Chardev string
	Name    string

	Options map[string]string `json:",omitempty"`
}

type SGuestPvpanic struct {
	Ioport uint // default ioport 1285(0x505)
	Id     string
}

// -device pcie-pci-bridge,id=pci.1,bus=pcie.0 \
// -device pci-bridge,id=pci.2,bus=pci.1,chassis_nr=1,addr=0x01 \

type PCIController struct {
	*PCIDevice

	CType PCI_CONTROLLER_TYPE
}

type SGuestMachine struct {
	Accel string

	// arm only
	GicVersion *string `json:",omitempty"`
}

type SGuestDisk struct {
	api.GuestdiskJsonDesc

	// disk driver virtio
	Pci *PCIDevice `json:",omitempty"`
	// disk driver scsi/pvscsi
	Scsi *SCSIDevice `json:",omitempty"`
	// disk driver ide/sata
	Ide *IDEDevice `json:",omitempty"`
}

// -device ide-cd,drive=ide0-cd0,bus=ide.1
// -drive id=ide0-cd0,media=cdrom,if=none,file=%s
// --- mac os
// -device ide-drive,drive=MacDVD,bus=ide.%d
// -drive id=MacDVD,if=none,snapshot=on,file=%s

type SGuestCdrom struct {
	Id        string
	Path      string
	Ordinal   int64
	BootIndex *int8

	Ide          *IDEDevice        `json:",omitempty"`
	Scsi         *SCSIDevice       `json:",omitempty"`
	DriveOptions map[string]string `json:",omitempty"`
}

type SGuestFloppy struct {
	Id      string
	Path    string
	Ordinal int64

	Floppy       *FloppyDevice     `json:",omitempty"`
	DriveOptions map[string]string `json:",omitempty"`
}

type SGuestNetwork struct {
	api.GuestnetworkJsonDesc
	Pci *PCIDevice `json:",omitempty"`
}

type VFIODevice struct {
	*PCIDevice

	HostAddr string
	XVga     bool
}

type SGuestIsolatedDevice struct {
	api.IsolatedDeviceJsonDesc
	VfioDevs []*VFIODevice `json:",omitempty"`
	Usb      *UsbDevice
}

type SGuestVga struct {
	*PCIDevice `json:",omitempty"`
}

type SGuestRng struct {
	*PCIDevice `json:",omitempty"`

	RngRandom *Object
}

type SoundCard struct {
	*PCIDevice `json:",omitempty"`
	Codec      *Codec
}

type Codec struct {
	Id   string
	Type string
	Cad  int
}

type SSpiceDesc struct {
	// Intel High Definition Audio
	IntelHDA *SoundCard

	// vdagent
	VdagentSerial     *SGuestVirtioSerial
	Vdagent           *CharDev
	VdagentSerialPort *VirtSerialPort

	// usb redirect
	UsbRedirct *UsbRedirctDesc

	Options map[string]string `json:",omitempty"`
}

type UsbMasterBus struct {
	Masterbus string
	Port      int
}

type UsbController struct {
	*PCIDevice

	MasterBus *UsbMasterBus `json:",omitempty"`
}

type UsbAddr struct {
	Bus  int
	Port int
}

type UsbDevice struct {
	*UsbAddr

	Id      string
	DevType string

	Options map[string]string `json:",omitempty"`
}

type UsbRedirctDesc struct {
	// EHCI adapter: Enhanced Host Controller Interface, USB2.0
	// Depends on UHCI to support full of usb devices
	EHCI1 *UsbController

	// UHCI controllers: Universal Host Controller Interface, USB 1.0, 1.1
	UHCI1 *UsbController
	UHCI2 *UsbController
	UHCI3 *UsbController

	UsbRedirDev1 *UsbRedir
	UsbRedirDev2 *UsbRedir
}

type UsbRedir struct {
	Id     string
	Source *CharDev
}

type SGuestVdi struct {
	Spice *SSpiceDesc
}

type SGuestVirtioSerial struct {
	*PCIDevice
}

type SGuestVirtioScsi struct {
	*PCIDevice

	NumQueues *uint8 `json:"num_queues"`
}

type SGuestPvScsi struct {
	*PCIDevice
}

type SGuestProjectDesc struct {
	Tenant        string
	TenantId      string
	DomainId      string
	ProjectDomain string
}

type SGuestRegionDesc struct {
	Zone     string
	Domain   string
	HostId   string
	Hostname string
}

type SGuestControlDesc struct {
	IsDaemon bool
	IsMaster bool
	IsSlave  bool

	// is volatile host meaning guest not running on this host right now
	IsVolatileHost bool

	ScalingGroupId     string
	SecurityRules      string
	AdminSecurityRules string
	SrcIpCheck         bool
	SrcMacCheck        bool

	EncryptKeyId string

	LightMode  bool // light mode
	Hypervisor string
}

type SGuestMetaDesc struct {
	Name         string
	Uuid         string
	OsName       string
	Pubkey       string
	Keypair      string
	Secgroup     string
	Flavor       string
	UserData     string
	Metadata     map[string]string
	ExtraOptions map[string]jsonutils.JSONObject
}

type SGuestContainerDesc struct {
	Containers []*api.ContainerDesc
}

type SGuestDesc struct {
	SGuestProjectDesc
	SGuestRegionDesc
	SGuestControlDesc
	SGuestHardwareDesc
	SGuestMetaDesc
	SGuestContainerDesc
}
