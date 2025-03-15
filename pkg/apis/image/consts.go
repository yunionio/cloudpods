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

package image

import "yunion.io/x/onecloud/pkg/apis"

type TImageType string

const (
	SERVICE_TYPE    = apis.SERVICE_TYPE_IMAGE
	SERVICE_VERSION = ""

	// https://docs.openstack.org/glance/pike/user/statuses.html
	//
	IMAGE_STATUS_QUEUED     = "queued"
	IMAGE_STATUS_SAVING     = "saving"
	IMAGE_STATUS_SAVE_FAIL  = "save_fail"
	IMAGE_STATUS_SAVED      = "saved"
	IMAGE_STATUS_ACTIVE     = "active"
	IMAGE_STATUS_CONVERTING = "converting"
	IMAGE_STATUS_PROBING    = "probing"

	IMAGE_ENCRYPT_STATUS_UNENCRYPTED = ""
	IMAGE_ENCRYPT_STATUS_ENCRYPTED   = "encrypted"
	IMAGE_ENCRYPT_STATUS_ENCRYPTING  = "encrypting"

	IMAGE_STATUS_DEACTIVATED    = "deactivated"
	IMAGE_STATUS_KILLED         = "killed"
	IMAGE_STATUS_DELETED        = "deleted"
	IMAGE_STATUS_PENDING_DELETE = "pending_delete"

	IMAGE_STATUS_UNKNOWN = "unkown"

	IMAGE_STATUS_SYNC_CLASS_METADATA_FAILEd = "sync_class_metadata_failed"

	ImageTypeTemplate = TImageType("image")
	ImageTypeISO      = TImageType("iso")
	ImageTypeTarGzip  = TImageType("tgz")

	LocalFilePrefix = "file://"
	S3Prefix        = "s3://"

	IMAGE_STORAGE_DRIVER_LOCAL = "local"
	IMAGE_STORAGE_DRIVER_S3    = "s3"

	// image properties
	IMAGE_OS_ARCH              = "os_arch"
	IMAGE_OS_DISTRO            = "os_distribution"
	IMAGE_OS_TYPE              = "os_type"
	IMAGE_OS_VERSION           = "os_version"
	IMAGE_DISK_FORMAT          = "disk_format"
	IMAGE_UEFI_SUPPORT         = "uefi_support"
	IMAGE_IS_LVM_PARTITION     = "is_lvm_partition"
	IMAGE_IS_READONLY          = "is_readonly"
	IMAGE_PARTITION_TYPE       = "partition_type"
	IMAGE_INSTALLED_CLOUDINIT  = "installed_cloud_init"
	IMAGE_DISABLE_USB_KBD      = "disable_usb_kbd"
	IMAGE_VDI_PROTOCOL         = "vdi_protocol"
	IMAGE_INTERNAL_PATH_MAP    = "internal_path_map"
	IMAGE_USED_BY_POST_OVERLAY = "used_by_post_overlay"

	IMAGE_STATUS_UPDATING = "updating"
)

var (
	ImageDeadStatus = []string{IMAGE_STATUS_DEACTIVATED, IMAGE_STATUS_KILLED, IMAGE_STATUS_DELETED, IMAGE_STATUS_PENDING_DELETE}
)

const (
	IMAGE_DISK_FORMAT_RAW    = "raw"
	IMAGE_DISK_FORMAT_QCOW2  = "qcow2"
	IMAGE_DISK_FORMAT_ISO    = "iso"
	IMAGE_DISK_FORMAT_VMDK   = "vmdk"
	IMAGE_DISK_FORMAT_DOCKER = "docker"
	IMAGE_DISK_FORMAT_VHD    = "vhd"
	IMAGE_DISK_FORMAT_TGZ    = "tgz"
)
