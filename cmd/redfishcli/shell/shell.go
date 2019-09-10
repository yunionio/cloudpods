package shell

import (
	"context"
	"fmt"

	"yunion.io/x/onecloud/pkg/util/redfish"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type ExploreInput struct {
		Element []string `help:"explore path" positional:"true" optional:"true"`
	}
	shellutils.R(&ExploreInput{}, "explore", "explore redfish API", func(cli redfish.IRedfishDriver, args *ExploreInput) error {
		path, resp, err := cli.GetResource(context.Background(), args.Element...)
		if err != nil {
			return err
		}
		fmt.Println(path)
		fmt.Println(resp.PrettyString())
		return nil
	})
}
