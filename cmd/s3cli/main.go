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

	"yunion.io/x/structarg"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud/objectstore"
	"yunion.io/x/onecloud/pkg/multicloud/objectstore/ceph"
	_ "yunion.io/x/onecloud/pkg/multicloud/objectstore/shell"
	"yunion.io/x/onecloud/pkg/multicloud/objectstore/xsky"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

type BaseOptions struct {
	Debug      bool   `help:"debug mode"`
	Help       bool   `help:"Show help"`
	AccessUrl  string `help:"Access url" default:"$S3_ACCESS_URL"`
	AccessKey  string `help:"Access key" default:"$S3_ACCESS_KEY"`
	Secret     string `help:"Secret" default:"$S3_SECRET"`
	Backend    string `help:"Backend driver" default:"$S3_BACKEND"`
	SUBCOMMAND string `help:"s3cli subcommand" subcommand:"true"`
}

func getSubcommandParser() (*structarg.ArgumentParser, error) {
	parse, e := structarg.NewArgumentParser(&BaseOptions{},
		"s3cli",
		"Command-line interface to standard S3 API.",
		`See "s3cli help COMMAND" for help on a specific command.`)

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

func newClient(options *BaseOptions) (cloudprovider.ICloudRegion, error) {
	if len(options.AccessUrl) == 0 {
		return nil, fmt.Errorf("Missing accessUrl")
	}

	if len(options.AccessKey) == 0 {
		return nil, fmt.Errorf("Missing accessKey")
	}

	if len(options.Secret) == 0 {
		return nil, fmt.Errorf("Missing secret")
	}

	if options.Backend == api.CLOUD_PROVIDER_CEPH {
		return ceph.NewCephRados("", "", options.AccessUrl, options.AccessKey, options.Secret, options.Debug)
	} else if options.Backend == api.CLOUD_PROVIDER_XSKY {
		return xsky.NewXskyClient("", "", options.AccessUrl, options.AccessKey, options.Secret, options.Debug)
	}
	return objectstore.NewObjectStoreClient("", "", options.AccessUrl, options.AccessKey, options.Secret, options.Debug)
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
				var client cloudprovider.ICloudRegion
				client, e = newClient(options)
				if e != nil {
					showErrorAndExit(e)
				}
				e = subcmd.Invoke(client, suboptions)
			}
			if e != nil {
				showErrorAndExit(e)
			}
		}
	}
}
