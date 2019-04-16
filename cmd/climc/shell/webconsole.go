package shell

import (
	"fmt"
	"net/url"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	o "yunion.io/x/onecloud/pkg/mcclient/options"
	"yunion.io/x/onecloud/pkg/webconsole/command"
	"yunion.io/x/onecloud/pkg/webconsole/session"
)

const (
	DefaultWebconsoleServer = "https://console.yunion.cn/web-console"
)

func init() {
	handleResult := func(opt o.WebConsoleOptions, obj jsonutils.JSONObject) error {
		if opt.WebconsoleUrl == "" {
			opt.WebconsoleUrl = DefaultWebconsoleServer
		}
		u, err := url.Parse(opt.WebconsoleUrl)
		if err != nil {
			return err
		}
		connParams, err := obj.GetString("connect_params")
		if err != nil {
			return err
		}
		var query url.Values
		connUrl, err := url.ParseRequestURI(connParams)
		if err == nil {
			query = connUrl.Query()
		} else {
			query, err = url.ParseQuery(connParams)
			if err != nil {
				return err
			}
		}
		protocol := query.Get("protocol")
		if !utils.IsInStringArray(protocol, []string{
			command.PROTOCOL_TTY, session.VNC,
			session.SPICE, session.WMKS,
		}) {
			fmt.Println(connParams)
			return nil
		}

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
		handleResult(args.WebConsoleOptions, ret)
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
		handleResult(args.WebConsoleOptions, ret)
		return nil
	})

	R(&o.WebConsoleBaremetalOptions{}, "webconsole-baremetal", "Connect baremetal host webconsole", func(s *mcclient.ClientSession, args *o.WebConsoleBaremetalOptions) error {
		ret, err := modules.WebConsole.DoBaremetalConnect(s, args.ID, nil)
		if err != nil {
			return err
		}
		handleResult(args.WebConsoleOptions, ret)
		return nil
	})

	R(&o.WebConsoleSshOptions{}, "webconsole-ssh", "Connect ssh webconsole", func(s *mcclient.ClientSession, args *o.WebConsoleSshOptions) error {
		params, err := args.Params()
		if err != nil {
			return err
		}
		ret, err := modules.WebConsole.DoSshConnect(s, args.IP, params)
		if err != nil {
			return err
		}
		handleResult(args.WebConsoleOptions, ret)
		return nil
	})

	R(&o.WebConsoleServerOptions{}, "webconsole-server", "Connect server remote graphic console", func(s *mcclient.ClientSession, args *o.WebConsoleServerOptions) error {
		ret, err := modules.WebConsole.DoServerConnect(s, args.ID, nil)
		if err != nil {
			return err
		}
		handleResult(args.WebConsoleOptions, ret)
		return nil
	})
}
