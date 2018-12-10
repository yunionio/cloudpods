package shell

import (
	"fmt"

	"yunion.io/x/onecloud/pkg/baremetal/utils/ipmitool"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	shellutils.R(&EmptyOptions{}, "get-chassis-power-status", "Get chassis power status", func(cli ipmitool.IPMIExecutor, _ *EmptyOptions) error {
		status, err := ipmitool.GetChassisPowerStatus(cli)
		if err != nil {
			return err
		}
		fmt.Println(status)
		return nil
	})
}
