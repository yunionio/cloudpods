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

	"github.com/sirupsen/logrus"
	"golang.org/x/net/http/httpproxy"

	"yunion.io/x/log"
	"yunion.io/x/structarg"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud/jdcloud"
	_ "yunion.io/x/onecloud/pkg/multicloud/jdcloud/shell"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

type Options struct {
	Debug        bool   `help:"Show debug" default:"false"`
	AccessKey    string `help:"Access key" default:"$JDCLOUD_ACCESS_KEY"`
	AccessSecret string `help:"Secret" default:"$JDCLOUD_SECRET"`
	RegionId     string `help:"RegionId" default:"$JDCLOUD_REGION"`
	SUBCOMMAND   string `help:"jdcloudcli subcommand" subcommand:"true"`
}

func getSubcommandParser() (*structarg.ArgumentParser, error) {
	parse, err := structarg.NewArgumentParserWithHelp(&Options{},
		"jdcloudcli",
		"Command-line interface to ecloud API.",
		`See "jdcloudcli COMMAND --help" for help on a specific command.`,
	)
	if err != nil {
		return nil, err
	}

	subcmd := parse.GetSubcommand()
	if subcmd == nil {
		return nil, fmt.Errorf("No subcommand argument")
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

func newClient(options *Options) (*jdcloud.SJDCloudClient, error) {
	if len(options.AccessKey) == 0 {
		return nil, fmt.Errorf("Missing access key")
	}
	if len(options.AccessSecret) == 0 {
		return nil, fmt.Errorf("Missing access secret")
	}
	regionId := options.RegionId
	if regionId == "" {
		regionId = jdcloud.JDCLOUD_DEFAULT_REGION
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

	cfcg := cloudprovider.ProviderConfig{
		ProxyFunc: proxyFunc,
	}

	return jdcloud.NewJDCloudClient(
		jdcloud.NewJDCloudClientConfig(
			options.AccessKey,
			options.AccessSecret,
		).CloudproviderConfig(cfcg).Debug(options.Debug),
	)
}

func main() {
	parser, err := getSubcommandParser()
	if err != nil {
		showErrorAndExit(err)
	}
	err = parser.ParseArgs(os.Args[1:], false)
	options := parser.Options().(*Options)

	if parser.IsHelpSet() {
		fmt.Print(parser.HelpString())
		return
	}
	if options.Debug {
		log.SetLogLevel(log.Logger(), logrus.DebugLevel)
	} else {
		log.SetLogLevel(log.Logger(), logrus.InfoLevel)
	}
	subcmd := parser.GetSubcommand()
	subparser := subcmd.GetSubParser()
	if err != nil || subparser == nil {
		if subparser != nil {
			fmt.Print(subparser.Usage())
		} else {
			fmt.Print(parser.Usage())
		}
		showErrorAndExit(err)
	}
	suboptions := subparser.Options()
	if subparser.IsHelpSet() {
		fmt.Print(subparser.HelpString())
		return
	}
	client, err := newClient(options)
	if err != nil {
		showErrorAndExit(err)
	}
	region := client.GetRegion(options.RegionId)
	if region == nil {
		fmt.Printf("not found region: %s", options.RegionId)
		return
	}
	err = subcmd.Invoke(region, suboptions)
	if err != nil {
		showErrorAndExit(err)
	}
}
