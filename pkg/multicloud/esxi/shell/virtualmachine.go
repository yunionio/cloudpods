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
	"context"
	"fmt"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/multicloud/esxi"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type VirtualMachineListOptions struct {
		Datacenter string `help:"Datacenter"`
		HostIP     string `help:"HostIP"`
		FakeRegex  string `help:"regex for fake template' name"`
		Datastore  string `help:"Datastore"`
		Template   bool   `help:"Whether it is tempalte virtual machine, default:false"`
	}
	shellutils.R(&VirtualMachineListOptions{}, "vm-list", "List vms of a host", func(cli *esxi.SESXiClient, args *VirtualMachineListOptions) error {
		var (
			vms []*esxi.SVirtualMachine
			err error
		)
		switch {
		case len(args.HostIP) > 0:
			host, err := cli.FindHostByIp(args.HostIP)
			if err != nil {
				return err
			}
			if args.Template {
				vms, err := host.GetTemplateVMs()
				if err != nil {
					return err
				}
				printList(vms, []string{})
				return nil
			}
			vms, err := host.GetIVMs2()
			if err != nil {
				return err
			}
			printList(vms, []string{})
			return nil
		case len(args.Datacenter) > 0:
			var fetcher esxi.VMFetcher
			if len(args.Datastore) > 0 {
				ds, err := getDatastore(cli, args.Datacenter, args.Datastore)
				if err != nil {
					return err
				}
				fetcher = ds
			} else {
				dc, err := cli.FindDatacenterByMoId(args.Datacenter)
				if err != nil {
					return errors.Wrap(err, "FindDatacenterByMoId")
				}
				fetcher = dc
			}
			if args.Template {
				if len(args.FakeRegex) > 0 {
					fmt.Printf("regex: %s\n", args.FakeRegex)
					vms, err = fetcher.FetchFakeTempateVMs(args.FakeRegex)
					if err != nil {
						return errors.Wrap(err, "FetchFakeTempateVMs")
					}
				} else {
					vms, err = fetcher.FetchTemplateVMs()
					if err != nil {
						return errors.Wrap(err, "FetchTemplateVMs")
					}
				}
			} else {
				vms, err = fetcher.FetchNoTemplateVMs()
				if err != nil {
					return errors.Wrap(err, "FetchNoTemplateVMs")
				}
			}
		default:
			return fmt.Errorf("Both Datacenter and HostIP cannot be empty")
		}
		printList(vms, []string{})
		return nil
	})

	type VirtualMachineCloneOptions struct {
		HOSTIP     string `help:"Host IP"`
		TEMPLATEID string `help:"id of template ma"`
		NAME       string `help:"New VM's name'"`
		Uuid       string `help:"Uuid of new VM"`
		CpuNum     int    `help:"Number of CPU"`
		MemSize    int    `help:"Size of Memory(MB)"`
	}
	shellutils.R(&VirtualMachineCloneOptions{}, "vm-clone", "Clone vm", func(cli *esxi.SESXiClient,
		args *VirtualMachineCloneOptions) error {
		host, err := cli.FindHostByIp(args.HOSTIP)
		if err != nil {
			return err
		}
		idss, err := host.GetDataStores()
		if err != nil {
			return err
		}
		if len(idss) == 0 {
			return fmt.Errorf("no datastore")
		}
		temVm, err := host.GetTemplateVMById(args.TEMPLATEID)
		if err != nil {
			return err
		}
		createParams := esxi.SCreateVMParam{
			Name: args.NAME,
			Uuid: args.Uuid,
			Cpu:  args.CpuNum,
			Mem:  args.MemSize,
		}
		vm, err := host.CloneVM(context.Background(), temVm, nil, idss[0].(*esxi.SDatastore), createParams)
		if err != nil {
			return errors.Wrap(err, "SHost.CloneVMFromTemplate")
		}
		printObject(vm)
		return nil
	})

	type VirtualMachineShowOptions struct {
		Datacenter string `help:"Datacenter"`
		HostIP     string `help:"Host IP"`
		VMID       string `help:"VM ID"`
	}
	getVM := func(cli *esxi.SESXiClient, args *VirtualMachineShowOptions) (*esxi.SVirtualMachine, error) {
		var vm *esxi.SVirtualMachine
		switch {
		case len(args.HostIP) > 0:
			host, err := cli.FindHostByIp(args.HostIP)
			if err != nil {
				return nil, errors.Wrap(err, "FindHostByIp")
			}
			ivm, err := host.GetIVMById(args.VMID)
			if err != nil && errors.Cause(err) != errors.ErrNotFound {
				return nil, err
			}
			if err != nil {
				vm, err = host.GetTemplateVMById(args.VMID)
				if err != nil {
					return nil, errors.Wrap(err, "GetTemplateVMById")
				}
			}
			vm = ivm.(*esxi.SVirtualMachine)
		case len(args.Datacenter) > 0:
			dc, err := cli.FindDatacenterByMoId(args.Datacenter)
			if err != nil {
				return nil, errors.Wrap(err, "FindDatacenterByMoId")
			}
			vm, err = dc.FetchVMById(args.VMID)
			if err != nil {
				return nil, errors.Wrap(err, "FetchVMById")
			}
		default:
			return nil, fmt.Errorf("Both Datacenter and HostIP cannot be empty")
		}
		return vm, nil
	}
	shellutils.R(&VirtualMachineShowOptions{}, "vm-show", "Show vm details", func(cli *esxi.SESXiClient, args *VirtualMachineShowOptions) error {
		vm, err := getVM(cli, args)
		if err != nil {
			return err
		}
		printObject(vm)
		return nil
	})

	shellutils.R(&VirtualMachineShowOptions{}, "vm-nics", "Show vm nics details", func(cli *esxi.SESXiClient, args *VirtualMachineShowOptions) error {
		vm, err := getVM(cli, args)
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

	shellutils.R(&VirtualMachineShowOptions{}, "vm-snapshots", "Show vm snapshots details", func(cli *esxi.SESXiClient, args *VirtualMachineShowOptions) error {
		vm, err := getVM(cli, args)
		if err != nil {
			return err
		}
		vmsps, err := vm.GetInstanceSnapshots()
		if err != nil {
			return err
		}
		printList(vmsps, []string{})
		return nil
	})

	type VirtualMachineSnapshotCreateOptions struct {
		VirtualMachineShowOptions
		NAME string `help:"Name of snapshot"`
		Desc string `help:"Description of snapshot"`
	}
	shellutils.R(&VirtualMachineSnapshotCreateOptions{}, "vm-snapshot-create", "Create vm snapshot", func(cli *esxi.SESXiClient, args *VirtualMachineSnapshotCreateOptions) error {
		vm, err := getVM(cli, &args.VirtualMachineShowOptions)
		if err != nil {
			return err
		}
		sp, err := vm.CreateInstanceSnapshot(context.Background(), args.NAME, args.Desc)
		if err != nil {
			return err
		}
		printObject(sp)
		return nil
	})

	type VirtualMachineSnapshotOptions struct {
		VirtualMachineShowOptions
		ID string `help:"ID of snapshot"`
	}
	shellutils.R(&VirtualMachineSnapshotOptions{}, "vm-snapshot-delete", "Delete vm snapshot", func(cli *esxi.SESXiClient, args *VirtualMachineSnapshotOptions) error {
		vm, err := getVM(cli, &args.VirtualMachineShowOptions)
		if err != nil {
			return err
		}
		sp, err := vm.GetInstanceSnapshot(args.ID)
		if err != nil {
			return err
		}
		err = sp.Delete()
		if err != nil {
			return err
		}
		return nil
	})
	shellutils.R(&VirtualMachineSnapshotOptions{}, "vm-snapshot-reset", "Reset vm to snapshot", func(cli *esxi.SESXiClient, args *VirtualMachineSnapshotOptions) error {
		vm, err := getVM(cli, &args.VirtualMachineShowOptions)
		if err != nil {
			return err
		}
		err = vm.ResetToInstanceSnapshot(context.Background(), args.ID)
		if err != nil {
			return err
		}
		return nil
	})

	shellutils.R(&VirtualMachineShowOptions{}, "vm-disks", "Show vm disks details", func(cli *esxi.SESXiClient, args *VirtualMachineShowOptions) error {
		vm, err := getVM(cli, args)
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

	type VirtualMachineDiskResizeOptions struct {
		VirtualMachineShowOptions
		DISKIDX int   `help:"disk index"`
		SIZEGB  int64 `help:"new size of disk"`
	}
	shellutils.R(&VirtualMachineDiskResizeOptions{}, "vm-disk-resize", "Resize a vm disk", func(cli *esxi.SESXiClient, args *VirtualMachineDiskResizeOptions) error {
		vm, err := getVM(cli, &args.VirtualMachineShowOptions)
		if err != nil {
			return err
		}
		vmdisks, err := vm.GetIDisks()
		if err != nil {
			return err
		}
		if args.DISKIDX < 0 || args.DISKIDX >= len(vmdisks) {
			return fmt.Errorf("Out of index: %d", args.DISKIDX)
		}
		disk := vmdisks[args.DISKIDX]
		ctx := context.Background()
		return disk.Resize(ctx, args.SIZEGB*1024)
	})

	shellutils.R(&VirtualMachineShowOptions{}, "vm-vnc", "Show vm VNC details", func(cli *esxi.SESXiClient, args *VirtualMachineShowOptions) error {
		vm, err := getVM(cli, args)
		if err != nil {
			return err
		}
		info, err := vm.GetVNCInfo(nil)
		if err != nil {
			return err
		}
		printObject(info)
		return nil
	})

	shellutils.R(&VirtualMachineShowOptions{}, "vm-vmrc", "Show vm VMRC connection", func(cli *esxi.SESXiClient, args *VirtualMachineShowOptions) error {
		vm, err := getVM(cli, args)
		if err != nil {
			return err
		}
		info, err := vm.GetVmrcInfo()
		if err != nil {
			return err
		}
		printObject(info)
		return nil
	})

	shellutils.R(&VirtualMachineShowOptions{}, "vm-webmks", "Show vm webmks connection", func(cli *esxi.SESXiClient, args *VirtualMachineShowOptions) error {
		vm, err := getVM(cli, args)
		if err != nil {
			return err
		}
		info, err := vm.GetWebmksInfo()
		if err != nil {
			return err
		}
		printObject(info)
		return nil
	})

	shellutils.R(&VirtualMachineShowOptions{}, "vm-file-status", "Show vm files details", func(cli *esxi.SESXiClient, args *VirtualMachineShowOptions) error {
		vm, err := getVM(cli, args)
		if err != nil {
			return err
		}
		err = vm.CheckFileInfo(context.Background())
		if err != nil {
			return err
		}
		return nil
	})

}
