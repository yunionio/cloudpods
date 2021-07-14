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

package options

import (
	"fmt"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/structarg"

	"yunion.io/x/onecloud/pkg/util/shellutils"
)

type CloudMonOptions struct {
	structarg.BaseOptions

	Region       string `help:"Region name"`
	EndpointType string `default:"internalURL" help:"Defaults to internalURL" choices:"publicURL|internalURL|adminURL"`
	ApiVersion   string `help:"override default modules service api version"`

	Debug    bool   `help:"Show debug information"`
	Timeout  int    `default:"600" help:"Number of seconds to wait for a response"`
	Insecure bool   `default:"true" help:"Allow skip server cert verification if URL is https" short-token:"k"`
	CertFile string `help:"certificate file"`
	KeyFile  string `help:"private key file"`

	AuthURL            string `help:"Keystone auth URL" alias:"auth-uri"`
	AdminUser          string `help:"Admin username"`
	AdminDomain        string `help:"Admin user domain"`
	AdminPassword      string `help:"Admin password" alias:"admin-passwd"`
	AdminProject       string `help:"Admin project" default:"system" alias:"admin-tenant-name"`
	AdminProjectDomain string `help:"Domain of Admin project"`

	SessionEndpointType string `help:"Client session end point type"`

	InfluxDatabase string `help:"influxdb database name, default telegraf" default:"telegraf"`

	SUBCOMMAND string `help:"climc subcommand" subcommand:"true"`
}

func GetArgumentParser() (*structarg.ArgumentParser, error) {
	parse, e := structarg.NewArgumentParser(&Options,
		"cloudmon",
		`Command-line interface to collect cloud monitoring data.`,
		`See "cloudmon help COMMAND" for help on a specific command.`)
	if e != nil {
		return nil, e
	}
	subcmd := parse.GetSubcommand()
	if subcmd == nil {
		return nil, errors.Error("No subcommand argument")
	}
	type HelpOptions struct {
		SUBCOMMAND string `help:"Sub-command name"`
	}
	shellutils.R(&HelpOptions{}, "help", "Show help information of any subcommand", func(suboptions *HelpOptions) error {
		helpstr, e := subcmd.SubHelpString(suboptions.SUBCOMMAND)
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

var (
	Options CloudMonOptions
)
