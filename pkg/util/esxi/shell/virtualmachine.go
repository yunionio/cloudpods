package shell

import (
	"yunion.io/x/onecloud/pkg/util/esxi"
	"yunion.io/x/onecloud/pkg/util/printutils"
	"yunion.io/x/onecloud/pkg/util/shellutils"
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
		VMID   string `help:"VM ID"`
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

	shellutils.R(&VirtualMachineShowOptions{}, "vm-nics", "Show vm nics details", func(cli *esxi.SESXiClient, args *VirtualMachineShowOptions) error {
		host, err := cli.FindHostByIp(args.HOSTIP)
		if err != nil {
			return err
		}
		vm, err := host.GetIVMById(args.VMID)
		if err != nil {
			return err
		}
		vmnics, err := vm.GetINics()
		if err != nil {
			return err
		}
		printList(vmnics, []string{})
		return nil
	})

	shellutils.R(&VirtualMachineShowOptions{}, "vm-disks", "Show vm disks details", func(cli *esxi.SESXiClient, args *VirtualMachineShowOptions) error {
		host, err := cli.FindHostByIp(args.HOSTIP)
		if err != nil {
			return err
		}
		vm, err := host.GetIVMById(args.VMID)
		if err != nil {
			return err
		}
		vmdisks, err := vm.GetIDisks()
		if err != nil {
			return err
		}
		printList(vmdisks, []string{})
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
