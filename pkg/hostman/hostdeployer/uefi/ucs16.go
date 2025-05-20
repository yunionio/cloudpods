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

package uefi

import (
	"unicode/utf16"
)

// ExtractUCS16String extracts a UCS-16 string from a byte array
// Returns the string data and the total length (including null terminator)
func ExtractUCS16String(data []byte) ([]byte, uint32) {
	// Find the null terminator (two consecutive zero bytes)
	var i int
	for i = 0; i < len(data)-1; i += 2 {
		if data[i] == 0 && data[i+1] == 0 {
			break
		}
	}

	// Include the null terminator in the length
	strLen := i + 2

	// If we reached the end without finding a null terminator,
	// use the entire data length
	if i >= len(data)-1 {
		strLen = len(data)
		i = len(data)
		if i%2 != 0 {
			i--
		}
	}

	// Return the string data and length
	return data[:i], uint32(strLen)
}

// DecodeUTF16LE decodes a UTF-16LE byte array to a string
func DecodeUTF16LE(b []byte) string {
	// Check if the byte array is empty
	if len(b) == 0 {
		return ""
	}

	// Convert bytes to uint16 array
	u16s := make([]uint16, len(b)/2)
	for i := range u16s {
		// Little-endian: low byte first, then high byte
		u16s[i] = uint16(b[i*2]) | (uint16(b[i*2+1]) << 8)
	}

	// Decode UTF-16 to UTF-8
	return string(utf16.Decode(u16s))
}

// EncodeUTF16LE encodes a string to UTF-16LE bytes
func EncodeUTF16LE(s string) []byte {
	u16s := utf16.Encode([]rune(s))
	bytes := make([]byte, len(u16s)*2)
	for i, u16 := range u16s {
		bytes[i*2] = byte(u16)
		bytes[i*2+1] = byte(u16 >> 8)
	}
	return bytes
}
