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

	"yunion.io/x/onecloud/pkg/baremetal/utils/ipmitool"
	_ "yunion.io/x/onecloud/pkg/util/ipmitool/shell"
	"yunion.io/x/onecloud/pkg/util/ssh"
)

type BaseOptions struct {
	MODE       string `help:"Execute command mode" choices:"ssh|rmcp"`
	HOST       string `help:"IP address of remote host"`
	PASSWD     string `help:"Password"`
	User       string `help:"Username" short-token:"u" default:"root"`
	Port       int    `help:"Remote service port"`
	SUBCOMMAND string `help:"ipmicli subcommand" subcommand:"true"`
}

func showErrorAndExit(e error) {
	fmt.Fprintf(os.Stderr, "%s", e)
	fmt.Fprintln(os.Stderr)
	os.Exit(1)
}

func getSubcommandParser() (*structarg.ArgumentParser, error) {
	parser, err := structarg.NewArgumentParserWithHelp(
		&BaseOptions{},
		"ipmicli",
		"Command-line interface to ipmitool",
		`See "ipmicli COMMAND --help" for help on a specific command.`,
	)
	if err != nil {
		return nil, err
	}
	subcmd := parser.GetSubcommand()
	if subcmd == nil {
		return nil, fmt.Errorf("No subcommand argument.")
	}
	for _, v := range shellutils.CommandTable {
		_, e := subcmd.AddSubParserWithHelp(v.Options, v.Command, v.Desc, v.Callback)
		if e != nil {
			return nil, e
		}
	}
	return parser, nil
}

func newExecutor(options *BaseOptions) (ipmitool.IPMIExecutor, error) {
	if options.MODE == "ssh" {
		port := 22
		if options.Port > 0 {
			port = options.Port
		}
		sshCli, err := ssh.NewClient(options.HOST, port, options.User, options.PASSWD, "")
		if err != nil {
			return nil, err
		}
		return ipmitool.NewSSHIPMI(sshCli), nil
	}
	if options.MODE == "rmcp" {
		port := 623
		if options.Port > 0 {
			port = options.Port
		}
		return ipmitool.NewLanPlusIPMIWithPort(options.HOST, options.User, options.PASSWD, port), nil
	}
	return nil, fmt.Errorf("Unsupported mode: %s", options.MODE)
}

func main() {
	parser, err := getSubcommandParser()
	if err != nil {
		showErrorAndExit(err)
	}

	err = parser.ParseArgs(os.Args[1:], false)
	options := parser.Options().(*BaseOptions)

	if parser.IsHelpSet() {
		fmt.Print(parser.HelpString())
		return
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
		return
	}

	suboptions := subparser.Options()
	var args []interface{}
	if subparser.IsHelpSet() {
		fmt.Print(subparser.HelpString())
		return
	}
	executor, err := newExecutor(options)
	if err != nil {
		showErrorAndExit(err)
	}
	args = append(args, executor, suboptions)
	err = subcmd.Invoke(args...)
	if err != nil {
		showErrorAndExit(err)
	}
}
