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
	"bytes"
	"encoding/binary"
	"testing"
	"unicode/utf16"

	"yunion.io/x/pkg/util/imagetools"
)

func encodeWimUTF16XML(s string) []byte {
	u16 := utf16.Encode([]rune(s))
	out := make([]byte, 2+len(u16)*2)
	binary.LittleEndian.PutUint16(out[0:], 0xfeff) // BOM
	for i, v := range u16 {
		binary.LittleEndian.PutUint16(out[2+i*2:], v)
	}
	return out
}

func buildMinimalWimWithXML(xmlASCII string) []byte {
	xmlData := encodeWimUTF16XML(xmlASCII)
	hdrSize := binary.Size(wimHeaderDisk{})
	offset := int64(hdrSize)

	var hdr wimHeaderDisk
	hdr.ImageTag = wimImageTag
	hdr.Size = uint32(hdrSize)
	hdr.Version = 0x10d00
	hdr.PartNumber = 1
	hdr.TotalParts = 1
	hdr.ImageCount = 1
	// uncompressed XML resource: flags=0, compressed size = original size
	hdr.XMLData = wimResourceDesc{
		FlagsAndCompressedSize: uint64(len(xmlData)),
		Offset:                 offset,
		OriginalSize:           int64(len(xmlData)),
	}

	buf := &bytes.Buffer{}
	_ = binary.Write(buf, binary.LittleEndian, &hdr)
	buf.Write(xmlData)
	return buf.Bytes()
}

func TestParseWimXmlMetadataWindows11(t *testing.T) {
	xml := `<?xml version="1.0"?>
<WIM>
  <IMAGE INDEX="1">
    <NAME>Windows 11 Pro</NAME>
    <WINDOWS>
      <ARCH>9</ARCH>
      <PRODUCTNAME>Microsoft® Windows® Operating System</PRODUCTNAME>
      <EDITIONID>Professional</EDITIONID>
      <PRODUCTTYPE>WinNT</PRODUCTTYPE>
      <LANGUAGES>
        <LANGUAGE>zh-CN</LANGUAGE>
        <DEFAULT>zh-CN</DEFAULT>
      </LANGUAGES>
      <VERSION>
        <MAJOR>10</MAJOR>
        <MINOR>0</MINOR>
        <BUILD>22631</BUILD>
      </VERSION>
    </WINDOWS>
  </IMAGE>
</WIM>`
	data := buildMinimalWimWithXML(xml)
	info, err := parseWimXmlMetadata(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("parseWimXmlMetadata: %v", err)
	}
	if info.Distro != imagetools.OS_DIST_WINDOWS {
		t.Fatalf("distro: got %s want %s", info.Distro, imagetools.OS_DIST_WINDOWS)
	}
	if info.Version != "Windows 11" {
		t.Fatalf("version: got %s want Windows 11", info.Version)
	}
	if info.Arch != "x86_64" {
		t.Fatalf("arch: got %s want x86_64", info.Arch)
	}
	if info.Language != "zh-CN" {
		t.Fatalf("language: got %s want zh-CN", info.Language)
	}
}

func TestParseWimXmlMetadataWindowsServer2022(t *testing.T) {
	xml := `<?xml version="1.0"?>
<WIM>
  <IMAGE INDEX="1">
    <NAME>Windows Server 2022 SERVERSTANDARD</NAME>
    <WINDOWS>
      <ARCH>9</ARCH>
      <PRODUCTTYPE>ServerNT</PRODUCTTYPE>
      <LANGUAGES>
        <DEFAULT>en-US</DEFAULT>
      </LANGUAGES>
      <VERSION>
        <MAJOR>10</MAJOR>
        <MINOR>0</MINOR>
        <BUILD>20348</BUILD>
      </VERSION>
    </WINDOWS>
  </IMAGE>
</WIM>`
	data := buildMinimalWimWithXML(xml)
	info, err := parseWimXmlMetadata(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("parseWimXmlMetadata: %v", err)
	}
	if info.Distro != imagetools.OS_DIST_WINDOWS_SERVER {
		t.Fatalf("distro: got %s want %s", info.Distro, imagetools.OS_DIST_WINDOWS_SERVER)
	}
	if info.Version != "Windows Server 2022" {
		t.Fatalf("version: got %s want Windows Server 2022", info.Version)
	}
}

func TestParseWimXmlMetadataRejectsCompressedXML(t *testing.T) {
	xmlData := encodeWimUTF16XML(`<?xml version="1.0"?><WIM></WIM>`)
	hdrSize := binary.Size(wimHeaderDisk{})
	var hdr wimHeaderDisk
	hdr.ImageTag = wimImageTag
	hdr.Size = uint32(hdrSize)
	hdr.PartNumber = 1
	hdr.TotalParts = 1
	hdr.XMLData = wimResourceDesc{
		FlagsAndCompressedSize: uint64(wimResFlagCompressed)<<56 | uint64(len(xmlData)),
		Offset:                 int64(hdrSize),
		OriginalSize:           int64(len(xmlData)),
	}
	buf := &bytes.Buffer{}
	_ = binary.Write(buf, binary.LittleEndian, &hdr)
	buf.Write(xmlData)

	_, err := parseWimXmlMetadata(bytes.NewReader(buf.Bytes()))
	if err == nil {
		t.Fatal("expected error for compressed XML")
	}
}

func TestMapWindowsVersion(t *testing.T) {
	cases := []struct {
		name    string
		win     wimWindowsInfo
		distro  string
		version string
	}{
		{
			name: "win10",
			win: wimWindowsInfo{
				Arch: 9, ProductType: "WinNT",
				Version: wimXMLVersion{Major: 10, Minor: 0, Build: 19045},
			},
			distro: imagetools.OS_DIST_WINDOWS, version: "Windows 10",
		},
		{
			name: "win7",
			win: wimWindowsInfo{
				Arch: 9, ProductType: "WinNT",
				Version: wimXMLVersion{Major: 6, Minor: 1, Build: 7601},
			},
			distro: imagetools.OS_DIST_WINDOWS, version: "Windows 7",
		},
		{
			name: "server2019",
			win: wimWindowsInfo{
				Arch: 9, ProductType: "ServerNT",
				Version: wimXMLVersion{Major: 10, Minor: 0, Build: 17763},
			},
			distro: imagetools.OS_DIST_WINDOWS_SERVER, version: "Windows Server 2019",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			info := mapWindowsVersion(&c.win)
			if info.Distro != c.distro || info.Version != c.version {
				t.Fatalf("got %s/%s want %s/%s", info.Distro, info.Version, c.distro, c.version)
			}
		})
	}
}
