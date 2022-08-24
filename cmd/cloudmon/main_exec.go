package main

import (
	"context"
	"fmt"
	"os"
	"path"
	"time"

	"yunion.io/x/log"
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
		options.ReqTimeout,
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
	)

	return session, nil
}

func TestCron() {
	parser, err := options.GetArgumentParser()
	if err != nil {
		showErrorAndExit(err)
	}

	err = parser.ParseArgs2(os.Args[1:], false, false)

	opts := parser.Options().(*options.SubCloudMonOptions)

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
	if opts.Subcommand == "help" {
		err = subcmd.Invoke(suboptions)
	} else {
		timout := suboptions.(*options.ReportOptions).Timeout
		endChan := make(chan int, 1)
		go func() {
			ticker := time.NewTicker(time.Duration(timout) * time.Second)
			for {
				select {
				case <-ticker.C:
					log.Errorf("cmd: %s,provider: %v,end due to timeout: %d s", opts.Subcommand,
						suboptions.(*options.ReportOptions).Provider, suboptions.(*options.ReportOptions).Timeout)
					os.Exit(3)
				case <-endChan:
				}
			}
		}()

		var session *mcclient.ClientSession
		session, err = newClientSession(&opts.CloudMonOptions)
		if err != nil {
			showErrorAndExit(err)
		}
		err = subcmd.Invoke(session, suboptions)
		endChan <- 1
	}
	if err != nil {
		showErrorAndExit(err)
	}
}

//func main() {
//	TestCron()
//}
