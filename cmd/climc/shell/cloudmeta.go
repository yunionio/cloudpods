package shell

import (
	"yunion.io/x/onecloud/pkg/mcclient"
)

func init() {
	type CloudmetaOptions struct {
		PROVIDER_ID string `help:"provider_id"`
		REGION_ID   string `help:"region_id"`
		ZONE_ID     string `help:"zone_id"`
	}
	R(&CloudmetaOptions{}, "instance-type-list", "query backend service for its version", func(s *mcclient.ClientSession, args *CloudmetaOptions) error {
		return nil
	})
}
