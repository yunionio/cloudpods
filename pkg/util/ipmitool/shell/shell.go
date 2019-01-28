package shell

import (
	"yunion.io/x/onecloud/pkg/baremetal/utils/ipmitool"
	"yunion.io/x/onecloud/pkg/util/printutils"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

type EmptyOptions struct{}

type BootFlagOptions struct {
	FLAG string `help:"Boot flag" choices:"pxe|disk|bios"`
}

type ShutdownOptions struct {
	Soft bool `help:"Do soft shutdown"`
}

func init() {
	shellutils.R(&EmptyOptions{}, "get-sysinfo", "Get system info", func(client ipmitool.IPMIExecutor, _ *EmptyOptions) error {
		info, err := ipmitool.GetSysInfo(client)
		if err != nil {
			return err
		}
		printutils.PrintInterfaceObject(info)
		return nil
	})
}
