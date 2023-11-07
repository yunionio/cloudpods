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
	"fmt"

	"yunion.io/x/pkg/errors"
)

type PCI_CONTROLLER_TYPE string

const (
	CONTROLLER_TYPE_PCI_ROOT                    PCI_CONTROLLER_TYPE = "pci-root"
	CONTROLLER_TYPE_PCIE_ROOT                                       = "pcie-root"
	CONTROLLER_TYPE_PCI_BRIDGE                                      = "pci-bridge"
	CONTROLLER_TYPE_DMI_TO_PCI_BRIDGE                               = "dmi-to-pci-bridge"
	CONTROLLER_TYPE_PCIE_TO_PCI_BRIDGE                              = "pcie-to-pci-bridge"
	CONTROLLER_TYPE_PCIE_ROOT_PORT                                  = "pcie-root-port"
	CONTROLLER_TYPE_PCIE_SWITCH_UPSTREAM_PORT                       = "pcie-switch-upstream-port"
	CONTROLLER_TYPE_PCIE_SWITCH_DOWNSTREAM_PORT                     = "pcie-switch-downstream-port"
	CONTROLLER_TYPE_PCI_EXPANDER_BUS                                = "pci-expander-bus"
	CONTROLLER_TYPE_PCIE_EXPANDER_BUS                               = "pcie-expander-bus"
)

type PCIDevice struct {
	*PCIAddr `json:",omitempty"`

	DevType    string
	Id         string
	Controller PCI_CONTROLLER_TYPE
	Options    map[string]string `json:",omitempty"`
}

// %04x:%02x:%02x.%x, domain, bus, slot, function
type PCIAddr struct {
	Domain   uint
	Bus      uint
	Slot     uint
	Function uint

	Multi *bool
}

func (addr *PCIAddr) String() string {
	return fmt.Sprintf("%04x:%02x:%02x.%x", addr.Domain, addr.Bus, addr.Slot, addr.Function)
}

func (addr *PCIAddr) Copy() *PCIAddr {
	return &PCIAddr{
		Domain:   addr.Domain,
		Bus:      addr.Bus,
		Slot:     addr.Slot,
		Function: addr.Function,
	}
}

func (d *PCIDevice) BusStr() string {
	switch d.Controller {
	case CONTROLLER_TYPE_PCIE_ROOT, CONTROLLER_TYPE_PCIE_EXPANDER_BUS:
		return fmt.Sprintf("pcie.%d", d.Bus)
	default:
		return fmt.Sprintf("pci.%d", d.Bus)
	}
}

func (d *PCIDevice) SlotFunc() string {
	if d.Function > 0 {
		return fmt.Sprintf("0x%02x.%x", d.Slot, d.Function)
	} else {
		return fmt.Sprintf("0x%02x", d.Slot)
	}
}

func (d *PCIDevice) MultiFunction() string {
	if d.Multi == nil {
		return ""
	} else {
		if *d.Multi {
			return "multifunction=on"
		} else {
			return "multifunction=off"
		}
	}
}

func OptionsToString(options map[string]string) string {
	var cmd string
	for key, value := range options {
		if value != "" {
			cmd += fmt.Sprintf(",%s=%s", key, value)
		} else {
			cmd += fmt.Sprintf(",%s", key)
		}
	}
	return cmd
}

func (d *PCIDevice) OptionsStr() string {
	cmd := ""
	if d.PCIAddr != nil {
		cmd += fmt.Sprintf("bus=%s,addr=%s", d.BusStr(), d.SlotFunc())
		if d.Multi != nil {
			cmd += fmt.Sprintf(",%s", d.MultiFunction())
		}
	}

	cmd += OptionsToString(d.Options)
	return cmd
}

// pvscsi or virtio-scsi-pci
type SCSIAddr struct {
	// The LUN identifies the specific logical unit
	// to the SCSI initiator when combined with
	// information such as the target port identifier.
	Lun        uint
	Bus        uint
	Controller uint
}

type SCSIDevice struct {
	*SCSIAddr `json:",omitempty"`

	Id           string
	DevType      string
	ControllerId string
	Options      map[string]string `json:",omitempty"`
}

type IDEAddr struct {
	Unit       uint
	Bus        uint
	Controller uint
}

type IDEDevice struct {
	*IDEAddr

	Id      string
	DevType string
	Options map[string]string `json:",omitempty"`
}

type FloppyAddr struct {
	Bus        uint
	Controller uint
}

type FloppyDevice struct {
	*FloppyAddr

	Id      string
	DevType string
	Options map[string]string `json:",omitempty"`
}

type Object struct {
	ObjType string
	Id      string
	Options map[string]string `json:",omitempty"`
}

type CharDev struct {
	Backend string
	Id      string
	Name    string
	Options map[string]string `json:",omitempty"`
}

const (
	PCI_ADDRESS_SLOT_MAX = 31
)

const QEMU_GUEST_PCIE_BUS_MAX = 16

type SGuestPCIAddressSlot struct {
	Function uint8
}

type SGuestPCIAddressBus struct {
	Slots []*SGuestPCIAddressSlot

	/* usually 0,0 or 0,31, or 1,31 */
	MinSlot uint
	MaxSlot uint

	Contorller PCI_CONTROLLER_TYPE
	// Hotplugable bool
}

type SGuestPCIAddresses struct {
	Buses []*SGuestPCIAddressBus
}

func (b *SGuestPCIAddresses) ReleasePCIAddress(addr *PCIAddr) error {
	if int(addr.Bus+1) > len(b.Buses) {
		return errors.Errorf("release pci address bus %02x out of range", addr.Bus)
	}
	bus := b.Buses[addr.Bus]
	return bus.ReleaseSlotFunction(addr.Slot, addr.Function)
}

func (b *SGuestPCIAddresses) IsAddrInUse(addr *PCIAddr) (error, bool) {
	if int(addr.Bus+1) > len(b.Buses) {
		return errors.Errorf("release pci address bus %02x out of range", addr.Bus), false
	}
	bus := b.Buses[addr.Bus]
	return bus.IsSlotFunctionInUse(addr.Slot, addr.Function)
}

func (b *SGuestPCIAddressBus) EnsureSlotFunction(slot, function uint) error {
	if b.Slots == nil {
		b.Slots = make([]*SGuestPCIAddressSlot, 0, b.MaxSlot+1)
	}
	if slot < b.MinSlot || slot > b.MaxSlot {
		return errors.Errorf("slot %02x out of range %02x~%02x", slot, b.MinSlot, b.MaxSlot)
	}
	if function >= 8 {
		return errors.Errorf("function %x out of range 0~7", function)
	}

	if int(slot+1) > len(b.Slots) {
		b.Slots = b.Slots[:slot+1]
	}

	if b.Slots[slot] == nil {
		b.Slots[slot] = new(SGuestPCIAddressSlot)
	}

	if (b.Slots[slot].Function & (1 << function)) > 0 {
		return errors.Errorf("slot %02x function %x is in use", slot, function)
	}

	b.setSlotFunction(slot, function)
	return nil
}

func (b *SGuestPCIAddressBus) setSlotFunction(slot, function uint) {
	b.Slots[slot].Function |= 1 << function
}

func (b *SGuestPCIAddressBus) IsSlotFunctionInUse(slot, function uint) (error, bool) {
	if slot < b.MinSlot || slot > b.MaxSlot {
		return errors.Errorf("slot %02x out of range %02x~%02x", slot, b.MinSlot, b.MaxSlot), false
	}
	if function >= 8 {
		return errors.Errorf("function %x out of range 0~7", function), false
	}
	return nil, (b.Slots[slot].Function & (1 << function)) > 0
}

func (b *SGuestPCIAddressBus) ReleaseSlotFunction(slot, function uint) error {
	if slot < b.MinSlot || slot > b.MaxSlot {
		return errors.Errorf("slot %02x out of range %02x~%02x", slot, b.MinSlot, b.MaxSlot)
	}
	if function >= 8 {
		return errors.Errorf("function %x out of range 0~7", function)
	}

	b.Slots[slot].Function &= ^(1 << function)
	return nil
}

func (b *SGuestPCIAddressBus) FindNextUnusedSlot(function uint) int {
	if b.Slots == nil {
		b.Slots = make([]*SGuestPCIAddressSlot, 0, b.MaxSlot+1)
	}

	var slot = b.MinSlot
	for ; slot <= b.MaxSlot; slot++ {
		if int(slot+1) > len(b.Slots) {
			b.Slots = b.Slots[:slot+1]
		}
		if b.Slots[slot] == nil {
			b.Slots[slot] = new(SGuestPCIAddressSlot)
		}

		if (b.Slots[slot].Function & (1 << function)) > 0 {
			continue
		} else {
			break
		}
	}

	if slot > b.MaxSlot {
		return -1
	}
	return int(slot)
}

func NewGuestPCIAddressBus(controller PCI_CONTROLLER_TYPE) (*SGuestPCIAddressBus, error) {
	bus := &SGuestPCIAddressBus{
		Contorller: controller,
	}
	switch controller {
	case CONTROLLER_TYPE_PCI_ROOT,
		CONTROLLER_TYPE_PCIE_ROOT,
		CONTROLLER_TYPE_PCI_BRIDGE,
		CONTROLLER_TYPE_PCIE_TO_PCI_BRIDGE:
		bus.MaxSlot = PCI_ADDRESS_SLOT_MAX
		bus.MinSlot = 1
	case CONTROLLER_TYPE_PCIE_ROOT_PORT, CONTROLLER_TYPE_PCIE_SWITCH_DOWNSTREAM_PORT:
		bus.MaxSlot = 0
		bus.MinSlot = 0
	case CONTROLLER_TYPE_DMI_TO_PCI_BRIDGE,
		CONTROLLER_TYPE_PCIE_SWITCH_UPSTREAM_PORT,
		CONTROLLER_TYPE_PCI_EXPANDER_BUS,
		CONTROLLER_TYPE_PCIE_EXPANDER_BUS:
		bus.MaxSlot = PCI_ADDRESS_SLOT_MAX
		bus.MinSlot = 0
	default:
		return nil, errors.Errorf("unknown controller type")
	}
	return bus, nil
}

func NewPCIController(controller PCI_CONTROLLER_TYPE) *PCIController {
	return &PCIController{
		CType: controller,
	}
}

func NewPCIDevice(controller PCI_CONTROLLER_TYPE, deviceType, id string) *PCIDevice {
	return &PCIDevice{
		Id:         id,
		Controller: controller,
		DevType:    deviceType,
	}
}

func NewVfioDevice(controller PCI_CONTROLLER_TYPE, deviceType, id, hostAddr string, hasXVga bool) *VFIODevice {
	return &VFIODevice{
		PCIDevice: NewPCIDevice(controller, deviceType, id),
		HostAddr:  hostAddr,
		XVga:      hasXVga,
	}
}

func NewScsiDevice(controller, deviceType, id string) *SCSIDevice {
	return &SCSIDevice{
		ControllerId: controller,
		DevType:      deviceType,
		Id:           id,
	}
}

func NewIdeDevice(deviceType, id string) *IDEDevice {
	return &IDEDevice{
		Id:      id,
		DevType: deviceType,
	}
}

func NewUsbDevice(deviceType, id string) *UsbDevice {
	return &UsbDevice{
		Id:      id,
		DevType: deviceType,
	}
}

func NewObject(objType, id string) *Object {
	return &Object{
		ObjType: objType,
		Id:      id,
	}
}

func NewCharDev(backend, id, name string) *CharDev {
	return &CharDev{
		Backend: backend,
		Id:      id,
		Name:    name,
	}
}

func NewUsbController(masterbus string, port int) *UsbController {
	uc := &UsbController{}
	if len(masterbus) > 0 {
		uc.MasterBus = &UsbMasterBus{
			Masterbus: masterbus,
			Port:      port,
		}
	}
	return uc
}
