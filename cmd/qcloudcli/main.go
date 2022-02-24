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

package main

import (
	"fmt"
	"net/http"
	"net/url"
	"os"

	"golang.org/x/net/http/httpproxy"

	"yunion.io/x/structarg"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud/qcloud"
	_ "yunion.io/x/onecloud/pkg/multicloud/qcloud/shell"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

type BaseOptions struct {
	Debug      bool   `help:"debug mode"`
	AppID      string `help:"AppID" default:"$QCLOUD_APPID" metavar:"QCLOUD_APPID"`
	SecretID   string `help:"Secret" default:"$QCLOUD_SECRET_ID" metavar:"QCLOUD_SECRET_ID"`
	SecretKey  string `help:"Access key" default:"$QCLOUD_SECRET_KEY" metavar:"QCLOUD_SECRET_KEY"`
	RegionId   string `help:"RegionId" default:"$QCLOUD_REGION" metavar:"QCLOUD_REGION"`
	SUBCOMMAND string `help:"azurecli subcommand" subcommand:"true"`
}

func getSubcommandParser() (*structarg.ArgumentParser, error) {
	parse, e := structarg.NewArgumentParserWithHelp(&BaseOptions{},
		"qcloudcli",
		"Command-line interface to tencentcloud API.",
		`See "qcloudcli COMMAND --help" for help on a specific command.`)

	if e != nil {
		return nil, e
	}

	subcmd := parse.GetSubcommand()
	if subcmd == nil {
		return nil, fmt.Errorf("No subcommand argument.")
	}
	for _, v := range shellutils.CommandTable {
		_, e := subcmd.AddSubParserWithHelp(v.Options, v.Command, v.Desc, v.Callback)
		if e != nil {
			return nil, e
		}
	}
	return parse, nil
}

func showErrorAndExit(e error) {
	fmt.Fprintf(os.Stderr, "%s", e)
	fmt.Fprintln(os.Stderr)
	os.Exit(1)
}

func newClient(options *BaseOptions) (*qcloud.SRegion, error) {
	if len(options.SecretKey) == 0 {
		return nil, fmt.Errorf("Missing SecretKey")
	}

	if len(options.SecretID) == 0 {
		return nil, fmt.Errorf("Missing SecretID")
	}

	cfg := &httpproxy.Config{
		HTTPProxy:  os.Getenv("HTTP_PROXY"),
		HTTPSProxy: os.Getenv("HTTPS_PROXY"),
		NoProxy:    os.Getenv("NO_PROXY"),
	}
	cfgProxyFunc := cfg.ProxyFunc()
	proxyFunc := func(req *http.Request) (*url.URL, error) {
		return cfgProxyFunc(req.URL)
	}

	if cli, err := qcloud.NewQcloudClient(
		qcloud.NewQcloudClientConfig(
			options.SecretID,
			options.SecretKey,
		).AppId(options.AppID).
			Debug(options.Debug).
			CloudproviderConfig(
				cloudprovider.ProviderConfig{
					ProxyFunc: proxyFunc,
				},
			),
	); err != nil {
		return nil, err
	} else if region := cli.GetRegion(options.RegionId); region == nil {
		return nil, fmt.Errorf("No such region %s", options.RegionId)
	} else {
		return region, nil
	}
}

func main() {
	parser, e := getSubcommandParser()
	if e != nil {
		showErrorAndExit(e)
	}
	e = parser.ParseArgs(os.Args[1:], false)
	options := parser.Options().(*BaseOptions)

	if parser.IsHelpSet() {
		fmt.Print(parser.HelpString())
		return
	}
	subcmd := parser.GetSubcommand()
	subparser := subcmd.GetSubParser()
	if e != nil || subparser == nil {
		if subparser != nil {
			fmt.Print(subparser.Usage())
		} else {
			fmt.Print(parser.Usage())
		}
		showErrorAndExit(e)
	}
	suboptions := subparser.Options()
	if subparser.IsHelpSet() {
		fmt.Print(subparser.HelpString())
		return
	}
	var region *qcloud.SRegion
	if len(options.RegionId) == 0 {
		options.RegionId = qcloud.QCLOUD_DEFAULT_REGION
	}
	region, e = newClient(options)
	if e != nil {
		showErrorAndExit(e)
	}
	e = subcmd.Invoke(region, suboptions)
	if e != nil {
		showErrorAndExit(e)
	}
}
