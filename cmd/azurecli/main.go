package main

import (
	"fmt"
	"os"

	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/util/azure"
	"yunion.io/x/onecloud/pkg/util/shellutils"
	"yunion.io/x/structarg"

	_ "yunion.io/x/onecloud/pkg/util/azure/shell"
)

type BaseOptions struct {
	Help       bool   `help:"Show help"`
	AccessKey  string `help:"Access key" default:"$AZURE_ACCESS_KEY"`
	Secret     string `help:"Secret" default:"$AZURE_SECRET"`
	RegionId   string `help:"RegionId" default:"$AZURE_REGION"`
	AccessURL  string `help:"Access URL" default:"$AZURE_ACCESS_URL"`
	SUBCOMMAND string `help:"azurecli subcommand" subcommand:"true"`
}

func getSubcommandParser() (*structarg.ArgumentParser, error) {
	parse, e := structarg.NewArgumentParser(&BaseOptions{},
		"azurecli",
		"Command-line interface to azure API.",
		`See "azurecli help COMMAND" for help on a specific command.`)

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

func newClient(options *BaseOptions) (*azure.SRegion, error) {
	if len(options.AccessKey) == 0 {
		return nil, fmt.Errorf("Missing accessKey")
	}

	if len(options.Secret) == 0 {
		return nil, fmt.Errorf("Missing secret")
	}

	if len(options.AccessURL) == 0 {
		return nil, fmt.Errorf("Missing AccessURL")
	}

	if cli, err := azure.NewAzureClient("", "", options.AccessKey, options.Secret, options.AccessURL); err != nil {
		return nil, err
	} else if region := cli.GetRegion(options.RegionId); region == nil {
		return nil, fmt.Errorf("No such region %s", options.RegionId)
	} else {
		return region, nil
	}
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
				var region *azure.SRegion
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
