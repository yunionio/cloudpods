package shell

import (
	"fmt"

	"github.com/yunionio/jsonutils"
	"github.com/yunionio/onecloud/pkg/mcclient"
	"github.com/yunionio/onecloud/pkg/mcclient/modules"
)

func init() {
	type HostListOptions struct {
		Schedtag  string `help:"List hosts in schedtag"`
		Zone      string `help:"List hosts in zone"`
		Wire      string `help:"List hosts in wire"`
		VCenter   string `help:"List hosts in vcenter"`
		Image     string `help:"List hosts cached images"`
		Storage   string `help:"List hosts attached to storages"`
		Baremetal string `help:"List hosts that is managed by baremetal system" choices:"true|false"`
		Empty     bool   `help:"show empty host"`
		Occupied  bool   `help:"show occupid host"`
		Enabled   bool   `help:"Show enabled host only"`
		Disabled  bool   `help:"Show disabled host only"`
		HostType  string `help:"Host type filter" choices:"baremetal|hypervisor|esxi|kubelet|hyperv"`
		AnyMac    string `help:"Mac matches one of the host's interface"`

		Manager string `help:"Show regions belongs to the cloud provider"`

		BaseListOptions
	}
	R(&HostListOptions{}, "host-list", "List hosts", func(s *mcclient.ClientSession, args *HostListOptions) error {
		params := FetchPagingParams(args.BaseListOptions)
		if len(args.Schedtag) > 0 {
			params.Add(jsonutils.NewString(args.Schedtag), "schedtag")
		}
		if len(args.Zone) > 0 {
			params.Add(jsonutils.NewString(args.Zone), "zone")
		}
		if len(args.Wire) > 0 {
			params.Add(jsonutils.NewString(args.Wire), "wire")
		}
		if len(args.VCenter) > 0 {
			params.Add(jsonutils.NewString(args.VCenter), "vcenter")
		}
		if len(args.Image) > 0 {
			params.Add(jsonutils.NewString(args.Image), "cachedimage")
		}
		if len(args.Storage) > 0 {
			params.Add(jsonutils.NewString(args.Storage), "storage")
		}
		if len(args.Baremetal) > 0 {
			params.Add(jsonutils.NewString(args.Baremetal), "baremetal")
		}
		if len(args.HostType) > 0 {
			params.Add(jsonutils.NewString(args.HostType), "host_type")
		}

		if len(args.Manager) > 0 {
			params.Add(jsonutils.NewString(args.Manager), "manager")
		}

		if args.Empty {
			params.Add(jsonutils.JSONTrue, "is_empty")
		} else if args.Occupied {
			params.Add(jsonutils.JSONFalse, "is_empty")
		}
		if args.Enabled {
			params.Add(jsonutils.NewInt(1), "enabled")
		} else if args.Disabled {
			params.Add(jsonutils.NewInt(0), "enabled")
		}
		if len(args.AnyMac) > 0 {
			params.Add(jsonutils.NewString(args.AnyMac), "any_mac")
		}
		result, err := modules.Hosts.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.Hosts.GetColumns(s))
		return nil
	})

	type HostDetailOptions struct {
		ID string `help:"ID or name of host"`
	}
	R(&HostDetailOptions{}, "host-show", "Show details of a host", func(s *mcclient.ClientSession, args *HostDetailOptions) error {
		result, err := modules.Hosts.Get(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&HostDetailOptions{}, "host-ping", "Ping a host", func(s *mcclient.ClientSession, args *HostDetailOptions) error {
		result, err := modules.Hosts.PerformAction(s, args.ID, "ping", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&HostDetailOptions{}, "host-metadata", "Show metadata of a host", func(s *mcclient.ClientSession, args *HostDetailOptions) error {
		result, err := modules.Hosts.GetMetadata(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&HostDetailOptions{}, "host-enable", "Enable a host", func(s *mcclient.ClientSession, args *HostDetailOptions) error {
		result, err := modules.Hosts.PerformAction(s, args.ID, "enable", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&HostDetailOptions{}, "host-disable", "Disable a host", func(s *mcclient.ClientSession, args *HostDetailOptions) error {
		result, err := modules.Hosts.PerformAction(s, args.ID, "disable", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&HostDetailOptions{}, "host-syncstatus", "Synchronize status of a host", func(s *mcclient.ClientSession, args *HostDetailOptions) error {
		result, err := modules.Hosts.PerformAction(s, args.ID, "syncstatus", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&HostDetailOptions{}, "host-prepare", "Prepare a host for installation", func(s *mcclient.ClientSession, args *HostDetailOptions) error {
		result, err := modules.Hosts.PerformAction(s, args.ID, "prepare", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&HostDetailOptions{}, "host-ipmi", "Get IPMI information of a host", func(s *mcclient.ClientSession, args *HostDetailOptions) error {
		result, err := modules.Hosts.GetSpecific(s, args.ID, "ipmi", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&HostDetailOptions{}, "host-logininfo", "Get SSH login information of a host", func(s *mcclient.ClientSession, args *HostDetailOptions) error {
		srvid, e := modules.Hosts.GetId(s, args.ID, nil)
		if e != nil {
			return e
		}
		i, e := modules.Hosts.GetLoginInfo(s, srvid, nil)
		if e != nil {
			return e
		}
		printObject(i)
		return nil
	})

	R(&HostDetailOptions{}, "host-vnc", "Get VNC information of a host", func(s *mcclient.ClientSession, args *HostDetailOptions) error {
		result, err := modules.Hosts.GetSpecific(s, args.ID, "vnc", nil)
		if err != nil {
			return err
		}
		printObject(result)
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

	type HostUpdateOptions struct {
		ID                string  `help:"ID or Name of Host"`
		Name              string  `help:"New name of the host"`
		Desc              string  `help:"New Description of the host"`
		CpuCommitBound    float64 `help:"CPU overcommit upper bound at this host"`
		MemoryCommitBound float64 `help:"Memory overcommit upper bound at this host"`
		MemoryReserved    string  `help:"Memory reserved"`
		CpuReserved       int64   `help:"CPU reserved"`
		HostType          string  `help:"Change host type, CAUTION!!!!" choices:"hypervisor|kubelet|esxi|baremetal"`
		AccessIp          string  `help:"Change access ip, CAUTION!!!!"`
	}
	R(&HostUpdateOptions{}, "host-update", "Update information of a host", func(s *mcclient.ClientSession, args *HostUpdateOptions) error {
		params := jsonutils.NewDict()
		if len(args.Name) > 0 {
			params.Add(jsonutils.NewString(args.Name), "name")
		}
		if len(args.Desc) > 0 {
			params.Add(jsonutils.NewString(args.Desc), "description")
		}
		if args.CpuCommitBound > 0.0 {
			params.Add(jsonutils.NewFloat(args.CpuCommitBound), "cpu_cmtbound")
		}
		if args.MemoryCommitBound > 0.0 {
			params.Add(jsonutils.NewFloat(args.MemoryCommitBound), "mem_cmtbound")
		}
		if len(args.MemoryReserved) > 0 {
			params.Add(jsonutils.NewString(args.MemoryReserved), "mem_reserved")
		}
		if args.CpuReserved > 0 {
			params.Add(jsonutils.NewInt(args.CpuReserved), "cpu_reserved")
		}
		if len(args.HostType) > 0 {
			params.Add(jsonutils.NewString(args.HostType), "host_type")
		}
		if len(args.AccessIp) > 0 {
			params.Add(jsonutils.NewString(args.AccessIp), "access_ip")
		}
		if params.Size() == 0 {
			return fmt.Errorf("Not data to update")
		}
		result, err := modules.Hosts.Update(s, args.ID, params)
		if err != nil {
			return err
		}
		printObject(result)
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

	R(&HostDetailOptions{}, "host-undo-convert", "Undo converting a host to hypervisor", func(s *mcclient.ClientSession, args *HostDetailOptions) error {
		result, err := modules.Hosts.PerformAction(s, args.ID, "undo-convert", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&HostDetailOptions{}, "host-maintenance", "Reboot host into PXE offline OS, do maintenance jobs", func(s *mcclient.ClientSession, args *HostDetailOptions) error {
		result, err := modules.Hosts.PerformAction(s, args.ID, "maintenance", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&HostDetailOptions{}, "host-unmaintenance", "Reboot host back into disk installed OS", func(s *mcclient.ClientSession, args *HostDetailOptions) error {
		result, err := modules.Hosts.PerformAction(s, args.ID, "unmaintenance", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&HostDetailOptions{}, "host-start", "Power on host", func(s *mcclient.ClientSession, args *HostDetailOptions) error {
		result, err := modules.Hosts.PerformAction(s, args.ID, "start", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&HostDetailOptions{}, "host-stop", "Power on host", func(s *mcclient.ClientSession, args *HostDetailOptions) error {
		result, err := modules.Hosts.PerformAction(s, args.ID, "stop", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&HostDetailOptions{}, "host-reset", "Power reset host", func(s *mcclient.ClientSession, args *HostDetailOptions) error {
		result, err := modules.Hosts.PerformAction(s, args.ID, "reset", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	R(&HostDetailOptions{}, "host-delete", "Delete host record", func(s *mcclient.ClientSession, args *HostDetailOptions) error {
		result, err := modules.Hosts.Delete(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type HostAddNetIfOptions struct {
		ID     string `help:"ID or Name of host"`
		WIRE   string `help:"ID or Name of wire to attach"`
		MAC    string `help:"Mac address of NIC"`
		Type   string `help:"Nic type" choices:"admin|ipmi"`
		IpAddr string `help:"IP address"`
	}
	R(&HostAddNetIfOptions{}, "host-add-netif", "Host add a NIC", func(s *mcclient.ClientSession, args *HostAddNetIfOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.WIRE), "wire")
		params.Add(jsonutils.NewString(args.MAC), "mac")
		params.Add(jsonutils.JSONTrue, "link_up")
		if len(args.Type) > 0 {
			params.Add(jsonutils.NewString(args.Type), "nic_type")
		}
		if len(args.IpAddr) > 0 {
			params.Add(jsonutils.NewString(args.IpAddr), "ip_addr")
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
		ID  string `help:"ID or Name of host"`
		MAC string `help:"MAC of NIC to enable"`
	}
	R(&HostEnableNetIfOptions{}, "host-enable-netif", "Enable a network interface for a host", func(s *mcclient.ClientSession, args *HostEnableNetIfOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.MAC), "mac")
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

	R(&HostDetailOptions{}, "host-remove-all-netifs", "Remvoe all netifs expect admin&ipmi netifs", func(s *mcclient.ClientSession, args *HostDetailOptions) error {
		result, err := modules.Hosts.PerformAction(s, args.ID, "remove-all-netifs", nil)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type HostCacheImageActionOptions struct {
		ID    string `help:"ID or name of host"`
		IMAGE string `help:"ID or name of image"`
		Force bool   `help:"Force refresh cache, even if the image exists in cache"`
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

}
