// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
	handleResult := func(opt o.WebConsoleOptions, obj jsonutils.JSONObject) error {
		if opt.WebconsoleUrl == "" {
			printObject(obj)
			return nil
		}
		u, err := url.Parse(opt.WebconsoleUrl)
		if err != nil {
			return err
		}
		connParams, err := obj.GetString("connect_params")
		if err != nil {
			return err
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
		ret, err := modules.WebConsole.DoSshConnect(s, args.IP, nil)
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
