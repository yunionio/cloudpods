package main

import (
	"fmt"
	"os"

	"yunion.io/x/log"
	"yunion.io/x/structarg"

	"yunion.io/x/onecloud/pkg/util/azure"
	"yunion.io/x/onecloud/pkg/util/printutils"
	"yunion.io/x/onecloud/pkg/util/shellutils"

	_ "yunion.io/x/onecloud/pkg/util/azure/shell"
)

type BaseOptions struct {
	Help           bool   `help:"Show help"`
	DirectoryID    string `help:"Azure account Directory ID/Tenant ID" default:"$AZURE_DIRECTORY_ID"`
	SubscriptionID string `help:"Azure account subscription ID" default:"$AZURE_SUBSCRIPTION_ID"`
	ApplicationID  string `help:"Azure application ID" default:"$AZURE_APPLICATION_ID"`
	ApplicationKey string `help:"Azure application key" default:"$AZURE_APPLICATION_KEY"`
	RegionId       string `help:"RegionId" default:"$AZURE_REGION_ID"`
	CloudEnv       string `help:"Cloud Environment" default:"$AZURE_CLOUD_ENV" choices:"AzureGermanCloud|AzureChinaCloud|AzureUSGovernmentCloud|AzurePublicCloud"`
	SUBCOMMAND     string `help:"azurecli subcommand" subcommand:"true"`
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
	if len(options.DirectoryID) == 0 {
		return nil, fmt.Errorf("Missing Directory ID")
	}

	if len(options.SubscriptionID) == 0 {
		return nil, fmt.Errorf("Missing subscription ID")
	}

	if len(options.ApplicationID) == 0 {
		return nil, fmt.Errorf("Missing Application ID")
	}

	if len(options.ApplicationKey) == 0 {
		return nil, fmt.Errorf("Missing Application Key")
	}

	if len(options.CloudEnv) == 0 {
		return nil, fmt.Errorf("Missing Cloud Environment")
	}

	account := fmt.Sprintf("%s/%s", options.DirectoryID, options.SubscriptionID)
	secret := fmt.Sprintf("%s/%s", options.ApplicationID, options.ApplicationKey)
	cli, err := azure.NewAzureClient("", "", account, secret, options.CloudEnv)
	if err != nil {
		return nil, err
	}
	region := cli.GetRegion(options.RegionId)
	if region == nil {
		fmt.Println("Please chooce which region you are going to use:")
		regions := cli.GetRegions()
		printutils.PrintInterfaceList(regions, 0, 0, 0, nil)
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
