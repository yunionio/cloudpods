package qemuimg

type TImageFormat string

const (
	QCOW2 = TImageFormat("qcow2")
	VMDK  = TImageFormat("vmdk")
	VHD   = TImageFormat("vhd")
	ISO   = TImageFormat("iso")
	RAW   = TImageFormat("raw")
)

func (fmt TImageFormat) String() string {
	switch string(fmt) {
	case "vhd":
		return "vpc"
	default:
		return string(fmt)
	}
}