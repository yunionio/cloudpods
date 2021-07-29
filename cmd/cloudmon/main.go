package main

import (
	"context"
	"fmt"
	"os"
	"path"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/version"

	_ "yunion.io/x/onecloud/pkg/cloudmon/collectors"
	"yunion.io/x/onecloud/pkg/cloudmon/options"
	"yunion.io/x/onecloud/pkg/mcclient"
)

func showErrorAndExit(e error) {
	fmt.Fprintf(os.Stderr, "%s", e)
	fmt.Fprintln(os.Stderr)
	os.Exit(1)
}

func newClientSession(options *options.CloudMonOptions) (*mcclient.ClientSession, error) {
	if len(options.AuthURL) == 0 {
		return nil, errors.Error("empty auth_url")
	}
	if len(options.AdminUser) == 0 {
		return nil, errors.Error("empty admin_user")
	}
	if len(options.AdminPassword) == 0 {
		return nil, errors.Error("empty admin_password")
	}
	if len(options.AdminProject) == 0 {
		return nil, errors.Error("empty admin_project")
	}

	client := mcclient.NewClient(
		options.AuthURL,
		options.Timeout,
		options.Debug,
		options.Insecure,
		options.CertFile,
		options.KeyFile,
	)

	token, err := client.AuthenticateWithSource(
		options.AdminUser,
		options.AdminPassword,
		options.AdminDomain,
		options.AdminProject,
		options.AdminProjectDomain,
		mcclient.AuthSourceAPI)
	if err != nil {
		return nil, err
	}

	session := client.NewSession(
		context.Background(),
		options.Region,
		"",
		options.EndpointType,
		token,
		options.ApiVersion)

	return session, nil
}

func main() {
	parser, err := options.GetArgumentParser()
	if err != nil {
		showErrorAndExit(err)
	}

	err = parser.ParseArgs2(os.Args[1:], false, false)

	opts := parser.Options().(*options.CloudMonOptions)

	if opts.Help {
		fmt.Println(parser.HelpString())
		os.Exit(0)
	}

	if opts.Version {
		fmt.Println(version.GetJsonString())
		os.Exit(0)
	}

	if len(opts.Config) == 0 {
		for _, p := range []string{"./etc", "/etc/yunion"} {
			confTmp := path.Join(p, "cloudmon.conf")
			if _, err := os.Stat(confTmp); err == nil {
				opts.Config = confTmp
				break
			}
		}
	}

	if len(opts.Config) > 0 {
		err := parser.ParseFile(opts.Config)
		if err != nil {
			showErrorAndExit(err)
		}
	}

	if err != nil {
		showErrorAndExit(err)
	}

	parser.SetDefault()

	subcmd := parser.GetSubcommand()
	subparser := subcmd.GetSubParser()
	if err != nil {
		if subparser != nil {
			fmt.Print(subparser.Usage())
		} else {
			fmt.Print(parser.Usage())
		}
		showErrorAndExit(err)
	}

	suboptions := subparser.Options()
	if opts.SUBCOMMAND == "help" {
		err = subcmd.Invoke(suboptions)
	} else {
		var session *mcclient.ClientSession
		session, err = newClientSession(opts)
		if err != nil {
			showErrorAndExit(err)
		}
		err = subcmd.Invoke(session, suboptions)
	}
	if err != nil {
		showErrorAndExit(err)
	}
}
