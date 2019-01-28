package main

import (
	"fmt"
	"os"

	"yunion.io/x/log"
	"yunion.io/x/structarg"

	"yunion.io/x/onecloud/pkg/baremetal/utils/ipmitool"
	"yunion.io/x/onecloud/pkg/util/shellutils"
	"yunion.io/x/onecloud/pkg/util/ssh"

	_ "yunion.io/x/onecloud/pkg/util/ipmitool/shell"
)

type BaseOptions struct {
	Help       bool   `help:"Show help" short-token:"h"`
	MODE       string `help:"Execute command mode" choices:"ssh|rmcp"`
	HOST       string `help:"IP address of remote host"`
	PASSWD     string `help:"Password"`
	User       string `help:"Username" short-token:"u" default:"root"`
	Port       int    `help:"Remote service port"`
	SUBCOMMAND string `help:"ipmicli subcommand" subcommand:"true"`
}

func showErrorAndExit(err error) {
	log.Errorf("%s", err)
	os.Exit(1)
}

func getSubcommandParser() (*structarg.ArgumentParser, error) {
	parser, err := structarg.NewArgumentParser(
		&BaseOptions{},
		"ipmicli",
		"Command-line interface to ipmitool",
		`See "ipmicli help COMMAND" for help on a specific command.`,
	)
	if err != nil {
		return nil, err
	}
	subcmd := parser.GetSubcommand()
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

	if options.Help {
		fmt.Print(parser.HelpString())
		return
	}

	subcmd := parser.GetSubcommand()
	subparser := subcmd.GetSubParser()
	if err != nil {
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
	if options.SUBCOMMAND == "help" {
		args = append(args, suboptions)
	} else {
		executor, err := newExecutor(options)
		if err != nil {
			showErrorAndExit(err)
		}
		args = append(args, executor, suboptions)
	}
	err = subcmd.Invoke(args...)
	if err != nil {
		showErrorAndExit(err)
	}
}
