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
	"time"
)

type Options struct {
	ProxyAgentId               string
	ProxyAgentInitWait         string `help:"duration to try and wait for init" default:"15s"`
	proxyAgentInitWaitDuration time.Duration

	APISyncIntervalSeconds int `default:"10"`

	APIListBatchSize int `default:"1024"`
}

func (opts *Options) GetProxyAgentInitWaitDuration() time.Duration {
	return opts.proxyAgentInitWaitDuration
}

func (opts *Options) ValidateThenInit() error {
	if opts.ProxyAgentId == "" {
		return fmt.Errorf("empty proxy_agent_id")
	}

	if d, err := time.ParseDuration(opts.ProxyAgentInitWait); err != nil {
		return fmt.Errorf("parse proxy_agent_init_wait: %v", err)
	} else {
		opts.proxyAgentInitWaitDuration = d
	}

	if opts.APIListBatchSize <= 20 {
		opts.APIListBatchSize = 20
	}
	if opts.APISyncIntervalSeconds <= 10 {
		opts.APISyncIntervalSeconds = 10
	}

	return nil
}
