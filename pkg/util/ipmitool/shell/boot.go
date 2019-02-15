package shell

import (
	"fmt"

	"yunion.io/x/onecloud/pkg/baremetal/utils/ipmitool"
	"yunion.io/x/onecloud/pkg/util/printutils"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	shellutils.R(&EmptyOptions{}, "get-boot-flags", "Get boot flags info", func(client ipmitool.IPMIExecutor, _ *EmptyOptions) error {
		info, err := ipmitool.GetBootFlags(client)
		if err != nil {
			return err
		}
		printutils.PrintInterfaceObject(info)
		return nil
	})

	shellutils.R(&EmptyOptions{}, "do-reboot", "Do reboot", func(client ipmitool.IPMIExecutor, _ *EmptyOptions) error {
		return ipmitool.DoReboot(client)
	})

	shellutils.R(&BootFlagOptions{}, "set-boot-flag", "Set bootflag, do reboot to make it work", func(cli ipmitool.IPMIExecutor, args *BootFlagOptions) error {
		switch args.FLAG {
		case "pxe":
			return ipmitool.SetRebootToPXE(cli)
		case "disk":
			return ipmitool.SetRebootToDisk(cli)
		case "bios":
			return ipmitool.SetRebootToBIOS(cli)
		default:
			return fmt.Errorf("Invalid boot flag: %s", args.FLAG)
		}
	})

	shellutils.R(&ShutdownOptions{}, "do-shutdown", "Do shutdown", func(client ipmitool.IPMIExecutor, args *ShutdownOptions) error {
		if args.Soft {
			return ipmitool.DoSoftShutdown(client)
		}
		return ipmitool.DoHardShutdown(client)
	})

	shellutils.R(&EmptyOptions{}, "do-power-on", "Do power on", func(client ipmitool.IPMIExecutor, _ *EmptyOptions) error {
		return ipmitool.DoPowerOn(client)
	})

	shellutils.R(&EmptyOptions{}, "do-power-reset", "Do power on", func(client ipmitool.IPMIExecutor, _ *EmptyOptions) error {
		return ipmitool.DoPowerReset(client)
	})
}
