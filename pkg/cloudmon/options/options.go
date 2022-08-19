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

	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

type CloudMonOptions struct {
	common_options.CommonOptions
	ReportOptions

	EndpointType string `default:"internalURL" help:"Defaults to internalURL" choices:"publicURL|internalURL|adminURL"`
	ApiVersion   string `help:"override default modules service api version"`

	ReqTimeout int    `default:"600" help:"Number of seconds to wait for a response"`
	Insecure   bool   `default:"true" help:"Allow skip server cert verification if URL is https" short-token:"k"`
	CertFile   string `help:"certificate file"`
	KeyFile    string `help:"private key file"`

	InfluxDatabase string `help:"influxdb database name, default telegraf" default:"telegraf"`
}

type SubCloudMonOptions struct {
	CloudMonOptions
	Subcommand string `help:"climc subcommand" subcommand:"true"`
}

type ReportOptions struct {
	Batch                      int   `help:"batch"`
	Count                      int   `help:"count" json:"count"`
	CloudproviderSyncInterval  int64 `help:"CloudproviderSyncInterval unit:minute" default:"30"`
	AlertRecordHistoryInterval int64 `help:"AlertRecordHistoryInterval unit:day"  default:"1"`
	// 定时执行间隔，同时也会影响metric拉取间隔
	Interval  string   `help:"interval" default:"6" unit:"minute"`
	Timeout   int64    `help:"command timeout unit:second" default:"10"`
	SinceTime string   `help:"sinceTime"`
	EndTime   string   `help:"endTime"`
	Provider  []string `help:"List objects from the provider" choices:"VMware|Aliyun|Qcloud|Azure|Aws|Huawei
|ZStack|Google|Apsara|JDcloud|Ecloud|HCSO|BingoCloud" json:"provider,omitempty"`
	MetricInterval string `help:"metric interval eg:PT1M"`
	PingProbeOptions
}

type PingProbeOptions struct {
	Debug         bool `help:"debug"`
	ProbeCount    int  `help:"probe count, default is 3" default:"3"`
	TimeoutSecond int  `help:"probe timeout in second, default is 1 second" default:"1"`

	DisablePingProbe bool `help:"enable ping probe"`
}

func GetArgumentParser() (*structarg.ArgumentParser, error) {
	parse, e := structarg.NewArgumentParser(&SubOptions,
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
	Options    CloudMonOptions
	SubOptions SubCloudMonOptions
)
