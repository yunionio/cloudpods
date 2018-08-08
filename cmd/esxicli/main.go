package main

import (
	"fmt"
	"os"

	"github.com/yunionio/log"
	"github.com/yunionio/structarg"

	"github.com/yunionio/onecloud/pkg/util/esxi"
	"github.com/yunionio/onecloud/pkg/util/shellutils"

	_ "github.com/yunionio/onecloud/pkg/util/esxi/shell"
)

type BaseOptions struct {
	Help       bool   `help:"Show help"`
	Host       string `help:"Host IP or NAME" default:"$VMWARE_HOST"`
	Port       int    `help:"Service port" default:"$VMWARE_PORT"`
	Account    string `help:"VCenter or ESXi Account" default:"$VMWARE_ACCOUNT"`
	Password   string `help:"Password" default:"$VMWARE_PASSWORD"`
	SUBCOMMAND string `help:"aliyuncli subcommand" subcommand:"true"`
}

func getSubcommandParser() (*structarg.ArgumentParser, error) {
	parse, e := structarg.NewArgumentParser(&BaseOptions{},
		"govmcli",
		"Command-line interface to VMware VSphere Webservice API.",
		`See "govmcli help COMMAND" for help on a specific command.`)

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
	log.Errorf("%s", e)
	os.Exit(1)
}

func newClient(options *BaseOptions) (*esxi.SESXiClient, error) {
	if len(options.Host) == 0 {
		return nil, fmt.Errorf("Missing host")
	}

	if len(options.Account) == 0 {
		return nil, fmt.Errorf("Missing account")
	}

	if len(options.Password) == 0 {
		return nil, fmt.Errorf("Missing password")
	}

	return esxi.NewESXiClient("", "", options.Host, options.Port, options.Account, options.Password)
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
				var esxicli *esxi.SESXiClient
				esxicli, e = newClient(options)
				if e != nil {
					showErrorAndExit(e)
				}
				e = subcmd.Invoke(esxicli, suboptions)
			}
			if e != nil {
				showErrorAndExit(e)
			}
		}
	}
}
