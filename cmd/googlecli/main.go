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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/structarg"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud/google"
	_ "yunion.io/x/onecloud/pkg/multicloud/google/shell"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

type BaseOptions struct {
	Debug        bool   `help:"debug mode"`
	AuthFile     string `help:"google cloud auth json file path" default:"$GOOGLE_AUTH_FILE" metavar:"GOOGLE_AUTH_FILE"`
	ClientEmail  string `help:"Client email" default:"$GOOGLE_CLIENT_EMAIL" metavar:"GOOGLE_CLIENT_EMAIL"`
	ProjectID    string `help:"Project ID" default:"$GOOGLE_PROJECT_ID" metavar:"GOOGLE_PROJECT_ID"`
	PrivateKeyID string `help:"Private Key ID" default:"$GOOGLE_PRIVATE_KEY_ID" metavar:"GOOGLE_PRIVATE_KEY_ID"`
	PrivateKey   string `help:"Private Key" default:"$GOOGLE_PRIVATE_KEY" metavar:"GOOGLE_PRIVATE_KEY"`
	RegionID     string `help:"RegionID" default:"$GOOGLE_REGION" metavar:"GOOGLE_REGION"`
	SUBCOMMAND   string `help:"googlecli subcommand" subcommand:"true"`
}

func getSubcommandParser() (*structarg.ArgumentParser, error) {
	parse, e := structarg.NewArgumentParserWithHelp(&BaseOptions{},
		"googlecli",
		"Command-line interface to google API.",
		`See "googlecli COMMAND --help" for help on a specific command.`)

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
	log.Errorf("%s", e)
	os.Exit(1)
}

func newClient(options *BaseOptions) (*google.SRegion, error) {
	if len(options.AuthFile) > 0 {
		jsonStr, err := fileutils2.FileGetContents(options.AuthFile)
		if err != nil {
			return nil, errors.Wrap(err, "FileGetContents")
		}
		jsonCfg, err := jsonutils.ParseString(jsonStr)
		if err != nil {
			return nil, errors.Wrap(err, "jsonutils.ParseString")
		}
		options.ClientEmail, _ = jsonCfg.GetString("client_email")
		options.PrivateKeyID, _ = jsonCfg.GetString("private_key_id")
		options.PrivateKey, _ = jsonCfg.GetString("private_key")
		options.ProjectID, _ = jsonCfg.GetString("project_id")
	}
	if len(options.ClientEmail) == 0 {
		return nil, fmt.Errorf("Missing ClientEmail")
	}

	if len(options.PrivateKeyID) == 0 {
		return nil, fmt.Errorf("Missing PrivateKeyID")
	}

	if len(options.PrivateKey) == 0 {
		return nil, fmt.Errorf("Missing PrivateKey")
	}

	if len(options.ProjectID) == 0 {
		return nil, fmt.Errorf("Missing ProjectID")
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

	cli, err := google.NewGoogleClient(
		google.NewGoogleClientConfig(
			options.ProjectID,
			options.ClientEmail,
			options.PrivateKeyID,
			options.PrivateKey,
		).Debug(options.Debug).
			CloudproviderConfig(
				cloudprovider.ProviderConfig{
					ProxyFunc: proxyFunc,
				},
			),
	)
	if err != nil {
		return nil, err
	}

	region := cli.GetRegion(options.RegionID)
	if region == nil {
		return nil, fmt.Errorf("No such region %s", options.RegionID)
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
		return
	}
	suboptions := subparser.Options()
	if subparser.IsHelpSet() {
		fmt.Print(subparser.HelpString())
		return
	}
	var region *google.SRegion
	region, e = newClient(options)
	if e != nil {
		showErrorAndExit(e)
	}
	e = subcmd.Invoke(region, suboptions)
	if e != nil {
		showErrorAndExit(e)
	}
}
