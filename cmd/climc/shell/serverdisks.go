package shell

import (
	"fmt"
	"github.com/yunionio/jsonutils"
	"github.com/yunionio/mcclient"
	"github.com/yunionio/mcclient/modules"
)

func init() {
	type ServerDiskListOptions struct {
		BaseListOptions
		Server string `help:"ID or Name of Server"`
	}
	R(&ServerDiskListOptions{}, "server-disk-list", "List server disk pairs", func(s *mcclient.ClientSession, args *ServerDiskListOptions) error {
		params := FetchPagingParams(args.BaseListOptions)
		var result *modules.ListResult
		var err error
		if len(args.Server) > 0 {
			result, err = modules.Serverdisks.ListDescendent(s, args.Server, params)
		} else {
			result, err = modules.Serverdisks.List(s, params)
		}
		if err != nil {
			return err
		}
		printList(result, modules.Serverdisks.GetColumns(s))
		return nil
	})

	type ServerDiskDetailOptions struct {
		SERVER string `help:"ID or Name of Server"`
		DISK   string `help:"ID or Name of Disk"`
	}
	R(&ServerDiskDetailOptions{}, "server-disk-show", "Show server disk details", func(s *mcclient.ClientSession, args *ServerDiskDetailOptions) error {
		result, err := modules.Serverdisks.Get(s, args.SERVER, args.DISK, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type ServerDiskUpdateOptions struct {
		SERVER string `help:"ID or Name of server"`
		DISK   string `help:"ID or Name of Disk"`
		Driver string `help:"Driver of vDisk" choices:"virtio|ide|scsi|pvscsi"`
		Cache  string `help:"Cache mode of vDisk" choices:"writethrough|none|writeback"`
		Aio    string `help:"Asynchronous IO mode of vDisk" choices:"native|threads"`
		Index  int64  `help:"Index of vDisk" default:"-1"`
	}
	R(&ServerDiskUpdateOptions{}, "server-disk-update", "Update details of a virtual disk of a virtual server", func(s *mcclient.ClientSession, args *ServerDiskUpdateOptions) error {
		params := jsonutils.NewDict()
		if len(args.Driver) > 0 {
			params.Add(jsonutils.NewString(args.Driver), "driver")
		}
		if len(args.Cache) > 0 {
			params.Add(jsonutils.NewString(args.Cache), "cache_mode")
		}
		if len(args.Aio) > 0 {
			params.Add(jsonutils.NewString(args.Aio), "aio_mode")
		}
		if args.Index >= 0 {
			params.Add(jsonutils.NewInt(args.Index), "index")
		}
		if params.Size() == 0 {
			return InvalidUpdateError()
		}
		srv, err := modules.Serverdisks.Update(s, args.SERVER, args.DISK, params)
		if err != nil {
			return err
		}
		printObject(srv)
		return nil
	})

	type ServerCreateDiskOptions struct {
		SERVER string   `help:"ID or Name of server"`
		DISK   []string `help:"Disk description of a virtual disk"`
	}
	R(&ServerCreateDiskOptions{}, "server-create-disk", "Create a disk and attach it to a virtual server", func(s *mcclient.ClientSession, args *ServerCreateDiskOptions) error {
		params := jsonutils.NewDict()
		for i, d := range args.DISK {
			params.Add(jsonutils.NewString(d), fmt.Sprintf("disk.%d", i))
		}
		srv, err := modules.Servers.PerformAction(s, args.SERVER, "createdisk", params)
		if err != nil {
			return err
		}
		printObject(srv)
		return nil
	})

	type ServerAttachDiskOptions struct {
		SERVER string `help:"ID or name of server"`
		DISK   string `help:"ID of name of disk to attach"`
		Driver string `help:"Driver" choices:"virtio|ide|scsi"`
		Cache  string `help:"Cache mode" choices:"writeback|none|writethrought"`
	}
	R(&ServerAttachDiskOptions{}, "server-attach-disk", "Attach an existing virtual disks to a virtual server", func(s *mcclient.ClientSession, args *ServerAttachDiskOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.DISK), "disk_id")
		if len(args.Driver) > 0 {
			params.Add(jsonutils.NewString(args.Driver), "driver")
		}
		if len(args.Cache) > 0 {
			params.Add(jsonutils.NewString(args.Cache), "cache")
		}
		srv, err := modules.Servers.PerformAction(s, args.SERVER, "attachdisk", params)
		if err != nil {
			return err
		}
		printObject(srv)
		return nil
	})

	type ServerDetachDiskOptions struct {
		SERVER     string `help:"ID or name of server"`
		DISK       string `help:"ID or name of disk to detach"`
		DeleteDisk bool   `help:"Delete disk if the disk not has flag of auto_delete when detached"`
	}
	R(&ServerDetachDiskOptions{}, "server-detach-disk", "Detach a disk from a virtual server", func(s *mcclient.ClientSession, args *ServerDetachDiskOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.DISK), "disk_id")
		if args.DeleteDisk {
			params.Add(jsonutils.NewInt(0), "keep_disk")
		}
		srv, err := modules.Servers.PerformAction(s, args.SERVER, "detachdisk", params)
		if err != nil {
			return err
		}
		printObject(srv)
		return nil
	})

}
