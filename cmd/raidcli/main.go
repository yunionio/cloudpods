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

	"yunion.io/x/pkg/util/shellutils"
	"yunion.io/x/structarg"

	_ "yunion.io/x/onecloud/cmd/raidcli/shell"
	"yunion.io/x/onecloud/pkg/baremetal/utils/raid"
	"yunion.io/x/onecloud/pkg/baremetal/utils/raid/drivers"
	"yunion.io/x/onecloud/pkg/util/ssh"
)

type BaseOptions struct {
	Debug      bool   `help:"debug mode"`
	Host       string `help:"SSH Host IP" default:"$RAID_HOST" metavar:"RAID_HOST"`
	Username   string `help:"Username, usually root" default:"$RAID_USERNAME" metavar:"RAID_USERNAME"`
	Password   string `help:"Password" default:"$RAID_PASSWORD" metavar:"RAID_PASSWORD"`
	Driver     string `help:"Raid driver" default:"$RAID_DRIVER" metavar:"RAID_DRIVER" choices:"MegaRaid|HPSARaid|Mpt2SAS|MarvelRaid"`
	LocalHost  bool   `help:"Run raidcli in localhost"`
	SUBCOMMAND string `help:"s3cli subcommand" subcommand:"true"`
}

var (
	options = &BaseOptions{}
)

func getSubcommandParser() (*structarg.ArgumentParser, error) {
	parse, e := structarg.NewArgumentParserWithHelp(options,
		"raidcli",
		"Command-line interface to test RAID drivers.",
		`See "raidcli COMMAND --help" for help on a specific command.`)

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

func newClient() (raid.IRaidDriver, error) {
	if options.Debug {
		raid.Debug = true
	}

	if len(options.Driver) == 0 {
		return nil, fmt.Errorf("Missing driver")
	}

	var drv raid.IRaidDriver
	if !options.LocalHost {
		if len(options.Host) == 0 {
			return nil, fmt.Errorf("Missing host")
		}

		if len(options.Username) == 0 {
			return nil, fmt.Errorf("Missing username")
		}

		if len(options.Password) == 0 {
			return nil, fmt.Errorf("Missing password")
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

		drv = drivers.GetDriver(options.Driver, sshClient)
		if drv == nil {
			return nil, fmt.Errorf("not supported driver %s", options.Driver)
		}
	} else {
		drv = drivers.GetLocalDriver(options.Driver)
	}

	err := drv.ParsePhyDevs()
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
	var client raid.IRaidDriver
	client, e = newClient()
	if e != nil {
		showErrorAndExit(e)
	}
	e = subcmd.Invoke(client, suboptions)
	if e != nil {
		showErrorAndExit(e)
	}
}
