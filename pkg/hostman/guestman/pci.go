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
	"strconv"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/hostman/guestman/desc"
	"yunion.io/x/onecloud/pkg/hostman/guestman/qemu"
	"yunion.io/x/onecloud/pkg/hostman/monitor"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/scheduler/api"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
)

func (s *SKVMGuestInstance) addPCIController(controllerType, bus desc.PCI_CONTROLLER_TYPE) *desc.PCIController {
	cont := desc.NewPCIController(controllerType)
	cont.PCIDevice = desc.NewPCIDevice(bus, "", "")
	if s.Desc.PCIControllers == nil {
		s.Desc.PCIControllers = make([]*desc.PCIController, 0)
	}
	s.Desc.PCIControllers = append(s.Desc.PCIControllers, cont)
	return cont
}

func (s *SKVMGuestInstance) pciControllerFind(cont desc.PCI_CONTROLLER_TYPE) *desc.PCIController {
	for i := 0; i < len(s.Desc.PCIControllers); i++ {
		if s.Desc.PCIControllers[i].CType == cont {
			return s.Desc.PCIControllers[i]
		}
	}
	return nil
}

func (s *SKVMGuestInstance) initGuestDesc() error {
	err := s.initCpuDesc(0)
	if err != nil {
		return err
	}
	s.initMemDesc(s.Desc.Mem)
	s.initMachineDesc()

	pciRoot, pciBridge := s.initGuestPciControllers(s.manager.host.IsKvmSupport())
	err = s.initGuestPciAddresses()
	if err != nil {
		return errors.Wrap(err, "init guest pci addresses")
	}

	err = s.initGuestDevicesDesc(pciRoot, pciBridge)
	if err != nil {
		return err
	}

	if err := s.initMachineDefaultAddresses(); err != nil {
		return errors.Wrap(err, "init machine default devices")
	}
	return s.ensurePciAddresses()
}

func (s *SKVMGuestInstance) initGuestDevicesDesc(pciRoot, pciBridge *desc.PCIController) error {
	// vdi device for spice
	s.Desc.VdiDevice = new(desc.SGuestVdi)
	if s.IsVdiSpice() {
		s.initSpiceDevices(pciRoot)
	}

	s.initVirtioSerial(pciRoot)
	s.initGuestVga(pciRoot)
	s.initCdromDesc()
	s.initFloppyDesc()
	s.initGuestDisks(pciRoot, pciBridge, false)
	if err := s.initGuestNetworks(pciRoot, pciBridge); err != nil {
		return errors.Wrap(err, "init guest networks")
	}

	s.initIsolatedDevices(pciRoot, pciBridge)
	s.initUsbController(pciRoot)
	s.initRandomDevice(pciRoot, options.HostOptions.EnableVirtioRngDevice)
	s.initQgaDesc()
	s.initPvpanicDesc()
	s.initIsaSerialDesc()
	return nil
}

func (s *SKVMGuestInstance) loadGuestPciAddresses() error {
	err := s.initGuestPciAddresses()
	if err != nil {
		return errors.Wrap(err, "init guest pci addresses")
	}

	if err := s.initMachineDefaultAddresses(); err != nil {
		return errors.Wrap(err, "init machine default devices")
	}
	err = s.ensurePciAddresses()
	if err != nil {
		return errors.Wrap(err, "load desc ensure pci address")
	}
	if err = SaveLiveDesc(s, s.Desc); err != nil {
		return errors.Wrap(err, "loadGuestPciAddresses save desc")
	}
	return nil
}

func (s *SKVMGuestInstance) initMachineDefaultAddresses() error {
	switch s.Desc.Machine {
	case "pc":
		I440FXSlot1 := &desc.PCIDevice{
			PCIAddr:    &desc.PCIAddr{Bus: 0, Slot: 1},
			Controller: desc.CONTROLLER_TYPE_PCI_ROOT,
		}
		// ISA bridge
		multiFunc := true
		if err := s.ensureDevicePciAddress(I440FXSlot1, 0, &multiFunc); err != nil {
			return err
		}
		// primary IDE controller
		if err := s.ensureDevicePciAddress(I440FXSlot1, 1, nil); err != nil {
			return err
		}
		// PIIX3 USB controller
		if err := s.ensureDevicePciAddress(I440FXSlot1, 2, nil); err != nil {
			return err
		}
		// ACPI (power management) and SMBus controller
		if err := s.ensureDevicePciAddress(I440FXSlot1, 3, nil); err != nil {
			return err
		}
	case "q35":
		Q35Slot31 := &desc.PCIDevice{
			PCIAddr:    &desc.PCIAddr{Bus: 0, Slot: 31},
			Controller: desc.CONTROLLER_TYPE_PCIE_ROOT,
		}
		// ISA bridge
		multiFunc := true
		if err := s.ensureDevicePciAddress(Q35Slot31, 0, &multiFunc); err != nil {
			return err
		}
		// primary SATA controller
		if err := s.ensureDevicePciAddress(Q35Slot31, 2, nil); err != nil {
			return err
		}
		// SMBus
		if err := s.ensureDevicePciAddress(Q35Slot31, 3, nil); err != nil {
			return err
		}
		// usb controllers
		Q35Slot29 := &desc.PCIDevice{
			PCIAddr:    &desc.PCIAddr{Bus: 0, Slot: 29},
			Controller: desc.CONTROLLER_TYPE_PCIE_ROOT,
		}
		multiFunc = true
		if err := s.ensureDevicePciAddress(Q35Slot29, 0, &multiFunc); err != nil {
			return err
		}
		if err := s.ensureDevicePciAddress(Q35Slot29, 1, nil); err != nil {
			return err
		}
		if err := s.ensureDevicePciAddress(Q35Slot29, 2, nil); err != nil {
			return err
		}
		if err := s.ensureDevicePciAddress(Q35Slot29, 7, nil); err != nil {
			return err
		}
	case "virt":
		// do nothing
	}
	return nil
}

func (s *SKVMGuestInstance) isMachineDefaultAddress(pciAddr *desc.PCIAddr) bool {
	switch s.Desc.Machine {
	case "pc":
		if pciAddr.Bus == 0 && pciAddr.Slot == 1 && pciAddr.Function < 4 {
			return true
		}
	case "q35":
		if pciAddr.Bus == 0 && pciAddr.Slot == 29 {
			switch pciAddr.Function {
			case 0, 1, 2, 7:
				return true
			}
		} else if pciAddr.Bus == 0 && pciAddr.Slot == 31 {
			switch pciAddr.Function {
			case 0, 2, 3:
				return true
			}
		}
	}
	return false
}

func (s *SKVMGuestInstance) initGuestPciControllers(pciExtend bool) (*desc.PCIController, *desc.PCIController) {
	var pciRoot, pciBridge *desc.PCIController
	if len(s.Desc.PCIControllers) > 0 {
		pciRoot = s.Desc.PCIControllers[0]
		if len(s.Desc.PCIControllers) > 1 {
			pciBridge = s.Desc.PCIControllers[1]
		}
		return pciRoot, pciBridge
	}
	if s.isPcie() {
		pciRoot = s.addPCIController(desc.CONTROLLER_TYPE_PCIE_ROOT, "")
		if pciExtend {
			s.addPCIController(desc.CONTROLLER_TYPE_PCIE_TO_PCI_BRIDGE, desc.CONTROLLER_TYPE_PCIE_ROOT)
			pciBridge = s.addPCIController(desc.CONTROLLER_TYPE_PCI_BRIDGE, desc.CONTROLLER_TYPE_PCIE_TO_PCI_BRIDGE)
			for i := 0; i < s.vfioDevCount()+options.HostOptions.PcieRootPortCount; i++ {
				s.addPCIController(desc.CONTROLLER_TYPE_PCIE_ROOT_PORT, desc.CONTROLLER_TYPE_PCIE_ROOT)
			}
		}
	} else {
		pciRoot = s.addPCIController(desc.CONTROLLER_TYPE_PCI_ROOT, "")
	}
	return pciRoot, pciBridge
}

func (s *SKVMGuestInstance) initGuestPciAddresses() error {
	s.pciAddrs = &desc.SGuestPCIAddresses{
		Buses: make([]*desc.SGuestPCIAddressBus, len(s.Desc.PCIControllers)),
	}
	var err error
	for i := 0; i < len(s.pciAddrs.Buses); i++ {
		s.pciAddrs.Buses[i], err = desc.NewGuestPCIAddressBus(s.Desc.PCIControllers[i].CType)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *SKVMGuestInstance) cleanGuestPciAddressed() {
	s.pciAddrs = nil
}

func (s *SKVMGuestInstance) initRandomDevice(pciRoot *desc.PCIController, enableVirtioRngDevice bool) {
	if !enableVirtioRngDevice {
		return
	}

	var randev string
	if fileutils2.Exists("/dev/urandom") {
		randev = "/dev/urandom"
	} else if fileutils2.Exists("/dev/random") {
		randev = "/dev/random"
	} else {
		return
	}

	s.Desc.Rng = &desc.SGuestRng{
		PCIDevice: desc.NewPCIDevice(pciRoot.CType, "virtio-rng-pci", "random0"),
		RngRandom: desc.NewObject("rng-random", "rng0"),
	}
	s.Desc.Rng.Options = map[string]string{
		"max-bytes": "1024",
		"period":    "1000",
	}
	s.Desc.Rng.RngRandom.Options = map[string]string{
		"filename": randev,
	}
}

func (s *SKVMGuestInstance) initUsbController(pciRoot *desc.PCIController) {
	contType := s.getUsbControllerType()
	s.Desc.Usb = &desc.UsbController{
		PCIDevice: desc.NewPCIDevice(pciRoot.CType, contType, "usb"),
	}
}

func (s *SKVMGuestInstance) initGuestNetworks(pciRoot, pciBridge *desc.PCIController) error {
	cont := pciRoot
	if pciBridge != nil {
		cont = pciBridge
	}

	if s.GetOsName() == OS_NAME_ANDROID {
		s.Desc.Nics = s.Desc.Nics[:1]
	}

	for i := 0; i < len(s.Desc.Nics); i++ {
		if err := s.generateNicScripts(s.Desc.Nics[i]); err != nil {
			return errors.Wrapf(err, "generateNicScripts for nic: %v", s.Desc.Nics[i])
		}
		s.Desc.Nics[i].UpscriptPath = s.getNicUpScriptPath(s.Desc.Nics[i])
		s.Desc.Nics[i].DownscriptPath = s.getNicDownScriptPath(s.Desc.Nics[i])

		if s.Desc.Nics[i].Driver != "vfio-pci" {
			switch s.GetOsName() {
			case OS_NAME_MACOS:
				vectors := 0
				s.Desc.Nics[i].Vectors = &vectors
				s.Desc.Nics[i].Driver = "e1000"
			case OS_NAME_VMWARE:
				s.Desc.Nics[i].Driver = "vmxnet3"
			}
			if s.Desc.Nics[i].NumQueues > 1 {
				vectors := s.Desc.Nics[i].NumQueues*2 + 1
				s.Desc.Nics[i].Vectors = &vectors
			}

			id := fmt.Sprintf("netdev-%s", s.Desc.Nics[i].Ifname)
			switch s.Desc.Nics[i].Driver {
			case "virtio":
				s.Desc.Nics[i].Pci = desc.NewPCIDevice(cont.CType, "virtio-net-pci", id)
			case "e1000":
				s.Desc.Nics[i].Pci = desc.NewPCIDevice(cont.CType, "e1000-82545em", id)
			case "vmxnet3":
				s.Desc.Nics[i].Pci = desc.NewPCIDevice(cont.CType, "vmxnet3", id)
			}
		}
	}
	return nil
}

func (s *SKVMGuestInstance) initIsolatedDevices(pciRoot, pciBridge *desc.PCIController) {
	cType := s.getVfioDeviceHotPlugPciControllerType()
	if cType == nil {
		cType = &pciRoot.CType
		if pciBridge != nil {
			cType = &pciBridge.CType
		}
	}

	manager := s.manager.GetHost().GetIsolatedDeviceManager()
	for i := 0; i < len(s.Desc.IsolatedDevices); i++ {
		dev := manager.GetDeviceByAddr(s.Desc.IsolatedDevices[i].Addr)
		if s.Desc.IsolatedDevices[i].DevType == api.USB_TYPE {
			s.Desc.IsolatedDevices[i].Usb = desc.NewUsbDevice("usb-host", dev.GetQemuId())
			s.Desc.IsolatedDevices[i].Usb.Options = dev.GetPassthroughOptions()
		} else {
			id := dev.GetQemuId()
			s.Desc.IsolatedDevices[i].VfioDevs = make([]*desc.VFIODevice, 0)
			vfioDev := desc.NewVfioDevice(
				*cType, "vfio-pci", id, dev.GetAddr(), dev.GetDeviceType() == api.GPU_VGA_TYPE,
			)
			s.Desc.IsolatedDevices[i].VfioDevs = append(s.Desc.IsolatedDevices[i].VfioDevs, vfioDev)

			groupDevAddrs := dev.GetIOMMUGroupRestAddrs()
			for j := 0; j < len(groupDevAddrs); j++ {
				gid := fmt.Sprintf("%s-%d", id, j+1)
				vfioDev = desc.NewVfioDevice(*cType, "vfio-pci", gid, groupDevAddrs[j], false)
				s.Desc.IsolatedDevices[i].VfioDevs = append(
					s.Desc.IsolatedDevices[i].VfioDevs, vfioDev,
				)
			}
		}
	}
}

func (s *SKVMGuestInstance) initCdromDesc() {
	if s.Desc.Cdroms == nil {
		s.Desc.Cdroms = make([]*desc.SGuestCdrom, options.HostOptions.CdromCount)
		for i := range s.Desc.Cdroms {
			s.Desc.Cdroms[i] = new(desc.SGuestCdrom)
			s.Desc.Cdroms[i].Ordinal = int64(i)
		}
	}
	for i := range s.Desc.Cdroms {
		s.archMan.GenerateCdromDesc(s.GetOsName(), s.Desc.Cdroms[i])
	}
}

func (s *SKVMGuestInstance) initFloppyDesc() {
	if s.Desc.Machine != "pc" || s.GetOsName() != OS_NAME_WINDOWS {
		return
	}
	if s.Desc.Floppys == nil {
		s.Desc.Floppys = make([]*desc.SGuestFloppy, options.HostOptions.FloppyCount)
		for i := range s.Desc.Floppys {
			s.Desc.Floppys[i] = new(desc.SGuestFloppy)
			s.Desc.Floppys[i].Ordinal = int64(i)
		}
	}
	for i := range s.Desc.Floppys {
		s.archMan.GenerateFloppyDesc(s.GetOsName(), s.Desc.Floppys[i])
	}
}

func (s *SKVMGuestInstance) initGuestDisks(pciRoot, pciBridge *desc.PCIController, loadGuest bool) {
	if !loadGuest {
		hasVirtioScsi, hasPvScsi := s.fixDiskDriver()
		if hasVirtioScsi && s.Desc.VirtioScsi == nil {
			s.Desc.VirtioScsi = &desc.SGuestVirtioScsi{
				PCIDevice: desc.NewPCIDevice(pciRoot.CType, "virtio-scsi-pci", "scsi"),
			}
		} else if hasPvScsi && s.Desc.PvScsi == nil {
			s.Desc.PvScsi = &desc.SGuestPvScsi{
				PCIDevice: desc.NewPCIDevice(pciRoot.CType, "pvscsi", "scsi"),
			}
		}
	}

	cont := pciRoot
	if pciBridge != nil {
		cont = pciBridge
	}
	for i := 0; i < len(s.Desc.Disks); i++ {
		devType := qemu.GetDiskDeviceModel(s.Desc.Disks[i].Driver)
		id := fmt.Sprintf("drive_%d", s.Desc.Disks[i].Index)
		switch s.Desc.Disks[i].Driver {
		case DISK_DRIVER_VIRTIO:
			if s.Desc.Disks[i].Pci == nil {
				s.Desc.Disks[i].Pci = desc.NewPCIDevice(cont.CType, devType, id)
			}
		case DISK_DRIVER_SCSI:
			s.Desc.Disks[i].Scsi = desc.NewScsiDevice(s.Desc.VirtioScsi.Id, devType, id)
		case DISK_DRIVER_PVSCSI:
			s.Desc.Disks[i].Scsi = desc.NewScsiDevice(s.Desc.PvScsi.Id, devType, id)
		case DISK_DRIVER_IDE:
			s.Desc.Disks[i].Ide = desc.NewIdeDevice(devType, id)
		case DISK_DRIVER_SATA: // -device ahci,id=ahci pci device
			s.Desc.Disks[i].Ide = desc.NewIdeDevice(devType, id)
		}
	}
}

func (s *SKVMGuestInstance) fixDiskDriver() (bool, bool) {
	var virtioScsi, pvScsi = false, false
	isArm := s.manager.host.IsAarch64()
	osname := s.GetOsName()

	for i := 0; i < len(s.Desc.Disks); i++ {
		if isArm && (s.Desc.Disks[i].Driver == DISK_DRIVER_IDE ||
			s.Desc.Disks[i].Driver == DISK_DRIVER_SATA) {
			s.Desc.Disks[i].Driver = DISK_DRIVER_SCSI
		} else if osname == OS_NAME_MACOS {
			s.Desc.Disks[i].Driver = DISK_DRIVER_SATA
		}

		if s.Desc.Disks[i].Driver == DISK_DRIVER_SCSI {
			virtioScsi = true
		} else if s.Desc.Disks[i].Driver == DISK_DRIVER_PVSCSI {
			pvScsi = true
		}
	}
	return virtioScsi, pvScsi
}

func (s *SKVMGuestInstance) initVirtioSerial(pciRoot *desc.PCIController) {
	s.Desc.VirtioSerial = new(desc.SGuestVirtioSerial)
	s.Desc.VirtioSerial.PCIDevice = desc.NewPCIDevice(pciRoot.CType, "virtio-serial-pci", "virtio-serial0")
}

func (s *SKVMGuestInstance) initGuestVga(pciRoot *desc.PCIController) {
	var isAarch64 = s.manager.host.IsAarch64()
	if s.gpusHasVga() {
		s.Desc.Vga = "none"
	} else if isAarch64 {
		s.Desc.Vga = "virtio-gpu"
	} else if s.Desc.Vga == "" {
		s.Desc.Vga = "std"
	}
	s.Desc.VgaDevice = new(desc.SGuestVga)

	var vgaDevName string
	var options map[string]string
	switch s.Desc.Vga {
	case "std":
		vgaDevName = "VGA"
	case "qxl":
		vgaDevName = "qxl-vga"
		options = map[string]string{
			"ram_size":  "141557760",
			"vram_size": "141557760",
		}
	case "cirros":
		vgaDevName = "cirrus-vga"
	case "vmware":
		vgaDevName = "vmware-svga"
	case "virtio":
		vgaDevName = "virtio-vga"
	case "virtio-gpu":
		vgaDevName = "virtio-gpu-pci"
		options = map[string]string{
			"max_outputs": "1",
		}
	case "", "none":
		vgaDevName = "none"
	}
	if vgaDevName != "none" {
		s.Desc.VgaDevice.PCIDevice = desc.NewPCIDevice(pciRoot.CType, vgaDevName, "video0")
		s.Desc.VgaDevice.PCIDevice.Options = options
	}
}

func (s *SKVMGuestInstance) initSpiceDevices(pciRoot *desc.PCIController) {
	spice := new(desc.SSpiceDesc)
	spice.IntelHDA = &desc.SoundCard{
		PCIDevice: desc.NewPCIDevice(pciRoot.CType, "intel-hda", "sound0"),
		Codec: &desc.Codec{
			Id:   "sound0-codec0",
			Type: "hda-duplex",
			Cad:  0,
		},
	}
	var ehciId = "usbspice"
	spice.UsbRedirct = &desc.UsbRedirctDesc{
		EHCI1: desc.NewUsbController("", -1),
		UHCI1: desc.NewUsbController(ehciId, 0),
		UHCI2: desc.NewUsbController(ehciId, 2),
		UHCI3: desc.NewUsbController(ehciId, 4),
	}
	spice.UsbRedirct.UsbRedirDev1 = &desc.UsbRedir{
		Id:     "usbredirdev1",
		Source: desc.NewCharDev("spicevmc", "usbredirchardev1", "usbredir"),
	}
	spice.UsbRedirct.UsbRedirDev2 = &desc.UsbRedir{
		Id:     "usbredirdev2",
		Source: desc.NewCharDev("spicevmc", "usbredirchardev2", "usbredir"),
	}
	spice.UsbRedirct.EHCI1.PCIDevice = desc.NewPCIDevice(pciRoot.CType, "ich9-usb-ehci1", ehciId)
	spice.UsbRedirct.UHCI1.PCIDevice = desc.NewPCIDevice(pciRoot.CType, "ich9-usb-uhci1", "uhci1")
	spice.UsbRedirct.UHCI2.PCIDevice = desc.NewPCIDevice(pciRoot.CType, "ich9-usb-uhci2", "uhci2")
	spice.UsbRedirct.UHCI3.PCIDevice = desc.NewPCIDevice(pciRoot.CType, "ich9-usb-uhci3", "uhci3")

	spice.VdagentSerial = &desc.SGuestVirtioSerial{
		PCIDevice: desc.NewPCIDevice(pciRoot.CType, "virtio-serial-pci", "vdagent-serial0"),
	}
	spice.Vdagent = desc.NewCharDev("spicevmc", "vdagent", "vdagent")
	spice.VdagentSerialPort = &desc.VirtSerialPort{
		Chardev: "vdagent",
		Name:    "com.redhat.spice.0",
		Options: map[string]string{
			"nr": "1",
		},
	}
	spice.Options = map[string]string{
		"disable-ticketing":  "off",
		"seamless-migration": "on",
	}

	s.Desc.VdiDevice.Spice = spice
}

func (s *SKVMGuestInstance) ensureDevicePciAddress(
	dev *desc.PCIDevice, function int, multiFunc *bool,
) error {
	if function >= 8 {
		return errors.Errorf("invalid function %x", function)
	} else if function < 0 {
		// if function not given, give them default function 0
		if dev.PCIAddr != nil {
			function = int(dev.Function)
		} else {
			function = 0
		}
	}

	if dev.PCIAddr != nil {
		dev.Function = uint(function)
		return s.pciAddrs.Buses[dev.Bus].EnsureSlotFunction(dev.Slot, dev.Function)
	}

	bus, slot, found := s.findUnusedSlotForController(dev.Controller, function)
	if !found {
		return errors.Errorf("no valid pci address found ?")
	}

	s.pciAddrs.Buses[bus].EnsureSlotFunction(uint(slot), uint(function))
	dev.PCIAddr = &desc.PCIAddr{
		Bus:      uint(bus),
		Slot:     uint(slot),
		Function: uint(function),
		Multi:    multiFunc,
	}
	return nil
}

func (s *SKVMGuestInstance) findUnusedSlotForController(cont desc.PCI_CONTROLLER_TYPE, function int) (int, int, bool) {
	var (
		bus, slot = 0, 0
		found     = false
	)
	for ; bus < len(s.pciAddrs.Buses); bus++ {
		if cont == s.pciAddrs.Buses[bus].Contorller {
			slot = s.pciAddrs.Buses[bus].FindNextUnusedSlot(uint(function))
			if slot >= 0 {
				found = true
				break
			}
		}
	}
	return bus, slot, found
}

func (s *SKVMGuestInstance) ensurePciAddresses() error {
	var err error
	if s.Desc.VgaDevice != nil && s.Desc.VgaDevice.PCIDevice != nil {
		err = s.ensureDevicePciAddress(s.Desc.VgaDevice.PCIDevice, -1, nil)
		if err != nil {
			return errors.Wrap(err, "ensure vga pci address")
		}
	}
	if s.Desc.VirtioSerial != nil {
		err = s.ensureDevicePciAddress(s.Desc.VirtioSerial.PCIDevice, -1, nil)
	}

	if s.Desc.VdiDevice != nil && s.Desc.VdiDevice.Spice != nil {
		err = s.ensureDevicePciAddress(s.Desc.VdiDevice.Spice.IntelHDA.PCIDevice, -1, nil)
		if err != nil {
			return errors.Wrap(err, "ensure vdi hda pci address")
		}

		err = s.ensureDevicePciAddress(s.Desc.VdiDevice.Spice.VdagentSerial.PCIDevice, -1, nil)
		if err != nil {
			return errors.Wrap(err, "ensure vdagent serial pci address")
		}

		multiFunc := true
		err = s.ensureDevicePciAddress(s.Desc.VdiDevice.Spice.UsbRedirct.UHCI1.PCIDevice, 0, &multiFunc)
		if err != nil {
			return errors.Wrap(err, "ensure vdi usb ehci1 pci address")
		}
		s.Desc.VdiDevice.Spice.UsbRedirct.UHCI2.PCIAddr = s.Desc.VdiDevice.Spice.UsbRedirct.UHCI1.PCIAddr.Copy()
		err = s.ensureDevicePciAddress(s.Desc.VdiDevice.Spice.UsbRedirct.UHCI2.PCIDevice, 1, &multiFunc)
		if err != nil {
			return errors.Wrap(err, "ensure vdi usb ehci1 pci address")
		}
		s.Desc.VdiDevice.Spice.UsbRedirct.UHCI3.PCIAddr = s.Desc.VdiDevice.Spice.UsbRedirct.UHCI1.PCIAddr.Copy()
		err = s.ensureDevicePciAddress(s.Desc.VdiDevice.Spice.UsbRedirct.UHCI3.PCIDevice, 2, &multiFunc)
		if err != nil {
			return errors.Wrap(err, "ensure vdi usb ehci1 pci address")
		}
		s.Desc.VdiDevice.Spice.UsbRedirct.EHCI1.PCIAddr = s.Desc.VdiDevice.Spice.UsbRedirct.UHCI1.PCIAddr.Copy()
		err = s.ensureDevicePciAddress(s.Desc.VdiDevice.Spice.UsbRedirct.EHCI1.PCIDevice, 7, &multiFunc)
		if err != nil {
			return errors.Wrap(err, "ensure vdi usb ehci1 pci address")
		}
	}

	if s.Desc.VirtioScsi != nil {
		err = s.ensureDevicePciAddress(s.Desc.VirtioScsi.PCIDevice, -1, nil)
		if err != nil {
			return errors.Wrap(err, "ensure virtio scsi pci address")
		}
	} else if s.Desc.PvScsi != nil {
		err = s.ensureDevicePciAddress(s.Desc.PvScsi.PCIDevice, -1, nil)
		if err != nil {
			return errors.Wrap(err, "ensure pvscsi pci address")
		}
	}

	// skip pci root or pcie root
	for i := 1; i < len(s.Desc.PCIControllers); i++ {
		err = s.ensureDevicePciAddress(s.Desc.PCIControllers[i].PCIDevice, -1, nil)
		if err != nil {
			return errors.Wrap(err, "ensure pci controllers address")
		}
	}

	for i := 0; i < len(s.Desc.Disks); i++ {
		if s.Desc.Disks[i].Pci != nil {
			err = s.ensureDevicePciAddress(s.Desc.Disks[i].Pci, -1, nil)
			if err != nil {
				return errors.Wrapf(err, "ensure disk %d pci address", s.Desc.Disks[i].Index)
			}
		}
	}
	for i := 0; i < len(s.Desc.Nics); i++ {
		if s.Desc.Nics[i].Pci != nil {
			err = s.ensureDevicePciAddress(s.Desc.Nics[i].Pci, -1, nil)
			if err != nil {
				return errors.Wrapf(err, "ensure nic %s pci address", s.Desc.Nics[i].Ifname)
			}
		}
	}

	for i := 0; i < len(s.Desc.IsolatedDevices); i++ {
		if len(s.Desc.IsolatedDevices[i].VfioDevs) > 0 {
			multiFunc := len(s.Desc.IsolatedDevices[i].VfioDevs) > 1
			err = s.ensureDevicePciAddress(s.Desc.IsolatedDevices[i].VfioDevs[0].PCIDevice, 0, &multiFunc)
			if err != nil {
				return errors.Wrapf(err, "ensure isolated device %s pci address", s.Desc.IsolatedDevices[i].VfioDevs[0].PCIAddr)
			}
			for j := 1; j < len(s.Desc.IsolatedDevices[i].VfioDevs); j++ {
				s.Desc.IsolatedDevices[i].VfioDevs[j].PCIAddr = s.Desc.IsolatedDevices[i].VfioDevs[0].PCIAddr.Copy()
				err = s.ensureDevicePciAddress(s.Desc.IsolatedDevices[i].VfioDevs[j].PCIDevice, j, nil)
				if err != nil {
					return errors.Wrapf(err, "ensure isolated device %s pci address", s.Desc.IsolatedDevices[i].VfioDevs[j].PCIAddr)
				}
			}
		}
	}

	if s.Desc.Usb != nil {
		err = s.ensureDevicePciAddress(s.Desc.Usb.PCIDevice, -1, nil)
		if err != nil {
			return errors.Wrap(err, "ensure usb controller pci address")
		}
	}

	if s.Desc.Rng != nil {
		err = s.ensureDevicePciAddress(s.Desc.Rng.PCIDevice, -1, nil)
		if err != nil {
			return errors.Wrap(err, "ensure random device pci address")
		}
	}

	anonymousPCIDevs := s.Desc.AnonymousPCIDevs[:0]
	for i := 0; i < len(s.Desc.AnonymousPCIDevs); i++ {
		if s.isMachineDefaultAddress(s.Desc.AnonymousPCIDevs[i].PCIAddr) {
			if _, inUse := s.pciAddrs.IsAddrInUse(s.Desc.AnonymousPCIDevs[i].PCIAddr); inUse {
				log.Infof("guest %s anonymous dev addr %s in use", s.GetName(), s.Desc.AnonymousPCIDevs[i].String())
				continue
			}
		}
		err = s.ensureDevicePciAddress(s.Desc.AnonymousPCIDevs[i], -1, nil)
		if err != nil {
			return errors.Wrap(err, "ensure anonymous pci dev pci address")
		}
		anonymousPCIDevs = append(anonymousPCIDevs, s.Desc.AnonymousPCIDevs[i])
	}
	if len(anonymousPCIDevs) == 0 {
		anonymousPCIDevs = nil
	}
	s.Desc.AnonymousPCIDevs = anonymousPCIDevs
	return nil
}

func (s *SKVMGuestInstance) getNetdevOfThePciAddress(qtree string, addr *desc.PCIAddr) string {
	var slotFunc = fmt.Sprintf("addr = %02x.%x", addr.Slot, addr.Function)
	var addressFound = false
	var lines = strings.Split(strings.TrimSuffix(qtree, "\r\n"), "\\r\\n")
	var currentIndentLevel = -1
	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "" {
			continue
		}

		if currentIndentLevel > 0 {
			newIndentLevel := len(line) - len(trimmedLine)
			if newIndentLevel <= currentIndentLevel {
				if addressFound {
					break
				}

				currentIndentLevel = -1
				continue
			}

			if strings.HasPrefix(trimmedLine, slotFunc) {
				addressFound = true
				continue
			}

			if strings.HasPrefix(trimmedLine, "netdev =") {
				segs := strings.Split(trimmedLine, " ")
				netdev := strings.Trim(segs[2], `\\"`)
				log.Infof("found netdev %s: %s", netdev, trimmedLine)
				return netdev
			} else {
				continue
			}
		}

		if strings.HasPrefix(trimmedLine, "dev: virtio-net-pci") {
			currentIndentLevel = len(line) - len(trimmedLine)
		} else {
			continue
		}
	}
	return ""
}

// guests description no pci description before host-agent assign pci device address info
// in this case wo need query pci address info by `query-pci` command. Also memory devices.
func (s *SKVMGuestInstance) initGuestDescFromExistingGuest(
	cpuList []monitor.HotpluggableCPU, pciInfoList []monitor.PCIInfo,
	memoryDevicesInfoList []monitor.MemoryDeviceInfo, memDevs []monitor.Memdev,
	scsiNumQueues int64, qtree string,
) error {
	if len(pciInfoList) > 1 {
		return errors.Errorf("unsupported pci info list with multi bus")
	}
	unknownDevices := make([]monitor.PCIDeviceInfo, 0)

	err := s.initCpuDesc(uint(len(cpuList)))
	if err != nil {
		return err
	}
	err = s.initMemDescFromMemoryInfo(memoryDevicesInfoList, memDevs)
	if err != nil {
		return errors.Wrap(err, "init guest memory devices")
	}
	s.initMachineDesc()

	// This code is designed to ensure compatibility with older guests.
	// However, it is not recommended for new guests to generate a desc file from it
	pciRoot, _ := s.initGuestPciControllers(false)
	err = s.initGuestPciAddresses()
	if err != nil {
		return errors.Wrap(err, "init guest pci addresses")
	}

	if err := s.initGuestNetworks(pciRoot, nil); err != nil {
		return errors.Wrap(err, "init guest networks")
	}
	//s.initIsolatedDevices(pciRoot, nil)
	s.initQgaDesc()
	s.initPvpanicDesc()
	s.initIsaSerialDesc()
	s.Desc.VdiDevice = new(desc.SGuestVdi)

	for i := 0; i < len(pciInfoList[0].Devices); i++ {
		pciAddr := &desc.PCIAddr{
			Bus:      uint(pciInfoList[0].Devices[i].Bus),
			Slot:     uint(pciInfoList[0].Devices[i].Slot),
			Function: uint(pciInfoList[0].Devices[i].Function),
		}
		switch pciInfoList[0].Devices[i].QdevID {
		case "scsi":
			_, hasPvScsi := s.fixDiskDriver()
			if hasPvScsi && s.Desc.PvScsi == nil {
				s.Desc.PvScsi = &desc.SGuestPvScsi{
					PCIDevice: desc.NewPCIDevice(pciRoot.CType, "pvscsi", "scsi"),
				}
			} else if s.Desc.VirtioScsi == nil {
				s.Desc.VirtioScsi = &desc.SGuestVirtioScsi{
					PCIDevice: desc.NewPCIDevice(pciRoot.CType, "virtio-scsi-pci", "scsi"),
				}
			}

			if s.Desc.VirtioScsi != nil {
				s.Desc.VirtioScsi.PCIAddr = pciAddr
				err = s.ensureDevicePciAddress(s.Desc.VirtioScsi.PCIDevice, -1, nil)
				if err != nil {
					return errors.Wrap(err, "ensure virtio scsi pci address")
				}
				if scsiNumQueues > 1 {
					numQueues := uint8(scsiNumQueues)
					s.Desc.VirtioScsi.NumQueues = &numQueues
				}
			} else if s.Desc.PvScsi != nil {
				s.Desc.PvScsi.PCIAddr = pciAddr
				err = s.ensureDevicePciAddress(s.Desc.PvScsi.PCIDevice, -1, nil)
				if err != nil {
					return errors.Wrap(err, "ensure pvscsi pci address")
				}
			}
		case "video0":
			if s.Desc.VgaDevice == nil {
				s.initGuestVga(pciRoot)
			}

			s.Desc.VgaDevice.PCIAddr = pciAddr
			err = s.ensureDevicePciAddress(s.Desc.VgaDevice.PCIDevice, -1, nil)
			if err != nil {
				return errors.Wrap(err, "ensure vga pci address")
			}
		case "random0":
			if s.Desc.Rng == nil {
				// in case rng device disable by host options
				s.initRandomDevice(pciRoot, true)
			}
			s.Desc.Rng.PCIAddr = pciAddr
			err = s.ensureDevicePciAddress(s.Desc.Rng.PCIDevice, -1, nil)
			if err != nil {
				return errors.Wrap(err, "ensure random device pci address")
			}
		case "usb":
			if s.Desc.Usb == nil {
				s.initUsbController(pciRoot)
			}

			s.Desc.Usb.PCIAddr = pciAddr
			err = s.ensureDevicePciAddress(s.Desc.Usb.PCIDevice, -1, nil)
			if err != nil {
				return errors.Wrap(err, "ensure usb controller pci address")
			}
		case "sound0":
			if s.Desc.VdiDevice.Spice == nil {
				s.Desc.Vdi = "spice"
				s.initSpiceDevices(pciRoot)
			}
			s.Desc.VdiDevice.Spice.IntelHDA.PCIAddr = pciAddr
			err = s.ensureDevicePciAddress(s.Desc.VdiDevice.Spice.IntelHDA.PCIDevice, -1, nil)
			if err != nil {
				return errors.Wrap(err, "ensure vdi hda pci address")
			}
		case "vdagent-serial0":
			if s.Desc.VdiDevice.Spice == nil {
				s.Desc.Vdi = "spice"
				s.initSpiceDevices(pciRoot)
			}
			s.Desc.VdiDevice.Spice.VdagentSerial.PCIAddr = pciAddr
			err = s.ensureDevicePciAddress(s.Desc.VdiDevice.Spice.VdagentSerial.PCIDevice, -1, nil)
			if err != nil {
				return errors.Wrap(err, "ensure vdagent serial pci address")
			}
		case "virtio-serial0":
			if s.Desc.VirtioSerial == nil {
				s.initVirtioSerial(pciRoot)
			}

			s.Desc.VirtioSerial.PCIAddr = pciAddr
			err = s.ensureDevicePciAddress(s.Desc.VirtioSerial.PCIDevice, -1, nil)
			if err != nil {
				return errors.Wrap(err, "ensure virtio serial address")
			}
		case "usbspice":
			if s.Desc.VdiDevice.Spice == nil {
				s.Desc.Vdi = "spice"
				s.initSpiceDevices(pciRoot)
			}
			s.Desc.VdiDevice.Spice.UsbRedirct.EHCI1.PCIAddr = pciAddr
			multiFunc := true
			err = s.ensureDevicePciAddress(s.Desc.VdiDevice.Spice.UsbRedirct.EHCI1.PCIDevice, -1, &multiFunc)
			if err != nil {
				return errors.Wrap(err, "ensure vdi usb ehci1 pci address")
			}
		case "uhci1":
			if s.Desc.VdiDevice.Spice == nil {
				s.Desc.Vdi = "spice"
				s.initSpiceDevices(pciRoot)
			}
			multiFunc := true
			s.Desc.VdiDevice.Spice.UsbRedirct.UHCI1.PCIAddr = pciAddr
			err = s.ensureDevicePciAddress(s.Desc.VdiDevice.Spice.UsbRedirct.UHCI1.PCIDevice, -1, &multiFunc)
			if err != nil {
				return errors.Wrap(err, "ensure vdi usb uhci1 pci address")
			}
		case "uhci2":
			if s.Desc.VdiDevice.Spice == nil {
				s.Desc.Vdi = "spice"
				s.initSpiceDevices(pciRoot)
			}
			multiFunc := true
			s.Desc.VdiDevice.Spice.UsbRedirct.UHCI2.PCIAddr = pciAddr
			err = s.ensureDevicePciAddress(s.Desc.VdiDevice.Spice.UsbRedirct.UHCI2.PCIDevice, -1, &multiFunc)
			if err != nil {
				return errors.Wrap(err, "ensure vdi usb uhci2 pci address")
			}
		case "uhci3":
			if s.Desc.VdiDevice.Spice == nil {
				s.Desc.Vdi = "spice"
				s.initSpiceDevices(pciRoot)
			}
			multiFunc := true
			s.Desc.VdiDevice.Spice.UsbRedirct.UHCI3.PCIAddr = pciAddr
			err = s.ensureDevicePciAddress(s.Desc.VdiDevice.Spice.UsbRedirct.UHCI3.PCIDevice, -1, &multiFunc)
			if err != nil {
				return errors.Wrap(err, "ensure vdi usb uhci3 pci address")
			}
		default:
			switch {
			case strings.HasPrefix(pciInfoList[0].Devices[i].QdevID, "drive_"):
				indexStr := strings.TrimPrefix(pciInfoList[0].Devices[i].QdevID, "drive_")
				index, err := strconv.Atoi(indexStr)
				if err != nil {
					log.Errorf("failed parse disk pci id %s", pciInfoList[0].Devices[i].QdevID)
					unknownDevices = append(unknownDevices, pciInfoList[0].Devices[i])
					continue
				}
				for i := 0; i < len(s.Desc.Disks); i++ {
					if s.Desc.Disks[i].Index == int8(index) {
						if s.Desc.Disks[i].Pci == nil {
							devType := qemu.GetDiskDeviceModel(DISK_DRIVER_VIRTIO)
							s.Desc.Disks[i].Pci = desc.NewPCIDevice(pciRoot.CType, devType, pciInfoList[0].Devices[i].QdevID)
							s.Desc.Disks[i].Scsi = nil
							s.Desc.Disks[i].Ide = nil
							s.Desc.Disks[i].Driver = DISK_DRIVER_VIRTIO
						}
						s.Desc.Disks[i].Pci.PCIAddr = pciAddr
						err = s.ensureDevicePciAddress(s.Desc.Disks[i].Pci, -1, nil)
						if err != nil {
							return errors.Wrapf(err, "ensure disk %d pci address", s.Desc.Disks[i].Index)
						}
					}
				}
			case strings.HasPrefix(pciInfoList[0].Devices[i].QdevID, "netdev-"):
				ifname := strings.TrimPrefix(pciInfoList[0].Devices[i].QdevID, "netdev-")
				for i := 0; i < len(s.Desc.Nics); i++ {
					if s.Desc.Nics[i].Ifname == ifname {
						s.Desc.Nics[i].Pci.PCIAddr = pciAddr
						err = s.ensureDevicePciAddress(s.Desc.Nics[i].Pci, -1, nil)
						if err != nil {
							return errors.Wrapf(err, "ensure nic %s pci address", s.Desc.Nics[i].Ifname)
						}
					}
				}
			default:
				class := pciInfoList[0].Devices[i].ClassInfo.Class
				vendor := pciInfoList[0].Devices[i].ID.Vendor
				device := pciInfoList[0].Devices[i].ID.Device
				switch { // qemu: docs/specs/pci-ids.txt
				case class == 3075 && vendor == 6966 && device == 13: // { 0x0c03, "USB controller", "usb"}, 1b36:000d PCI xhci usb host adapter
					if s.Desc.Usb == nil {
						s.initUsbController(pciRoot)
					}

					s.Desc.Usb.PCIAddr = pciAddr
					err = s.ensureDevicePciAddress(s.Desc.Usb.PCIDevice, -1, nil)
					if err != nil {
						return errors.Wrap(err, "ensure usb controller pci address")
					}
				case class == 255 && vendor == 6900 && device == 4101: // 0x00ff, 1af4:1005  entropy generator device (legacy)
					if s.Desc.Rng == nil {
						// in case rng device disable by host options
						s.initRandomDevice(pciRoot, true)
					}

					s.Desc.Rng.PCIAddr = pciAddr
					err = s.ensureDevicePciAddress(s.Desc.Rng.PCIDevice, -1, nil)
					if err != nil {
						return errors.Wrap(err, "ensure random device pci address")
					}
				case class == 1920 && vendor == 6900 && device == 4099: // 0x0780, 1af4:1003  console device (legacy)
					if s.Desc.VirtioSerial == nil {
						s.initVirtioSerial(pciRoot)
					}

					s.Desc.VirtioSerial.PCIAddr = pciAddr
					err = s.ensureDevicePciAddress(s.Desc.VirtioSerial.PCIDevice, -1, nil)
					if err != nil {
						return errors.Wrap(err, "ensure virtio serial address")
					}
				case class == 768 && vendor == 4660 && device == 4369: // { 0x0300, "VGA controller", "display", 0x00ff}, PCI ID: 1234:1111
					if s.Desc.VgaDevice == nil {
						s.initGuestVga(pciRoot)
					}

					s.Desc.VgaDevice.PCIAddr = pciAddr
					err = s.ensureDevicePciAddress(s.Desc.VgaDevice.PCIDevice, -1, nil)
					if err != nil {
						return errors.Wrap(err, "ensure vga pci address")
					}
				case class == 512 && vendor == 6900 && device == 4096: // { 0x0200, "Ethernet controller", "ethernet"}, 1af4:1000  network device (legacy)
					// virtio nics has no ids
					ifname := s.getNetdevOfThePciAddress(qtree, pciAddr)

					index := 0
					for ; index < len(s.Desc.Nics); index++ {
						if s.Desc.Nics[index].Ifname == ifname {
							s.Desc.Nics[index].Pci.PCIAddr = pciAddr
							err = s.ensureDevicePciAddress(s.Desc.Nics[index].Pci, -1, nil)
							if err != nil {
								return errors.Wrapf(err, "ensure nic %s pci address", s.Desc.Nics[index].Ifname)
							}
							break
						}
					}
					if index >= len(s.Desc.Nics) {
						return errors.Errorf("failed find nics ifname")
					}
				default:
					unknownDevices = append(unknownDevices, pciInfoList[0].Devices[i])
				}
			}
		}
	}

	s.initGuestDisks(pciRoot, nil, true)

	for i := 0; i < len(unknownDevices); i++ {
		if unknownDevices[i].Bus == 0 && unknownDevices[i].Slot == 0 {
			continue // host bridge
		}
		pciDev := desc.NewPCIDevice(pciRoot.CType, "", unknownDevices[i].QdevID)
		pciDev.PCIAddr = &desc.PCIAddr{
			Bus:      uint(unknownDevices[i].Bus),
			Slot:     uint(unknownDevices[i].Slot),
			Function: uint(unknownDevices[i].Function),
		}
		err = s.ensureDevicePciAddress(pciDev, -1, nil)
		if err != nil {
			return errors.Wrap(err, "ensure anonymous pci dev address")
		}
		if s.isMachineDefaultAddress(pciDev.PCIAddr) {
			continue
		}
		if s.Desc.AnonymousPCIDevs == nil {
			s.Desc.AnonymousPCIDevs = make([]*desc.PCIDevice, 0)
		}
		s.Desc.AnonymousPCIDevs = append(s.Desc.AnonymousPCIDevs, pciDev)
	}
	return nil
}
