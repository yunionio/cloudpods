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

	_ "yunion.io/x/onecloud/cmd/raidcli/shell"
	"yunion.io/x/onecloud/pkg/baremetal/utils/raid"
	"yunion.io/x/onecloud/pkg/baremetal/utils/raid/drivers"
	"yunion.io/x/onecloud/pkg/util/shellutils"
	"yunion.io/x/onecloud/pkg/util/ssh"
)

type BaseOptions struct {
	Debug      bool   `help:"debug mode"`
	Help       bool   `help:"Show help"`
	Host       string `help:"SSH Host IP" default:"$RAID_HOST" metavar:"RAID_HOST"`
	Username   string `help:"Username, usually root" default:"$RAID_USERNAME" metavar:"RAID_USERNAME"`
	Password   string `help:"Password" default:"$RAID_PASSWORD" metavar:"RAID_PASSWORD"`
	Driver     string `help:"Password" default:"$RAID_DRIVER" metavar:"RAID_DRIVER" choices:"MegaRaid|HPSARaid|Mpt2SAS|MarvelRaid"`
	SUBCOMMAND string `help:"s3cli subcommand" subcommand:"true"`
}

var (
	options = &BaseOptions{}
)

func getSubcommandParser() (*structarg.ArgumentParser, error) {
	parse, e := structarg.NewArgumentParser(options,
		"raidcli",
		"Command-line interface to test RAID drivers.",
		`See "raidcli help COMMAND" for help on a specific command.`)

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

func newClient() (raid.IRaidDriver, error) {
	if len(options.Host) == 0 {
		return nil, fmt.Errorf("Missing host")
	}

	if len(options.Username) == 0 {
		return nil, fmt.Errorf("Missing username")
	}

	if len(options.Password) == 0 {
		return nil, fmt.Errorf("Missing password")
	}

	if len(options.Driver) == 0 {
		return nil, fmt.Errorf("Missing driver")
	}

	if options.Debug {
		raid.Debug = true
	}

	sshClient, err := ssh.NewClient(
		options.Host,
		22,
		options.Username,
		options.Password,
		"",
	)
	if err != nil {
		return nil, fmt.Errorf("ssh client init fail: %s", err)
	}

	drv := drivers.GetDriver(options.Driver, sshClient)
	if drv == nil {
		return nil, fmt.Errorf("not supported driver %s", options.Driver)
	}

	err = drv.ParsePhyDevs()
	if err != nil {
		return nil, fmt.Errorf("parse phyical devices error %s", err)
	}

	return drv, nil
}

func main() {
	parser, e := getSubcommandParser()
	if e != nil {
		showErrorAndExit(e)
	}
	e = parser.ParseArgs(os.Args[1:], false)
	// options := parser.Options().(*BaseOptions)

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
				var client raid.IRaidDriver
				client, e = newClient()
				if e != nil {
					showErrorAndExit(e)
				}
				e = subcmd.Invoke(client, suboptions)
			}
			if e != nil {
				showErrorAndExit(e)
			}
		}
	}
}
