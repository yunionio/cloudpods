/*
Copyright (c) 2024-2024 VMware, Inc. All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package types

import (
	"fmt"
	"regexp"
	"strconv"
)

// HardwareVersion is a VMX hardware version.
type HardwareVersion uint8

const (
	VMX3 HardwareVersion = iota + 3
	VMX4

	vmx5 // invalid

	VMX6
	VMX7
	VMX8
	VMX9
	VMX10
	VMX11
	VMX12
	VMX13
	VMX14
	VMX15
	VMX16
	VMX17
	VMX18
	VMX19
	VMX20
	VMX21
)

const (
	// MinValidHardwareVersion is the minimum, valid hardware version supported
	// by VMware hypervisors in the wild.
	MinValidHardwareVersion = VMX3

	// MaxValidHardwareVersion is the maximum, valid hardware version supported
	// by VMware hypervisors in the wild.
	MaxValidHardwareVersion = VMX21
)

func (hv HardwareVersion) IsValid() bool {
	return hv != vmx5 &&
		hv >= MinValidHardwareVersion &&
		hv <= MaxValidHardwareVersion
}

func (hv HardwareVersion) String() string {
	if hv.IsValid() {
		return fmt.Sprintf("vmx-%d", hv)
	}
	return ""
}

func (hv HardwareVersion) MarshalText() ([]byte, error) {
	return []byte(hv.String()), nil
}

func (hv *HardwareVersion) UnmarshalText(text []byte) error {
	v, err := ParseHardwareVersion(string(text))
	if err != nil {
		return err
	}
	*hv = v
	return nil
}

var vmxRe = regexp.MustCompile(`(?i)^vmx-(\d+)$`)

// MustParseHardwareVersion parses the provided string into a hardware version.
func MustParseHardwareVersion(s string) HardwareVersion {
	v, err := ParseHardwareVersion(s)
	if err != nil {
		panic(err)
	}
	return v
}

// ParseHardwareVersion parses the provided string into a hardware version.
func ParseHardwareVersion(s string) (HardwareVersion, error) {
	var u uint64
	if m := vmxRe.FindStringSubmatch(s); len(m) > 0 {
		u, _ = strconv.ParseUint(m[1], 10, 8)
	} else {
		u, _ = strconv.ParseUint(s, 10, 8)
	}
	v := HardwareVersion(u)
	if !v.IsValid() {
		return 0, fmt.Errorf("invalid version: %q", s)
	}
	return v, nil
}

var hardwareVersions []HardwareVersion

func init() {
	for i := MinValidHardwareVersion; i <= MaxValidHardwareVersion; i++ {
		if i.IsValid() {
			hardwareVersions = append(hardwareVersions, i)
		}
	}
}

// GetHardwareVersions returns a list of hardware versions.
func GetHardwareVersions() []HardwareVersion {
	dst := make([]HardwareVersion, len(hardwareVersions))
	copy(dst, hardwareVersions)
	return dst
}
