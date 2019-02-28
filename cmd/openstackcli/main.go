package main

import (
	"fmt"
	"os"

	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/util/openstack"
	"yunion.io/x/onecloud/pkg/util/shellutils"
	"yunion.io/x/structarg"

	_ "yunion.io/x/onecloud/pkg/util/openstack/shell"
)

type BaseOptions struct {
	Help         bool   `help:"Show help"`
	AuthURL      string `help:"Auth URL" default:"$OPENSTACK_AUTH_URL"`
	Username     string `help:"Username" default:"$OPENSTACK_USERNAME"`
	Password     string `help:"Password" default:"$OPENSTACK_PASSWORD"`
	Project      string `help:"Project" default:"$OPENSTACK_PROJECT"`
	EndpointType string `help:"Project" default:"$OPENSTACK_ENDPOINT_TYPE|internal"`
	RegionID     string `help:"RegionId" default:"$OPENSTACK_REGION_ID"`
	SUBCOMMAND   string `help:"openstackcli subcommand" subcommand:"true"`
}

func getSubcommandParser() (*structarg.ArgumentParser, error) {
	parse, e := structarg.NewArgumentParser(&BaseOptions{},
		"openstackcli",
		"Command-line interface to openstack API.",
		`See "openstackcli help COMMAND" for help on a specific command.`)

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

func newClient(options *BaseOptions) (*openstack.SRegion, error) {
	if len(options.AuthURL) == 0 {
		return nil, fmt.Errorf("Missing AuthURL")
	}

	if len(options.Username) == 0 {
		return nil, fmt.Errorf("Missing Username")
	}

	if len(options.Password) == 0 {
		return nil, fmt.Errorf("Missing Password")
	}

	cli, err := openstack.NewOpenStackClient("", "", options.AuthURL, options.Username, options.Password, options.Project, options.EndpointType)
	if err != nil {
		return nil, err
	}
	region := cli.GetRegion(options.RegionID)
	if region == nil {
		return nil, fmt.Errorf("No such region %s", options.RegionID)
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
				var region *openstack.SRegion
				if len(options.RegionID) == 0 {
					options.RegionID = openstack.OPENSTACK_DEFAULT_REGION
				}
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
