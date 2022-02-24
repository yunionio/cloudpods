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
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"

	"yunion.io/x/structarg"

	_ "yunion.io/x/onecloud/cmd/redfishcli/shell"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/redfish"
	"yunion.io/x/onecloud/pkg/util/redfish/bmconsole"
	_ "yunion.io/x/onecloud/pkg/util/redfish/loader"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

type BaseOptions struct {
	Debug      bool   `help:"debug mode"`
	Endpoint   string `help:"Endpoint, usually https://<host_ipmi_ip>" default:"$REDFISH_ENDPOINT" metavar:"REDFISH_ENDPOINT"`
	Username   string `help:"Username, usually root" default:"$REDFISH_USERNAME" metavar:"REDFISH_USERNAME"`
	Password   string `help:"Password" default:"$REDFISH_PASSWORD" metavar:"REDFISH_PASSWORD"`
	SUBCOMMAND string `help:"s3cli subcommand" subcommand:"true"`
}

var (
	options = &BaseOptions{}
)

func getSubcommandParser() (*structarg.ArgumentParser, error) {
	parse, e := structarg.NewArgumentParserWithHelp(options,
		"redfishcli",
		"Command-line interface to redfish API.",
		`See "redfishcli COMMAND --help" for help on a specific command.`)

	if e != nil {
		return nil, e
	}

	subcmd := parse.GetSubcommand()
	if subcmd == nil {
		return nil, fmt.Errorf("No subcommand argument.")
	}
	bmcJnlp()
	for _, v := range shellutils.CommandTable {
		_, e := subcmd.AddSubParserWithHelp(v.Options, v.Command, v.Desc, v.Callback)
		if e != nil {
			return nil, e
		}
	}
	return parse, nil
}

func bmcJnlp() {
	type BmcGetOptions struct {
		BRAND string `help:"brand of baremetal" choices:"Lenovo|Huawei|HPE|Dell|Supermicro|Dell6|Dell7|Dell9"`
		Save  string `help:"save to file"`
		Debug bool   `help:"turn on debug mode"`
		Sku   string `help:"sku"`
		Model string `help:"model"`
	}
	shellutils.R(&BmcGetOptions{}, "bmc-jnlp", "Get Java Console JNLP file", func(args *BmcGetOptions) error {
		ctx := context.Background()
		parts, err := url.Parse(options.Endpoint)
		if err != nil {
			return err
		}
		bmc := bmconsole.NewBMCConsole(parts.Hostname(), options.Username, options.Password, args.Debug)
		var jnlp string
		switch strings.ToLower(args.BRAND) {
		case "hp", "hpe":
			jnlp, err = bmc.GetIloConsoleJNLP(ctx)
		case "dell", "dell inc.":
			jnlp, err = bmc.GetIdracConsoleJNLP(ctx, args.Sku, args.Model)
		case "dell6":
			jnlp, err = bmc.GetIdrac6ConsoleJNLP(ctx, args.Sku, args.Model)
		case "dell7":
			jnlp, err = bmc.GetIdrac7ConsoleJNLP(ctx, args.Sku, args.Model)
		case "dell9":
			jnlp, err = bmc.GetIdrac9ConsoleJNLP(ctx)
		case "supermicro":
			jnlp, err = bmc.GetSupermicroConsoleJNLP(ctx)
		case "lenovo":
			jnlp, err = bmc.GetLenovoConsoleJNLP(ctx)
		}
		if err != nil {
			return err
		}
		if len(args.Save) > 0 {
			return fileutils2.FilePutContents(args.Save, jnlp, false)
		} else {
			fmt.Println(jnlp)
			return nil
		}
	})
}

func showErrorAndExit(e error) {
	fmt.Fprintf(os.Stderr, "%s", e)
	fmt.Fprintln(os.Stderr)
	os.Exit(1)
}

func newClient() (redfish.IRedfishDriver, error) {
	if len(options.Endpoint) == 0 {
		return nil, fmt.Errorf("Missing endpoint")
	}

	if len(options.Username) == 0 {
		return nil, fmt.Errorf("Missing username")
	}

	if len(options.Password) == 0 {
		return nil, fmt.Errorf("Missing password")
	}

	cli := redfish.NewRedfishDriver(context.Background(), options.Endpoint, options.Username, options.Password, options.Debug)
	if cli == nil {
		return nil, fmt.Errorf("no approriate driver")
	}

	return cli, nil
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
	} else if options.SUBCOMMAND == "bmc-jnlp" {
		e = subcmd.Invoke(suboptions)
	} else {
		var client redfish.IRedfishDriver
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
