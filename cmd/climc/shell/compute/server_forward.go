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
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/cmd/climc/shell"
	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	cmd := shell.NewResourceCmd(&modules.Servers)
	cmd.Perform("open-forward", &options.ServerOpenForwardOptions{})
	cmd.Perform("close-forward", &options.ServerCloseForwardOptions{})
	cmd.Perform("list-forward", &options.ServerListForwardOptions{})
}

type forwardInfo struct {
	ProxyAddr string `json:"proxy_addr"`
	ProxyPort int    `json:"proxy_port"`
}

func dump(input jsonutils.JSONObject) (*forwardInfo, error) {
	ret := new(forwardInfo)
	if err := input.Unmarshal(ret); err != nil {
		return nil, errors.Wrap(err, "JSONObject unmarshal failed")
	}

	if ret.ProxyAddr == "" {
		return nil, errors.Errorf("proxy_addr is empty")
	}

	if ret.ProxyPort <= 0 {
		return nil, errors.Errorf("invalid proxy_port %d", ret.ProxyPort)
	}

	return ret, nil
}

func openForward(session *mcclient.ClientSession, srvid string) (*forwardInfo, error) {
	opt := &options.ServerOpenForwardOptions{
		ServerIdOptions: options.ServerIdOptions{
			ID: srvid,
		},
		Proto: "tcp",
		Port:  22,
	}

	params, err := opt.Params()
	if err != nil {
		return nil, errors.Wrap(err, "get open forward params")
	}
	jsonItem, err := modules.Servers.PerformAction(session, opt.ID, "open-forward", params)
	if err != nil {
		return nil, err
	}

	return dump(jsonItem)
}

func closeForward(session *mcclient.ClientSession, srvid string, fItem *forwardInfo) (jsonutils.JSONObject, error) {
	opt := &options.ServerCloseForwardOptions{
		ServerIdOptions: options.ServerIdOptions{
			ID: srvid,
		},
		Proto:     "tcp",
		ProxyAddr: fItem.ProxyAddr,
		ProxyPort: fItem.ProxyPort,
	}

	params, err := opt.Params()
	if err != nil {
		return nil, errors.Wrap(err, "close forward params")
	}
	return modules.Servers.PerformAction(session, opt.ID, "close-forward", params)
}
