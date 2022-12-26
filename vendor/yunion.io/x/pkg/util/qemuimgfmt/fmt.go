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

package qemuimgfmt

import (
	"strings"
)

type TImageFormat string

const (
	QCOW2 = TImageFormat("qcow2")
	VMDK  = TImageFormat("vmdk")
	VHD   = TImageFormat("vhd")
	ISO   = TImageFormat("iso")
	RAW   = TImageFormat("raw")
)

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
