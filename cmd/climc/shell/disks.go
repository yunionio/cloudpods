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
		Unused   bool   `help:"Show unused disks"`
		Share    bool   `help:"Show Share storage disks"`
		Local    bool   `help:"Show Local storage disks"`
		Guest    string `help:"Guest ID or name"`
		Storage  string `help:"Storage ID or name"`
		Provider string `help:"Provider for disk" choices:"Aliyun|VMware|Azure"`
	}
	R(&DiskListOptions{}, "disk-list", "List virtual disks", func(s *mcclient.ClientSession, suboptions *DiskListOptions) error {
		var params *jsonutils.JSONDict
		{
			var err error
			params, err = suboptions.BaseListOptions.Params()
			if err != nil {
				return err

			}
		}
		if suboptions.Unused {
			params.Add(jsonutils.JSONTrue, "unused")
		}
		if suboptions.Share {
			params.Add(jsonutils.JSONTrue, "share")
		}
		if suboptions.Local {
			params.Add(jsonutils.JSONTrue, "local")
		}
		if len(suboptions.Guest) > 0 {
			params.Add(jsonutils.NewString(suboptions.Guest), "guest")
		}
		if len(suboptions.Storage) > 0 {
			params.Add(jsonutils.NewString(suboptions.Storage), "storage")
		}
		if len(suboptions.Provider) > 0 {
			params.Add(jsonutils.NewString(suboptions.Provider), "provider")
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
		OverridePendingDelete bool     `help:"Delete disk directly instead of pending delete"`
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

	R(&DiskDetailOptions{}, "disk-purge", "Delete a disk record in database, not actually do deletion", func(s *mcclient.ClientSession, args *DiskDetailOptions) error {
		disk, e := modules.Disks.PerformAction(s, args.ID, "purge", nil)
		if e != nil {
			return e
		}
		printObject(disk)
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

	type DiskCreateOptions struct {
		STORAGE    string `help:"ID or name of storage where the disk is created"`
		NAME       string `help:"Name of the disk"`
		DISKDESC   string `help:"Image size or size of virtual disk"`
		Desc       string `help:"Description" metavar:"Description"`
		TaskNotify bool   `help:"Setup task notify"`
	}
	R(&DiskCreateOptions{}, "disk-create", "Create a virtual disk", func(s *mcclient.ClientSession, args *DiskCreateOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.NAME), "name")
		params.Add(jsonutils.NewString(args.DISKDESC), "disk")
		if len(args.Desc) > 0 {
			params.Add(jsonutils.NewString(args.Desc), "description")
		}
		if args.TaskNotify {
			s.PrepareTask()
		}
		disk, err := modules.Disks.CreateInContext(s, params, &modules.Storages, args.STORAGE)
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
}
