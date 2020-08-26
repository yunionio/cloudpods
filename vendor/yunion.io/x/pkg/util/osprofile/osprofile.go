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

package osprofile

import (
	"fmt"
	"strings"

	"yunion.io/x/pkg/utils"
)

const (
	OS_TYPE_MACOS   = "macOS"
	OS_TYPE_VMWARE  = "VMWare"
	OS_TYPE_LINUX   = "Linux"
	OS_TYPE_WINDOWS = "Windows"
	OS_TYPE_ANDROID = "Android"
)

var OS_TYPES = []string{OS_TYPE_MACOS, OS_TYPE_VMWARE, OS_TYPE_LINUX, OS_TYPE_WINDOWS, OS_TYPE_ANDROID}

var FS_TYPES = []string{"swap", "ext2", "ext3", "ext4", "xfs", "ntfs", "fat", "hfsplus"}

var IMAGE_FORMAT_TYPES = []string{"qcow2", "raw", "docker", "iso", "vmdk", "vmdkflatver1", "vmdkflatver2", "vmdkflat",
	"vmdksparse", "vmdksparsever1", "vmdksparsever2", "vmdksesparse", "vhd"}

var DISK_DRIVERS = []string{"virtio", "ide", "scsi", "sata", "pvscsi"}

var DISK_CACHE_MODES = []string{"writeback", "none", "writethrough"}

type SOSProfile struct {
	DiskDriver string
	NetDriver  string
	FsFormat   string
	OSType     string
	Hypervisor string
}

func GetOSProfile(osname string, hypervisor string) SOSProfile {
	switch osname {
	case OS_TYPE_MACOS:
		return SOSProfile{
			DiskDriver: "sata",
			NetDriver:  "e1000",
			FsFormat:   "hfsplus",
		}
	case OS_TYPE_VMWARE:
		return SOSProfile{
			DiskDriver: "ide",
			NetDriver:  "vmxnet3",
		}
	case OS_TYPE_WINDOWS:
		if hypervisor == "esxi" {
			return SOSProfile{
				DiskDriver: "ide",
				NetDriver:  "e1000",
				FsFormat:   "ntfs",
			}
		} else {
			return SOSProfile{
				DiskDriver: "scsi",
				NetDriver:  "virtio",
				FsFormat:   "ntfs",
			}
		}
	case OS_TYPE_LINUX:
		if hypervisor == "esxi" {
			return SOSProfile{
				DiskDriver: "pvscsi",
				NetDriver:  "vmxnet3",
				FsFormat:   "ext4",
			}
		} else {
			return SOSProfile{
				DiskDriver: "scsi",
				NetDriver:  "virtio",
				FsFormat:   "ext4",
			}
		}
	default:
		if hypervisor == "esxi" {
			return SOSProfile{
				DiskDriver: "scsi",
				NetDriver:  "e1000",
			}
		} else {
			return SOSProfile{
				DiskDriver: "ide",
				NetDriver:  "e1000",
			}
		}
	}
}

func NormalizeOSType(osname string) string {
	for _, n := range OS_TYPES {
		if strings.ToLower(n) == osname {
			return n
		}
	}
	return osname
}

func GetOSProfileFromImageProperties(imgProp map[string]string, hypervisor string) (SOSProfile, error) {
	osType, ok := imgProp["os_type"]
	if !ok {
		return SOSProfile{}, fmt.Errorf("Missing os_type in image properties")
	}
	var imgHypers []string
	imgHyperStr, ok := imgProp["hypervisor"]
	if ok && len(imgHyperStr) > 0 {
		imgHypers = strings.Split(imgHyperStr, ",")
	} else {
		imgHypers = []string{}
	}
	if len(hypervisor) == 0 && len(imgHypers) > 0 {
		hypervisor = imgHypers[0]
	} else if len(imgHypers) > 0 && len(hypervisor) > 0 && !utils.IsInStringArray(hypervisor, imgHypers) {
		return SOSProfile{}, fmt.Errorf("The template requires hypervisor %s", hypervisor)
	}
	osprofile := GetOSProfile(osType, hypervisor)
	diskDriver, ok := imgProp["disk_driver"]
	if ok && len(diskDriver) > 0 {
		osprofile.DiskDriver = diskDriver
	}
	netDriver, ok := imgProp["net_driver"]
	if ok && len(netDriver) > 0 {
		osprofile.NetDriver = netDriver
	}
	osprofile.OSType = osType
	if len(hypervisor) > 0 {
		osprofile.Hypervisor = hypervisor
	}
	return osprofile, nil
}
