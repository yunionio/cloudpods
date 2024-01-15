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

package netutils

import (
	"fmt"
	"strconv"
	"strings"
)

type SMacAddr [6]byte

func ErrMacFormat(macStr string) error {
	return fmt.Errorf("invalid mac format: %s", macStr)
}

func ParseMac(macStr string) (SMacAddr, error) {
	mac := SMacAddr{}
	macStr = FormatMacAddr(macStr)
	parts := strings.Split(macStr, ":")
	if len(parts) != 6 {
		return mac, ErrMacFormat(macStr)
	}
	for i := 0; i < 6; i += 1 {
		bt, err := strconv.ParseInt(parts[i], 16, 64)
		if err != nil {
			return mac, ErrMacFormat(macStr)
		}
		mac[i] = byte(bt)
	}
	return mac, nil
}

func (mac SMacAddr) Add(step int) SMacAddr {
	mac2 := SMacAddr{}
	leftOver := step
	for i := 5; i >= 0; i -= 1 {
		newByte := int(mac[i]) + leftOver
		res := 0
		if newByte < 0 {
			res = ((-newByte) / 0x100) + 1
			newByte = newByte + res*0x100
		}
		mac2[i] = byte(newByte % 0x100)
		leftOver = newByte/0x100 - res
	}
	return mac2
}

func (mac SMacAddr) String() string {
	var parts [6]string
	for i := 0; i < len(parts); i += 1 {
		parts[i] = fmt.Sprintf("%02x", mac[i])
	}
	return strings.Join(parts[:], ":")
}
