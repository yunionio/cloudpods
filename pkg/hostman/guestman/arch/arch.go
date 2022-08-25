package arch

import "yunion.io/x/onecloud/pkg/hostman/guestman/desc"

const (
	Arch_x86_64  string = "x86_64"
	Arch_aarch64 string = "aarch64"
)

type Arch interface {
	GenerateCpuDesc(cpus uint, osName string, enableKVM, hideKVM bool) *desc.SGuestCpu
	GenerateMemDesc() *desc.SGuestMem
	GenerateMachineDesc(accel string) *desc.SGuestMachine
	GenerateCdromDesc(osName string, cdrom *desc.SGuestCdrom)
	GenerateQgaDesc(qgaPath string) *desc.SGuestQga
	GeneratePvpanicDesc() *desc.SGuestPvpanic
	GenerateIsaSerialDesc() *desc.SGuestIsaSerial
}

func NewArch(arch string) Arch {
	switch arch {
	case Arch_x86_64:
		return &X86{}
	case Arch_aarch64:
		return &ARM{}
	}
	return nil
}

type archBase struct {
}

func (*archBase) GenerateQgaDesc(qgaPath string) *desc.SGuestQga {
	charDev := "qga0"
	socket := &desc.CharDev{
		Backend: "socket",
		Id:      charDev,
		Options: map[string]string{
			"path":   qgaPath,
			"server": "",
			"nowait": "",
		},
	}

	serialPort := &desc.VirtSerialPort{
		Chardev: charDev,
		Name:    "org.qemu.guest_agent.0",
	}

	return &desc.SGuestQga{
		Socket:     socket,
		SerialPort: serialPort,
	}
}

func (*archBase) GeneratePvpanicDesc() *desc.SGuestPvpanic {
	return nil
}

func (*archBase) GenerateIsaSerialDesc() *desc.SGuestIsaSerial {
	return nil
}
