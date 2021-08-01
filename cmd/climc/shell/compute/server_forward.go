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
	"os"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	
	"yunion.io/x/onecloud/cmd/climc/shell"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	cmd := shell.NewResourceCmd(&modules.Servers)
	cmd.Perform("open-forward", &options.ServerOpenForwardOptions{})
	cmd.Perform("close-forward", &options.ServerCloseForwardOptions{})
	cmd.Perform("list-forward", &options.ServerListForwardOptions{})
}

func openForward(srvid string) (jsonutils.JSONObject, error) {
	parser, e := entry.GetSubcommandsParser()
	if e != nil {
		entry.ShowErrorAndExit(e)
	}
	e = parser.ParseArgs(os.Args[1:], false)
	session, _ := entry.NewClientSession(parser.Options().(*entry.BaseOptions))
	opt := &options.ServerOpenForwardOptions{ServerIdOptions: options.ServerIdOptions{ID: srvid}, Proto: "tcp", Port: 22}

	params, err := opt.Params()
	if err != nil {
		return nil, errors.Wrap(err, "get open forward params")
	}
	return modules.Servers.PerformAction(session, opt.ID, "open-forward", params)
}

func closeForward(srvid string, proxy_addr string, proxy_port int) (jsonutils.JSONObject, error) {
	parser, e := entry.GetSubcommandsParser()
	if e != nil {
		entry.ShowErrorAndExit(e)
	}
	e = parser.ParseArgs(os.Args[1:], false)
	session, _ := entry.NewClientSession(parser.Options().(*entry.BaseOptions))
	opt := &options.ServerCloseForwardOptions{ServerIdOptions: options.ServerIdOptions{ID: srvid}, Proto: "tcp", ProxyAddr: proxy_addr, ProxyPort: proxy_port}

	params, err := opt.Params()
	if err != nil {
		return nil, errors.Wrap(err, "close forward params")
	}
	return modules.Servers.PerformAction(session, opt.ID, "close-forward", params)
}


