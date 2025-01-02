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
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"
	"yunion.io/x/pkg/util/printutils"

	"yunion.io/x/onecloud/cmd/climc/shell"
	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/mcclient/options"
	"yunion.io/x/onecloud/pkg/mcclient/options/compute"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/ssh"
)

func init() {
	cmd := shell.NewResourceCmd(&modules.Hosts)
	cmd.List(&compute.HostListOptions{})
	cmd.GetMetadata(&options.BaseIdOptions{})
	cmd.GetProperty(&compute.HostStatusStatisticsOptions{})
	cmd.Update(&compute.HostUpdateOptions{})

	cmd.Perform("ping", &options.BaseIdOptions{})
	cmd.Perform("purge", &options.BaseIdOptions{})
	cmd.Perform("undo-convert", &options.BaseIdOptions{})
	cmd.Perform("maintenance", &options.BaseIdOptions{})
	cmd.Perform("unmaintenance", &options.BaseIdOptions{})
	cmd.Perform("start", &options.BaseIdOptions{})
	cmd.Perform("stop", &options.BaseIdOptions{})
	cmd.Perform("reset", &options.BaseIdOptions{})
	cmd.BatchDelete(&options.BaseIdsOptions{})
	cmd.Perform("remove-all-netifs", &options.BaseIdOptions{})
	cmd.Perform("probe-isolated-devices", &options.BaseIdOptions{})
	cmd.Perform("class-metadata", &options.ResourceMetadataOptions{})
	cmd.Perform("set-class-metadata", &options.ResourceMetadataOptions{})
	cmd.PerformClass("validate-ipmi", &compute.HostValidateIPMI{})
	cmd.Perform("set-commit-bound", &compute.HostSetCommitBoundOptions{})

	cmd.BatchPerform("enable", &options.BaseIdsOptions{})
	cmd.BatchPerform("disable", &options.BaseIdsOptions{})
	cmd.BatchPerform("syncstatus", &options.BaseIdsOptions{})
	cmd.BatchPerform("sync-config", &options.BaseIdsOptions{})
	cmd.BatchPerform("prepare", &options.BaseIdsOptions{})
	cmd.BatchPerform("ipmi-probe", &options.BaseIdsOptions{})
	cmd.BatchPerform("reserve-cpus", &compute.HostReserveCpusOptions{})
	cmd.BatchPerform("unreserve-cpus", &options.BaseIdsOptions{})
	cmd.BatchPerform("auto-migrate-on-host-down", &compute.HostAutoMigrateOnHostDownOptions{})
	cmd.BatchPerform("restart-host-agent", &options.BaseIdsOptions{})

	cmd.Get("ipmi", &options.BaseIdOptions{})
	cmd.Get("vnc", &options.BaseIdOptions{})
	cmd.Get("app-options", &options.BaseIdOptions{})
	cmd.Get("tap-config", &options.BaseIdOptions{})
	cmd.GetWithCustomShow("nics", func(data jsonutils.JSONObject) {
		results := printutils.ListResult{}
		err := data.Unmarshal(&results)
		if err == nil {
			printutils.PrintJSONList(&results, []string{
				"mac",
				"vlan_id",
				"interface",
				"bridge",
				"ip_addr",
				"net",
				"wire",
			})
		} else {
			fmt.Println("error", err)
		}
	}, &options.BaseIdOptions{})

	R(&compute.HostShowOptions{}, "host-show", "Show details of a host", func(s *mcclient.ClientSession, args *compute.HostShowOptions) error {
		params, err := args.Params()
		if err != nil {
			return err
		}
		result, err := modules.Hosts.Get(s, args.ID, params)
		if err != nil {
			return err
		}
		resultDict := result.(*jsonutils.JSONDict)
		if !args.ShowAll {
			if !args.ShowMetadata {
				print("ShowMetadata\n")
				resultDict.Remove("metadata")
			}
			if !args.ShowNicInfo {
				print("ShowMetadata\n")
				resultDict.Remove("nic_info")
			}
			if !args.ShowSysInfo {
				print("ShowSysInfo\n")
				resultDict.Remove("sys_info")
			}
		}
		printObject(resultDict)
		return nil
	})

	R(&options.BaseIdOptions{}, "host-logininfo", "Get SSH login information of a host", func(s *mcclient.ClientSession, args *options.BaseIdOptions) error {
		i, e := modules.Hosts.PerformAction(s, args.ID, "login-info", nil)
		if e != nil {
			return e
		}
		printObject(i)
		return nil
	})

	type HostPropertyOptions struct {
	}

	R(&HostPropertyOptions{}, "baremetal-register-script", "Get online baremetal register script", func(s *mcclient.ClientSession, args *HostPropertyOptions) error {
		result, err := modules.Hosts.Get(s, "bm-start-register-script", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	countFunc := func(s *mcclient.ClientSession, opts *compute.HostListOptions, action string) error {
		params, err := options.ListStructToParams(opts)
		if err != nil {
			return err
		}

		if opts.Empty {
			params.Add(jsonutils.JSONTrue, "is_empty")
		} else if opts.Occupied {
			params.Add(jsonutils.JSONFalse, "is_empty")
		}
		if opts.Enabled {
			params.Add(jsonutils.NewInt(1), "enabled")
		} else if opts.Disabled {
			params.Add(jsonutils.NewInt(0), "enabled")
		}
		if len(opts.Uuid) > 0 {
			params.Add(jsonutils.NewString(opts.Uuid), "uuid")
		}
		if len(opts.Sn) > 0 {
			params.Add(jsonutils.NewString(opts.Sn), "sn")
		}
		result, err := modules.Hosts.Get(s, action, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	}
	R(&compute.HostListOptions{}, "host-node-count", "Get host node count", func(s *mcclient.ClientSession, opts *compute.HostListOptions) error {
		return countFunc(s, opts, "node-count")
	})
	R(&compute.HostListOptions{}, "host-type-count", "Get host type count", func(s *mcclient.ClientSession, opts *compute.HostListOptions) error {
		return countFunc(s, opts, "host-type-count")
	})

	type HostSysInfoOpt struct {
		options.BaseIdOptions
		Key    string `help:"The key for extract, e.g. 'cpu_info.processors'"`
		Format string `help:"Output format" choices:"yaml|json" default:"yaml"`
	}

	R(&HostSysInfoOpt{}, "host-sysinfo", "Get host system info", func(s *mcclient.ClientSession, args *HostSysInfoOpt) error {
		obj, err := modules.Hosts.Get(s, args.GetId(), nil)
		if err != nil {
			return err
		}
		keys := []string{"sys_info"}
		if args.Key != "" {
			keys = append(keys, strings.Split(args.Key, ".")...)
		}
		sysInfo, err := obj.Get(keys...)
		if err != nil {
			return errors.Wrap(err, "Get sys_info")
		}
		if args.Format == "yaml" {
			fmt.Print(sysInfo.YAMLString())
		} else {
			fmt.Print(sysInfo.PrettyString())
		}
		return nil
	})

	type HostConvertOptions struct {
		ID         string   `help:"Host ID or Name"`
		Name       string   `help:"New name of the converted host"`
		HOSTTYPE   string   `help:"Convert host type" choices:"hypervisor|esxi|kubelet|hyperv"`
		Image      string   `help:"Template image to install"`
		Raid       string   `help:"Raid to deploy" choices:"raid0|raid1|raid10|raid5|none"`
		RaidConfig []string `help:"Baremetal raid config"`
		Disk       []string `help:"Disk descriptions" metavar:"DISK"`
		Net        []string `help:"Network descriptions" metavar:"NETWORK"`
	}
	R(&HostConvertOptions{}, "host-convert-hypervisor", "Convert a baremetal into a hypervisor", func(s *mcclient.ClientSession, args *HostConvertOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.HOSTTYPE), "host_type")
		if len(args.Name) > 0 {
			params.Add(jsonutils.NewString(args.Name), "name")
		}
		if len(args.RaidConfig) > 0 && len(args.Raid) > 0 {
			return fmt.Errorf("Cannot specifiy raidconfig and raid simultaneously")
		} else if len(args.RaidConfig) > 0 {
			for i := 0; i < len(args.RaidConfig); i += 1 {
				params.Add(jsonutils.NewString(args.RaidConfig[i]), fmt.Sprintf("baremetal_disk_config.%d", i))
			}
		} else if len(args.Raid) > 0 {
			params.Add(jsonutils.NewString(args.Raid), "raid")
		}
		if len(args.Disk) > 0 && len(args.Image) > 0 {
			return fmt.Errorf("Cannot specify disk and image simultaneously")
		} else if len(args.Disk) > 0 {
			for i := 0; i < len(args.Disk); i += 1 {
				params.Add(jsonutils.NewString(args.Disk[i]), fmt.Sprintf("disk.%d", i))
			}
		} else if len(args.Image) > 0 {
			params.Add(jsonutils.NewString(args.Image), "image")
		}
		if len(args.Net) > 0 {
			for i := 0; i < len(args.Net); i += 1 {
				params.Add(jsonutils.NewString(args.Net[i]), fmt.Sprintf("net.%d", i))
			}
		}
		result, err := modules.Hosts.PerformAction(s, args.ID, "convert-hypervisor", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
	type HostAddNetIfOptions struct {
		ID        string `help:"ID or Name of host"`
		WIRE      string `help:"ID or Name of wire to attach"`
		MAC       string `help:"Mac address of NIC"`
		INDEX     int64  `help:"nic index"`
		Type      string `help:"Nic type" choices:"admin|ipmi"`
		IpAddr    string `help:"IP address"`
		Bridge    string `help:"Bridge of hostwire"`
		Interface string `help:"Interface name, eg:eth0, en0"`
	}
	R(&HostAddNetIfOptions{}, "host-add-netif", "Host add a NIC", func(s *mcclient.ClientSession, args *HostAddNetIfOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.WIRE), "wire")
		params.Add(jsonutils.NewString(args.MAC), "mac")
		params.Add(jsonutils.JSONTrue, "link_up")
		params.Add(jsonutils.NewInt(args.INDEX), "index")
		if len(args.Type) > 0 {
			params.Add(jsonutils.NewString(args.Type), "nic_type")
		}
		if len(args.IpAddr) > 0 {
			params.Add(jsonutils.NewString(args.IpAddr), "ip_addr")
		}
		if len(args.Bridge) > 0 {
			params.Add(jsonutils.NewString(args.Bridge), "bridge")
		}
		if len(args.Interface) > 0 {
			params.Add(jsonutils.NewString(args.Interface), "interface")
		}
		result, err := modules.Hosts.PerformAction(s, args.ID, "add-netif", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type HostRemoveNetIfOptions struct {
		ID  string `help:"ID or Name of host"`
		MAC string `help:"MAC of NIC to remove"`
	}
	R(&HostRemoveNetIfOptions{}, "host-remove-netif", "Remove NIC from host", func(s *mcclient.ClientSession, args *HostRemoveNetIfOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.MAC), "mac")
		result, err := modules.Hosts.PerformAction(s, args.ID, "remove-netif", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type HostEnableNetIfOptions struct {
		ID       string `help:"ID or Name of host"`
		MAC      string `help:"MAC of NIC to enable"`
		Ip       string `help:"IP address"`
		Network  string `help:"network to connect"`
		Reserved bool   `help:"fetch IP from reserved pool"`
	}
	R(&HostEnableNetIfOptions{}, "host-enable-netif", "Enable a network interface for a host", func(s *mcclient.ClientSession, args *HostEnableNetIfOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.MAC), "mac")
		if len(args.Ip) > 0 {
			params.Add(jsonutils.NewString(args.Ip), "ip_addr")
			if args.Reserved {
				params.Add(jsonutils.JSONTrue, "reserve")
			}
		}
		if len(args.Network) > 0 {
			params.Add(jsonutils.NewString(args.Network), "network")
		}
		result, err := modules.Hosts.PerformAction(s, args.ID, "enable-netif", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type HostDisableNetIfOptions struct {
		ID      string `help:"ID or Name of host"`
		MAC     string `help:"MAC of NIC to disable"`
		Reserve bool   `help:"Reserve the IP address"`
	}
	R(&HostDisableNetIfOptions{}, "host-disable-netif", "Disable a network interface", func(s *mcclient.ClientSession, args *HostDisableNetIfOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.MAC), "mac")
		if args.Reserve {
			params.Add(jsonutils.JSONTrue, "reserve")
		}
		result, err := modules.Hosts.PerformAction(s, args.ID, "disable-netif", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type HostCacheImageActionOptions struct {
		ID     string `help:"ID or name of host"`
		IMAGE  string `help:"ID or name of image"`
		Force  bool   `help:"Force refresh cache, even if the image exists in cache"`
		Format string `help:"image format" choices:"iso|vmdk|qcow2|vhd"`
	}
	R(&HostCacheImageActionOptions{}, "host-cache-image", "Ask a host to cache a image", func(s *mcclient.ClientSession, args *HostCacheImageActionOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.IMAGE), "image")
		if args.Force {
			params.Add(jsonutils.JSONTrue, "is_force")
		}
		host, err := modules.Hosts.PerformAction(s, args.ID, "cache-image", params)
		if err != nil {
			return err
		}
		printObject(host)
		return nil
	})

	type HostUncacheImageActionOptions struct {
		ID    string `help:"ID or name of host"`
		IMAGE string `help:"ID or name of image"`
	}
	R(&HostUncacheImageActionOptions{}, "host-uncache-image", "Ask a host to remove image from a cache", func(s *mcclient.ClientSession, args *HostUncacheImageActionOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.IMAGE), "image")
		host, err := modules.Hosts.PerformAction(s, args.ID, "uncache-image", params)
		if err != nil {
			return err
		}
		printObject(host)
		return nil
	})

	type HostCreateOptions struct {
		NAME       string `help:"Name of baremetal"`
		MAC        string `help:"Default MAC address of baremetal"`
		Rack       string `help:"Rack number of baremetal"`
		Slots      string `help:"Slots number of baremetal"`
		IpmiUser   string `help:"IPMI user name"`
		IpmiPasswd string `help:"IPMI user password"`
		IpmiAddr   string `help:"IPMI IP address"`

		AccessIp   string `help:"Access IP address"`
		AccessNet  string `help:"Access network"`
		AccessWire string `help:"Access wire"`

		NoProbe   bool `help:"just save the record, do not probe"`
		NoBMC     bool `help:"No BMC hardware"`
		NoPrepare bool `help:"just probe, do not reboot baremetal to prepare"`

		DisablePxeBoot bool `help:"set enable_pxe_boot to false, which is true by default"`

		Uuid string `help:"host uuid"`
	}
	R(&HostCreateOptions{}, "host-create", "Create a baremetal host", func(s *mcclient.ClientSession, args *HostCreateOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.NAME), "name")
		params.Add(jsonutils.NewString(args.MAC), "access_mac")
		params.Add(jsonutils.NewString("baremetal"), "host_type")
		if len(args.Rack) > 0 {
			params.Add(jsonutils.NewString(args.Rack), "rack")
		}
		if len(args.Slots) > 0 {
			params.Add(jsonutils.NewString(args.Slots), "slots")
		}
		if len(args.IpmiUser) > 0 {
			params.Add(jsonutils.NewString(args.IpmiUser), "ipmi_username")
		}
		if len(args.IpmiPasswd) > 0 {
			params.Add(jsonutils.NewString(args.IpmiPasswd), "ipmi_password")
		}
		if len(args.IpmiAddr) > 0 {
			params.Add(jsonutils.NewString(args.IpmiAddr), "ipmi_ip_addr")
		}
		if len(args.AccessIp) > 0 {
			params.Add(jsonutils.NewString(args.AccessIp), "access_ip")
		}
		if len(args.AccessNet) > 0 {
			params.Add(jsonutils.NewString(args.AccessNet), "access_net")
		}
		if len(args.AccessWire) > 0 {
			params.Add(jsonutils.NewString(args.AccessWire), "access_wire")
		}
		if args.NoProbe {
			params.Add(jsonutils.JSONTrue, "no_probe")
		}
		if args.NoPrepare {
			params.Add(jsonutils.JSONTrue, "no_prepare")
		}
		if args.NoBMC {
			params.Add(jsonutils.JSONTrue, "no_bmc")
		}
		if args.DisablePxeBoot {
			params.Add(jsonutils.JSONFalse, "enable_pxe_boot")
		}
		if len(args.Uuid) > 0 {
			params.Add(jsonutils.NewString(args.Uuid), "uuid")
		}
		result, err := modules.Hosts.Create(s, params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type HostUndoPrepaidRecycleOptions struct {
		ID string `help:"ID or name of host to undo recycle"`
	}
	R(&HostUndoPrepaidRecycleOptions{}, "host-undo-recycle", "Pull a prepaid server from recycle pool, so that it will not be shared any more", func(s *mcclient.ClientSession, args *HostUndoPrepaidRecycleOptions) error {
		params := jsonutils.NewDict()
		result, err := modules.Hosts.PerformAction(s, args.ID, "undo-prepaid-recycle", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type PrepaidRecycleHostRenewOptions struct {
		ID       string `help:"ID or name of server to renew"`
		DURATION string `help:"Duration of renew, ADMIN only command"`
	}
	R(&PrepaidRecycleHostRenewOptions{}, "host-renew-prepaid-recycle", "Renew a prepaid recycle host", func(s *mcclient.ClientSession, args *PrepaidRecycleHostRenewOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.DURATION), "duration")
		result, err := modules.Hosts.PerformAction(s, args.ID, "renew-prepaid-recycle", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type HostSpecOptions struct {
		ID string `help:"ID or name of host"`
	}
	R(&HostSpecOptions{}, "host-spec", "Get host spec info", func(s *mcclient.ClientSession, args *HostSpecOptions) error {
		spec, err := modules.Hosts.GetSpecific(s, args.ID, "spec", nil)
		if err != nil {
			return err
		}
		printObject(spec)
		return nil
	})

	type HostJnlpOptions struct {
		ID   string `help:"ID or name of host"`
		Save string `help:"save xml into this file"`
	}
	R(&HostJnlpOptions{}, "host-jnlp", "Get host jnlp file contentn", func(s *mcclient.ClientSession, args *HostJnlpOptions) error {
		spec, err := modules.Hosts.GetSpecific(s, args.ID, "jnlp", nil)
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

	type HostInsertIsoOptions struct {
		ID    string `help:"ID or name of host" json:"-"`
		Image string `help:"ID or name or ISO image name" json:"image"`
		Boot  bool   `help:"Boot from ISO on next reset" json:"boot"`
	}
	R(&HostInsertIsoOptions{}, "host-insert-iso", "Insert ISO into virtual cd-rom of host", func(s *mcclient.ClientSession, args *HostInsertIsoOptions) error {
		params := jsonutils.Marshal(args)
		result, err := modules.Hosts.PerformAction(s, args.ID, "insert-iso", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type HostEjectIsoOptions struct {
		ID string `help:"ID or name of host" json:"-"`
	}
	R(&HostEjectIsoOptions{}, "host-eject-iso", "Eject ISO from virtual cd-rom of host", func(s *mcclient.ClientSession, args *HostEjectIsoOptions) error {
		params := jsonutils.Marshal(args)
		result, err := modules.Hosts.PerformAction(s, args.ID, "eject-iso", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type HostSSHLoginOptions struct {
		ID   string `help:"ID or name of host"`
		Port int    `help:"SSH service port" default:"22"`
	}
	R(&HostSSHLoginOptions{}, "host-ssh", "SSH login of a host", func(s *mcclient.ClientSession, args *HostSSHLoginOptions) error {
		i, e := modules.Hosts.PerformAction(s, args.ID, "login-info", nil)
		privateKey := ""
		if e != nil {
			if httputils.ErrorCode(e) == 404 || strings.Contains(e.Error(), "ciphertext too short") {
				var err error
				privateKey, err = modules.Sshkeypairs.FetchPrivateKeyBySession(context.Background(), s)
				if err != nil {
					return errors.Wrap(err, "fetch private key")
				}
				params := jsonutils.NewDict()
				params.Add(jsonutils.NewString(string(args.ID)), "ID")
				ret, err := modules.Hosts.Get(s, args.ID, params)
				if err != nil {
					return errors.Wrap(err, "get host by ID")
				}
				ip, err := ret.GetString("access_ip")
				if err != nil {
					return errors.Wrap(err, "get the ip of the host")
				}

				jsonItem := jsonutils.NewDict()
				jsonItem.Add(jsonutils.NewString(ip), "ip")
				jsonItem.Add(jsonutils.NewString("root"), "username")
				jsonItem.Add(jsonutils.NewString(""), "password")
				jsonItem.Add(jsonutils.NewInt(22), "port")
				i = jsonItem
			} else {
				return e
			}
		}

		host, err := i.GetString("ip")
		if err != nil {
			return err
		}
		user, err := i.GetString("username")
		if err != nil {
			return err
		}
		passwd, err := i.GetString("password")
		if err != nil {
			return err
		}
		port := 22
		if args.Port != 22 {
			port = args.Port
		}
		sshCli, err := ssh.NewClient(host, port, user, passwd, privateKey)
		if err != nil {
			return err
		}
		log.Infof("ssh %s:%d", host, port)
		if err := sshCli.RunTerminal(); err != nil {
			return err
		}
		return nil
	})

	type HostChangeOwnerOptions struct {
		ID            string `help:"ID or name of host" json:"-"`
		ProjectDomain string `json:"project_domain" help:"target domain"`
	}
	R(&HostChangeOwnerOptions{}, "host-change-owner", "Change owner domain of host", func(s *mcclient.ClientSession, args *HostChangeOwnerOptions) error {
		if len(args.ProjectDomain) == 0 {
			return fmt.Errorf("empty project_domain")
		}
		params := jsonutils.Marshal(args)
		ret, err := modules.Hosts.PerformAction(s, args.ID, "change-owner", params)
		if err != nil {
			return err
		}
		printObject(ret)
		return nil
	})

	type HostPublicOptions struct {
		ID            string   `help:"ID or name of host" json:"-"`
		Scope         string   `help:"sharing scope" choices:"system|domain"`
		SharedDomains []string `help:"share to domains"`
	}
	R(&HostPublicOptions{}, "host-public", "Make a host public", func(s *mcclient.ClientSession, args *HostPublicOptions) error {
		params := jsonutils.Marshal(args)
		result, err := modules.Hosts.PerformAction(s, args.ID, "public", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type HostPrivateOptions struct {
		ID string `help:"ID or name of host" json:"-"`
	}
	R(&HostPrivateOptions{}, "host-private", "Make a host private", func(s *mcclient.ClientSession, args *HostPrivateOptions) error {
		params := jsonutils.Marshal(args)
		result, err := modules.Hosts.PerformAction(s, args.ID, "private", params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type HostChangeOwnerCandidateDomainsOptions struct {
		ID string `help:"ID or name of host"`
	}
	R(&HostChangeOwnerCandidateDomainsOptions{}, "host-change-owner-candidate-domains", "Get change owner candidate domain list", func(s *mcclient.ClientSession, args *HostChangeOwnerCandidateDomainsOptions) error {
		result, err := modules.Hosts.GetSpecific(s, args.ID, "change-owner-candidate-domains", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type HostSetReservedResourceForIsolatedDevice struct {
		ID              []string `help:"ID or name of host" json:"-"`
		ReservedCpu     *int     `help:"reserved cpu count"`
		ReservedMem     *int     `help:"reserved mem count"`
		ReservedStorage *int     `help:"reserved storage count"`
	}
	R(&HostSetReservedResourceForIsolatedDevice{},
		"host-set-reserved-resource-for-isolated-device",
		"Set reserved resource for isolated device",
		func(s *mcclient.ClientSession, args *HostSetReservedResourceForIsolatedDevice) error {
			res := modules.Hosts.BatchPerformAction(s, args.ID, "set-reserved-resource-for-isolated-device", jsonutils.Marshal(args))
			printBatchResults(res, modules.Hosts.GetColumns(s))
			return nil
		},
	)
}
