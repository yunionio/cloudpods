package shell

import (
	"fmt"

	"yunion.io/x/onecloud/pkg/util/azure"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type InstanceListOptions struct {
		Limit  int `help:"page size"`
		Offset int `help:"page offset"`
	}
	shellutils.R(&InstanceListOptions{}, "instance-list", "List intances", func(cli *azure.SRegion, args *InstanceListOptions) error {
		if instances, err := cli.GetInstances(); err != nil {
			return err
		} else {
			printList(instances, len(instances), args.Offset, args.Limit, []string{})
			return nil
		}
	})

	type InstanceCrateOptions struct {
		NAME      string `help:"name of instance"`
		IMAGE     string `help:"image ID"`
		CPU       int    `help:"CPU count"`
		MEMORYGB  int    `help:"MemoryGB"`
		Disk      []int  `help:"Data disk sizes int GB"`
		STORAGE   string `help:"Storage type"`
		NETWORK   string `help:"Network ID"`
		PASSWD    string `help:"password"`
		PublicKey string `help:"PublicKey"`
	}
	shellutils.R(&InstanceCrateOptions{}, "instance-create", "Create a instance", func(cli *azure.SRegion, args *InstanceCrateOptions) error {
		instance, e := cli.CreateInstanceSimple(args.NAME, args.IMAGE, args.CPU, args.MEMORYGB, args.STORAGE, args.Disk, args.NETWORK, args.PASSWD, args.PublicKey)
		if e != nil {
			return e
		}
		printObject(instance)
		return nil
	})

	type InstanceOptions struct {
		ID string `help:"Instance ID"`
	}
	shellutils.R(&InstanceOptions{}, "instance-show", "Show intance detail", func(cli *azure.SRegion, args *InstanceOptions) error {
		if instance, err := cli.GetInstance(args.ID); err != nil {
			return err
		} else {
			printObject(instance)
			return nil
		}
	})

	shellutils.R(&InstanceOptions{}, "instance-delete", "Delete intance", func(cli *azure.SRegion, args *InstanceOptions) error {
		return cli.DeleteVM(args.ID)
	})

	shellutils.R(&InstanceOptions{}, "instance-deallocate", "Deallocate intance", func(cli *azure.SRegion, args *InstanceOptions) error {
		return cli.DeallocateVM(args.ID)
	})

	type InstanceRebuildOptions struct {
		ID        string `help:"Instance ID"`
		Image     string `help:"Image ID"`
		Password  string `help:"pasword"`
		PublicKey string `help:"Public Key"`
		Size      int32  `help:"system disk size in GB"`
	}
	shellutils.R(&InstanceRebuildOptions{}, "instance-rebuild-root", "Reinstall virtual server system image", func(cli *azure.SRegion, args *InstanceRebuildOptions) error {
		if diskID, err := cli.ReplaceSystemDisk(args.ID, args.Image, args.Password, args.PublicKey, args.Size); err != nil {
			return err
		} else {
			fmt.Printf("New diskID is %s", diskID)
			return nil
		}
	})

	type InstanceDiskOptions struct {
		ID   string `help:"Instance ID"`
		DISK string `help:"Disk ID"`
	}
	shellutils.R(&InstanceDiskOptions{}, "instance-attach-disk", "Attach a disk to intance", func(cli *azure.SRegion, args *InstanceDiskOptions) error {
		return cli.AttachDisk(args.ID, args.DISK)
	})

	shellutils.R(&InstanceDiskOptions{}, "instance-detach-disk", "Attach a disk to intance", func(cli *azure.SRegion, args *InstanceDiskOptions) error {
		return cli.DetachDisk(args.ID, args.DISK)
	})

	type InstanceConfigOptions struct {
		ID     string `help:"Instance ID"`
		NCPU   int    `help:"Number of cpu core"`
		MEMERY int    `helo:"Instance memery in mb"`
	}

	shellutils.R(&InstanceConfigOptions{}, "instance-change-conf", "Attach a disk to intance", func(cli *azure.SRegion, args *InstanceConfigOptions) error {
		return cli.ChangeVMConfig(args.ID, args.NCPU, args.MEMERY)
	})

	type InstanceDeployOptions struct {
		ID        string `help:"Instance ID"`
		Password  string `help:"Password for instance"`
		PublicKey string `helo:"Deploy ssh_key for instance"`
	}

	shellutils.R(&InstanceDeployOptions{}, "instance-reset-password", "Reset intance password", func(cli *azure.SRegion, args *InstanceDeployOptions) error {
		return cli.DeployVM(args.ID, "", args.Password, args.PublicKey, false, "")
	})
}
