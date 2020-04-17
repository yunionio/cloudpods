package shell

import (
	"fmt"

	"yunion.io/x/onecloud/pkg/multicloud/huawei"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type OwnerShowOptions struct {
	}
	shellutils.R(&OwnerShowOptions{}, "owner-show", "Get aksk owner id", func(cli *huawei.SRegion, args *OwnerShowOptions) error {
		result, err := cli.GetClient().GetOwnerId()
		if err != nil {
			return err
		}
		fmt.Println(result)
		return nil
	})
}
