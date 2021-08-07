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
	"os"

	"yunion.io/x/structarg"

	"yunion.io/x/onecloud/pkg/multicloud/cloudpods"
	_ "yunion.io/x/onecloud/pkg/multicloud/cloudpods/shell"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

type BaseOptions struct {
	Debug        bool   `help:"debug mode"`
	Help         bool   `help:"Show help"`
	AuthURL      string `help:"Auth URL" default:"$CLOUDPODS_AUTH_URL" metavar:"CLOUDPODS_AUTH_URL"`
	AccessKey    string `help:"AccessKey" default:"$CLOUDPODS_ACCESS_KEY" metavar:"CLOUDPODS_ACCESS_KEY"`
	AccessSecret string `help:"AccessSecret" default:"$CLOUDPODS_ACCESS_SECRET" metavar:"CLOUDPODS_ACCESS_SECRET"`
	RegionId     string `help:"RegionId" default:"$CLOUDPODS_REGION_ID|default" metavar:"CLOUDPODS_REGION_ID"`
	SUBCOMMAND   string `help:"cloudpodscli subcommand" subcommand:"true"`
}

func getSubcommandParser() (*structarg.ArgumentParser, error) {
	parse, err := structarg.NewArgumentParser(&BaseOptions{},
		"cloudpodscli",
		"Command-line interface to cloudpods API.",
		`See "cloudpodscli help COMMAND" for help on a specific command.`)

	if err != nil {
		return nil, err
	}

	subcmd := parse.GetSubcommand()
	if subcmd == nil {
		return nil, fmt.Errorf("No subcommand argument.")
	}
	type HelpOptions struct {
		SUBCOMMAND string `help:"sub-command name"`
	}
	shellutils.R(&HelpOptions{}, "help", "Show help of a subcommand", func(args *HelpOptions) error {
		helpstr, err := subcmd.SubHelpString(args.SUBCOMMAND)
		if err != nil {
			return err
		}
		fmt.Print(helpstr)
		return nil
	})
	for _, v := range shellutils.CommandTable {
		_, err = subcmd.AddSubParser(v.Options, v.Command, v.Desc, v.Callback)
		if err != nil {
			return nil, err
		}
	}
	return parse, nil
}

func showErrorAndExit(e error) {
	fmt.Fprintf(os.Stderr, "%s", e)
	fmt.Fprintln(os.Stderr)
	os.Exit(1)
}

func newClient(options *BaseOptions) (*cloudpods.SRegion, error) {
	if len(options.AuthURL) == 0 {
		return nil, fmt.Errorf("Missing AuthURL")
	}

	if len(options.AccessKey) == 0 {
		return nil, fmt.Errorf("Missing access key")
	}

	if len(options.AccessSecret) == 0 {
		return nil, fmt.Errorf("Missing access secret")
	}

	cli, err := cloudpods.NewCloudpodsClient(
		cloudpods.NewCloudpodsClientConfig(
			options.AuthURL,
			options.AccessKey,
			options.AccessSecret,
		).Debug(options.Debug),
	)
	if err != nil {
		return nil, err
	}
	region, err := cli.GetRegion(options.RegionId)
	if err != nil {
		return nil, err
	}
	return region, nil
}

func main() {
	parser, err := getSubcommandParser()
	if err != nil {
		showErrorAndExit(err)
	}
	err = parser.ParseArgs(os.Args[1:], false)
	options := parser.Options().(*BaseOptions)

	if options.Help {
		fmt.Print(parser.HelpString())
		return
	}
	subcmd := parser.GetSubcommand()
	subparser := subcmd.GetSubParser()
	if err != nil {
		if subparser != nil {
			fmt.Print(subparser.Usage())
		} else {
			fmt.Print(parser.Usage())
		}
		showErrorAndExit(err)
		return
	}
	suboptions := subparser.Options()
	if options.SUBCOMMAND == "help" {
		err = subcmd.Invoke(suboptions)
	} else {
		var region *cloudpods.SRegion
		region, err = newClient(options)
		if err != nil {
			showErrorAndExit(err)
		}
		err = subcmd.Invoke(region, suboptions)
	}
	if err != nil {
		showErrorAndExit(err)
	}
}
