package guestman

import (
	"fmt"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/hostman/guestman/desc"
	"yunion.io/x/onecloud/pkg/hostman/guestman/qemu"
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
	s.initMemDesc()
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
	s.initGuestDisks(pciRoot, pciBridge)
	if err = s.initGuestNetworks(pciRoot, pciBridge); err != nil {
		return errors.Wrap(err, "init guest networks")
	}

	s.initIsolatedDevices(pciRoot, pciBridge)
	s.initUsbController(pciRoot)
	s.initRandomDevice(pciRoot)
	s.initQgaDesc()
	s.initPvpanicDesc()
	s.initIsaSerialDesc()

	return s.ensurePciAddresses()
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
	if isPcie {
		pciRoot = s.addPCIController(desc.CONTROLLER_TYPE_PCIE_ROOT, "")
		if s.hasPcieExtendBus() {
			s.addPCIController(desc.CONTROLLER_TYPE_PCIE_TO_PCI_BRIDGE, desc.CONTROLLER_TYPE_PCIE_ROOT)
			pciBridge = s.addPCIController(desc.CONTROLLER_TYPE_PCI_BRIDGE, desc.CONTROLLER_TYPE_PCIE_TO_PCI_BRIDGE)
			for i := 0; i < options.HostOptions.PcieRootPortCount; i++ {
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

func (s *SKVMGuestInstance) initRandomDevice(pciRoot *desc.PCIController) {
	if !options.HostOptions.EnableVirtioRngDevice {
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
	s.Desc.Usb = &desc.UsbController{
		PCIDevice: desc.NewPCIDevice(pciRoot.CType, " qemu-xhci", "usb"),
	}
	s.Desc.Usb.Options = map[string]string{
		"p2": "8", // usb2 port count
		"p3": "8", // usb3 port count
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
	if s.Desc.Cdrom == nil {
		s.Desc.Cdrom = new(desc.SGuestCdrom)
		return
	}

	s.archMan.GenerateCdromDesc(s.getOsname(), s.Desc.Cdrom)
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
	if s.Desc.VgaDevice != nil {
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

	return nil
}
