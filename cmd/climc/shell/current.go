package shell

import (
	"yunion.io/x/onecloud/pkg/mcclient"
)

func init() {
	type CurrentUserOptions struct {
	}
	R(&CurrentUserOptions{}, "session-show", "show information of current account", func(s *mcclient.ClientSession, args *CurrentUserOptions) error {
		printObject(s.ToJson())
		return nil
	})
}
