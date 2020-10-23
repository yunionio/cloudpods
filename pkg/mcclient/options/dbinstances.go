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
	"strings"

	"yunion.io/x/jsonutils"
)

type DBInstanceCreateOptions struct {
	NAME          string   `help:"DBInstance Name"`
	InstanceType  string   `help:"InstanceType for DBInstance"`
	VcpuCount     int      `help:"Core of cpu for DBInstance"`
	VmemSizeMb    int      `help:"Memory size of DBInstance"`
	Port          int      `help:"Port of DBInstance"`
	Category      string   `help:"Category of DBInstance"`
	Network       string   `help:"Network of DBInstance"`
	Address       string   `help:"Address of DBInstance"`
	Engine        string   `help:"Engine of DBInstance"`
	EngineVersion string   `help:"EngineVersion of DBInstance Engine"`
	StorageType   string   `help:"StorageTyep of DBInstance"`
	Secgroup      string   `help:"Secgroup name or Id for DBInstance"`
	Zone          string   `help:"ZoneId or name for DBInstance"`
	DiskSizeGB    int      `help:"Storage size for DBInstance"`
	Duration      string   `help:"Duration for DBInstance"`
	AllowDelete   *bool    `help:"not lock dbinstance" `
	Tags          []string `help:"Tags info,prefix with 'user:', eg: user:project=default" json:"-"`
}

func (opts *DBInstanceCreateOptions) Params() (*jsonutils.JSONDict, error) {
	params, err := StructToParams(opts)
	if err != nil {
		return nil, err
	}
	Tagparams := jsonutils.NewDict()
	for _, tag := range opts.Tags {
		info := strings.Split(tag, "=")
		if len(info) == 2 {
			if len(info[0]) == 0 {
				return nil, fmt.Errorf("invalidate tag info %s", tag)
			}
			Tagparams.Add(jsonutils.NewString(info[1]), info[0])
		} else if len(info) == 1 {
			Tagparams.Add(jsonutils.NewString(info[0]), info[0])
		} else {
			return nil, fmt.Errorf("invalidate tag info %s", tag)
		}
	}
	params.Add(Tagparams, "__meta__")
	return params, nil
}
