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

package isoutils

import (
	"encoding/binary"
	"encoding/xml"
	"fmt"
	"io"
	"unicode/utf16"

	"yunion.io/x/log"
	"yunion.io/x/pkg/util/imagetools"
)

var wimImageTag = [8]byte{'M', 'S', 'W', 'I', 'M', 0, 0, 0}

const (
	wimResFlagCompressed = 0x04
)

// wimResourceDesc matches the on-disk WIM resource descriptor (24 bytes).
type wimResourceDesc struct {
	FlagsAndCompressedSize uint64
	Offset                 int64
	OriginalSize           int64
}

func (r wimResourceDesc) flags() byte {
	return byte(r.FlagsAndCompressedSize >> 56)
}

func (r wimResourceDesc) compressedSize() int64 {
	return int64(r.FlagsAndCompressedSize & 0xffffffffffffff)
}

// wimHeaderDisk is the on-disk WIM header (208 bytes / 0xd0).
type wimHeaderDisk struct {
	ImageTag        [8]byte
	Size            uint32
	Version         uint32
	Flags           uint32
	CompressionSize uint32
	WIMGuid         [16]byte
	PartNumber      uint16
	TotalParts      uint16
	ImageCount      uint32
	OffsetTable     wimResourceDesc
	XMLData         wimResourceDesc
	BootMetadata    wimResourceDesc
	BootIndex       uint32
	Padding         uint32
	Integrity       wimResourceDesc
	Unused          [60]byte
}

type wimXMLInfo struct {
	Image []wimXMLImage `xml:"IMAGE"`
}

type wimXMLImage struct {
	Name    string          `xml:"NAME"`
	Index   int             `xml:"INDEX,attr"`
	Windows *wimWindowsInfo `xml:"WINDOWS"`
}

type wimWindowsInfo struct {
	Arch            byte          `xml:"ARCH"`
	ProductName     string        `xml:"PRODUCTNAME"`
	EditionID       string        `xml:"EDITIONID"`
	ProductType     string        `xml:"PRODUCTTYPE"`
	DefaultLanguage string        `xml:"LANGUAGES>DEFAULT"`
	Version         wimXMLVersion `xml:"VERSION"`
}

type wimXMLVersion struct {
	Major int `xml:"MAJOR"`
	Minor int `xml:"MINOR"`
	Build int `xml:"BUILD"`
}

// parseWimXmlMetadata reads WIM/ESD header + uncompressed XML metadata and maps Windows version.
// Content compression (LZMS/XPRESS) is ignored; only the XML blob is required for edition detection.
func parseWimXmlMetadata(r io.ReaderAt) (*ISOInfo, error) {
	var hdr wimHeaderDisk
	if err := binary.Read(io.NewSectionReader(r, 0, int64(binary.Size(hdr))), binary.LittleEndian, &hdr); err != nil {
		return nil, fmt.Errorf("read WIM header: %w", err)
	}
	if hdr.ImageTag != wimImageTag {
		return nil, fmt.Errorf("not a WIM/ESD file")
	}
	if hdr.XMLData.compressedSize() == 0 || hdr.XMLData.OriginalSize == 0 {
		return nil, fmt.Errorf("WIM/ESD has no XML metadata")
	}
	if hdr.XMLData.flags()&wimResFlagCompressed != 0 {
		return nil, fmt.Errorf("compressed WIM XML metadata is not supported")
	}

	xmlBytes := make([]byte, hdr.XMLData.OriginalSize)
	if _, err := r.ReadAt(xmlBytes, hdr.XMLData.Offset); err != nil {
		return nil, fmt.Errorf("read WIM XML metadata: %w", err)
	}

	xmlStr, err := decodeWimUTF16XML(xmlBytes)
	if err != nil {
		return nil, err
	}

	var info wimXMLInfo
	if err := xml.Unmarshal([]byte(xmlStr), &info); err != nil {
		return nil, fmt.Errorf("parse WIM XML: %w", err)
	}

	for _, image := range info.Image {
		if image.Windows == nil {
			continue
		}
		result := mapWindowsVersion(image.Windows)
		ver := fmt.Sprintf("%d.%d.%d", image.Windows.Version.Major, image.Windows.Version.Minor, image.Windows.Version.Build)
		log.Debugf("识别到 %s 版本: %s -> %s", result.Distro, ver, result.Version)
		return result, nil
	}
	return nil, fmt.Errorf("no WINDOWS metadata found in WIM/ESD XML")
}

func decodeWimUTF16XML(data []byte) (string, error) {
	if len(data) < 2 || len(data)%2 != 0 {
		return "", fmt.Errorf("invalid WIM XML encoding")
	}
	u16 := make([]uint16, len(data)/2)
	for i := 0; i < len(u16); i++ {
		u16[i] = binary.LittleEndian.Uint16(data[i*2:])
	}
	// BOM is little-endian UTF-16 (0xFEFF)
	if u16[0] != 0xfeff {
		return "", fmt.Errorf("invalid WIM XML BOM")
	}
	return string(utf16.Decode(u16[1:])), nil
}

func mapWindowsVersion(win *wimWindowsInfo) *ISOInfo {
	result := &ISOInfo{
		Distro:   imagetools.OS_DIST_WINDOWS,
		Language: win.DefaultLanguage,
	}
	switch win.Arch {
	case 9:
		result.Arch = "x86_64"
	case 12:
		result.Arch = "arm64"
	case 0:
		result.Arch = "x86"
	}

	majMin := fmt.Sprintf("%d.%d", win.Version.Major, win.Version.Minor)
	switch majMin {
	case "6.0":
		result.Version = "Windows Vista"
	case "6.1":
		result.Version = "Windows 7"
	case "6.2":
		result.Version = "Windows 8"
	case "6.3":
		result.Version = "Windows 8.1"
	case "10.0":
		if win.Version.Build >= 27500 {
			result.Version = "Windows 12"
		} else if win.Version.Build >= 22000 {
			result.Version = "Windows 11"
		} else {
			result.Version = "Windows 10"
		}
	}

	if win.ProductType == "ServerNT" {
		result.Distro = imagetools.OS_DIST_WINDOWS_SERVER
		switch majMin {
		case "6.0":
			result.Version = "Windows Server 2008"
		case "6.1":
			result.Version = "Windows Server 2008 R2"
		case "6.2":
			result.Version = "Windows Server 2012"
		case "6.3":
			result.Version = "Windows Server 2012 R2"
		case "10.0":
			if win.Version.Build >= 26040 {
				result.Version = "Windows Server 2025"
			} else if win.Version.Build >= 20348 {
				result.Version = "Windows Server 2022"
			} else if win.Version.Build >= 17763 {
				result.Version = "Windows Server 2019"
			} else if win.Version.Build >= 14393 {
				result.Version = "Windows Server 2016"
			}
		}
	}
	return result
}
