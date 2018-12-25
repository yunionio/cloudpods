package qemuimg

<<<<<<< HEAD
import (
	"strings"
	// "yunion.io/x/log"
)

=======
>>>>>>> move util to pkg/utils
type TImageFormat string

const (
	QCOW2 = TImageFormat("qcow2")
	VMDK  = TImageFormat("vmdk")
	VHD   = TImageFormat("vhd")
	ISO   = TImageFormat("iso")
	RAW   = TImageFormat("raw")
)

<<<<<<< HEAD
var supportedImageFormats = []TImageFormat{
	QCOW2, VMDK, VHD, ISO, RAW,
}

func IsSupportedImageFormat(fmtStr string) bool {
	for i := 0; i < len(supportedImageFormats); i += 1 {
		if fmtStr == string(supportedImageFormats[i]) {
			return true
		}
	}
	return false
}

=======
>>>>>>> move util to pkg/utils
func (fmt TImageFormat) String() string {
	switch string(fmt) {
	case "vhd":
		return "vpc"
	default:
		return string(fmt)
	}
}
<<<<<<< HEAD

func String2ImageFormat(fmt string) TImageFormat {
	switch strings.ToLower(fmt) {
	case "vhd", "vpc":
		return VHD
	case "qcow2":
		return QCOW2
	case "vmdk":
		return VMDK
	case "iso":
		return ISO
	case "raw":
		return RAW
	}
	// log.Fatalf("unknown image format!!! %s", fmt)
	return TImageFormat(fmt)
}
=======
>>>>>>> move util to pkg/utils
