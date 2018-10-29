package shell

import (
	"fmt"
	"io/ioutil"

	"yunion.io/x/onecloud/pkg/util/aws"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type InstanceListOptions struct {
		Id     []string `help:"IDs of instances to show"`
		Zone   string   `help:"Zone ID"`
		Limit  int      `help:"page size"`
		Offset int      `help:"page offset"`
	}
	shellutils.R(&InstanceListOptions{}, "instance-list", "List intances", func(cli *aws.SRegion, args *InstanceListOptions) error {
		instances, total, e := cli.GetInstances(args.Zone, args.Id, args.Offset, args.Limit)
		if e != nil {
			return e
		}
		printList(instances, total, args.Offset, args.Limit, []string{})
		return nil
	})

	type InstanceCrateOptions struct {
		NAME      string `help:"name of instance"`
		IMAGE     string `help:"image ID"`
		CPU       int    `help:"CPU count"`
		MEMORYGB  int    `help:"MemoryGB"`
		Disk      []int  `help:"Data disk sizes int GB"`
		STORAGE   string `help:"Storage type" choices:"gp2|io1|st1|sc1|standard"`
		NETWORK   string `help:"Network ID"`
		PUBLICKEY string `help:"PublicKey file path"`
	}
	shellutils.R(&InstanceCrateOptions{}, "instance-create", "Create a instance", func(cli *aws.SRegion, args *InstanceCrateOptions) error {
		content, err := ioutil.ReadFile(args.PUBLICKEY)
		if err != nil {
			return err
		}
		instance, e := cli.CreateInstanceSimple(args.NAME, args.IMAGE, args.CPU, args.MEMORYGB, args.STORAGE, args.Disk, args.NETWORK, string(content))
		if e != nil {
			return e
		}
		printObject(instance)
		return nil
	})

	type InstanceDiskOperationOptions struct {
		ID   string `help:"instance ID"`
		DISK string `help:"disk ID"`
	}

	type InstanceDiskAttachOptions struct {
		ID   string `help:"instance ID"`
		DISK string `help:"disk ID"`
		DEVICE string `help:"disk device name. eg. /dev/sdb"`
	}

	shellutils.R(&InstanceDiskAttachOptions{}, "instance-attach-disk", "Attach a disk to instance", func(cli *aws.SRegion, args *InstanceDiskOperationOptions) error {
		err := cli.AttachDisk(args.ID, args.DISK, args.DEVICE)
		if err != nil {
			return err
		}
		return nil
	})

	shellutils.R(&InstanceDiskOperationOptions{}, "instance-detach-disk", "Detach a disk to instance", func(cli *aws.SRegion, args *InstanceDiskOperationOptions) error {
		err := cli.DetachDisk(args.ID, args.DISK)
		if err != nil {
			return err
		}
		return nil
	})

	type InstanceOperationOptions struct {
		ID string `help:"instance ID"`
	}
	shellutils.R(&InstanceOperationOptions{}, "instance-start", "Start a instance", func(cli *aws.SRegion, args *InstanceOperationOptions) error {
		err := cli.StartVM(args.ID)
		if err != nil {
			return err
		}
		return nil
	})

	type InstanceStopOptions struct {
		ID    string `help:"instance ID"`
		Force bool   `help:"Force stop instance"`
	}
	shellutils.R(&InstanceStopOptions{}, "instance-stop", "Stop a instance", func(cli *aws.SRegion, args *InstanceStopOptions) error {
		err := cli.StopVM(args.ID, args.Force)
		if err != nil {
			return err
		}
		return nil
	})
	shellutils.R(&InstanceOperationOptions{}, "instance-delete", "Delete a instance", func(cli *aws.SRegion, args *InstanceOperationOptions) error {
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
		ID            string `help:"instance ID"`
		Name          string `help:"new instance name"`
		Hostname      string `help:"new hostname"`
		Keypair       string `help:"Keypair Name"`
		DeleteKeypair bool   `help:"Remove SSH keypair"`
		Password      string `help:"new password"`
		// ResetPassword bool   `help:"Force reset password"`
		Description string `help:"new instances description"`
	}

	shellutils.R(&InstanceDeployOptions{}, "instance-deploy", "Deploy keypair/password to a stopped virtual server", func(cli *aws.SRegion, args *InstanceDeployOptions) error {
		err := cli.DeployVM(args.ID, args.Name, args.Password, args.Keypair, args.DeleteKeypair, args.Description)
		if err != nil {
			return err
		}
		return nil
	})

	type InstanceRebuildRootOptions struct {
		ID       string `help:"instance ID"`
		Image    string `help:"Image ID"`
		Password string `help:"pasword"`
		Keypair  string `help:"keypair name"`
		Size     int    `help:"system disk size in GB"`
	}

	shellutils.R(&InstanceRebuildRootOptions{}, "instance-rebuild-root", "Reinstall virtual server system image", func(cli *aws.SRegion, args *InstanceRebuildRootOptions) error {
		diskID, err := cli.ReplaceSystemDisk(args.ID, args.Image, args.Password, args.Keypair, args.Size)
		if err != nil {
			return err
		}
		fmt.Printf("New diskID is %s", diskID)
		return nil
	})

	type InstanceChangeConfigOptions struct {
		ID   string `help:"instance ID"`
		Ncpu int    `help:"number of CPU"`
		Vmem int    `help:"MiB of memory"`
		Disk []int  `help:"Data disk sizes int GB"`
	}

	shellutils.R(&InstanceChangeConfigOptions{}, "instance-change-config", "Deploy keypair/password to a stopped virtual server", func(cli *aws.SRegion, args *InstanceChangeConfigOptions) error {
		instance, e := cli.GetInstance(args.ID)
		if e != nil {
			return e
		}

		// todo : add create disks
		err := cli.ChangeVMConfig(instance.ZoneId, args.ID, args.Ncpu, args.Vmem, nil)
		if err != nil {
			return err
		}
		return nil
	})
}
