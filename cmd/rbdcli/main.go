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

	"yunion.io/x/pkg/errors"
	"yunion.io/x/structarg"

	_ "yunion.io/x/onecloud/cmd/rbdcli/shell"
	"yunion.io/x/onecloud/pkg/util/rbdutils"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

type BaseOptions struct {
	Help       bool   `help:"Show help"`
	Debug      bool   `help:"debug mode"`
	MonHost    string `help:"Ceph Mon Host" default:"$MON_HOST"`
	Pool       string `help:"Ceph pool" default:"$CEPH_POOL" metavar:"CEPH_POOL"`
	Key        string `help:"Secret" default:"$CEPH_KEY" metavar:"CEPH_KEY"`
	SUBCOMMAND string `help:"rbdcli subcommand" subcommand:"true"`
}

func getSubcommandParser() (*structarg.ArgumentParser, error) {
	parse, e := structarg.NewArgumentParser(&BaseOptions{},
		"rbdcli",
		"Command-line interface to rbd API.",
		`See "rbdcli help COMMAND" for help on a specific command.`)

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

func newPool(opts *BaseOptions) (*rbdutils.SPool, error) {
	cli, err := rbdutils.NewCluster(opts.MonHost, opts.Key)
	if err != nil {
		return nil, err
	}

	pool, err := cli.GetPool(opts.Pool)
	if err != nil {
		return nil, errors.Wrapf(err, "GetPool(%s)", opts.Pool)
	}

	return pool, nil
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
		return
	}
	subcmd := parser.GetSubcommand()
	subparser := subcmd.GetSubParser()
	if e != nil {
		if subparser != nil {
			fmt.Print(subparser.Usage())
		} else {
			fmt.Print(parser.Usage())
		}
		showErrorAndExit(e)
		return
	}
	suboptions := subparser.Options()
	if options.SUBCOMMAND == "help" {
		e = subcmd.Invoke(suboptions)
	} else {
		var pool *rbdutils.SPool
		pool, e = newPool(options)
		if e != nil {
			showErrorAndExit(e)
		}
		e = subcmd.Invoke(pool, suboptions)
	}
	if e != nil {
		showErrorAndExit(e)
	}
}
