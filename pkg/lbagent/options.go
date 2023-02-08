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

package lbagent

import (
	"fmt"
	"os"
	"path/filepath"

	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
	agentutils "yunion.io/x/onecloud/pkg/lbagent/utils"
)

type LbagentCommonOptions struct {
	common_options.HostCommonOptions

	// ApiLbagentId                  string `require:"true"`
	ApiLbagentHbInterval          int `default:"10"`
	ApiLbagentHbTimeoutRelaxation int `default:"120" help:"If agent is to stale out in specified seconds in the future, consider it staled to avoid race condition when doing incremental api data fetch"`

	ApiSyncIntervalSeconds  int `default:"10"`
	ApiRunDelayMilliseconds int `default:"10"`

	ApiListBatchSize int `default:"1024"`

	DataPreserveN int `default:"8" help:"number of recent data to preserve on disk"`

	BaseDataDir      string // `required:"true"`
	apiDataStoreDir  string
	haproxyConfigDir string
	haproxyRunDir    string
	haproxyShareDir  string
	// haStateChan      chan string

	KeepalivedBin string `default:"keepalived"`
	HaproxyBin    string `default:"haproxy"`
	GobetweenBin  string `default:"gobetween"`
	TelegrafBin   string `default:"telegraf"`
}

type Options struct {
	LbagentCommonOptions

	CommonConfigFile string `help:"common config file for container"`

	ListenInterface string `help:"listening interface of lbagent" default:"eth0"`
	AccessIp        string `help:"access ip of lbagent, if there are multiple IPs on listen interface"`
}

func (opts *Options) ValidateThenInit() error {
	if opts.ApiListBatchSize <= 0 {
		return fmt.Errorf("negative api batch list size: %d",
			opts.ApiListBatchSize)
	}
	if err := opts.initDirs(); err != nil {
		return err
	}

	return nil
}

func (opts *Options) initDirs() error {
	opts.apiDataStoreDir = filepath.Join(opts.BaseDataDir, "data")
	opts.haproxyConfigDir = filepath.Join(opts.BaseDataDir, "configs")
	opts.haproxyRunDir = filepath.Join(opts.BaseDataDir, "run")
	opts.haproxyShareDir = filepath.Join(opts.BaseDataDir, "share")
	dirs := []string{
		opts.apiDataStoreDir,
		opts.haproxyConfigDir,
		opts.haproxyRunDir,
		opts.haproxyShareDir,
	}
	for _, dir := range dirs {
		err := os.MkdirAll(dir, agentutils.FileModeDir)
		if err != nil {
			return fmt.Errorf("mkdir -p %q: %s",
				dir, err)
		}
	}

	return nil
}
