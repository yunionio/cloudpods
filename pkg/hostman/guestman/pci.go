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
	s.initCpuDesc()
	s.initMemDesc(s.Desc.Mem)
	s.initMachineDesc()

	pciRoot, pciBridge := s.initGuestPciControllers()
	err := s.initGuestPciAddresses()
	if err != nil {
		return errors.Wrap(err, "init guest pci addresses")
	}
	if err := s.initMachineDefaultDevices(); err != nil {
		return errors.Wrap(err, "init machine default devices")
	}

	// vdi device for spice
	s.Desc.VdiDevice = new(desc.SGuestVdi)
	if s.IsVdiSpice() {
		s.initSpiceDevices(pciRoot)
	}

	s.initVirtioSerial(pciRoot)
	s.initGuestVga(pciRoot)
	s.initCdromDesc()
	s.initFloppyDesc()
	s.initGuestDisks(pciRoot, pciBridge)
	if err = s.initGuestNetworks(pciRoot, pciBridge); err != nil {
		return errors.Wrap(err, "init guest networks")
	}

	s.initIsolatedDevices(pciRoot, pciBridge)
	s.initUsbController(pciRoot)
	s.initRandomDevice(pciRoot, options.HostOptions.EnableVirtioRngDevice)
	s.initQgaDesc()
	s.initPvpanicDesc()
	s.initIsaSerialDesc()

	return s.ensurePciAddresses()
}

func (s *SKVMGuestInstance) loadGuestPciAddresses() error {
	err := s.initGuestPciAddresses()
	if err != nil {
		return errors.Wrap(err, "init guest pci addresses")
	}
	if err := s.initMachineDefaultDevices(); err != nil {
		return errors.Wrap(err, "init machine default devices")
	}
	err = s.ensurePciAddresses()
	if err != nil {
		return errors.Wrap(err, "load desc ensure pci address")
	}
	return nil
}

func (s *SKVMGuestInstance) initMachineDefaultDevices() error {
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

func (s *SKVMGuestInstance) initGuestPciControllers() (*desc.PCIController, *desc.PCIController) {
	var isPcie = s.isPcie()
	var pciRoot, pciBridge *desc.PCIController
	if isPcie && s.hasPcieExtendBus() {
		pciRoot = s.addPCIController(desc.CONTROLLER_TYPE_PCIE_ROOT, "")
		s.addPCIController(desc.CONTROLLER_TYPE_PCIE_TO_PCI_BRIDGE, desc.CONTROLLER_TYPE_PCIE_ROOT)
		pciBridge = s.addPCIController(desc.CONTROLLER_TYPE_PCI_BRIDGE, desc.CONTROLLER_TYPE_PCIE_TO_PCI_BRIDGE)
		for i := 0; i < options.HostOptions.PcieRootPortCount; i++ {
			s.addPCIController(desc.CONTROLLER_TYPE_PCIE_ROOT_PORT, desc.CONTROLLER_TYPE_PCIE_ROOT)
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
	if contType == "qemu-xhci" {
		s.Desc.Usb.Options = map[string]string{
			"p2": "8", // usb2 port count
			"p3": "8", // usb3 port count
		}
	}
}

func (s *SKVMGuestInstance) initGuestNetworks(pciRoot, pciBridge *desc.PCIController) error {
	cont := pciRoot
	if pciBridge != nil {
		cont = pciBridge
	}

	if s.getOsname() == OS_NAME_ANDROID {
		s.Desc.Nics = s.Desc.Nics[:1]
	}

	for i := 0; i < len(s.Desc.Nics); i++ {
		switch s.getOsname() {
		case OS_NAME_MACOS:
			vectors := 0
			s.Desc.Nics[i].Vectors = &vectors
			s.Desc.Nics[i].Driver = "e1000"
		case OS_NAME_VMWARE:
			s.Desc.Nics[i].Driver = "vmxnet3"
		}

		if s.Desc.Nics[i].NumQueues > 1 {
			vectors := s.Desc.Nics[i].NumQueues * 2
			s.Desc.Nics[i].Vectors = &vectors
		}

		if err := s.generateNicScripts(s.Desc.Nics[i]); err != nil {
			return errors.Wrapf(err, "generateNicScripts for nic: %v", s.Desc.Nics[i])
		}
		s.Desc.Nics[i].UpscriptPath = s.getNicUpScriptPath(s.Desc.Nics[i])
		s.Desc.Nics[i].DownscriptPath = s.getNicDownScriptPath(s.Desc.Nics[i])

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
	return nil
}

func (s *SKVMGuestInstance) initIsolatedDevices(pciRoot, pciBridge *desc.PCIController) {
	cont := pciRoot
	if pciBridge != nil {
		cont = pciBridge
	}
	manager := s.manager.GetHost().GetIsolatedDeviceManager()
	for i := 0; i < len(s.Desc.IsolatedDevices); i++ {
		dev := manager.GetDeviceByAddr(s.Desc.IsolatedDevices[i].Addr)
		if s.Desc.IsolatedDevices[i].DevType == api.USB_TYPE {
			s.Desc.IsolatedDevices[i].Usb = desc.NewUsbDevice("usb-host", fmt.Sprintf("usb%d", i))
			s.Desc.IsolatedDevices[i].Usb.Options = dev.GetPassthroughOptions()
		} else {
			id := fmt.Sprintf("vfio-%d", i)
			s.Desc.IsolatedDevices[i].VfioDevs = make([]*desc.VFIODevice, 0)
			vfioDev := &desc.VFIODevice{
				PCIDevice: desc.NewPCIDevice(cont.CType, "vfio-pci", id),
			}
			s.Desc.IsolatedDevices[i].VfioDevs = append(
				s.Desc.IsolatedDevices[i].VfioDevs, vfioDev,
			)
			s.Desc.IsolatedDevices[i].VfioDevs[0].HostAddr = dev.GetAddr()
			if dev.GetDeviceType() == api.GPU_VGA_TYPE {
				s.Desc.IsolatedDevices[i].VfioDevs[0].XVga = true
			}

			groupDevAddrs := dev.GetIOMMUGroupRestAddrs()
			for j := 0; j < len(groupDevAddrs); j++ {
				gid := fmt.Sprintf("%s-%s", id, groupDevAddrs[i])
				vfioDev = &desc.VFIODevice{
					PCIDevice: desc.NewPCIDevice(cont.CType, "vfio-pci", gid),
				}
				s.Desc.IsolatedDevices[i].VfioDevs = append(
					s.Desc.IsolatedDevices[i].VfioDevs, vfioDev,
				)
				s.Desc.IsolatedDevices[i].VfioDevs[j].HostAddr = groupDevAddrs[i]
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
		s.archMan.GenerateCdromDesc(s.getOsname(), s.Desc.Cdroms[i])
	}
}

func (s *SKVMGuestInstance) initFloppyDesc() {
	if s.getOsname() != OS_NAME_WINDOWS {
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
		s.archMan.GenerateFloppyDesc(s.getOsname(), s.Desc.Floppys[i])
	}
}

func (s *SKVMGuestInstance) initGuestDisks(pciRoot, pciBridge *desc.PCIController) {
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
	cont := pciRoot
	if pciBridge != nil {
		cont = pciBridge
	}
	for i := 0; i < len(s.Desc.Disks); i++ {
		devType := qemu.GetDiskDeviceModel(s.Desc.Disks[i].Driver)
		id := fmt.Sprintf("drive_%d", s.Desc.Disks[i].Index)
		switch s.Desc.Disks[i].Driver {
		case DISK_DRIVER_VIRTIO:
			s.Desc.Disks[i].Pci = desc.NewPCIDevice(cont.CType, devType, id)
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
	osname := s.getOsname()

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
	} else {
		if s.Desc.Vga == "" {
			s.Desc.Vga = "std"
		}
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

	var (
		bus, slot = 0, 0
		found     = false
	)
	for ; bus < len(s.pciAddrs.Buses); bus++ {
		if dev.Controller == s.pciAddrs.Buses[bus].Contorller {
			slot = s.pciAddrs.Buses[bus].FindNextUnusedSlot(uint(function))
			if slot >= 0 {
				found = true
				break
			}
		}
	}
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
			multiFunc := true
			err = s.ensureDevicePciAddress(s.Desc.IsolatedDevices[i].VfioDevs[0].PCIDevice, 0, &multiFunc)
			if err != nil {
				return errors.Wrapf(err, "ensure isolated device %s pci address", s.Desc.IsolatedDevices[i].VfioDevs[0].PCIAddr)
			}
			for j := 1; j < len(s.Desc.IsolatedDevices[i].VfioDevs); i++ {
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

	for i := 0; i < len(s.Desc.AnonymousPCIDevs); i++ {
		err = s.ensureDevicePciAddress(s.Desc.AnonymousPCIDevs[i], -1, nil)
		if err != nil {
			return errors.Wrap(err, "ensure anonymous pci dev pci address")
		}
	}
	return nil
}

// guests description no pci description before host-agent assign pci device address info
// in this case wo need query pci address info by `query-pci` command. Also memory devices.
func (s *SKVMGuestInstance) initGuestDescFromExistingGuest(
	pciInfoList []monitor.PCIInfo, memoryDevicesInfoList []monitor.MemoryDeviceInfo,
) error {
	if len(pciInfoList) > 1 {
		return errors.Errorf("unsupported pci info list with multi bus")
	}

	unknownDevices := make([]monitor.PCIDeviceInfo, 0)

	s.initCpuDesc()
	err := s.initMemDescFromMemoryInfo(memoryDevicesInfoList)
	if err != nil {
		return errors.Wrap(err, "init guest memory devices")
	}
	s.initMachineDesc()

	pciRoot, _ := s.initGuestPciControllers()
	err = s.initGuestPciAddresses()
	if err != nil {
		return errors.Wrap(err, "init guest pci addresses")
	}

	// vdi device for spice
	s.Desc.VdiDevice = new(desc.SGuestVdi)
	if s.IsVdiSpice() {
		s.initSpiceDevices(pciRoot)
	}

	s.initVirtioSerial(pciRoot)
	s.initGuestVga(pciRoot)
	s.initCdromDesc()
	s.initGuestDisks(pciRoot, nil)
	if err = s.initGuestNetworks(pciRoot, nil); err != nil {
		return errors.Wrap(err, "init guest networks")
	}

	s.initIsolatedDevices(pciRoot, nil)
	s.initUsbController(pciRoot)
	s.initRandomDevice(pciRoot, options.HostOptions.EnableVirtioRngDevice)
	s.initQgaDesc()
	s.initPvpanicDesc()
	s.initIsaSerialDesc()

	for i := 0; i < len(pciInfoList[0].Devices); i++ {
		pciAddr := &desc.PCIAddr{
			Bus:      uint(pciInfoList[0].Devices[i].Bus),
			Slot:     uint(pciInfoList[0].Devices[i].Slot),
			Function: uint(pciInfoList[0].Devices[i].Function),
		}
		switch pciInfoList[0].Devices[i].QdevID {
		case "scsi":
			if s.Desc.VirtioScsi != nil {
				s.Desc.VirtioScsi.PCIAddr = pciAddr
				err = s.ensureDevicePciAddress(s.Desc.VirtioScsi.PCIDevice, -1, nil)
				if err != nil {
					return errors.Wrap(err, "ensure virtio scsi pci address")
				}
			} else if s.Desc.PvScsi != nil {
				s.Desc.PvScsi.PCIAddr = pciAddr
				err = s.ensureDevicePciAddress(s.Desc.PvScsi.PCIDevice, -1, nil)
				if err != nil {
					return errors.Wrap(err, "ensure pvscsi pci address")
				}
			}
		case "video0":
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
		case "virtio-serial0":
			if s.Desc.VdiDevice.Spice == nil {
				s.Desc.Vdi = "spice"
				s.initSpiceDevices(pciRoot)
			}
			s.Desc.VdiDevice.Spice.VdagentSerial.PCIAddr = pciAddr
			err = s.ensureDevicePciAddress(s.Desc.VdiDevice.Spice.VdagentSerial.PCIDevice, -1, nil)
			if err != nil {
				return errors.Wrap(err, "ensure vdagent serial pci address")
			}
		case "usbspice":
			if s.Desc.VdiDevice.Spice == nil {
				s.Desc.Vdi = "spice"
				s.initSpiceDevices(pciRoot)
			}
			s.Desc.VdiDevice.Spice.UsbRedirct.EHCI1.PCIAddr = pciAddr
			multiFunc := true
			err = s.ensureDevicePciAddress(s.Desc.VdiDevice.Spice.UsbRedirct.EHCI1.PCIDevice, 7, &multiFunc)
			if err != nil {
				return errors.Wrap(err, "ensure vdi usb ehci1 pci address")
			}
			s.Desc.VdiDevice.Spice.UsbRedirct.UHCI1.PCIAddr = s.Desc.VdiDevice.Spice.UsbRedirct.EHCI1.PCIAddr.Copy()
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
				unknownDevices = append(unknownDevices, pciInfoList[0].Devices[i])
			}
		}
	}

	if len(unknownDevices) > 0 {
		s.Desc.AnonymousPCIDevs = make([]*desc.PCIDevice, len(unknownDevices))
	}
	for i := 0; i < len(unknownDevices); i++ {
		if unknownDevices[i].Bus == 0 && unknownDevices[i].Slot == 0 {
			// host bridge
			continue
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
		s.Desc.AnonymousPCIDevs = append(s.Desc.AnonymousPCIDevs, pciDev)
	}
	return nil
}
