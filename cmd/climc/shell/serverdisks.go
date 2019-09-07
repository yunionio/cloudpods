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
	type ServerDiskListOptions struct {
		options.BaseListOptions
		Server string `help:"ID or Name of Server"`
		Disk   string `help:"ID or name of disk"`
		Index  int64  `help:"disk index" default:"-1"`
	}
	R(&ServerDiskListOptions{}, "server-disk-list", "List server disk pairs", func(s *mcclient.ClientSession, args *ServerDiskListOptions) error {
		var params *jsonutils.JSONDict
		{
			var err error
			params, err = args.BaseListOptions.Params()
			if err != nil {
				return err

			}
		}
		if args.Index >= 0 {
			params.Add(jsonutils.NewInt(args.Index), "index")
		}
		var result *modulebase.ListResult
		var err error
		if len(args.Server) > 0 {
			result, err = modules.Serverdisks.ListDescendent(s, args.Server, params)
		} else if len(args.Disk) > 0 {
			result, err = modules.Serverdisks.ListDescendent2(s, args.Disk, params)
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
		srv, err := modules.Serverdisks.Update(s, args.SERVER, args.DISK, nil, params)
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
