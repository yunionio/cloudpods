package shell

import (
	"fmt"
	"net/url"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	o "yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	handleResult := func(opt o.WebConsoleFrontendOptions, obj jsonutils.JSONObject) error {
		if opt.WebUrl == "" {
			printObject(obj)
			return nil
		}
		u, err := url.Parse(opt.WebUrl)
		if err != nil {
			return err
		}
		connParams, err := obj.GetString("connect_params")
		if err != nil {
			return err
		}
		u.Path = "index.html"
		u.RawQuery = connParams
		fmt.Println(u.String())
		return nil
	}

	R(&o.PodShellOptions{}, "webconsole-k8s-pod", "Show TTY console of given pod", func(s *mcclient.ClientSession, args *o.PodShellOptions) error {
		params, err := args.Params()
		if err != nil {
			return err
		}
		ret, err := modules.WebConsole.DoK8sShellConnect(s, args.NAME, params)
		if err != nil {
			return err
		}
		handleResult(args.WebConsoleFrontendOptions, ret)
		return nil
	})

	R(&o.PodLogOptoins{}, "webconsole-k8s-pod-log", "Get logs of given pod", func(s *mcclient.ClientSession, args *o.PodLogOptoins) error {
		params, err := args.Params()
		if err != nil {
			return err
		}
		ret, err := modules.WebConsole.DoK8sLogConnect(s, args.NAME, params)
		if err != nil {
			return err
		}
		handleResult(args.WebConsoleFrontendOptions, ret)
		return nil
	})
}
