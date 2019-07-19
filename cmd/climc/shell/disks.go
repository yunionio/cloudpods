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
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type DiskListOptions struct {
		options.BaseListOptions
		Unused    *bool  `help:"Show unused disks"`
		Share     *bool  `help:"Show Share storage disks"`
		Local     *bool  `help:"Show Local storage disks"`
		Guest     string `help:"Guest ID or name"`
		Storage   string `help:"Storage ID or name"`
		Type      string `help:"Disk type" choices:"sys|data|swap|volume"`
		CloudType string `help:"Public cloud or private cloud" choices:"Public|Private"`

		BillingType string `help:"billing type" choices:"postpaid|prepaid"`
	}
	R(&DiskListOptions{}, "disk-list", "List virtual disks", func(s *mcclient.ClientSession, opts *DiskListOptions) error {
		params, err := options.ListStructToParams(opts)
		if err != nil {
			return err
		}
		if len(opts.CloudType) > 0 {
			if opts.CloudType == "Public" {
				params.Add(jsonutils.JSONTrue, "public_cloud")
			} else if opts.CloudType == "Private" {
				params.Add(jsonutils.JSONTrue, "private_cloud")
			}
		}

		result, err := modules.Disks.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.Disks.GetColumns(s))
		return nil
	})

	type DiskDetailOptions struct {
		ID string `help:"ID or Name of disk"`
	}
	R(&DiskDetailOptions{}, "disk-show", "Show details of disk", func(s *mcclient.ClientSession, args *DiskDetailOptions) error {
		disk, e := modules.Disks.Get(s, args.ID, nil)
		if e != nil {
			return e
		}
		printObject(disk)
		return nil
	})
	R(&DiskDetailOptions{}, "disk-cancel-delete", "Cancel pending delete disks", func(s *mcclient.ClientSession, args *DiskDetailOptions) error {
		disk, e := modules.Disks.PerformAction(s, args.ID, "cancel-delete", nil)
		if e != nil {
			return e
		}
		printObject(disk)
		return nil
	})

	type DiskDeleteOptions struct {
		ID                    []string `help:"ID of disks to delete" metavar:"DISK"`
		OverridePendingDelete bool     `help:"Delete disk directly instead of pending delete" short-token:"f"`
	}

	R(&DiskDeleteOptions{}, "disk-delete", "Delete a disk", func(s *mcclient.ClientSession, args *DiskDeleteOptions) error {
		params := jsonutils.NewDict()
		if args.OverridePendingDelete {
			params.Add(jsonutils.JSONTrue, "override_pending_delete")
		}
		ret := modules.Disks.BatchDeleteWithParam(s, args.ID, params, nil)
		printBatchResults(ret, modules.Disks.GetColumns(s))
		return nil
	})

	type DiskBatchOpsOptions struct {
		ID []string `help:"id list of disks to operate"`
	}
	R(&DiskBatchOpsOptions{}, "disk-purge", "Delete a disk record in database, not actually do deletion", func(s *mcclient.ClientSession, args *DiskBatchOpsOptions) error {
		ret := modules.Disks.BatchPerformAction(s, args.ID, "purge", nil)
		printBatchResults(ret, modules.Disks.GetColumns(s))
		return nil
	})

	R(&DiskDetailOptions{}, "disk-public", "Make a disk public", func(s *mcclient.ClientSession, args *DiskDetailOptions) error {
		disk, e := modules.Disks.PerformAction(s, args.ID, "public", nil)
		if e != nil {
			return e
		}
		printObject(disk)
		return nil
	})

	R(&DiskDetailOptions{}, "disk-private", "Make a disk private", func(s *mcclient.ClientSession, args *DiskDetailOptions) error {
		disk, e := modules.Disks.PerformAction(s, args.ID, "private", nil)
		if e != nil {
			return e
		}
		printObject(disk)
		return nil
	})

	R(&DiskDetailOptions{}, "disk-metadata", "Get metadata of a disk", func(s *mcclient.ClientSession, args *DiskDetailOptions) error {
		meta, e := modules.Disks.GetMetadata(s, args.ID, nil)
		if e != nil {
			return e
		}
		printObject(meta)
		return nil
	})

	type DiskUpdateOptions struct {
		ID           string `help:"ID or name of disk"`
		Name         string `help:"New name of disk"`
		Desc         string `help:"Description" metavar:"DESCRIPTION"`
		AutoDelete   string `help:"enable/disable auto delete of disk" choices:"enable|disable"`
		AutoSnapshot string `help:"enable/disable auto snapshot of disk" choices:"enable|disable"`
		DiskType     string `help:"Disk type" choices:"data|volume"`
	}
	R(&DiskUpdateOptions{}, "disk-update", "Update property of a virtual disk", func(s *mcclient.ClientSession, args *DiskUpdateOptions) error {
		params := jsonutils.NewDict()
		if len(args.Name) > 0 {
			params.Add(jsonutils.NewString(args.Name), "name")
		}
		if len(args.Desc) > 0 {
			params.Add(jsonutils.NewString(args.Desc), "description")
		}
		if len(args.AutoDelete) > 0 {
			if args.AutoDelete == "enable" {
				params.Add(jsonutils.JSONTrue, "auto_delete")
			} else {
				params.Add(jsonutils.JSONFalse, "auto_delete")
			}
		}
		if len(args.AutoSnapshot) > 0 {
			if args.AutoSnapshot == "enable" {
				params.Add(jsonutils.JSONTrue, "auto_snapshot")
			} else {
				params.Add(jsonutils.JSONFalse, "auto_snapshot")
			}
		}
		if len(args.DiskType) > 0 {
			params.Add(jsonutils.NewString(args.DiskType), "disk_type")
		}
		if params.Size() == 0 {
			return InvalidUpdateError()
		}
		disk, e := modules.Disks.Update(s, args.ID, params)
		if e != nil {
			return e
		}
		printObject(disk)
		return nil
	})

	R(&options.DiskCreateOptions{}, "disk-create", "Create a virtual disk", func(s *mcclient.ClientSession, args *options.DiskCreateOptions) error {
		params, err := args.Params()
		if err != nil {
			return err
		}
		if args.TaskNotify {
			s.PrepareTask()
		}
		disk, err := modules.Disks.Create(s, params.JSON(params))
		if err != nil {
			return err
		}
		printObject(disk)
		if args.TaskNotify {
			s.WaitTaskNotify()
		}
		return nil
	})

	type DiskResizeOptions struct {
		DISK string `help:"ID or name of disk"`
		SIZE string `help:"Size of disk"`
	}
	R(&DiskResizeOptions{}, "disk-resize", "Resize a disk", func(s *mcclient.ClientSession, args *DiskResizeOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.SIZE), "size")
		disk, err := modules.Disks.PerformAction(s, args.DISK, "resize", params)
		if err != nil {
			return err
		}
		printObject(disk)
		return nil
	})
	type DiskResetOptions struct {
		DISK      string `help:"ID or name of disk"`
		SNAPSHOT  string `help:"snapshots ID of disk"`
		AutoStart bool   `help:"Autostart guest"`
	}
	R(&DiskResetOptions{}, "disk-reset", "Resize a disk", func(s *mcclient.ClientSession, args *DiskResetOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.SNAPSHOT), "snapshot_id")
		if args.AutoStart {
			params.Add(jsonutils.JSONTrue, "auto_start")
		}
		disk, err := modules.Disks.PerformAction(s, args.DISK, "disk-reset", params)
		if err != nil {
			return err
		}
		printObject(disk)
		return nil
	})
	type DiskCreateSnapshotOptions struct {
		DISK          string `help:"ID or name of disk"`
		SNAPSHOT_NAME string `help:"Snapshot name"`
	}
	R(&DiskCreateSnapshotOptions{}, "disk-create-snapshot", "Disk create snapshot", func(s *mcclient.ClientSession, args *DiskCreateSnapshotOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.SNAPSHOT_NAME), "name")
		disk, err := modules.Disks.PerformAction(s, args.DISK, "create-snapshot", params)
		if err != nil {
			return err
		}
		printObject(disk)
		return nil
	})

	type DiskSaveOptions struct {
		ID     string `help:"ID or name of the disk" json:"-"`
		NAME   string `help:"Image name"`
		OSTYPE string `help:"Os type" choices:"Linux|Windows|VMware" json:"-"`
		Public *bool  `help:"Make the image public available" json:"is_public"`
		Format string `help:"image format" choices:"vmdk|qcow2"`
		Notes  string `help:"Notes about the image"`
	}
	R(&DiskSaveOptions{}, "disk-save", "Disk save image", func(s *mcclient.ClientSession, args *DiskSaveOptions) error {
		params, err := options.StructToParams(args)
		if err != nil {
			return err
		}
		params.Add(jsonutils.NewString(args.OSTYPE), "properties", "os_type")
		disk, err := modules.Disks.PerformAction(s, args.ID, "save", params)
		if err != nil {
			return err
		}
		printObject(disk)
		return nil
	})

	type DiskUpdateStatusOptions struct {
		ID     string `help:"ID or name of disk"`
		STATUS string `help:"Disk status" choices:"ready"`
	}
	R(&DiskUpdateStatusOptions{}, "disk-update-status", "Set disk status", func(s *mcclient.ClientSession, args *DiskUpdateStatusOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.STATUS), "status")
		disk, err := modules.Disks.PerformAction(s, args.ID, "status", params)
		if err != nil {
			return err
		}
		printObject(disk)
		return nil
	})
	type DiskApplySnapshotPolicy struct {
		ID             string `help:"ID or name of disk" json:"-"`
		Snapshotpolicy string `help:"ID or name of snapshot policy"`
	}
	R(&DiskApplySnapshotPolicy{}, "disk-apply-snapshot-policy", "Set disk snapshot policy", func(s *mcclient.ClientSession, args *DiskApplySnapshotPolicy) error {
		params, err := options.StructToParams(args)
		if err != nil {
			return err
		}
		disk, err := modules.Disks.PerformAction(s, args.ID, "apply-snapshot-policy", params)
		if err != nil {
			return err
		}
		printObject(disk)
		return nil
	})
	type DiskCancelSnapshotPolicy struct {
		ID string `help:"ID or name of disk"`
	}
	R(&DiskCancelSnapshotPolicy{}, "disk-cancel-snapshot-policy", "Cancel disk snapshot policy", func(s *mcclient.ClientSession, args *DiskCancelSnapshotPolicy) error {
		disk, err := modules.Disks.PerformAction(s, args.ID, "cancel-snapshot-policy", nil)
		if err != nil {
			return err
		}
		printObject(disk)
		return nil
	})

	type DiskChangeOwnerOptions struct {
		ID      string `help:"Disk to change owner" json:"-"`
		PROJECT string `help:"Project ID or change" json:"tenant"`
	}
	R(&DiskChangeOwnerOptions{}, "disk-change-owner", "Change owner porject of a disk", func(s *mcclient.ClientSession, opts *DiskChangeOwnerOptions) error {
		params, err := options.StructToParams(opts)
		if err != nil {
			return err
		}
		srv, err := modules.Disks.PerformAction(s, opts.ID, "change-owner", params)
		if err != nil {
			return err
		}
		printObject(srv)
		return nil
	})
}
