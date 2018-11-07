package shell

import (
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

func init() {
	type CapabilitiesOptions struct {
	}
	R(&CapabilitiesOptions{}, "capabilities", "Show backend capabilities", func(s *mcclient.ClientSession, args *CapabilitiesOptions) error {
		result, err := modules.Capabilities.List(s, nil)
		if err != nil {
			return err
		}
		printObject(result.Data[0])
		return nil
	})
}
