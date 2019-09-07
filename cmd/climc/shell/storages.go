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

package shell

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type StorageListOptions struct {
		options.BaseListOptions

		Share  *bool  `help:"Share storage list"`
		Local  *bool  `help:"Local storage list"`
		Usable *bool  `help:"Usable storage list"`
		Zone   string `help:"List storages in zone" json:"-"`
		Region string `help:"List storages in region"`
	}
	R(&StorageListOptions{}, "storage-list", "List storages", func(s *mcclient.ClientSession, opts *StorageListOptions) error {
		params, err := options.ListStructToParams(opts)
		if err != nil {
			return err
		}
		var result *modulebase.ListResult
		if len(opts.Zone) > 0 {
			result, err = modules.Storages.ListInContext(s, params, &modules.Zones, opts.Zone)
		} else {
			result, err = modules.Storages.List(s, params)
		}
		if err != nil {
			return err
		}
		printList(result, modules.Storages.GetColumns(s))
		return nil
	})

	type StorageUpdateOptions struct {
		ID                    string  `help:"ID or Name of storage to update"`
		Name                  string  `help:"New Name of storage"`
		Desc                  string  `help:"Description"`
		CommitBound           float64 `help:"Upper bound of storage overcommit rate"`
		MediumType            string  `help:"Medium type" choices:"ssd|rotate"`
		RbdRadosMonOpTimeout  int64   `help:"ceph rados_mon_op_timeout"`
		RbdRadosOsdOpTimeout  int64   `help:"ceph rados_osd_op_timeout"`
		RbdClientMountTimeout int64   `help:"ceph client_mount_timeout"`
		RbdKey                string  `help:"ceph rbd key"`
		Reserved              string  `help:"Reserved storage space"`
	}
	R(&StorageUpdateOptions{}, "storage-update", "Update a storage", func(s *mcclient.ClientSession, args *StorageUpdateOptions) error {
		params, err := options.StructToParams(args)
		if err != nil {
			return err
		}

		result, err := modules.Storages.Update(s, args.ID, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type StorageCreateOptions struct {
		NAME                  string `help:"Name of the Storage"`
		ZONE                  string `help:"Zone id of storage"`
		Capacity              int64  `help:"Capacity of the Storage"`
		MediumType            string `help:"Medium type" choices:"ssd|rotate"`
		StorageType           string `help:"Storage type" choices:"local|nas|vsan|rbd|nfs|gpfs|baremetal"`
		RbdMonHost            string `help:"Ceph mon_host config"`
		RbdRadosMonOpTimeout  int64  `help:"ceph rados_mon_op_timeout"`
		RbdRadosOsdOpTimeout  int64  `help:"ceph rados_osd_op_timeout"`
		RbdClientMountTimeout int64  `help:"ceph client_mount_timeout"`
		RbdKey                string `help:"Ceph key config"`
		RbdPool               string `help:"Ceph Pool Name"`
		NfsHost               string `help:"NFS host"`
		NfsSharedDir          string `help:"NFS shared dir"`
	}
	R(&StorageCreateOptions{}, "storage-create", "Create a Storage", func(s *mcclient.ClientSession, args *StorageCreateOptions) error {
		params, err := options.StructToParams(args)
		if err != nil {
			return err
		}

		if args.StorageType == "rbd" {
			if args.RbdMonHost == "" || args.RbdKey == "" || args.RbdPool == "" {
				return fmt.Errorf("Not enough arguments, missing mon_host、key or pool")
			}
		} else if args.StorageType == "nfs" {
			if len(args.NfsHost) == 0 || len(args.NfsSharedDir) == 0 {
				return fmt.Errorf("Storage type nfs missing conf host or shared dir")
			}
		}
		storage, err := modules.Storages.Create(s, params)
		if err != nil {
			return err
		}
		printObject(storage)
		return nil
	})

	type StorageShowOptions struct {
		ID string `help:"ID or Name of the storage to show"`
	}
	R(&StorageShowOptions{}, "storage-show", "Show storage details", func(s *mcclient.ClientSession, args *StorageShowOptions) error {
		result, err := modules.Storages.Get(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&StorageShowOptions{}, "storage-delete", "Delete a storage", func(s *mcclient.ClientSession, args *StorageShowOptions) error {
		result, err := modules.Storages.Delete(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&StorageShowOptions{}, "storage-enable", "Enable a storage", func(s *mcclient.ClientSession, args *StorageShowOptions) error {
		result, err := modules.Storages.PerformAction(s, args.ID, "enable", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&StorageShowOptions{}, "storage-disable", "Disable a storage", func(s *mcclient.ClientSession, args *StorageShowOptions) error {
		result, err := modules.Storages.PerformAction(s, args.ID, "disable", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&StorageShowOptions{}, "storage-online", "Online a storage", func(s *mcclient.ClientSession, args *StorageShowOptions) error {
		result, err := modules.Storages.PerformAction(s, args.ID, "online", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&StorageShowOptions{}, "storage-offline", "Offline a storage", func(s *mcclient.ClientSession, args *StorageShowOptions) error {
		result, err := modules.Storages.PerformAction(s, args.ID, "offline", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type StorageCacheImageActionOptions struct {
		ID     string `help:"ID or name of storage"`
		IMAGE  string `help:"ID or name of image"`
		Force  bool   `help:"Force refresh cache, even if the image exists in cache"`
		Format string `help:"Image force" choices:"iso|vmdk|qcow2|vhd"`
	}
	R(&StorageCacheImageActionOptions{}, "storage-cache-image", "Ask a storage to cache a image", func(s *mcclient.ClientSession, args *StorageCacheImageActionOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.IMAGE), "image")
		if args.Force {
			params.Add(jsonutils.JSONTrue, "is_force")
		}
		if len(args.Format) > 0 {
			params.Add(jsonutils.NewString(args.Format), "format")
		}
		storage, err := modules.Storages.PerformAction(s, args.ID, "cache-image", params)
		if err != nil {
			return err
		}
		printObject(storage)
		return nil
	})

	type StorageUncacheImageActionOptions struct {
		ID    string `help:"ID or name of storage"`
		IMAGE string `help:"ID or name of image"`
		Force bool   `help:"Force uncache, even if the image exists in cache"`
	}
	R(&StorageUncacheImageActionOptions{}, "storage-uncache-image", "Ask a storage to remove image from its cache", func(s *mcclient.ClientSession, args *StorageUncacheImageActionOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.IMAGE), "image")
		if args.Force {
			params.Add(jsonutils.JSONTrue, "is_force")
		}
		storage, err := modules.Storages.PerformAction(s, args.ID, "uncache-image", params)
		if err != nil {
			return err
		}
		printObject(storage)
		return nil
	})
}
