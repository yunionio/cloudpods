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
	"yunion.io/x/onecloud/pkg/multicloud/azure"
	_ "yunion.io/x/onecloud/pkg/multicloud/azure/shell"
	"yunion.io/x/onecloud/pkg/util/printutils"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

type BaseOptions struct {
	Debug          bool   `help:"debug mode"`
	DirectoryID    string `help:"Azure account Directory ID/Tenant ID" default:"$AZURE_DIRECTORY_ID" metavar:"AZURE_DIRECTORY_ID"`
	SubscriptionID string `help:"Azure account subscription ID" default:"$AZURE_SUBSCRIPTION_ID" metavar:"AZURE_SUBSCRIPTION_ID"`
	ApplicationID  string `help:"Azure application ID" default:"$AZURE_APPLICATION_ID" metavar:"AZURE_APPLICATION_ID"`
	ApplicationKey string `help:"Azure application key" default:"$AZURE_APPLICATION_KEY" metavar:"AZURE_APPLICATION_KEY"`
	RegionId       string `help:"RegionId" default:"$AZURE_REGION_ID" metavar:"AZURE_REGION_ID"`
	CloudEnv       string `help:"Cloud Environment" default:"$AZURE_CLOUD_ENV" choices:"AzureGermanCloud|AzureChinaCloud|AzureUSGovernmentCloud|AzurePublicCloud" metavar:"AZURE_CLOUD_ENV"`
	SUBCOMMAND     string `help:"azurecli subcommand" subcommand:"true"`
}

func getSubcommandParser() (*structarg.ArgumentParser, error) {
	parse, e := structarg.NewArgumentParserWithHelp(&BaseOptions{},
		"azurecli",
		"Command-line interface to azure API.",
		`See "azurecli COMMAND --help" for help on a specific command.`)

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

func newClient(options *BaseOptions) (*azure.SRegion, error) {
	if len(options.DirectoryID) == 0 {
		return nil, fmt.Errorf("Missing Directory ID")
	}

	if len(options.SubscriptionID) == 0 {
		return nil, fmt.Errorf("Missing subscription ID")
	}

	if len(options.ApplicationID) == 0 {
		return nil, fmt.Errorf("Missing Application ID")
	}

	if len(options.ApplicationKey) == 0 {
		return nil, fmt.Errorf("Missing Application Key")
	}

	if len(options.CloudEnv) == 0 {
		return nil, fmt.Errorf("Missing Cloud Environment")
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

	cli, err := azure.NewAzureClient(
		azure.NewAzureClientConfig(
			options.CloudEnv,
			options.DirectoryID,
			options.ApplicationID,
			options.ApplicationKey,
		).
			SubscriptionId(options.SubscriptionID).
			Debug(options.Debug).
			CloudproviderConfig(
				cloudprovider.ProviderConfig{
					ProxyFunc: proxyFunc,
				},
			),
	)

	if err != nil {
		return nil, err
	}
	region := cli.GetRegion(options.RegionId)
	if region == nil {
		fmt.Println("Please chooce which region you are going to use:")
		regions := cli.GetRegions()
		printutils.PrintInterfaceList(regions, 0, 0, 0, nil)
		return nil, fmt.Errorf("No such region %s", options.RegionId)
	}

	return region, nil
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
	var region *azure.SRegion
	region, e = newClient(options)
	if e != nil {
		showErrorAndExit(e)
	}
	e = subcmd.Invoke(region, suboptions)
	if e != nil {
		showErrorAndExit(e)
	}
}
