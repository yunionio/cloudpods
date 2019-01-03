package isolated_device

const (
	// TODO: merge  models/isolated_devices in new file
	DIRECT_PCI_TYPE = "PCI"
	GPU_HPC_TYPE    = "GPU-HPC" // # for compute
	GPU_VGA_TYPE    = "GPU-VGA" // # for display
	USB_TYPE        = "USB"
)

const (
	BUSID_REGEX = `[0-9a-fA-F]{2}:[0-9a-fA-F]{2}\.[0-9a-fA-F]`
	CODE_REGEX  = `[0-9a-fA-F]{4}`
	LABEL_REGEX = `[\w+\ \.\,\:\+\&\-\/\[\]\(\)]+`

	VFIO_PCI_KERNEL_DRIVER = "vfio-pci"
	DEFAULT_VGA_CMD        = " -vga std"
	DEFAULT_CPU_CMD        = "host,kvm=off"

	RESOURCE = "isolated_devices"
)

// # 在qemu/kvm下模拟Windows Hyper-V的一些半虚拟化特性，以便更好地使用Win虚拟机
// # http://blog.wikichoon.com/2014/07/enabling-hyper-v-enlightenments-with-kvm.html
// 但实际测试不行，虚拟机不能运行nvidia驱动
// #DEFAULT_CPU_CMD = "host,kvm=off,hv_relaxed,hv_spinlocks=0x1fff,hv_vapic,hv_time"
