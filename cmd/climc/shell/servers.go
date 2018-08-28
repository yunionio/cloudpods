package shell

import (
	"fmt"
	"io/ioutil"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

type ServerDeployInfo struct {
	Action  string
	Path    string
	Content string
}

func parseDeployInfo(info string) (ServerDeployInfo, error) {
	sdi := ServerDeployInfo{}
	colonPos := strings.IndexByte(info, ':')
	if colonPos <= 0 {
		return sdi, fmt.Errorf("Malformed deploy string")
	}
	if info[0] == '+' {
		sdi.Action = "append"
		sdi.Path = info[1:colonPos]
	} else {
		sdi.Action = "create"
		sdi.Path = info[:colonPos]
	}
	content := info[colonPos+1:]
	bytes, e := ioutil.ReadFile(content)
	if e != nil {
		sdi.Content = content
	} else {
		sdi.Content = string(bytes)
	}
	return sdi, nil
}

func parseDeployList(list []string, params *jsonutils.JSONDict) error {
	for i, info := range list {
		ret, e := parseDeployInfo(info)
		if e != nil {
			return e
		}
		params.Add(jsonutils.NewString(ret.Action), fmt.Sprintf("deploy.%d.action", i))
		params.Add(jsonutils.NewString(ret.Path), fmt.Sprintf("deploy.%d.path", i))
		params.Add(jsonutils.NewString(ret.Content), fmt.Sprintf("deploy.%d.content", i))
	}
	return nil
}

func init() {
	type ServerListOptions struct {
		Zone          string `help:"Zone ID or Name"`
		Wire          string `help:"Wire ID or Name"`
		Network       string `help:"Network ID or Name"`
		Disk          string `help:"Disk ID or Name"`
		Host          string `help:"Host ID or Name"`
		Baremetal     bool   `help:"Show baremetal servers"`
		Gpu           bool   `help:"Show gpu servers"`
		Secgroup      string `help:"Secgroup ID or Name"`
		AdminSecgroup string `help:"AdminSecgroup ID or Name"`
		Hypervisor    string `help:"Show server of hypervisor" choices:"kvm|esxi|container|baremetal|aliyun"`
		Manager       string `help:"Show servers imported from manager"`
		BaseListOptions
	}
	R(&ServerListOptions{}, "server-list", "List virtual servers", func(s *mcclient.ClientSession, args *ServerListOptions) error {
		params := FetchPagingParams(args.BaseListOptions)
		if len(args.Zone) > 0 {
			params.Add(jsonutils.NewString(args.Zone), "zone")
		}
		if len(args.Disk) > 0 {
			params.Add(jsonutils.NewString(args.Disk), "disk")
		}
		if len(args.Wire) > 0 {
			params.Add(jsonutils.NewString(args.Wire), "wire")
		}
		if len(args.Network) > 0 {
			params.Add(jsonutils.NewString(args.Network), "network")
		}
		if len(args.Host) > 0 {
			params.Add(jsonutils.NewString(args.Host), "host")
		}
		if args.Baremetal {
			params.Add(jsonutils.JSONTrue, "baremetal")
		}
		if args.Gpu {
			params.Add(jsonutils.JSONTrue, "gpu")
		}
		if len(args.Secgroup) > 0 {
			params.Add(jsonutils.NewString(args.Secgroup), "secgroup")
		}
		if len(args.AdminSecgroup) > 0 {
			params.Add(jsonutils.NewString(args.AdminSecgroup), "admin_secgroup")
		}
		if len(args.Hypervisor) > 0 {
			params.Add(jsonutils.NewString(args.Hypervisor), "hypervisor")
		}
		if len(args.Manager) > 0 {
			params.Add(jsonutils.NewString(args.Manager), "manager")
		}
		result, err := modules.Servers.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.Servers.GetColumns(s))
		return nil
	})

	type ServerShowOptions struct {
		ID       string `help:"ID or name of the server"`
		WithMeta bool   `help:"With meta data"`
	}
	R(&ServerShowOptions{}, "server-show", "Show details of a server", func(s *mcclient.ClientSession, args *ServerShowOptions) error {
		params := jsonutils.NewDict()
		if args.WithMeta {
			params.Add(jsonutils.JSONTrue, "with_meta")
		}
		result, err := modules.Servers.Get(s, args.ID, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&ServerShowOptions{}, "server-metadata", "Show metadata of a server", func(s *mcclient.ClientSession, args *ServerShowOptions) error {
		result, err := modules.Servers.GetMetadata(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type ServerCreateOptions struct {
		NAME             string   `help:"Name of server"`
		MEM              string   `help:"Memory size" metavar:"MEMORY"`
		Disk             []string `help:"Disk descriptions" nargs:"+"`
		Net              []string `help:"Network descriptions" metavar:"NETWORK"`
		IsolatedDevice   []string `help:"Isolated device model or ID" metavar:"ISOLATED_DEVICE"`
		Keypair          string   `help:"SSH Keypair"`
		Password         string   `help:"Default user password"`
		Iso              string   `help:"ISO image ID" metavar:"IMAGE_ID"`
		Ncpu             int64    `help:"#CPU cores of VM server, default 1" default:"1" metavar:"<SERVER_CPU_COUNT>"`
		Vga              string   `help:"VGA driver" choices:"std|vmware|cirrus|qxl"`
		Vdi              string   `help:"VDI protocool" choices:"vnc|spice"`
		Bios             string   `help:"BIOS" choices:"BIOS|UEFI"`
		Desc             string   `help:"Description" metavar:"<DESCRIPTION>"`
		Boot             string   `help:"Boot device" metavar:"<BOOT_DEVICE>" choices:"disk|cdrom"`
		NoAccountInit    bool     `help:"Not reset account password"`
		AllowDelete      bool     `help:"Unlock server to allow deleting"`
		ShutdownBehavior string   `help:"Behavior after VM server shutdown, stop or terminate server" metavar:"<SHUTDOWN_BEHAVIOR>" choices:"stop|terminate"`
		AutoStart        bool     `help:"Auto start server after it is created"`
		Zone             string   `help:"Preferred zone where virtual server should be created"`
		Host             string   `help:"Preferred host where virtual server should be created"`
		SchedTag         []string `help:"Schedule policy, key = aggregate name, value = require|exclude|prefer|avoid" metavar:"<KEY:VALUE>"`
		Deploy           []string `help:"Specify deploy files in virtual server file system"`
		Group            []string `help:"Group of virtual server"`
		Project          string   `help:"'Owner project ID or Name"`
		User             string   `help:"Owner user ID or Name"`
		System           bool     `help:"Create a system VM, sysadmin ONLY option"`
		Hypervisor       string   `help:"Hypervisor type" choices:"kvm|esxi|baremetal|container|aliyun"`
		TaskNotify       bool     `help:"Setup task notify"`
		Count            int      `help:"Create multiple simultaneously" default:"1"`
		DryRun           bool     `help:"Dry run to test scheduler"`
		RaidConfig       []string `help:"Baremetal raid config"`
	}
	R(&ServerCreateOptions{}, "server-create", "Create a server", func(s *mcclient.ClientSession, args *ServerCreateOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.NAME), "name")
		params.Add(jsonutils.NewString(args.MEM), "vmem_size")
		for i, d := range args.Disk {
			params.Add(jsonutils.NewString(d), fmt.Sprintf("disk.%d", i))
		}
		for i, n := range args.Net {
			params.Add(jsonutils.NewString(n), fmt.Sprintf("net.%d", i))
		}
		for i, g := range args.IsolatedDevice {
			params.Add(jsonutils.NewString(g), fmt.Sprintf("isolated_device.%d", i))
		}
		for i, t := range args.SchedTag {
			params.Add(jsonutils.NewString(t), fmt.Sprintf("schedtag.%d", i))
		}
		if args.Ncpu > 0 {
			params.Add(jsonutils.NewInt(args.Ncpu), "vcpu_count")
		}
		if len(args.Iso) > 0 {
			params.Add(jsonutils.NewString(args.Iso), "cdrom")
		}
		if len(args.Vga) > 0 {
			params.Add(jsonutils.NewString(args.Vga), "vga")
		}
		if args.NoAccountInit {
			params.Add(jsonutils.JSONFalse, "reset_password")
		}
		if args.Password != "" {
			params.Add(jsonutils.NewString(args.Password), "password")
		}
		if len(args.Vdi) > 0 {
			params.Add(jsonutils.NewString(args.Vdi), "vdi")
		}
		if len(args.Bios) > 0 {
			params.Add(jsonutils.NewString(args.Bios), "bios")
		}
		if len(args.Deploy) > 0 {
			err := parseDeployList(args.Deploy, params)
			if err != nil {
				return err
			}
		}
		if len(args.Boot) > 0 {
			if args.Boot == "disk" {
				params.Add(jsonutils.NewString("cdn"), "boot_order")
			} else {
				params.Add(jsonutils.NewString("dcn"), "boot_order")
			}
		}
		if len(args.Desc) > 0 {
			params.Add(jsonutils.NewString(args.Desc), "description")
		}
		if args.AllowDelete {
			params.Add(jsonutils.JSONFalse, "disable_delete")
		}
		if len(args.ShutdownBehavior) > 0 {
			params.Add(jsonutils.NewString(args.ShutdownBehavior), "shutdown_behavior")
		}
		if args.AutoStart {
			params.Add(jsonutils.JSONTrue, "auto_start")
		}
		if len(args.Group) > 0 {
			for i, g := range args.Group {
				params.Add(jsonutils.NewString(g), fmt.Sprintf("group.%d", i))
			}
		}
		if len(args.Host) > 0 {
			params.Add(jsonutils.NewString(args.Host), "prefer_host")
		} else {
			if len(args.Zone) > 0 {
				params.Add(jsonutils.NewString(args.Zone), "prefer_zone")
			}
		}
		if len(args.Project) > 0 {
			params.Add(jsonutils.NewString(args.Project), "tenant")
		}
		if len(args.User) > 0 {
			params.Add(jsonutils.NewString(args.User), "user")
		}
		if args.System {
			params.Add(jsonutils.JSONTrue, "is_system")
		}
		if len(args.Hypervisor) > 0 {
			params.Add(jsonutils.NewString(args.Hypervisor), "hypervisor")
		}
		if len(args.RaidConfig) > 0 {
			if args.Hypervisor != "baremetal" {
				return fmt.Errorf("RaidConfig is applicable to baremetal ONLY")
			}
			for i, conf := range args.RaidConfig {
				params.Add(jsonutils.NewString(conf), fmt.Sprintf("baremetal_disk_config.%d", i))
			}
		}
		if args.DryRun {
			params.Add(jsonutils.JSONTrue, "suggestion")
			results, err := modules.SchedManager.DoScheduleListResult(s, params, args.Count)
			if err != nil {
				return err
			}
			printList(results, []string{"id", "name", "rank", "capacity", "error"})
		} else {
			if args.TaskNotify {
				s.PrepareTask()
			}
			if args.Count > 1 {
				results := modules.Servers.BatchCreate(s, params, args.Count)
				printBatchResults(results, modules.Servers.GetColumns(s))
			} else {
				server, err := modules.Servers.Create(s, params)
				if err != nil {
					return err
				}
				printObject(server)
			}
			if args.TaskNotify {
				s.WaitTaskNotify()
			}
		}
		return nil
	})

	R(&ServerShowOptions{}, "server-logininfo", "Get login info of a server", func(s *mcclient.ClientSession, args *ServerShowOptions) error {
		srvid, e := modules.Servers.GetId(s, args.ID, nil)
		if e != nil {
			return e
		}
		i, e := modules.Servers.GetLoginInfo(s, srvid, nil)
		if e != nil {
			return e
		}
		printObject(i)
		return nil
	})

	type ServerOpsOptions struct {
		ID []string `help:"ID of servers to operate" metavar:"SERVER"`
	}
	R(&ServerOpsOptions{}, "server-start", "Start servers", func(s *mcclient.ClientSession, args *ServerOpsOptions) error {
		ret := modules.Servers.BatchPerformAction(s, args.ID, "start", nil)
		printBatchResults(ret, modules.Servers.GetColumns(s))
		return nil
	})

	R(&ServerOpsOptions{}, "server-syncstatus", "Sync servers status", func(s *mcclient.ClientSession, args *ServerOpsOptions) error {
		ret := modules.Servers.BatchPerformAction(s, args.ID, "syncstatus", nil)
		printBatchResults(ret, modules.Servers.GetColumns(s))
		return nil
	})

	R(&ServerOpsOptions{}, "server-sync", "Sync servers configures", func(s *mcclient.ClientSession, args *ServerOpsOptions) error {
		ret := modules.Servers.BatchPerformAction(s, args.ID, "sync", nil)
		printBatchResults(ret, modules.Servers.GetColumns(s))
		return nil
	})

	type ServerStopOptions struct {
		ID    []string `help:"ID or Name of server"`
		Force bool     `help:"Stop server forcefully"`
	}
	R(&ServerStopOptions{}, "server-stop", "Stop servers", func(s *mcclient.ClientSession, args *ServerStopOptions) error {
		params := jsonutils.NewDict()
		if args.Force {
			params.Add(jsonutils.JSONTrue, "is_force")
		} else {
			params.Add(jsonutils.JSONFalse, "is_force")
		}
		ret := modules.Servers.BatchPerformAction(s, args.ID, "stop", params)
		printBatchResults(ret, modules.Servers.GetColumns(s))
		return nil
	})

	R(&ServerOpsOptions{}, "server-suspend", "Suspend servers", func(s *mcclient.ClientSession, args *ServerOpsOptions) error {
		ret := modules.Servers.BatchPerformAction(s, args.ID, "suspend", nil)
		printBatchResults(ret, modules.Servers.GetColumns(s))
		return nil
	})

	type ServerResetOptions struct {
		ServerOpsOptions
		Hard bool `help:"Hard reset or not; default soft"`
	}
	R(&ServerResetOptions{}, "server-reset", "Reset servers", func(s *mcclient.ClientSession, args *ServerResetOptions) error {
		params := jsonutils.NewDict()
		if args.Hard {
			params.Add(jsonutils.JSONTrue, "is_hard")
		}
		ret := modules.Servers.BatchPerformAction(s, args.ID, "reset", params)
		printBatchResults(ret, modules.Servers.GetColumns(s))
		return nil
	})

	R(&ServerOpsOptions{}, "server-purge", "Purge obsolete servers", func(s *mcclient.ClientSession, args *ServerOpsOptions) error {
		ret := modules.Servers.BatchPerformAction(s, args.ID, "purge", nil)
		printBatchResults(ret, modules.Servers.GetColumns(s))
		return nil
	})

	type ServerDeleteOptions struct {
		OverridePendingDelete bool `help:"Delete server directly instead of pending delete"`
		ServerOpsOptions
	}
	R(&ServerDeleteOptions{}, "server-delete", "Delete servers", func(s *mcclient.ClientSession, args *ServerDeleteOptions) error {
		params := jsonutils.NewDict()
		if args.OverridePendingDelete {
			params.Add(jsonutils.JSONTrue, "override_pending_delete")
		}
		ret := modules.Servers.BatchDeleteWithParam(s, args.ID, params, nil)
		printBatchResults(ret, modules.Servers.GetColumns(s))
		return nil
	})

	R(&ServerOpsOptions{}, "server-cancel-delete", "Cancel pending delete servers", func(s *mcclient.ClientSession, args *ServerOpsOptions) error {
		ret := modules.Servers.BatchPerformAction(s, args.ID, "cancel-delete", nil)
		printBatchResults(ret, modules.Servers.GetColumns(s))
		return nil
	})

	R(&ServerShowOptions{}, "server-vnc", "Show vnc info of server", func(s *mcclient.ClientSession, args *ServerShowOptions) error {
		ret, e := modules.Servers.GetSpecific(s, args.ID, "vnc", nil)
		if e != nil {
			return e
		}
		printObject(ret)
		return nil
	})

	R(&ServerShowOptions{}, "server-desc", "Show desc info of server", func(s *mcclient.ClientSession, args *ServerShowOptions) error {
		ret, e := modules.Servers.GetSpecific(s, args.ID, "desc", nil)
		if e != nil {
			return e
		}
		printObject(ret)
		return nil
	})

	R(&ServerShowOptions{}, "server-status", "Show status of server", func(s *mcclient.ClientSession, args *ServerShowOptions) error {
		ret, e := modules.Servers.GetSpecific(s, args.ID, "status", nil)
		if e != nil {
			return e
		}
		printObject(ret)
		return nil
	})

	type ServerUpdateOptions struct {
		ID               []string `help:"IDs or Names of servers to update"`
		Name             string   `help:"New name to change"`
		Vmem             string   `help:"Memory size"`
		Ncpu             int64    `help:"CPU count"`
		Vga              string   `help:"VGA driver" choices:"std|vmware|cirrus|qxl"`
		Vdi              string   `help:"VDI protocol" choices:"vnc|spice"`
		Bios             string   `help:"BIOS" choices:"BIOS|UEFI"`
		Desc             string   `help:"Description"`
		Boot             string   `help:"Boot device" choices:"disk|cdrom"`
		Delete           string   `help:"Lock server to prevent from deleting" choices:"enable|disable"`
		ShutdownBehavior string   `help:"Behavior after VM server shutdown, stop or terminate server" choices:"stop|terminate"`
	}
	R(&ServerUpdateOptions{}, "server-update", "Update servers", func(s *mcclient.ClientSession, args *ServerUpdateOptions) error {
		params := jsonutils.NewDict()
		if len(args.Name) > 0 {
			params.Add(jsonutils.NewString(args.Name), "name")
		}
		if args.Ncpu > 0 {
			params.Add(jsonutils.NewInt(args.Ncpu), "vcpu_count")
		}
		if len(args.Vmem) > 0 {
			params.Add(jsonutils.NewString(args.Vmem), "vmem_size")
		}
		if len(args.Vga) > 0 {
			params.Add(jsonutils.NewString(args.Vga), "vga")
		}
		if len(args.Vdi) > 0 {
			params.Add(jsonutils.NewString(args.Vdi), "vdi")
		}
		if len(args.Bios) > 0 {
			params.Add(jsonutils.NewString(args.Bios), "bios")
		}
		if len(args.Desc) > 0 {
			params.Add(jsonutils.NewString(args.Desc), "description")
		}
		if len(args.Boot) > 0 {
			if args.Boot == "disk" {
				params.Add(jsonutils.NewString("cdn"), "boot_order")
			} else {
				params.Add(jsonutils.NewString("dcn"), "boot_order")
			}
		}
		if len(args.Delete) > 0 {
			if args.Delete == "disable" {
				params.Add(jsonutils.JSONTrue, "disable_delete")
			} else {
				params.Add(jsonutils.JSONFalse, "disable_delete")
			}
		}
		if len(args.ShutdownBehavior) > 0 {
			params.Add(jsonutils.NewString(args.ShutdownBehavior), "shutdown_behavior")
		}
		if params.Size() == 0 {
			return InvalidUpdateError()
		}
		result := modules.Servers.BatchPut(s, args.ID, params)
		printBatchResults(result, modules.Servers.GetColumns(s))
		return nil
	})

	type ServerSendKeyOptions struct {
		SERVER string `help:"ID of virtual server to get monitor info"`
		KEYS   string `help:"Special keys to send, eg. ctrl, alt, f12, shift, etc, separated by \"-\""`
		Hold   int64  `help:"Hold key for specified milliseconds"`
	}
	R(&ServerSendKeyOptions{}, "server-send-keys", "Send keys to server", func(s *mcclient.ClientSession, args *ServerSendKeyOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.KEYS), "keys")
		if args.Hold > 0 {
			params.Add(jsonutils.NewInt(args.Hold), "duration")
		}
		srv, err := modules.Servers.PerformAction(s, args.SERVER, "sendkeys", params)
		if err != nil {
			return err
		}
		printObject(srv)
		return nil
	})

	type ServerDeployOptions struct {
		ID            string   `help:"ID or Name of server"`
		Keypair       string   `help:"ssh Keypair used for login"`
		DeleteKeypair bool     `help:"Remove ssh Keypairs"`
		Deploy        []string `help:"Specify deploy files in virtual server file system"`
		ResetPassword bool     `help:"Force reset password"`
		Password      string   `help:"Default user password"`
	}
	R(&ServerDeployOptions{}, "server-deploy", "Deploy hostname and keypair to a stopped virtual server", func(s *mcclient.ClientSession, args *ServerDeployOptions) error {
		params := jsonutils.NewDict()
		if args.DeleteKeypair {
			params.Add(jsonutils.JSONTrue, "__delete_keypair__")
		} else if len(args.Keypair) > 0 {
			params.Add(jsonutils.NewString(args.Keypair), "keypair")
		}
		if len(args.Deploy) > 0 {
			e := parseDeployList(args.Deploy, params)
			if e != nil {
				return e
			}
		}
		if args.ResetPassword {
			params.Add(jsonutils.JSONTrue, "reset_password")
		}
		if args.Password != "" {
			params.Add(jsonutils.NewString(args.Password), "password")
		}
		srv, e := modules.Servers.PerformAction(s, args.ID, "deploy", params)
		if e != nil {
			return e
		}
		printObject(srv)
		return nil
	})

	type ServerSecGroupOptions struct {
		ID     string `help:"ID or Name of server" metavar:"Guest"`
		SecGrp string `help:"ID of Security Group" metavar:"Security Group" positional:"true"`
	}

	R(&ServerSecGroupOptions{}, "server-assign-secgroup", "Assign security group to a VM", func(s *mcclient.ClientSession, args *ServerSecGroupOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.SecGrp), "secgrp")
		srv, e := modules.Servers.PerformAction(s, args.ID, "assign-secgroup", params)
		if e != nil {
			return e
		}
		printObject(srv)
		return nil
	})

	R(&ServerSecGroupOptions{}, "server-assign-admin-secgroup", "Assign administrative security group to a VM", func(s *mcclient.ClientSession, args *ServerSecGroupOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.SecGrp), "secgrp")
		srv, e := modules.Servers.PerformAction(s, args.ID, "assign-admin-secgroup", params)
		if e != nil {
			return e
		}
		printObject(srv)
		return nil
	})

	type ServerRevokeSecGroupOptions struct {
		ID string `help:"ID or Name of server" metavar:"Guest"`
	}

	R(&ServerRevokeSecGroupOptions{}, "server-revoke-secgroup", "Assign security group to a VM", func(s *mcclient.ClientSession, args *ServerRevokeSecGroupOptions) error {
		srv, e := modules.Servers.PerformAction(s, args.ID, "revoke-secgroup", nil)
		if e != nil {
			return e
		}
		printObject(srv)
		return nil
	})

	R(&ServerRevokeSecGroupOptions{}, "server-revoke-admin-secgroup", "Assign administrative security group to a VM", func(s *mcclient.ClientSession, args *ServerRevokeSecGroupOptions) error {
		srv, e := modules.Servers.PerformAction(s, args.ID, "revoke-admin-secgroup", nil)
		if e != nil {
			return e
		}
		printObject(srv)
		return nil
	})

	type ServerMonitorOptions struct {
		SERVER string `help:"ID or Name of server"`
		CMD    string `help:"Qemu Monitor command to send"`
	}
	R(&ServerMonitorOptions{}, "server-monitor", "Send commands to qemu monitor", func(s *mcclient.ClientSession, args *ServerMonitorOptions) error {
		query := jsonutils.NewDict()
		query.Add(jsonutils.NewString(args.CMD), "command")
		ret, err := modules.Servers.GetSpecific(s, args.SERVER, "monitor", query)
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

	type ServerSaveImageOptions struct {
		SERVER string `help:"ID or name of server"`
		IMAGE  string `help:"Image name"`
		Public bool   `help:"Make the image public available"`
		Format string `help:"image format" choices:"vmdk|qcow2"`
		Notes  string `help:"Notes about the image"`
	}
	R(&ServerSaveImageOptions{}, "server-save-image", "Save root disk to new image and upload to glance.", func(s *mcclient.ClientSession, args *ServerSaveImageOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.IMAGE), "name")
		if args.Public {
			params.Add(jsonutils.JSONTrue, "is_public")
		} else {
			params.Add(jsonutils.JSONFalse, "is_public")
		}
		if len(args.Format) > 0 {
			params.Add(jsonutils.NewString(args.Format), "format")
		}
		if len(args.Notes) > 0 {
			params.Add(jsonutils.NewString(args.Notes), "notes")
		}
		srv, err := modules.Servers.PerformAction(s, args.SERVER, "save-image", params)
		if err != nil {
			return err
		}
		printObject(srv)
		return nil
	})

	type ServerChangeOwnerOptions struct {
		SERVER  string `help:"Server to change owner"`
		PROJECT string `help:"Project ID or change"`
		RawId   bool   `help:"User raw ID, instead of name"`
	}
	R(&ServerChangeOwnerOptions{}, "server-change-owner", "Change owner porject of a server", func(s *mcclient.ClientSession, args *ServerChangeOwnerOptions) error {
		params := jsonutils.NewDict()
		if args.RawId {
			projid, err := modules.Projects.GetId(s, args.PROJECT, nil)
			if err != nil {
				return err
			}
			params.Add(jsonutils.NewString(projid), "tenant")
			params.Add(jsonutils.JSONTrue, "raw_id")
		} else {
			params.Add(jsonutils.NewString(args.PROJECT), "tenant")
		}
		srv, err := modules.Servers.PerformAction(s, args.SERVER, "change-owner", params)
		if err != nil {
			return err
		}
		printObject(srv)
		return nil
	})

	type ServerRebuildRootOptions struct {
		SERVER string `help:"Server to rebuild root"`
		Image  string `help:"New root Image template ID"`
	}
	R(&ServerRebuildRootOptions{}, "server-rebuild-root", "Rebuild VM root image with new template", func(s *mcclient.ClientSession, args *ServerRebuildRootOptions) error {
		params := jsonutils.NewDict()
		if len(args.Image) > 0 {
			params.Add(jsonutils.NewString(args.Image), "image_id")
		}
		srv, err := modules.Servers.PerformAction(s, args.SERVER, "rebuild-root", params)
		if err != nil {
			return err
		}
		printObject(srv)
		return nil
	})

	type ServerChangeConfigOptions struct {
		SERVER string   `help:"Server to rebuild root"`
		Ncpu   int64    `help:"New number of Virtual CPU cores"`
		Vmem   string   `help:"New memory size"`
		Disk   []string `help:"Data disk description, from the 1st data disk to the last one, empty string if no change for this data disk"`
	}
	R(&ServerChangeConfigOptions{}, "server-change-config", "Change configuration of VM", func(s *mcclient.ClientSession, args *ServerChangeConfigOptions) error {
		params := jsonutils.NewDict()
		if args.Ncpu > 0 {
			params.Add(jsonutils.NewInt(args.Ncpu), "vcpu_count")
		}
		if len(args.Vmem) > 0 {
			params.Add(jsonutils.NewString(args.Vmem), "vmem_size")
		}
		if len(args.Disk) > 0 {
			for idx, disk := range args.Disk {
				params.Add(jsonutils.NewString(disk), fmt.Sprintf("disk.%d", (idx+1)))
			}
		}
		if params.Size() == 0 {
			return InvalidUpdateError()
		}
		srv, err := modules.Servers.PerformAction(s, args.SERVER, "change-config", params)
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
	R(&ServerInsertISOOptions{}, "server-insert-iso", "Insert an ISO image into server's cdrom", func(s *mcclient.ClientSession, args *ServerInsertISOOptions) error {
		img, err := modules.Images.Get(s, args.ISO, nil)
		if err != nil {
			return err
		}
		imgId, err := img.GetString("id")
		if err != nil {
			return err
		}
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(imgId), "image_id")
		result, err := modules.Servers.PerformAction(s, args.ID, "insertiso", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&ServerShowOptions{}, "server-eject-iso", "Eject iso from servers' cdrom", func(s *mcclient.ClientSession, args *ServerShowOptions) error {
		result, err := modules.Servers.PerformAction(s, args.ID, "ejectiso", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&ServerShowOptions{}, "server-iso", "Show server's mounting ISO information", func(s *mcclient.ClientSession, args *ServerShowOptions) error {
		results, err := modules.Servers.GetSpecific(s, args.ID, "iso", nil)
		if err != nil {
			return err
		}
		printObject(results)
		return nil
	})
}
