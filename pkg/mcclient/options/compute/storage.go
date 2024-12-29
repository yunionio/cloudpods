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

package compute

import (
	"fmt"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type StorageListOptions struct {
	options.BaseListOptions

	Share    *bool  `help:"Share storage list"`
	Local    *bool  `help:"Local storage list"`
	Usable   *bool  `help:"Usable storage list"`
	Zone     string `help:"List storages in zone" json:"-"`
	Region   string `help:"List storages in region"`
	Schedtag string `help:"filter storage by schedtag"`
	HostId   string `help:"filter storages which attached the specified host"`

	HostSchedtagId string `help:"filter storage by host schedtag"`
	ImageId        string `help:"filter storage by image"`
	IsBaremetal    *bool  `help:"Baremetal storage list"`
}

func (opts *StorageListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(opts)
}

func (opts *StorageListOptions) GetContextId() string {
	return opts.Zone
}

type StorageUpdateOptions struct {
	options.BaseUpdateOptions
	MediumType            string `help:"Medium type" choices:"ssd|rotate"`
	RbdRadosMonOpTimeout  int64  `help:"ceph rados_mon_op_timeout"`
	RbdRadosOsdOpTimeout  int64  `help:"ceph rados_osd_op_timeout"`
	RbdClientMountTimeout int64  `help:"ceph client_mount_timeout"`
	RbdEnableMessengerV2  bool   `help:"ceph enable Messenger V2"`
	RbdKey                string `help:"ceph rbd key"`
	Reserved              string `help:"Reserved storage space"`
	Capacity              int    `help:"Capacity for storage"`
	MasterHost            string `help:"slvm storage master host"`
}

func (opts *StorageUpdateOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(opts)
}

type StorageCreateOptions struct {
	NAME                  string `help:"Name of the Storage"`
	ZONE                  string `help:"Zone id of storage"`
	Capacity              int64  `help:"Capacity of the Storage"`
	MediumType            string `help:"Medium type" choices:"ssd|rotate" default:"ssd"`
	StorageType           string `help:"Storage type" choices:"local|nas|vsan|rbd|nfs|gpfs|baremetal|clvm|slvm"`
	RbdMonHost            string `help:"Ceph mon_host config"`
	RbdEnableMessengerV2  bool   `help:"ceph enable Messenger V2"`
	RbdRadosMonOpTimeout  int64  `help:"ceph rados_mon_op_timeout"`
	RbdRadosOsdOpTimeout  int64  `help:"ceph rados_osd_op_timeout"`
	RbdClientMountTimeout int64  `help:"ceph client_mount_timeout"`
	RbdKey                string `help:"Ceph key config"`
	RbdPool               string `help:"Ceph Pool Name"`
	NfsHost               string `help:"NFS host"`
	NfsSharedDir          string `help:"NFS shared dir"`
	ClvmVgName            string `help:"clvm vg name"`
	SlvmVgName            string `help:"slvm vg name"`
	Lvmlockd              bool   `help:"shared lvm storage use lvmlockd"`
	MasterHost            string `help:"slvm storage master host"`
}

func (opts *StorageCreateOptions) Params() (jsonutils.JSONObject, error) {
	if opts.StorageType == "rbd" {
		if opts.RbdMonHost == "" || opts.RbdPool == "" {
			return nil, fmt.Errorf("Not enough arguments, missing mon_hostor pool")
		}
	} else if opts.StorageType == "nfs" {
		if len(opts.NfsHost) == 0 || len(opts.NfsSharedDir) == 0 {
			return nil, fmt.Errorf("Storage type nfs missing conf host or shared dir")
		}
	} else if opts.StorageType == "clvm" {
		if len(opts.ClvmVgName) == 0 {
			return nil, fmt.Errorf("Storage type clvm missing conf clvm_vg_name")
		}
	} else if opts.StorageType == "slvm" {
		if len(opts.SlvmVgName) == 0 {
			return nil, fmt.Errorf("Storage type slvm missing conf slvm_vg_name")
		}
	}
	return options.StructToParams(opts)
}

type StorageForceDetachHost struct {
	options.BaseIdOptions
	HOST string `help:"ID or name of host"`
}

func (opts *StorageForceDetachHost) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(map[string]string{"host": opts.HOST}), nil
}

type StorageSetHardwareInfoOptions struct {
	options.BaseIdOptions
	compute.StorageHardwareInfo
}

func (o *StorageSetHardwareInfoOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(o), nil
}

type StorageSetCommitBoundOptions struct {
	options.BaseIdOptions
	Cmtbound *float32 `help:"Storage commit bound"`
}

func (o *StorageSetCommitBoundOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(o)
}
