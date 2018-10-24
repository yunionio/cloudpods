package shell

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

func init() {
	type SWebsocketNotifyOptions struct {
		Cmp bool `help:"websocket all the compute nodes automatically"`
	}

	R(&SWebsocketNotifyOptions{}, "websocket-notify", "post a Websocket msg", func(s *mcclient.ClientSession, args *SWebsocketNotifyOptions) error {
		params := jsonutils.NewDict()
		if args.Cmp {
			params.Add(jsonutils.JSONTrue, "cmp")
		}

		_, err := modules.Websockets.Create(s, params)
		if err != nil {
			return err
		}

		return nil
	})
}
