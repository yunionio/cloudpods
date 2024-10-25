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

package compute

import (
	"encoding/base64"
	"fmt"
	"net/url"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/utils"

	webconsole_api "yunion.io/x/onecloud/pkg/apis/webconsole"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules/identity"
	"yunion.io/x/onecloud/pkg/mcclient/modules/webconsole"
	o "yunion.io/x/onecloud/pkg/mcclient/options"
	"yunion.io/x/onecloud/pkg/webconsole/command"
)

func init() {
	handleResult := func(s *mcclient.ClientSession, opt o.WebConsoleOptions, obj jsonutils.JSONObject) error {
		if obj.Contains("access_url") {
			accessUrl, _ := obj.GetString("access_url")
			fmt.Println("AccessURL:", accessUrl)
			return nil
		}
		if opt.WebconsoleUrl == "" {
			resp, err := identity.ServicesV3.GetSpecific(s, "common", "config", nil)
			if err != nil {
				return err
			}
			apiServer, _ := resp.GetString("config", "default", "api_server")
			if len(apiServer) > 0 {
				opt.WebconsoleUrl = fmt.Sprintf("%s/web-console", apiServer)
			}
		}
		u, err := url.Parse(opt.WebconsoleUrl)
		if err != nil {
			return err
		}

		resp := &webconsole_api.ServerRemoteConsoleResponse{}
		if err := obj.Unmarshal(resp); err != nil {
			return err
		}
		connectParams := resp.GetConnectParams()
		protocol, err := resp.GetConnectProtocol()
		if err != nil {
			return err
		}
		if !utils.IsInStringArray(protocol, []string{
			command.PROTOCOL_TTY, webconsole_api.VNC,
			webconsole_api.SPICE, webconsole_api.WMKS, webconsole_api.WS,
		}) {
			fmt.Println(connectParams)
			return nil
		}
		if protocol == webconsole_api.WS {
			u, err = url.Parse(fmt.Sprintf("%s/ws", opt.WebconsoleUrl))
			if err != nil {
				return err
			}
		}

		newQuery := url.Values{}
		newQuery.Set("data", base64.StdEncoding.EncodeToString([]byte(connectParams)))
		u.RawQuery = newQuery.Encode()
		fmt.Println(u.String())
		return nil
	}

	R(&o.PodShellOptions{}, "webconsole-k8s-pod", "Show TTY console of given pod", func(s *mcclient.ClientSession, args *o.PodShellOptions) error {
		params, err := args.Params()
		if err != nil {
			return err
		}
		ret, err := webconsole.WebConsole.DoK8sShellConnect(s, args.NAME, params)
		if err != nil {
			return err
		}
		handleResult(s, args.WebConsoleOptions, ret)
		return nil
	})

	R(&o.PodLogOptoins{}, "webconsole-k8s-pod-log", "Get logs of given pod", func(s *mcclient.ClientSession, args *o.PodLogOptoins) error {
		params, err := args.Params()
		if err != nil {
			return err
		}
		ret, err := webconsole.WebConsole.DoK8sLogConnect(s, args.NAME, params)
		if err != nil {
			return err
		}
		handleResult(s, args.WebConsoleOptions, ret)
		return nil
	})

	R(&o.WebConsoleBaremetalOptions{}, "webconsole-baremetal", "Connect baremetal host webconsole", func(s *mcclient.ClientSession, args *o.WebConsoleBaremetalOptions) error {
		ret, err := webconsole.WebConsole.DoBaremetalConnect(s, args.ID, nil)
		if err != nil {
			return err
		}
		handleResult(s, args.WebConsoleOptions, ret)
		return nil
	})

	R(&o.WebConsoleSshOptions{}, "webconsole-ssh", "Connect ssh webconsole", func(s *mcclient.ClientSession, args *o.WebConsoleSshOptions) error {
		params, err := args.Params()
		if err != nil {
			return err
		}
		ret, err := webconsole.WebConsole.DoSshConnect(s, args.ID, params)
		if err != nil {
			return err
		}
		handleResult(s, args.WebConsoleOptions, ret)
		return nil
	})

	R(&o.WebConsoleServerOptions{}, "webconsole-server", "Connect server remote graphic console", func(s *mcclient.ClientSession, args *o.WebConsoleServerOptions) error {
		ret, err := webconsole.WebConsole.DoServerConnect(s, args.ID, nil)
		if err != nil {
			return err
		}
		handleResult(s, args.WebConsoleOptions, ret)
		return nil
	})

	R(&o.WebConsoleServerRdpOptions{}, "webconsole-server-rdp", "Connect server remote graphic console by rdp", func(s *mcclient.ClientSession, args *o.WebConsoleServerRdpOptions) error {
		ret, err := webconsole.WebConsole.DoServerRDPConnect(s, args.ID, jsonutils.Marshal(map[string]interface{}{"webconsole": args}))
		if err != nil {
			return err
		}
		handleResult(s, args.WebConsoleOptions, ret)
		return nil
	})

	R(&o.WebConsoleContainerExecOptions{}, "webconsole-container-exec", "Container exec", func(s *mcclient.ClientSession, args *o.WebConsoleContainerExecOptions) error {
		ret, err := webconsole.WebConsole.DoContainerExec(s, jsonutils.Marshal(map[string]interface{}{"container_id": args.ID}))
		if err != nil {
			return err
		}
		return handleResult(s, args.WebConsoleOptions, ret)
	})
}
