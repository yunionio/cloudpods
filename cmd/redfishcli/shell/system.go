package shell

import (
	"context"
	"fmt"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/util/redfish"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {

	type SystemGetOptions struct {
	}
	shellutils.R(&SystemGetOptions{}, "system-get", "Get details of a system", func(cli redfish.IRedfishDriver, args *SystemGetOptions) error {
		path, sysInfo, err := cli.GetSystemInfo(context.Background())
		if err != nil {
			return err
		}
		fmt.Println(path)
		fmt.Println(jsonutils.Marshal(sysInfo).PrettyString())
		return nil
	})

	shellutils.R(&SystemGetOptions{}, "bios-get", "Get details of a system Bios", func(cli redfish.IRedfishDriver, args *SystemGetOptions) error {
		bios, err := cli.GetBiosInfo(context.Background())
		if err != nil {
			return err
		}
		fmt.Println(jsonutils.Marshal(bios).PrettyString())
		return nil
	})

	type SetNextBootOptions struct {
		DEV string `help:"next boot device"`
	}
	shellutils.R(&SetNextBootOptions{}, "set-next-boot-dev", "Set next boot device", func(cli redfish.IRedfishDriver, args *SetNextBootOptions) error {
		var err error
		if args.DEV == "vcd" {
			err = cli.SetNextBootVirtualCdrom(context.Background())
		} else {
			err = cli.SetNextBootDev(context.Background(), args.DEV)
		}
		if err != nil {
			return err
		}
		fmt.Println("Success!")
		return nil
	})

	type SystemResetOptions struct {
		ACTION string `help:"reset action"`
	}
	shellutils.R(&SystemResetOptions{}, "system-reset", "Reset system", func(cli redfish.IRedfishDriver, args *SystemResetOptions) error {
		err := cli.Reset(context.Background(), args.ACTION)
		if err != nil {
			return err
		}
		fmt.Println("Success!")
		return nil
	})

}
