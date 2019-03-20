package compute

const (
	DIRECT_PCI_TYPE = "PCI"
	GPU_HPC_TYPE    = "GPU-HPC" // # for compute
	GPU_VGA_TYPE    = "GPU-VGA" // # for display
	USB_TYPE        = "USB"
	NIC_TYPE        = "NIC"

	NVIDIA_VENDOR_ID = "10de"
	AMD_VENDOR_ID    = "1002"
)

var VALID_GPU_TYPES = []string{GPU_HPC_TYPE, GPU_VGA_TYPE}

var VALID_PASSTHROUGH_TYPES = []string{DIRECT_PCI_TYPE, USB_TYPE, NIC_TYPE, GPU_HPC_TYPE, GPU_VGA_TYPE}

var ID_VENDOR_MAP = map[string]string{
	NVIDIA_VENDOR_ID: "NVIDIA",
	AMD_VENDOR_ID:    "AMD",
}

var VENDOR_ID_MAP = map[string]string{
	"NVIDIA": NVIDIA_VENDOR_ID,
	"AMD":    AMD_VENDOR_ID,
}
