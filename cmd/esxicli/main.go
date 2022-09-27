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
	"yunion.io/x/onecloud/pkg/multicloud/esxi"
	_ "yunion.io/x/onecloud/pkg/multicloud/esxi/shell"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

type BaseOptions struct {
	Host       string `help:"Host IP or NAME" default:"$VMWARE_HOST" metavar:"VMWARE_HOST"`
	Port       int    `help:"Service port" default:"$VMWARE_PORT" metavar:"VMWARE_PORT"`
	Account    string `help:"VCenter or ESXi Account" default:"$VMWARE_ACCOUNT" metavar:"VMWARE_ACCOUNT"`
	Password   string `help:"Password" default:"$VMWARE_PASSWORD" metavar:"VMWARE_PASSWORD"`
	Debug      bool
	Format     string `choices:"xml|json" default:"json"`
	SUBCOMMAND string `help:"aliyuncli subcommand" subcommand:"true"`
}

func getSubcommandParser() (*structarg.ArgumentParser, error) {
	parse, e := structarg.NewArgumentParserWithHelp(&BaseOptions{},
		"esxicli",
		"Command-line interface to VMware VSphere Webservice API.",
		`See "esxicli COMMAND --help" for help on a specific command.`)

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

func newClient(options *BaseOptions) (*esxi.SESXiClient, error) {
	if len(options.Host) == 0 {
		return nil, fmt.Errorf("Missing host")
	}

	if len(options.Account) == 0 {
		return nil, fmt.Errorf("Missing account")
	}

	if len(options.Password) == 0 {
		return nil, fmt.Errorf("Missing password")
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

	return esxi.NewESXiClient2(
		esxi.NewESXiClientConfig(
			options.Host,
			options.Port,
			options.Account,
			options.Password,
		).
			CloudproviderConfig(
				cloudprovider.ProviderConfig{
					ProxyFunc: proxyFunc,
				},
			).Debug(options.Debug).Format(options.Format),
	)
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
	var esxicli *esxi.SESXiClient
	esxicli, e = newClient(options)
	if e != nil {
		showErrorAndExit(e)
	}
	e = subcmd.Invoke(esxicli, suboptions)
	if e != nil {
		showErrorAndExit(e)
	}
}
