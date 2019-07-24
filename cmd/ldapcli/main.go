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

	"yunion.io/x/log"
	"yunion.io/x/structarg"

	_ "yunion.io/x/onecloud/cmd/ldapcli/shell"
	"yunion.io/x/onecloud/pkg/util/ldaputils"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

type BaseOptions struct {
	Debug    bool   `help:"debug mode"`
	Help     bool   `help:"Show help"`
	Url      string `help:"ldap url, like ldap://10.168.222.23:389 or ldaps://10.168.222.23:389" default:"$LDAP_URL"`
	Account  string `help:"LDAP Account" default:"$LDAP_ACCOUNT"`
	Password string `help:"LDAP password" default:"$LDAP_PASSWORD"`
	BaseDN   string `help:"LDAP base DN" default:"$LDAP_BASEDN"`

	SUBCOMMAND string `help:"aliyuncli subcommand" subcommand:"true"`
}

func getSubcommandParser() (*structarg.ArgumentParser, error) {
	parse, e := structarg.NewArgumentParser(&BaseOptions{},
		"ldapcli",
		"Command-line tool for LDAP API.",
		`See "ldapcli help COMMAND" for help on a specific command.`)

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

func newClient(options *BaseOptions) (*ldaputils.SLDAPClient, error) {
	if len(options.Url) == 0 {
		return nil, fmt.Errorf("Missing ldap URL")
	}
	if len(options.BaseDN) == 0 {
		return nil, fmt.Errorf("Missing BaseDN, e.g. DC=example,DC=com")
	}

	cli := ldaputils.NewLDAPClient(options.Url, options.Account, options.Password, options.BaseDN, options.Debug)

	err := cli.Connect()
	if err != nil {
		log.Errorf("connect fail %s", err)
		return nil, err
	}
	return cli, nil
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
			var cli *ldaputils.SLDAPClient
			suboptions := subparser.Options()
			if options.SUBCOMMAND == "help" {
				e = subcmd.Invoke(suboptions)
			} else {
				cli, e = newClient(options)
				if e != nil {
					showErrorAndExit(e)
				}
				e = subcmd.Invoke(cli, suboptions)
			}
			if e != nil {
				showErrorAndExit(e)
			}
		}
	}
}
