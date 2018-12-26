package qemuimg

import (
	"strings"

	"yunion.io/x/log"
)

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

func String2ImageFormat(fmt string) TImageFormat {
	switch strings.ToLower(fmt) {
	case "vhd":
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
	log.Fatalf("unknown image format!!! %s", fmt)
	return TImageFormat(fmt)
}
