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

	"yunion.io/x/onecloud/pkg/multicloud/ctyun"
	_ "yunion.io/x/onecloud/pkg/multicloud/ctyun/shell"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

type BaseOptions struct {
	Help       bool   `help:"Show help" default:"false"`
	Debug      bool   `help:"Show debug" default:"false"`
	AccessKey  string `help:"Access key" default:"$CTYUN_ACCESS_KEY"`
	Secret     string `help:"Secret" default:"$CTYUN_SECRET"`
	RegionId   string `help:"RegionId" default:"$CTYUN_REGION"`
	SUBCOMMAND string `help:"ctyuncli subcommand" subcommand:"true"`
}

func getSubcommandParser() (*structarg.ArgumentParser, error) {
	parse, e := structarg.NewArgumentParser(&BaseOptions{},
		"ctyuncli",
		"Command-line interface to ctyun API.",
		`See "ctyuncli help COMMAND" for help on a specific command.`)

	if e != nil {
		return nil, e
	}

	subcmd := parse.GetSubcommand()
	if subcmd == nil {
		return nil, fmt.Errorf("No subcommand argument.")
	}
	type HelpOptions struct {
		SUBCOMMAND string `help:"sub-command name"`
	}
	shellutils.R(&HelpOptions{}, "help", "Show help of a subcommand", func(args *HelpOptions) error {
		helpstr, e := subcmd.SubHelpString(args.SUBCOMMAND)
		if e != nil {
			return e
		} else {
			fmt.Print(helpstr)
			return nil
		}
	})
	for _, v := range shellutils.CommandTable {
		_, e := subcmd.AddSubParser(v.Options, v.Command, v.Desc, v.Callback)
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

func newClient(options *BaseOptions) (*ctyun.SRegion, error) {
	if len(options.AccessKey) == 0 {
		return nil, fmt.Errorf("Missing accessKey")
	}

	if len(options.Secret) == 0 {
		return nil, fmt.Errorf("Missing secret")
	}

	cli, err := ctyun.NewSCtyunClient("", "", "",
		options.AccessKey, options.Secret, options.Debug)
	if err != nil {
		return nil, err
	}

	region := cli.GetRegion(options.RegionId)
	if region == nil {
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

	if options.Help {
		fmt.Print(parser.HelpString())
	} else {
		subcmd := parser.GetSubcommand()
		subparser := subcmd.GetSubParser()
		if e != nil {
			if subparser != nil {
				fmt.Print(subparser.Usage())
			} else {
				fmt.Print(parser.Usage())
			}
			showErrorAndExit(e)
		} else {
			suboptions := subparser.Options()
			if options.SUBCOMMAND == "help" {
				e = subcmd.Invoke(suboptions)
			} else {
				var region *ctyun.SRegion
				region, e = newClient(options)
				if e != nil {
					showErrorAndExit(e)
				}
				e = subcmd.Invoke(region, suboptions)
			}
			if e != nil {
				showErrorAndExit(e)
			}
		}
	}
}
