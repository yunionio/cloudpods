package shell

import (
	"yunion.io/x/onecloud/pkg/util/shellutils"
	"yunion.io/x/onecloud/pkg/util/ucloud"
)

func init() {
	type InstanceListOptions struct {
	}
	shellutils.R(&InstanceListOptions{}, "instance-list", "List intances", func(cli *ucloud.SRegion, args *InstanceListOptions) error {
		instances, e := cli.GetInstances("", "")
		if e != nil {
			return e
		}
		printList(instances, 0, 0, 0, nil)
		return nil
	})

	type InstanceDiskOperationOptions struct {
		ZONE string `help:"zone ID"`
		ID   string `help:"instance ID"`
		DISK string `help:"disk ID"`
	}

	type InstanceDiskAttachOptions struct {
		ID     string `help:"instance ID"`
		DISK   string `help:"disk ID"`
		DEVICE string `help:"disk device name. eg. /dev/sdb"`
	}

	shellutils.R(&InstanceDiskAttachOptions{}, "instance-attach-disk", "Attach a disk to instance", func(cli *ucloud.SRegion, args *InstanceDiskAttachOptions) error {
		err := cli.AttachDisk(args.ID, args.DISK, args.DEVICE)
		if err != nil {
			return err
		}
		return nil
	})

	shellutils.R(&InstanceDiskOperationOptions{}, "instance-detach-disk", "Detach a disk to instance", func(cli *ucloud.SRegion, args *InstanceDiskOperationOptions) error {
		err := cli.DetachDisk(args.ZONE, args.ID, args.DISK)
		if err != nil {
			return err
		}
		return nil
	})

	type InstanceCrateOptions struct {
		NAME     string `help:"name of instance"`
		IMAGE    string `help:"image ID"`
		HOSTTYPE string `help:"host type" choices:"N1|N2|N3|C1||I2|G1|G2|G3|I1"`
		PASSWORD string `help:"password"`
		VPC      string `help:"VPC ID"`
		NETWORK  string `help:"Network ID"`
		SECGROUP string `help:"secgroup ID"`
		ZONE     string `help:"zone ID"`
		CPU      int    `help:"CPU count"`
		MEMORYGB int    `help:"MemoryGB"`
		STORAGE  string `help:"Storage type" choices:"CLOUD_NORMAL|CLOUD_SSD"`
		DISKSIZE int    `help:"root disk size GB"`
	}
	shellutils.R(&InstanceCrateOptions{}, "instance-create", "Create a instance", func(cli *ucloud.SRegion, args *InstanceCrateOptions) error {
		disk := ucloud.SDisk{DiskType: args.STORAGE, SizeGB: args.DISKSIZE}
		instance, e := cli.CreateInstance(args.NAME, args.IMAGE, args.HOSTTYPE, args.PASSWORD, args.VPC, args.NETWORK, args.SECGROUP, args.ZONE, "", "", args.CPU, args.MEMORYGB*1024, []ucloud.SDisk{disk}, nil)
		if e != nil {
			return e
		}
		printObject(instance)
		return nil
	})

	type InstanceOperationOptions struct {
		ID string `help:"instance ID"`
	}
	shellutils.R(&InstanceOperationOptions{}, "instance-start", "Start a instance", func(cli *ucloud.SRegion, args *InstanceOperationOptions) error {
		err := cli.StartVM(args.ID)
		if err != nil {
			return err
		}
		return nil
	})

	type InstanceStopOptions struct {
		ID string `help:"instance ID"`
	}
	shellutils.R(&InstanceStopOptions{}, "instance-stop", "Stop a instance", func(cli *ucloud.SRegion, args *InstanceStopOptions) error {
		err := cli.StopVM(args.ID)
		if err != nil {
			return err
		}
		return nil
	})
	shellutils.R(&InstanceOperationOptions{}, "instance-delete", "Delete a instance", func(cli *ucloud.SRegion, args *InstanceOperationOptions) error {
		err := cli.DeleteVM(args.ID)
		if err != nil {
			return err
		}
		return nil
	})

	/*
		server-change-config 更改系统配置
		server-reset
	*/
	type InstanceDeployOptions struct {
		ID       string `help:"instance ID"`
		Password string `help:"new password"`
	}

	shellutils.R(&InstanceDeployOptions{}, "instance-deploy", "Deploy keypair/password to a stopped virtual server", func(cli *ucloud.SRegion, args *InstanceDeployOptions) error {
		err := cli.ResetVMPasswd(args.ID, args.Password)
		if err != nil {
			return err
		}
		return nil
	})

	type InstanceRebuildRootOptions struct {
		ID       string `help:"instance ID"`
		Image    string `help:"Image ID"`
		Password string `help:"admin password"`
	}

	shellutils.R(&InstanceRebuildRootOptions{}, "instance-rebuild-root", "Reinstall virtual server system image", func(cli *ucloud.SRegion, args *InstanceRebuildRootOptions) error {
		err := cli.RebuildRoot(args.ID, args.Image, args.Password)
		if err != nil {
			return err
		}

		return nil
	})

	type InstanceChangeConfigOptions struct {
		ID   string `help:"instance ID"`
		NCPU int    `help:"cpu"`
		MEM  int    `help:"memory MB"`
	}

	shellutils.R(&InstanceChangeConfigOptions{}, "instance-change-config", "Deploy keypair/password to a stopped virtual server", func(cli *ucloud.SRegion, args *InstanceChangeConfigOptions) error {
		err := cli.ResizeVM(args.ID, args.NCPU, args.MEM)
		if err != nil {
			return err
		}
		return nil
	})
}
