package shell

import (
	"github.com/yunionio/onecloud/pkg/util/shellutils"
	"github.com/yunionio/onecloud/pkg/util/esxi"
	"github.com/yunionio/onecloud/pkg/util/printutils"
)

func init() {
	type VirtualMachineListOptions struct {
		HOSTIP string `help:"Host IP"`
	}
	shellutils.R(&VirtualMachineListOptions{}, "vm-list", "List vms of a host", func(cli *esxi.SESXiClient, args *VirtualMachineListOptions) error {
		host, err := cli.FindHostByIp(args.HOSTIP)
		if err != nil {
			return err
		}
		vms, err := host.GetIVMs()
		if err != nil {
			return err
		}
		printList(vms, []string{})
		return nil
	})

	type VirtualMachineShowOptions struct {
		HOSTIP string `help:"Host IP"`
		VMID string `help:"VM ID"`
	}
	shellutils.R(&VirtualMachineShowOptions{}, "vm-show", "Show vm details", func(cli *esxi.SESXiClient, args *VirtualMachineShowOptions) error {
		host, err := cli.FindHostByIp(args.HOSTIP)
		if err != nil {
			return err
		}
		vm, err := host.GetIVMById(args.VMID)
		if err != nil {
			return err
		}
		printObject(vm)
		return nil
	})

	shellutils.R(&VirtualMachineShowOptions{}, "vm-vnc", "Show vm VNC details", func(cli *esxi.SESXiClient, args *VirtualMachineShowOptions) error {
		host, err := cli.FindHostByIp(args.HOSTIP)
		if err != nil {
			return err
		}
		vm, err := host.GetIVMById(args.VMID)
		if err != nil {
			return err
		}
		info, err := vm.GetVNCInfo()
		if err != nil {
			return err
		}
		printutils.PrintJSONObject(info)
		return nil
	})
}
