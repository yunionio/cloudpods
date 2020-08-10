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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/cmdline"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/ssh"
)

func init() {

	R(&options.ServerListOptions{}, "server-list", "List virtual servers", func(s *mcclient.ClientSession, opts *options.ServerListOptions) error {
		params, err := options.ListStructToParams(opts)
		if err != nil {
			return err
		}
		result, err := modules.Servers.List(s, params)
		if err != nil {
			return err
		}
		if len(opts.ExportFile) > 0 {
			exportList(result, opts.ExportFile, opts.ExportKeys, opts.ExportTexts, modules.Servers.GetColumns(s))
		} else {
			printList(result, modules.Servers.GetColumns(s))
		}
		return nil
	})

	R(&options.ServerShowOptions{}, "server-show", "Show details of a server", func(s *mcclient.ClientSession, opts *options.ServerShowOptions) error {
		params, err := options.StructToParams(opts)
		if err != nil {
			return err
		}
		result, err := modules.Servers.Get(s, opts.ID, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type ServerTaskShowOptions struct {
		ID       string `help:"ID or name of server" json:"-"`
		Since    string `help:"show tasks since this time point"`
		Open     bool   `help:"show tasks that are not completed" json:"-"`
		Complete bool   `help:"show tasks that has been completed" json:"-"`
	}
	R(&ServerTaskShowOptions{}, "server-tasks", "Show tasks of a server", func(s *mcclient.ClientSession, opts *ServerTaskShowOptions) error {
		params, err := options.StructToParams(opts)
		if err != nil {
			return err
		}
		if opts.Open {
			params.Add(jsonutils.JSONTrue, "is_open")
		} else if opts.Complete {
			params.Add(jsonutils.JSONFalse, "is_open")
		}
		result, err := modules.Servers.GetSpecific(s, opts.ID, "tasks", params)
		if err != nil {
			return err
		}
		tasks, err := result.GetArray("tasks")
		if err != nil {
			return err
		}
		listResult := modulebase.ListResult{}
		listResult.Data = tasks
		printList(&listResult, nil)
		return nil
	})

	R(&options.ServerIdOptions{}, "server-metadata", "Show metadata of a server", func(s *mcclient.ClientSession, opts *options.ServerIdOptions) error {
		result, err := modules.Servers.GetMetadata(s, opts.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&options.ServerBatchMetadataOptions{}, "server-batch-add-tag", "add tags for some server", func(s *mcclient.ClientSession, opts *options.ServerBatchMetadataOptions) error {
		params, err := opts.Params()
		if err != nil {
			return err
		}
		result, err := modules.Servers.PerformClassAction(s, "batch-user-metadata", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&options.ServerBatchMetadataOptions{}, "server-batch-set-tag", "Set tags for some server", func(s *mcclient.ClientSession, opts *options.ServerBatchMetadataOptions) error {
		params, err := opts.Params()
		if err != nil {
			return err
		}
		result, err := modules.Servers.PerformClassAction(s, "batch-set-user-metadata", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&options.ResourceMetadataOptions{}, "server-add-tag", "Set tag of a server", func(s *mcclient.ClientSession, opts *options.ResourceMetadataOptions) error {
		params, err := opts.Params()
		if err != nil {
			return err
		}
		result, err := modules.Servers.PerformAction(s, opts.ID, "user-metadata", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&options.ResourceMetadataOptions{}, "server-set-tag", "Set tag of a server", func(s *mcclient.ClientSession, opts *options.ResourceMetadataOptions) error {
		params, err := opts.Params()
		if err != nil {
			return err
		}
		result, err := modules.Servers.PerformAction(s, opts.ID, "set-user-metadata", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&options.ServerCreateFromInstanceSnapshot{}, "server-create-from-instance-snapshot", "server create from instance snapshot",
		func(s *mcclient.ClientSession, opts *options.ServerCreateFromInstanceSnapshot) error {
			params := &compute.ServerCreateInput{}
			params.InstanceSnapshotId = opts.InstaceSnapshotId
			params.Name = opts.NAME
			params.AutoStart = opts.AutoStart
			params.Eip = opts.Eip
			params.EipChargeType = opts.EipChargeType
			params.EipBw = opts.EipBw

			server, err := modules.Servers.Create(s, params.JSON(params))
			if err != nil {
				return err
			}
			printObject(server)
			return nil
		},
	)

	R(&options.ServerCreateOptions{}, "server-check-create-data", "Check create server data", func(s *mcclient.ClientSession, opts *options.ServerCreateOptions) error {
		params, err := opts.Params()
		if err != nil {
			return err
		}
		server, err := modules.Servers.PerformClassAction(s, "check-create-data", params.JSON(params))
		if err != nil {
			return err
		}
		printObject(server)
		return nil
	})

	R(&options.ServerCreateOptions{}, "server-create", "Create a server", func(s *mcclient.ClientSession, opts *options.ServerCreateOptions) error {
		params, err := opts.Params()
		if err != nil {
			return err
		}
		count := params.Count
		if options.BoolV(opts.DryRun) {
			listFields := []string{"id", "name", "capacity", "count", "score", "capacity_details", "score_details"}
			input, err := opts.ToScheduleInput()
			if err != nil {
				return err
			}
			result, err := modules.SchedManager.Test(s, input)
			if err != nil {
				return err
			}
			if err != nil {
				return err
			}
			printList(modulebase.JSON2ListResult(result), listFields)
		} else {
			taskNotify := options.BoolV(opts.TaskNotify)
			if taskNotify {
				s.PrepareTask()
			}
			if count > 1 {
				results := modules.Servers.BatchCreate(s, params.JSON(params), count)
				printBatchResults(results, modules.Servers.GetColumns(s))
			} else {
				server, err := modules.Servers.Create(s, params.JSON(params))
				if err != nil {
					return err
				}
				printObject(server)
			}
			if taskNotify {
				s.WaitTaskNotify()
			}
		}
		return nil
	})

	R(&options.ServerCloneOptions{}, "server-clone", "Clone a server", func(s *mcclient.ClientSession, opts *options.ServerCloneOptions) error {
		params := jsonutils.Marshal(opts).(*jsonutils.JSONDict)
		res, err := modules.Servers.PerformAction(s, opts.SOURCE, "clone", params)
		if err != nil {
			return err
		}
		printObject(res)
		return nil
	})

	R(&options.ServerLoginInfoOptions{}, "server-logininfo", "Get login info of a server", func(s *mcclient.ClientSession, opts *options.ServerLoginInfoOptions) error {
		srvid, e := modules.Servers.GetId(s, opts.ID, nil)
		if e != nil {
			return e
		}

		params := jsonutils.NewDict()
		if len(opts.Key) > 0 {
			privateKey, e := ioutil.ReadFile(opts.Key)
			if e != nil {
				return e
			}
			params.Add(jsonutils.NewString(string(privateKey)), "private_key")
		}

		i, e := modules.Servers.GetLoginInfo(s, srvid, params)
		if e != nil {
			return e
		}
		printObject(i)
		return nil
	})

	R(&options.ServerIdsOptions{}, "server-start", "Start servers", func(s *mcclient.ClientSession, opts *options.ServerIdsOptions) error {
		ret := modules.Servers.BatchPerformAction(s, opts.ID, "start", nil)
		printBatchResults(ret, modules.Servers.GetColumns(s))
		return nil
	})

	R(&options.ServerIdsOptions{}, "server-syncstatus", "Sync servers status", func(s *mcclient.ClientSession, opts *options.ServerIdsOptions) error {
		ret := modules.Servers.BatchPerformAction(s, opts.ID, "syncstatus", nil)
		printBatchResults(ret, modules.Servers.GetColumns(s))
		return nil
	})

	R(&options.ServerIdsOptions{}, "server-sync", "Sync servers configurations", func(s *mcclient.ClientSession, opts *options.ServerIdsOptions) error {
		ret := modules.Servers.BatchPerformAction(s, opts.ID, "sync", nil)
		printBatchResults(ret, modules.Servers.GetColumns(s))
		return nil
	})

	R(&options.ServerSwitchToBackupOptions{}, "server-switch-to-backup", "Switch geust master to backup host", func(s *mcclient.ClientSession, opts *options.ServerSwitchToBackupOptions) error {
		params, err := options.StructToParams(opts)
		if err != nil {
			return err
		}
		ret, err := modules.Servers.PerformAction(s, opts.ID, "switch-to-backup", params)
		if err != nil {
			return err
		}
		printObject(ret)
		return nil
	})

	R(&options.ServerIdsOptions{}, "server-reconcile-backup", "Reconcile backup server", func(s *mcclient.ClientSession, opts *options.ServerIdsOptions) error {
		ret := modules.Servers.BatchPerformAction(s, opts.ID, "reconcile-backup", nil)
		printBatchResults(ret, modules.Servers.GetColumns(s))
		return nil
	})

	R(&options.ServerIdsOptions{}, "server-create-backup", "Create backup guest", func(s *mcclient.ClientSession, opts *options.ServerIdsOptions) error {
		ret := modules.Servers.BatchPerformAction(s, opts.ID, "create-backup", nil)
		printBatchResults(ret, modules.Servers.GetColumns(s))
		return nil
	})

	R(&options.ServerDeleteBackupOptions{}, "server-delete-backup", "Guest delete backup", func(s *mcclient.ClientSession, opts *options.ServerDeleteBackupOptions) error {
		params, err := options.StructToParams(opts)
		if err != nil {
			return err
		}
		ret, err := modules.Servers.PerformAction(s, opts.ID, "delete-backup", params)
		if err != nil {
			return err
		}
		printObject(ret)
		return nil
	})

	R(&options.ServerStopOptions{}, "server-stop", "Stop servers", func(s *mcclient.ClientSession, opts *options.ServerStopOptions) error {
		params, err := options.StructToParams(opts)
		if err != nil {
			return err
		}
		ret := modules.Servers.BatchPerformAction(s, opts.ID, "stop", params)
		printBatchResults(ret, modules.Servers.GetColumns(s))
		return nil
	})

	R(&options.ServerIdsOptions{}, "server-suspend", "Suspend servers", func(s *mcclient.ClientSession, opts *options.ServerIdsOptions) error {
		ret := modules.Servers.BatchPerformAction(s, opts.ID, "suspend", nil)
		printBatchResults(ret, modules.Servers.GetColumns(s))
		return nil
	})

	R(&options.ServerIdsOptions{}, "server-resume", "Resume servers", func(s *mcclient.ClientSession,
		opts *options.ServerIdsOptions) error {
		ret := modules.Servers.BatchPerformAction(s, opts.ID, "resume", nil)
		printBatchResults(ret, modules.Servers.GetColumns(s))
		return nil
	})

	R(&options.ServerMigrateOptions{}, "server-migrate", "Migrate server", func(s *mcclient.ClientSession, opts *options.ServerMigrateOptions) error {
		params, err := options.StructToParams(opts)
		if err != nil {
			return err
		}
		ret, err := modules.Servers.PerformAction(s, opts.ID, "migrate", params)
		if err != nil {
			return err
		}
		printObject(ret)
		return nil
	})

	R(&options.ServerLiveMigrateOptions{}, "server-live-migrate", "Migrate server", func(s *mcclient.ClientSession, opts *options.ServerLiveMigrateOptions) error {
		params, err := options.StructToParams(opts)
		if err != nil {
			return err
		}
		ret, err := modules.Servers.PerformAction(s, opts.ID, "live-migrate", params)
		if err != nil {
			return err
		}
		printObject(ret)
		return nil
	})

	R(&options.ServerResetOptions{}, "server-reset", "Reset servers", func(s *mcclient.ClientSession, opts *options.ServerResetOptions) error {
		params, err := options.StructToParams(opts)
		if err != nil {
			return err
		}
		ret := modules.Servers.BatchPerformAction(s, opts.ID, "reset", params)
		printBatchResults(ret, modules.Servers.GetColumns(s))
		return nil
	})

	R(&options.ServerRestartOptions{}, "server-restart", "Restart servers", func(s *mcclient.ClientSession, opts *options.ServerRestartOptions) error {
		params, err := options.StructToParams(opts)
		if err != nil {
			return err
		}
		ret := modules.Servers.BatchPerformAction(s, opts.ID, "restart", params)
		printBatchResults(ret, modules.Servers.GetColumns(s))
		return nil
	})

	R(&options.ServerIdsOptions{}, "server-purge", "Purge obsolete servers", func(s *mcclient.ClientSession, opts *options.ServerIdsOptions) error {
		ret := modules.Servers.BatchPerformAction(s, opts.ID, "purge", nil)
		printBatchResults(ret, modules.Servers.GetColumns(s))
		return nil
	})

	R(&options.ServerDeleteOptions{}, "server-delete", "Delete servers", func(s *mcclient.ClientSession, opts *options.ServerDeleteOptions) error {
		params, err := options.StructToParams(opts)
		if err != nil {
			return err
		}
		ret := modules.Servers.BatchDeleteWithParam(s, opts.ID, params, nil)
		printBatchResults(ret, modules.Servers.GetColumns(s))
		return nil
	})

	R(&options.ServerIdsOptions{}, "server-cancel-delete", "Cancel pending delete servers", func(s *mcclient.ClientSession, opts *options.ServerIdsOptions) error {
		ret := modules.Servers.BatchPerformAction(s, opts.ID, "cancel-delete", nil)
		printBatchResults(ret, modules.Servers.GetColumns(s))
		return nil
	})

	R(&options.ServerIdOptions{}, "server-vnc", "Show vnc info of server", func(s *mcclient.ClientSession, opts *options.ServerIdOptions) error {
		ret, e := modules.Servers.GetSpecific(s, opts.ID, "vnc", nil)
		if e != nil {
			return e
		}
		printObject(ret)
		return nil
	})

	R(&options.ServerIdOptions{}, "server-desc", "Show desc info of server", func(s *mcclient.ClientSession, opts *options.ServerIdOptions) error {
		ret, e := modules.Servers.GetSpecific(s, opts.ID, "desc", nil)
		if e != nil {
			return e
		}
		printObject(ret)
		return nil
	})

	R(&options.ServerIdOptions{}, "server-status", "Show status of server", func(s *mcclient.ClientSession, opts *options.ServerIdOptions) error {
		ret, e := modules.Servers.GetSpecific(s, opts.ID, "status", nil)
		if e != nil {
			return e
		}
		printObject(ret)
		return nil
	})

	R(&options.ServerUpdateOptions{}, "server-update", "Update servers", func(s *mcclient.ClientSession, opts *options.ServerUpdateOptions) error {
		params, err := opts.Params()
		if err != nil {
			return err
		}
		if params.Size() == 0 {
			return InvalidUpdateError()
		}
		result := modules.Servers.BatchPut(s, opts.ID, params)
		printBatchResults(result, modules.Servers.GetColumns(s))
		return nil
	})

	R(&options.ServerSendKeyOptions{}, "server-send-keys", "Send keys to server", func(s *mcclient.ClientSession, opts *options.ServerSendKeyOptions) error {
		params, err := options.StructToParams(opts)
		if err != nil {
			return err
		}
		srv, err := modules.Servers.PerformAction(s, opts.ID, "sendkeys", params)
		if err != nil {
			return err
		}
		printObject(srv)
		return nil
	})

	R(&options.ServerDeployOptions{}, "server-deploy", "Deploy hostname and keypair to a stopped virtual server", func(s *mcclient.ClientSession, opts *options.ServerDeployOptions) error {
		params, err := opts.Params()
		if err != nil {
			return err
		}
		srv, e := modules.Servers.PerformAction(s, opts.ID, "deploy", params.JSON(params))
		if e != nil {
			return e
		}
		printObject(srv)
		return nil
	})

	R(&options.ServerModifySrcCheckOptions{}, "server-modify-src-check", "Modify src ip, mac check settings", func(s *mcclient.ClientSession, opts *options.ServerModifySrcCheckOptions) error {
		params, err := options.StructToParams(opts)
		if err != nil {
			return err
		}
		srv, err := modules.Servers.PerformAction(s, opts.ID, "modify-src-check", params)
		if err != nil {
			return err
		}
		printObject(srv)
		return nil
	})

	R(&options.ServerSecGroupsOptions{}, "server-set-secgroup", "Set security groups to a VM", func(s *mcclient.ClientSession, opts *options.ServerSecGroupsOptions) error {
		srv, err := modules.Servers.PerformAction(s, opts.ID, "set-secgroup", opts.Parmas())
		if err != nil {
			return err
		}
		printObject(srv)
		return nil
	})

	R(&options.ServerSecGroupsOptions{}, "server-add-secgroup", "Add security group to a VM", func(s *mcclient.ClientSession, opts *options.ServerSecGroupsOptions) error {
		srv, err := modules.Servers.PerformAction(s, opts.ID, "add-secgroup", opts.Parmas())
		if err != nil {
			return err
		}
		printObject(srv)
		return nil
	})

	R(&options.ServerSecGroupOptions{}, "server-assign-secgroup", "Assign security group to a VM", func(s *mcclient.ClientSession, opts *options.ServerSecGroupOptions) error {
		params, err := options.StructToParams(opts)
		if err != nil {
			return err
		}
		srv, e := modules.Servers.PerformAction(s, opts.ID, "assign-secgroup", params)
		if e != nil {
			return e
		}
		printObject(srv)
		return nil
	})

	R(&options.ServerSecGroupOptions{}, "server-assign-admin-secgroup", "Assign administrative security group to a VM", func(s *mcclient.ClientSession, opts *options.ServerSecGroupOptions) error {
		params, err := options.StructToParams(opts)
		if err != nil {
			return err
		}
		srv, e := modules.Servers.PerformAction(s, opts.ID, "assign-admin-secgroup", params)
		if e != nil {
			return e
		}
		printObject(srv)
		return nil
	})

	R(&options.ServerSecGroupsOptions{}, "server-revoke-secgroup", "Revoke security group from VM", func(s *mcclient.ClientSession, opts *options.ServerSecGroupsOptions) error {
		srv, err := modules.Servers.PerformAction(s, opts.ID, "revoke-secgroup", opts.Parmas())
		if err != nil {
			return err
		}
		printObject(srv)
		return nil
	})

	R(&options.ServerIdOptions{}, "server-revoke-admin-secgroup", "Assign administrative security group to a VM", func(s *mcclient.ClientSession, opts *options.ServerIdOptions) error {
		srv, e := modules.Servers.PerformAction(s, opts.ID, "revoke-admin-secgroup", nil)
		if e != nil {
			return e
		}
		printObject(srv)
		return nil
	})

	R(&options.ServerMonitorOptions{}, "server-monitor", "Send commands to qemu monitor", func(s *mcclient.ClientSession, opts *options.ServerMonitorOptions) error {
		params, err := options.StructToParams(opts)
		if err != nil {
			return err
		}
		ret, err := modules.Servers.PerformAction(s, opts.ID, "monitor", params)
		if err != nil {
			return err
		}
		result, err := ret.GetString("results")
		if err != nil {
			return err
		}
		fmt.Println(result)
		return nil
	})

	R(&options.ServerSaveImageOptions{}, "server-save-image", "Save root disk to new image and upload to glance.", func(s *mcclient.ClientSession, opts *options.ServerSaveImageOptions) error {
		params, err := options.StructToParams(opts)
		if err != nil {
			return err
		}
		srv, err := modules.Servers.PerformAction(s, opts.ID, "save-image", params)
		if err != nil {
			return err
		}
		printObject(srv)
		return nil
	})

	R(&options.ServerSaveGuestImageOptions{}, "server-save-guest-image",
		"save root disk and data disks to new images and upload to glance.", func(s *mcclient.ClientSession,
			opts *options.ServerSaveGuestImageOptions) error {

			params, err := options.StructToParams(opts)
			if err != nil {
				return err
			}
			srv, err := modules.Servers.PerformAction(s, opts.ID, "save-guest-image", params)
			if err != nil {
				return err
			}
			printObject(srv)
			return nil
		},
	)

	type ServerChangeOwnerOptions struct {
		ID      string `help:"Server to change owner" json:"-"`
		PROJECT string `help:"Project ID or change" json:"tenant"`
	}
	R(&ServerChangeOwnerOptions{}, "server-change-owner", "Change owner porject of a server", func(s *mcclient.ClientSession, opts *ServerChangeOwnerOptions) error {
		params, err := options.StructToParams(opts)
		if err != nil {
			return err
		}
		srv, err := modules.Servers.PerformAction(s, opts.ID, "change-owner", params)
		if err != nil {
			return err
		}
		printObject(srv)
		return nil
	})

	R(&options.ServerRebuildRootOptions{}, "server-rebuild-root", "Rebuild VM root image with new template", func(s *mcclient.ClientSession, opts *options.ServerRebuildRootOptions) error {
		params, err := options.StructToParams(opts)
		if err != nil {
			return err
		}

		if opts.NoAccountInit != nil && *opts.NoAccountInit {
			params.Add(jsonutils.JSONFalse, "reset_password")
		}

		srv, err := modules.Servers.PerformAction(s, opts.ID, "rebuild-root", params)
		if err != nil {
			return err
		}
		printObject(srv)
		return nil
	})

	R(&options.ServerChangeConfigOptions{}, "server-change-config", "Change configuration of VM", func(s *mcclient.ClientSession, opts *options.ServerChangeConfigOptions) error {
		params, err := options.StructToParams(opts)
		if len(opts.Disk) > 0 {
			params.Remove("disk.0")
			disksConf := make([]*compute.DiskConfig, 0)
			for i, d := range opts.Disk {
				// params.Set(key, value)
				diskConfig, err := cmdline.ParseDiskConfig(d, i+1)
				if err != nil {
					return err
				}
				disksConf = append(disksConf, diskConfig)
			}
			params.Set("disks", jsonutils.Marshal(disksConf))
		}

		if err != nil {
			return err
		}
		if params.Size() == 0 {
			return InvalidUpdateError()
		}
		srv, err := modules.Servers.PerformAction(s, opts.ID, "change-config", params)
		if err != nil {
			return err
		}
		printObject(srv)
		return nil
	})

	type ServerDiskSnapshotOptions struct {
		SERVER       string `help:"server ID or Name"`
		DISK         string `help:"create snapshot disk id"`
		SNAPSHOTNAME string `help:"Snapshot name"`
	}
	R(&ServerDiskSnapshotOptions{}, "server-disk-create-snapshot", "Task server disk snapshot", func(s *mcclient.ClientSession, args *ServerDiskSnapshotOptions) error {
		params := jsonutils.NewDict()
		params.Set("disk_id", jsonutils.NewString(args.DISK))
		params.Set("name", jsonutils.NewString(args.SNAPSHOTNAME))
		srv, err := modules.Servers.PerformAction(s, args.SERVER, "disk-snapshot", params)
		if err != nil {
			return err
		}
		printObject(srv)
		return nil
	})

	type ServerInsertISOOptions struct {
		ID  string `help:"server ID or Name"`
		ISO string `help:"Glance image ID of the ISO"`
	}
	R(&ServerInsertISOOptions{}, "server-insert-iso", "Insert an ISO image into server's cdrom", func(s *mcclient.ClientSession, opts *ServerInsertISOOptions) error {
		img, err := modules.Images.Get(s, opts.ISO, nil)
		if err != nil {
			return err
		}
		imgId, err := img.GetString("id")
		if err != nil {
			return err
		}
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(imgId), "image_id")
		result, err := modules.Servers.PerformAction(s, opts.ID, "insertiso", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&options.ServerIdOptions{}, "server-eject-iso", "Eject iso from servers' cdrom", func(s *mcclient.ClientSession, opts *options.ServerIdOptions) error {
		result, err := modules.Servers.PerformAction(s, opts.ID, "ejectiso", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&options.ServerIdOptions{}, "server-iso", "Show server's mounting ISO information", func(s *mcclient.ClientSession, opts *options.ServerIdOptions) error {
		results, err := modules.Servers.GetSpecific(s, opts.ID, "iso", nil)
		if err != nil {
			return err
		}
		printObject(results)
		return nil
	})

	type ServerAssociateEipOptions struct {
		ID  string `help:"ID or name of server"`
		EIP string `help:"ID or name of EIP to associate"`
	}
	R(&ServerAssociateEipOptions{}, "server-associate-eip", "Associate a server and an eip", func(s *mcclient.ClientSession, args *ServerAssociateEipOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.EIP), "eip")
		results, err := modules.Servers.PerformAction(s, args.ID, "associate-eip", params)
		if err != nil {
			return err
		}
		printObject(results)
		return nil
	})

	type ServerDissociateEipOptions struct {
		ID         string `help:"ID or name of server" json:"-"`
		AutoDelete bool   `help:"automatically delete the dissociate EIP" json:"auto_delete,omitfalse"`
	}
	R(&ServerDissociateEipOptions{}, "server-dissociate-eip", "Dissociate an eip from a server", func(s *mcclient.ClientSession, args *ServerDissociateEipOptions) error {
		params, err := options.StructToParams(args)
		if err != nil {
			return err
		}
		result, err := modules.Servers.PerformAction(s, args.ID, "dissociate-eip", params)
		if err != nil {
			return nil
		}
		printObject(result)
		return nil
	})

	type ServerUserDataOptions struct {
		ID   string `help:"ID or name of server"`
		FILE string `help:"Path to user data file"`
	}
	R(&ServerUserDataOptions{}, "server-set-user-data", "Update server user_data", func(s *mcclient.ClientSession, args *ServerUserDataOptions) error {
		params := jsonutils.NewDict()
		content, err := ioutil.ReadFile(args.FILE)
		if err != nil {
			return err
		}
		params.Add(jsonutils.NewString(string(content)), "user_data")
		result, err := modules.Servers.PerformAction(s, args.ID, "user-data", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type ServerAddExtraOption struct {
		ID    string `help:"ID or name of server"`
		KEY   string `help:"Option key"`
		VALUE string `help:"Option value"`
	}
	R(&ServerAddExtraOption{}, "server-add-extra-options", "Add server extra options", func(s *mcclient.ClientSession, args *ServerAddExtraOption) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.KEY), "key")
		params.Add(jsonutils.NewString(args.VALUE), "value")
		result, err := modules.Servers.PerformAction(s, args.ID, "set-extra-option", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
	type ServerRemoveExtraOption struct {
		ID  string `help:"ID or name of server"`
		KEY string `help:"Option key"`
	}
	R(&ServerRemoveExtraOption{}, "server-remove-extra-options", "Remove server extra options", func(s *mcclient.ClientSession, args *ServerRemoveExtraOption) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.KEY), "key")
		result, err := modules.Servers.PerformAction(s, args.ID, "del-extra-option", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type ServerRenewOptions struct {
		ID       string `help:"ID or name of server to renew"`
		DURATION string `help:"Duration of renew, ADMIN only command"`
	}
	R(&ServerRenewOptions{}, "server-renew", "Renew a server", func(s *mcclient.ClientSession, args *ServerRenewOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.DURATION), "duration")
		result, err := modules.Servers.PerformAction(s, args.ID, "renew", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type ServerPrepaidRecycleOptions struct {
		ID         string `help:"ID or name of server to recycle"`
		AutoDelete bool   `help:"after joining the pool, remove the server automatically"`
	}
	R(&ServerPrepaidRecycleOptions{}, "server-enable-recycle", "Put a prepaid server into recycle pool, so that it can be shared", func(s *mcclient.ClientSession, args *ServerPrepaidRecycleOptions) error {
		params := jsonutils.NewDict()
		if args.AutoDelete {
			params.Add(jsonutils.JSONTrue, "auto_delete")
		}
		result, err := modules.Servers.PerformAction(s, args.ID, "prepaid-recycle", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&ServerPrepaidRecycleOptions{}, "server-disable-recycle", "Pull a prepaid server from recycle pool, so that it will not be shared anymore", func(s *mcclient.ClientSession, args *ServerPrepaidRecycleOptions) error {
		params := jsonutils.NewDict()
		result, err := modules.Servers.PerformAction(s, args.ID, "undo-prepaid-recycle", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type ServerImportOptions struct {
		LOCATION string `help:"Server desc file location, should be desc file or workspace directory"`
		HOST     string `help:"Host id or name for this server"`
	}
	R(&ServerImportOptions{}, "server-import", "Import a server by desc file", func(s *mcclient.ClientSession, args *ServerImportOptions) error {
		var descFiles []string
		err := filepath.Walk(args.LOCATION, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			if info.Name() == "desc" {
				descFiles = append(descFiles, path)
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("Find desc files: %v", err)
		}

		importF := func(desc string) error {
			ret, err := ioutil.ReadFile(desc)
			if err != nil {
				return fmt.Errorf("Read file %s: %v", desc, err)
			}
			jsonObj, err := jsonutils.Parse(ret)
			if err != nil {
				return fmt.Errorf("Parse %s to json: %v", string(ret), err)
			}
			params := jsonObj.(*jsonutils.JSONDict)
			disks, err := params.GetArray("disks")
			if err != nil || len(disks) == 0 {
				return fmt.Errorf("Desc %s not have disks, skip it", desc)
			}
			params.Add(jsonutils.NewString(args.HOST), "host_id")
			// project may not exists
			params.Remove("tenant")
			params.Remove("tenant_id")
			_, err = modules.Servers.PerformClassAction(s, "import", params)
			if err != nil {
				return err
			}
			//printObject(result)
			return nil
		}

		for _, descFile := range descFiles {
			if err := importF(descFile); err != nil {
				log.Errorf("Import %s error: %v", descFile, err)
			}
		}
		return nil
	})

	type ServersImportFromLibvirtOptions struct {
		CONFIG_FILE string `help:"File Path describing servers from libvirt"`
	}

	type Servers struct {
		Mac string `yaml:"mac"`
		Ip  string `yaml:"ip"`
	}

	type Hosts struct {
		HostIp      string    `yaml:"host_ip"`
		XmlFilePath string    `yaml:"xml_file_path"`
		MonitorPath string    `yaml:"monitor_path"`
		Servers     []Servers `yaml:"servers"`
	}

	type LibvirtImportOptions struct {
		Hosts []Hosts `yaml:"hosts"`
	}

	R(&ServersImportFromLibvirtOptions{}, "servers-import-from-libvirt", "Import servers from libvrt", func(s *mcclient.ClientSession, args *ServersImportFromLibvirtOptions) error {
		var (
			rawConfig []byte
			err       error
		)

		rawConfig, err = ioutil.ReadFile(args.CONFIG_FILE)
		if err != nil {
			return fmt.Errorf("Read config file %s error: %s", args.CONFIG_FILE, err)
		}

		var (
			params []jsonutils.JSONObject
			config = &compute.SLibvirtImportConfig{}
		)

		// Try parse as json first
		{
			err = json.Unmarshal(rawConfig, config)
			if err != nil {
				goto YAML
			}
			for i := 0; i < len(config.Hosts); i++ {
				if nIp := net.ParseIP(config.Hosts[i].HostIp); nIp == nil {
					return fmt.Errorf("Parse host ip %s failed", config.Hosts[i].HostIp)
				}
				for _, server := range config.Hosts[i].Servers {
					for mac, ip := range server.MacIp {
						if _, err := net.ParseMAC(mac); err != nil {
							return fmt.Errorf("Parse mac %s error %s", mac, err)
						}
						if nIp := net.ParseIP(ip); nIp == nil {
							return fmt.Errorf("Parse ip %s failed", ip)
						}
					}
				}
			}

			goto REQUEST
		}

	YAML: // Try Parse as yaml
		{
			yamlConfig := &LibvirtImportOptions{}
			err = yaml.Unmarshal(rawConfig, yamlConfig)
			if err != nil {
				return err
			}

			config.Hosts = make([]compute.SLibvirtHostConfig, len(yamlConfig.Hosts))
			for i := 0; i < len(yamlConfig.Hosts); i++ {
				if nIp := net.ParseIP(yamlConfig.Hosts[i].HostIp); nIp == nil {
					return fmt.Errorf("Parse host ip %s failed", yamlConfig.Hosts[i].HostIp)
				}
				config.Hosts[i].HostIp = yamlConfig.Hosts[i].HostIp
				config.Hosts[i].XmlFilePath = yamlConfig.Hosts[i].XmlFilePath
				config.Hosts[i].Servers = make([]compute.SLibvirtServerConfig, len(yamlConfig.Hosts[i].Servers))
				config.Hosts[i].MonitorPath = yamlConfig.Hosts[i].MonitorPath
				for j := 0; j < len(yamlConfig.Hosts[i].Servers); j++ {
					config.Hosts[i].Servers[j].MacIp = make(map[string]string)
					mac := yamlConfig.Hosts[i].Servers[j].Mac
					_, err := net.ParseMAC(mac)
					if err != nil {
						return fmt.Errorf("Parse mac address %s error %s", mac, err)
					}
					ip := yamlConfig.Hosts[i].Servers[j].Ip
					nIp := net.ParseIP(ip)
					if len(nIp) == 0 {
						return fmt.Errorf("Parse ip address %s failed", ip)
					}
					config.Hosts[i].Servers[j].MacIp[mac] = ip
				}
			}
		}

	REQUEST:
		params, err = jsonutils.Marshal(config.Hosts).GetArray()
		if err != nil {
			return err
		}
		//for i := 0; i < len(params); i++ {
		//	val := jsonutils.NewDict()
		//	val.Set(modules.Servers.KeywordPlural, params[i])
		//	params[i] = val
		//}

		results := modules.Servers.BatchPerformClassAction(s, "import-from-libvirt", params)
		printBatchResults(results, modules.Servers.GetColumns(s))
		return nil
	})

	R(&options.ServerIdOptions{}, "server-create-params", "Show server create params", func(s *mcclient.ClientSession, opts *options.ServerIdOptions) error {
		ret, e := modules.Servers.GetSpecific(s, opts.ID, "create-params", nil)
		if e != nil {
			return e
		}
		printObject(ret)
		return nil
	})

	type ServerExportVirtInstallCommand struct {
		ID            string   `help:"Server Id" json:"-"`
		LibvirtBridge string   `help:"Libvirt default bridge" json:"libvirt_bridge"`
		ExtraCmdline  []string `help:"Extra virt-install arguments add to script, eg:'--extra-args ...', '--console ...'" json:"extra_cmdline"`
	}
	R(&ServerExportVirtInstallCommand{}, "server-export-virt-install-command", "Export virt-install command line from existing guest", func(s *mcclient.ClientSession, args *ServerExportVirtInstallCommand) error {
		params := jsonutils.NewDict()
		if len(args.LibvirtBridge) > 0 {
			params.Set("libvirt_bridge", jsonutils.NewString(args.LibvirtBridge))
		}
		if len(args.ExtraCmdline) > 0 {
			params.Set("extra_cmdline", jsonutils.NewStringArray(args.ExtraCmdline))
		}
		result, err := modules.Servers.GetSpecific(s, args.ID, "virt-install", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&options.ServerIdOptions{}, "server-remote-nics", "Show remote nics of a server", func(s *mcclient.ClientSession, opts *options.ServerIdOptions) error {
		result, err := modules.Servers.GetSpecific(s, opts.ID, "remote-nics", nil)
		if err != nil {
			return err
		}
		listResult := modulebase.ListResult{}
		listResult.Data, _ = result.GetArray()
		printList(&listResult, nil)
		return nil
	})

	type ServerSyncFixNicsOptions struct {
		ID string   `help:"ID or name of VM" json:"-"`
		IP []string `help:"IP address of each NIC" json:"ip"`
	}
	R(&ServerSyncFixNicsOptions{}, "server-sync-fix-nics", "Fix missing IP for each nics after syncing VNICS", func(s *mcclient.ClientSession, opts *ServerSyncFixNicsOptions) error {
		params := jsonutils.Marshal(opts)
		result, err := modules.Servers.PerformAction(s, opts.ID, "sync-fix-nics", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type ServerResizeDiskOptions struct {
		Server string `help:"ID or name of VM" json:"-" optional:"false" positional:"true"`
		Disk   string `help:"ID or name of disk to resize" json:"disk" optional:"false" positional:"true"`
		Size   string `help:"new size of disk in MB" json:"size" optional:"false" positional:"true"`
	}
	R(&ServerResizeDiskOptions{}, "server-resize-disk", "Resize attached disk of a server", func(s *mcclient.ClientSession, args *ServerResizeDiskOptions) error {
		params := jsonutils.Marshal(args)
		result, err := modules.Servers.PerformAction(s, args.Server, "resize-disk", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type ServerIoThrottle struct {
		ID   string `help:"ID or name of VM" json:"-"`
		BPS  int    `help:"bps(MB) of throttle" json:"bps"`
		IOPS int    `help:"iops of throttle" json:"iops"`
	}
	R(&ServerIoThrottle{}, "server-io-throttle", "Guest io set throttle", func(s *mcclient.ClientSession, opts *ServerIoThrottle) error {
		params := jsonutils.Marshal(opts)
		result, err := modules.Servers.PerformAction(s, opts.ID, "io-throttle", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type ServerGroupsOptions struct {
		ID    string   `help:"ID or name of VM"`
		Group []string `help:"ids or names of group"`
	}

	R(&ServerGroupsOptions{}, "server-join-groups", "Join multiple groups", func(s *mcclient.ClientSession,
		opts *ServerGroupsOptions) error {

		params, err := options.StructToParams(opts)
		if err != nil {
			return err
		}
		server, err := modules.Servers.PerformAction(s, opts.ID, "bind-groups", params)
		if err != nil {
			return err
		}
		printObject(server)
		return nil
	})

	R(&ServerGroupsOptions{}, "server-leave-groups", "Leave multiple groups", func(s *mcclient.ClientSession,
		opts *ServerGroupsOptions) error {

		params, err := options.StructToParams(opts)
		if err != nil {
			return err
		}
		server, err := modules.Servers.PerformAction(s, opts.ID, "unbind-groups", params)
		if err != nil {
			return err
		}
		printObject(server)
		return nil
	})

	type ServerQemuParams struct {
		ID               string `help:"ID or name of VM"`
		DisableIsaSerial string `help:"disable isa serial device" choices:"true|false"`
		DisablePvpanic   string `help:"disable pvpanic device" choices:"true|false"`
	}

	R(&ServerQemuParams{}, "server-set-qemu-params", "config qemu params", func(s *mcclient.ClientSession,
		opts *ServerQemuParams) error {
		params := jsonutils.NewDict()
		if len(opts.DisableIsaSerial) > 0 {
			params.Set("disable_isa_serial", jsonutils.NewString(opts.DisableIsaSerial))
		}
		if len(opts.DisablePvpanic) > 0 {
			params.Set("disable_pvpanic", jsonutils.NewString(opts.DisablePvpanic))
		}
		result, err := modules.Servers.PerformAction(s, opts.ID, "set-qemu-params", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
	type ServerCreateSnapshot struct {
		ID       string `help:"ID or name of VM" json:"-"`
		SNAPSHOT string `help:"Instance snapshot name" json:"name"`
	}
	R(&ServerCreateSnapshot{}, "instance-snapshot-create", "create instance snapshot", func(s *mcclient.ClientSession, opts *ServerCreateSnapshot) error {
		params := jsonutils.Marshal(opts)
		result, err := modules.Servers.PerformAction(s, opts.ID, "instance-snapshot", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type ServerSnapshotAndClone struct {
		ID          string `help:"ID or name of VM" json:"-"`
		NAME        string `help:"Newly instance name" json:"name"`
		AutoStart   bool   `help:"Auto start new guest"`
		AllowDelete bool   `help:"Allow new guest delete" json:"-"`
		Count       int    `help:"Guest count"`
	}
	R(&ServerSnapshotAndClone{}, "instance-snapshot-and-clone", "create instance snapshot and clone new instance", func(s *mcclient.ClientSession, opts *ServerSnapshotAndClone) error {
		params := jsonutils.Marshal(opts)
		dictParams := params.(*jsonutils.JSONDict)
		if opts.AllowDelete {
			dictParams.Set("disable_delete", jsonutils.JSONFalse)
		}
		result, err := modules.Servers.PerformAction(s, opts.ID, "snapshot-and-clone", dictParams)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type ServerRollBackSnapshot struct {
		ID               string `help:"ID or name of VM" json:"-"`
		InstanceSnapshot string `help:"Instance snapshot id or name" json:"instance_snapshot"`
	}
	R(&ServerRollBackSnapshot{}, "instance-snapshot-reset", "reset instance snapshot", func(s *mcclient.ClientSession, opts *ServerRollBackSnapshot) error {
		params := jsonutils.Marshal(opts)
		result, err := modules.Servers.PerformAction(s, opts.ID, "instance-snapshot-reset", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type ServerJnlpOptions struct {
		ID   string `help:"ID or name of server"`
		Save string `help:"save xml into this file"`
	}
	R(&ServerJnlpOptions{}, "server-jnlp", "Get baremetal server jnlp file contentn", func(s *mcclient.ClientSession, args *ServerJnlpOptions) error {
		spec, err := modules.Servers.GetSpecific(s, args.ID, "jnlp", nil)
		if err != nil {
			return err
		}
		jnlp, err := spec.GetString("jnlp")
		if err != nil {
			return err
		}
		if len(args.Save) > 0 {
			return fileutils2.FilePutContents(args.Save, jnlp, false)
		} else {
			fmt.Println(jnlp)
		}
		return nil
	})

	R(&options.ServerSSHLoginOptions{}, "server-ssh", "Use SSH login a server", func(s *mcclient.ClientSession, opts *options.ServerSSHLoginOptions) error {
		srv, err := modules.Servers.Get(s, opts.ID, nil)
		if err != nil {
			return err
		}

		srvid, err := srv.GetString("id")
		if err != nil {
			return err
		}

		address := make([]string, 0)
		nics, err := srv.GetArray("nics")
		if err != nil {
			return err
		}
		for _, nic := range nics {
			if addr, err := nic.GetString("ip_addr"); err == nil {
				address = append(address, addr)
			}
		}
		if len(address) == 0 {
			return fmt.Errorf("Not found ip address from server %s", opts.ID)
		}

		params := jsonutils.NewDict()
		if len(opts.Key) > 0 {
			privateKey, e := ioutil.ReadFile(opts.Key)
			if e != nil {
				return e
			}
			params.Add(jsonutils.NewString(string(privateKey)), "private_key")
		}

		i, e := modules.Servers.GetLoginInfo(s, srvid, params)
		if e != nil {
			return e
		}
		passwd, err := i.GetString("password")
		if err != nil {
			return err
		}
		if opts.Password != "" {
			passwd = opts.Password
		}
		user, err := i.GetString("username")
		if err != nil {
			return err
		}
		if opts.User != "" {
			user = opts.User
		}

		host := address[0]
		if opts.Host != "" {
			host = opts.Host
		}
		port := 22
		if opts.Port != 22 {
			port = opts.Port
		}

		sshCli, err := ssh.NewClient(host, port, user, passwd, "")
		if err != nil {
			return err
		}
		log.Infof("ssh %s:%d", host, port)
		if err := sshCli.RunTerminal(); err != nil {
			return err
		}
		return nil
	})

	type ServerPublicipToEip struct {
		ID        string `help:"ID or name of VM" json:"-"`
		AutoStart bool   `help:"Auto start new guest"`
	}

	R(&ServerPublicipToEip{}, "server-publicip-to-eip", "Convert PublicIp to Eip for server", func(s *mcclient.ClientSession, opts *ServerPublicipToEip) error {
		params := jsonutils.NewDict()
		params.Set("auto_start", jsonutils.NewBool(opts.AutoStart))
		result, err := modules.Servers.PerformAction(s, opts.ID, "publicip-to-eip", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type ServerSetAutoRenew struct {
		ID        string `help:"ID or name of VM" json:"-"`
		AutoRenew bool   `help:"Set server auto renew or manual renew"`
	}

	R(&ServerSetAutoRenew{}, "server-set-auto-renew", "Set autorenew for server", func(s *mcclient.ClientSession, opts *ServerSetAutoRenew) error {
		params := jsonutils.NewDict()
		params.Set("auto_renew", jsonutils.NewBool(opts.AutoRenew))
		result, err := modules.Servers.PerformAction(s, opts.ID, "set-auto-renew", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&options.ServerConvertToKvmOptions{}, "server-convert-to-kvm", "Convert esxi server to kvm", func(s *mcclient.ClientSession, opts *options.ServerConvertToKvmOptions) error {
		params := jsonutils.Marshal(opts)
		dict := params.(*jsonutils.JSONDict)
		dict.Set("target_hypervisor", jsonutils.NewString("kvm"))
		result, err := modules.Servers.PerformAction(s, opts.ID, "convert", dict)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&options.ServerShowOptions{}, "server-change-owner-candidate-domains", "Get change owner candidate domain list", func(s *mcclient.ClientSession, args *options.ServerShowOptions) error {
		result, err := modules.Servers.GetSpecific(s, args.ID, "change-owner-candidate-domains", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type ServerSaveTemplateOptions struct {
		ID           string `help:"The ID or Name of server"`
		TemplateName string `help:"The name of guest template"`
	}

	R(&ServerSaveTemplateOptions{}, "server-save-template", "Save Guest Template of this Server",
		func(s *mcclient.ClientSession, args *ServerSaveTemplateOptions) error {
			dict := jsonutils.NewDict()
			dict.Set("name", jsonutils.NewString(args.TemplateName))
			result, err := modules.Servers.PerformAction(s, args.ID, "save-template", dict)
			if err != nil {
				return err
			}
			printObject(result)
			return nil
		},
	)
}
