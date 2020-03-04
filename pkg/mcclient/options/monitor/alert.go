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

package monitor

import (
	"fmt"
	"time"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type AlertListOptions struct {
	options.BaseListOptions
}

type AlertShowOptions struct {
	ID string `help:"ID or name of the alert" json:"-"`
}

type AlertDeleteOptions struct {
	ID []string `help:"ID of alert to delete"`
}

type AlertUpdateOptions struct {
	ID        string `help:"ID or name of the alert"`
	Name      string `help:"Update alert name"`
	Frequency string `help:"Alert execute frequency, e.g. '5m', '1h'"`
}

func (opt AlertUpdateOptions) Params() (*monitor.AlertUpdateInput, error) {
	input := new(monitor.AlertUpdateInput)
	if opt.Name != "" {
		input.Name = &opt.Name
	}
	if opt.Frequency != "" {
		freq, err := time.ParseDuration(opt.Frequency)
		if err != nil {
			return nil, fmt.Errorf("Invalid frequency time format %s: %v", opt.Frequency, err)
		}
		f := int64(freq / time.Second)
		input.Frequency = &f
	}
	return input, nil
}
