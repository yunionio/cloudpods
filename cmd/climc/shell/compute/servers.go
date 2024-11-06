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
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/printutils"

	"yunion.io/x/onecloud/cmd/climc/shell"
	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/mcclient/modules/image"
	"yunion.io/x/onecloud/pkg/mcclient/modules/scheduler"
	baseoptions "yunion.io/x/onecloud/pkg/mcclient/options"
	options "yunion.io/x/onecloud/pkg/mcclient/options/compute"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/ssh"
)

func init() {
	cmd := shell.NewResourceCmd(&modules.Servers)
	cmd.List(new(options.ServerListOptions))
	cmd.Show(new(options.ServerShowOptions))
	cmd.BatchDeleteWithParam(new(options.ServerDeleteOptions))
	cmd.BatchPerform("cancel-delete", new(options.ServerCancelDeleteOptions))
	cmd.BatchPut(new(options.ServerUpdateOptions))
	cmd.GetMetadata(new(options.ServerIdOptions))
	cmd.Perform("clone", new(options.ServerCloneOptions))
	cmd.BatchPerform("start", new(options.ServerStartOptions))
	cmd.BatchPerform("syncstatus", new(options.ServerIdsOptions))
	cmd.BatchPerform("sync", new(options.ServerIdsOptions))
	cmd.Perform("switch-to-backup", new(options.ServerSwitchToBackupOptions))
	cmd.BatchPerform("reconcile-backup", new(options.ServerIdsOptions))
	cmd.Perform("create-backup", new(options.ServerCreateBackupOptions))
	cmd.BatchPerform("start-backup", new(options.ServerIdsOptions))
	cmd.Perform("delete-backup", new(options.ServerDeleteBackupOptions))
	cmd.BatchPerform("stop", new(options.ServerStopOptions))
	cmd.BatchPerform("suspend", new(options.ServerIdsOptions))
	cmd.BatchPerform("resume", new(options.ServerIdsOptions))
	cmd.BatchPerform("reset", new(options.ServerResetOptions))
	cmd.BatchPerform("restart", new(options.ServerRestartOptions))
	cmd.BatchPerform("purge", new(options.ServerIdsOptions))
	cmd.BatchPerform("convert-to-kvm", new(options.ServerConvertToKvmOptions))
	cmd.PrintObjectYAML().Perform("migrate-forecast", new(options.ServerMigrateForecastOptions))
	cmd.Perform("migrate", new(options.ServerMigrateOptions))
	cmd.Perform("live-migrate", new(options.ServerLiveMigrateOptions))
	cmd.BatchPerform("cancel-live-migrate", new(options.ServerIdsOptions))
	cmd.Perform("set-live-migrate-params", new(options.ServerSetLiveMigrateParamsOptions))
	cmd.Perform("modify-src-check", new(options.ServerModifySrcCheckOptions))
	cmd.Perform("set-secgroup", new(options.ServerSecGroupsOptions))
	cmd.Perform("add-secgroup", new(options.ServerSecGroupsOptions))
	cmd.Perform("assign-secgroup", new(options.ServerSecGroupOptions))
	cmd.Perform("assign-admin-secgroup", new(options.ServerSecGroupOptions))
	cmd.Perform("revoke-secgroup", new(options.ServerSecGroupOptions))
	cmd.Perform("revoke-admin-secgroup", new(options.ServerIdOptions))
	cmd.Perform("save-image", new(options.ServerSaveImageOptions))
	cmd.Perform("save-guest-image", new(options.ServerSaveGuestImageOptions))
	cmd.Perform("change-owner", new(options.ServerChangeOwnerOptions))
	cmd.Perform("rebuild-root", new(options.ServerRebuildRootOptions))
	cmd.Perform("change-config", new(options.ServerChangeConfigOptions))
	cmd.Perform("ejectiso", new(options.ServerIdOptions))
	cmd.Perform("sendkeys", new(options.ServerSendKeyOptions))
	cmd.Perform("deploy", new(options.ServerDeployOptions))
	cmd.Perform("associate-eip", new(options.ServerAssociateEipOptions))
	cmd.Perform("dissociate-eip", new(options.ServerDissociateEipOptions))
	cmd.Perform("renew", new(options.ServerRenewOptions))
	cmd.Perform("io-throttle", new(options.ServerIoThrottle))
	cmd.Perform("publicip-to-eip", new(options.ServerPublicipToEip))
	cmd.Perform("set-auto-renew", new(options.ServerSetAutoRenew))
	cmd.Perform("save-template", new(options.ServerSaveImageOptions))
	cmd.Perform("remote-update", new(options.ServerRemoteUpdateOptions))
	cmd.Perform("create-eip", &options.ServerCreateEipOptions{})
	cmd.Perform("make-sshable", &options.ServerMakeSshableOptions{})
	cmd.Perform("migrate-network", &options.ServerMigrateNetworkOptions{})
	cmd.Perform("set-sshport", &options.ServerSetSshportOptions{})
	cmd.Perform("have-agent", &options.ServerHaveAgentOptions{})
	cmd.Perform("change-disk-storage", &options.ServerChangeDiskStorageOptions{})
	cmd.Perform("change-storage", &options.ServerChangeStorageOptions{})
	cmd.PerformClass("batch-user-metadata", &options.ServerBatchMetadataOptions{})
	cmd.PerformClass("batch-set-user-metadata", &options.ServerBatchMetadataOptions{})
	cmd.Perform("user-metadata", &baseoptions.ResourceMetadataOptions{})
	cmd.Perform("set-user-metadata", &baseoptions.ResourceMetadataOptions{})
	cmd.Perform("probe-isolated-devices", &options.ServerIdOptions{})
	cmd.Perform("cpuset", &options.ServerCPUSetOptions{})
	cmd.Perform("cpuset-remove", &options.ServerIdOptions{})
	cmd.Perform("calculate-record-checksum", &options.ServerIdOptions{})
	cmd.Perform("set-class-metadata", &baseoptions.ResourceMetadataOptions{})
	cmd.Perform("monitor", &options.ServerMonitorOptions{})
	cmd.BatchPerform("enable-memclean", new(options.ServerIdsOptions))
	cmd.Perform("qga-set-password", &options.ServerQgaSetPassword{})
	cmd.Perform("qga-command", &options.ServerQgaCommand{})
	cmd.Perform("qga-ping", &options.ServerQgaPing{})
	cmd.Perform("qga-guest-info-task", &options.ServerQgaGuestInfoTask{})
	cmd.Perform("qga-get-network", &options.ServerQgaGetNetwork{})
	cmd.Perform("set-password", &options.ServerSetPasswordOptions{})
	cmd.Perform("set-boot-index", &options.ServerSetBootIndexOptions{})
	cmd.Perform("reset-nic-traffic-limit", &options.ServerNicTrafficLimitOptions{})
	cmd.Perform("set-nic-traffic-limit", &options.ServerNicTrafficLimitOptions{})
	cmd.Perform("add-sub-ips", &options.ServerAddSubIpsOptions{})
	cmd.Perform("update-sub-ips", &options.ServerUpdateSubIpsOptions{})
	cmd.BatchPerform("set-os-info", &options.ServerSetOSInfoOptions{})
	cmd.BatchPerform("start-rescue", &options.ServerStartOptions{})
	cmd.BatchPerform("stop-rescue", &options.ServerStartOptions{})
	cmd.BatchPerform("sync-os-info", &options.ServerIdsOptions{})
	cmd.BatchPerform("set-root-disk-matcher", &options.ServerSetRootDiskMatcher{})

	cmd.Get("vnc", new(options.ServerVncOptions))
	cmd.Get("desc", new(options.ServerIdOptions))
	cmd.Get("status", new(options.ServerIdOptions))
	cmd.Get("iso", new(options.ServerIdOptions))
	cmd.Get("create-params", new(options.ServerIdOptions))
	cmd.Get("sshable", new(options.ServerIdOptions))
	cmd.Get("make-sshable-cmd", new(options.ServerIdOptions))
	cmd.Get("change-owner-candidate-domains", new(options.ServerChangeOwnerCandidateDomainsOptions))
	cmd.Get("change-owner-candidate-domains", new(options.ServerChangeOwnerCandidateDomainsOptions))
	cmd.Get("cpuset-cores", new(options.ServerIdOptions))
	cmd.Get("sshport", new(options.ServerIdOptions))
	cmd.Get("qemu-info", new(options.ServerIdOptions))
	cmd.Get("hardware-info", new(options.ServerIdOptions))

	cmd.GetProperty(&options.ServerStatusStatisticsOptions{})
	cmd.GetProperty(&options.ServerProjectStatisticsOptions{})
	cmd.GetProperty(&options.ServerDomainStatisticsOptions{})
	cmd.GetProperty(&options.ServerGetPropertyTagValuePairOptions{})
	cmd.GetProperty(&options.ServerGetPropertyTagValueTreeOptions{})
	cmd.GetProperty(&options.ServerGetPropertyProjectTagValuePairOptions{})
	cmd.GetProperty(&options.ServerGetPropertyProjectTagValueTreeOptions{})
	cmd.GetProperty(&options.ServerGetPropertyDomainTagValuePairOptions{})
	cmd.GetProperty(&options.ServerGetPropertyDomainTagValueTreeOptions{})

	type ServerTaskShowOptions struct {
		ID       string `help:"ID or name of server" json:"-"`
		Since    string `help:"show tasks since this time point"`
		Open     bool   `help:"show tasks that are not completed" json:"-"`
		Complete bool   `help:"show tasks that has been completed" json:"-"`
	}
	R(&ServerTaskShowOptions{}, "server-tasks", "Show tasks of a server", func(s *mcclient.ClientSession, opts *ServerTaskShowOptions) error {
		params, err := baseoptions.StructToParams(opts)
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
		listResult := printutils.ListResult{}
		listResult.Data = tasks
		printList(&listResult, nil)
		return nil
	})

	/*R(&options.ServerBatchMetadataOptions{}, "server-batch-update-user-tag", "add tags for some server", func(s *mcclient.ClientSession, opts *options.ServerBatchMetadataOptions) error {
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

	R(&options.ServerBatchMetadataOptions{}, "server-batch-replace-user-tag", "Set tags for some server", func(s *mcclient.ClientSession, opts *options.ServerBatchMetadataOptions) error {
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

	R(&options.ResourceMetadataOptions{}, "server-update-user-tag", "Set tag of a server", func(s *mcclient.ClientSession, opts *options.ResourceMetadataOptions) error {
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

	R(&options.ResourceMetadataOptions{}, "server-update-user-tag", "Set tag of a server", func(s *mcclient.ClientSession, opts *options.ResourceMetadataOptions) error {
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
	*/

	R(&baseoptions.ResourceMetadataOptions{}, "server-set-metadata", "Set raw metadata of a server", func(s *mcclient.ClientSession, opts *baseoptions.ResourceMetadataOptions) error {
		params, err := opts.Params()
		if err != nil {
			return err
		}
		result, err := modules.Servers.PerformAction(s, opts.ID, "metadata", params)
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

	R(&options.ServerCreateOptions{}, "server-create", "Create a server", func(s *mcclient.ClientSession, opts *options.ServerCreateOptions) error {
		params, err := opts.Params()
		if err != nil {
			return err
		}
		count := params.Count
		if baseoptions.BoolV(opts.DryRun) {
			listFields := []string{"id", "name", "capacity", "count", "score", "capacity_details", "score_details"}
			input, err := opts.ToScheduleInput()
			if err != nil {
				return err
			}
			result, err := scheduler.SchedManager.Test(s, input)
			if err != nil {
				return err
			}
			printList(modulebase.JSON2ListResult(result), listFields)
			return nil
		}
		taskNotify := baseoptions.BoolV(opts.TaskNotify)
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
		return nil
	})

	R(&options.ServerLoginInfoOptions{}, "server-logininfo", "Get login info of a server", func(s *mcclient.ClientSession, opts *options.ServerLoginInfoOptions) error {
		params := jsonutils.NewDict()
		if len(opts.Key) > 0 {
			privateKey, e := ioutil.ReadFile(opts.Key)
			if e != nil {
				return e
			}
			params.Add(jsonutils.NewString(string(privateKey)), "private_key")
		}

		i, e := modules.Servers.PerformAction(s, opts.ID, "login-info", params)
		if e != nil {
			return e
		}
		printObject(i)
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
		img, err := image.Images.Get(s, opts.ISO, nil)
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

		Value string `help:"Option value"`
	}
	R(&ServerRemoveExtraOption{}, "server-remove-extra-options", "Remove server extra options", func(s *mcclient.ClientSession, args *ServerRemoveExtraOption) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.KEY), "key")
		if len(args.Value) > 0 {
			params.Add(jsonutils.NewString(args.Value), "value")
		}
		result, err := modules.Servers.PerformAction(s, args.ID, "del-extra-option", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&options.ServerPrepaidRecycleOptions{}, "server-enable-recycle", "Put a prepaid server into recycle pool, so that it can be shared", func(s *mcclient.ClientSession, args *options.ServerPrepaidRecycleOptions) error {
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

	R(&options.ServerPrepaidRecycleOptions{}, "server-disable-recycle", "Pull a prepaid server from recycle pool, so that it will not be shared anymore", func(s *mcclient.ClientSession, args *options.ServerPrepaidRecycleOptions) error {
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
		listResult := printutils.ListResult{}
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

	type ServerGroupsOptions struct {
		ID    string   `help:"ID or name of VM"`
		Group []string `help:"ids or names of group"`
	}

	R(&ServerGroupsOptions{}, "server-join-groups", "Join multiple groups", func(s *mcclient.ClientSession,
		opts *ServerGroupsOptions) error {

		params, err := baseoptions.StructToParams(opts)
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

		params, err := baseoptions.StructToParams(opts)
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
		ID                string `help:"ID or name of VM"`
		DisableIsaSerial  string `help:"disable isa serial device" choices:"true|false"`
		DisablePvpanic    string `help:"disable pvpanic device" choices:"true|false"`
		DisableUsbKbd     string `help:"disable usb kbd" choices:"true|false"`
		UsbControllerType string `help:"usb controller type" choices:"usb-ehci|qemu-xhci"`
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
		if len(opts.DisableUsbKbd) > 0 {
			params.Set("disable_usb_kbd", jsonutils.NewString(opts.DisableUsbKbd))
		}
		if len(opts.UsbControllerType) > 0 {
			params.Set("usb_controller_type", jsonutils.NewString(opts.UsbControllerType))
		}
		result, err := modules.Servers.PerformAction(s, opts.ID, "set-qemu-params", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
	type ServerCreateSnapshot struct {
		ID         string `help:"ID or name of VM" json:"-"`
		SNAPSHOT   string `help:"Instance snapshot name" json:"name"`
		WithMemory bool   `help:"Save memory state" json:"with_memory"`
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
	type ServerCreateBackup struct {
		ID              string `help:"ID or name of VM" json:"-"`
		BACKUP          string `help:"Instance backup name" json:"name"`
		BACKUPSTORAGEID string `help:"backup storage id" json:"backup_storage_id"`
	}
	R(&ServerCreateBackup{}, "server-create-instance-backup", "create instance backup", func(s *mcclient.ClientSession, opts *ServerCreateBackup) error {
		params := jsonutils.Marshal(opts)
		result, err := modules.Servers.PerformAction(s, opts.ID, "instance-backup", params)
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
		WithMemory       bool   `help:"Memory restore" json:"with_memory"`
		AutoStart        bool   `help:"Auto start VM"`
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

		eip, err := srv.GetString("eip")
		if err != nil && err.Error() != "Get: key not found" {
			return err
		}

		vpcid, err := srv.GetString("vpc_id")
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

		privateKey := ""
		params := jsonutils.NewDict()
		if len(opts.Key) > 0 {
			key, e := ioutil.ReadFile(opts.Key)
			if e != nil {
				return e
			}
			params.Add(jsonutils.NewString(string(key)), "private_key")
			privateKey = string(key)
		}

		i, e := modules.Servers.PerformAction(s, srvid, "login-info", params)
		if e != nil {
			return e
		}
		passwd, err := i.GetString("password")
		if err != nil && !opts.UseCloudroot {
			return err
		}
		if opts.Password != "" {
			passwd = opts.Password
		}
		user, err := i.GetString("username")
		if err != nil && !opts.UseCloudroot {
			return err
		}
		if opts.User != "" {
			user = opts.User
		}
		port := 22
		if opts.Port != 22 {
			port = opts.Port
		}

		var forwardItem *forwardInfo = nil
		host := address[0]
		if opts.Host != "" {
			host = opts.Host
		} else {
			if eip != "" {
				host = eip
			} else {
				if vpcid != "default" {
					forwardItem, err = openForward(s, srvid)
					if err != nil {
						return err
					}
					host = forwardItem.ProxyAddr
					port = forwardItem.ProxyPort
				}
			}
		}

		if opts.UseCloudroot {
			var err error
			privateKey, err = modules.Sshkeypairs.FetchPrivateKeyBySession(context.Background(), s)
			if err != nil {
				return err
			}
			passwd = ""
			user = "cloudroot"
		}

		var sshCli *ssh.Client
		err = nil
		for ; sshCli == nil; sshCli, err = ssh.NewClient(host, port, user, passwd, privateKey) {
			if err == nil {
				continue
			}
			if opts.Host != "" {
				return err
			}
			if forwardItem != nil {
				closeForward(s, srvid, forwardItem)
				return err
			} else {
				if vpcid != "default" {
					forwardItem, e = openForward(s, srvid)
					if e != nil {
						return e
					}
					host = forwardItem.ProxyAddr
					port = forwardItem.ProxyPort
				}
			}
		}

		log.Infof("ssh %s:%d", host, port)
		if err := sshCli.RunTerminal(); err != nil {
			if forwardItem != nil {
				closeForward(s, srvid, forwardItem)
			}
			return err
		}

		if forwardItem != nil {
			closeForward(s, srvid, forwardItem)
		}
		return nil
	})
}
