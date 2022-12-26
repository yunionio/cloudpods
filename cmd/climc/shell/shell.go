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

package shell

import (
	"errors"

	"yunion.io/x/pkg/util/shellutils"
)

type CMD struct {
	Options  interface{}
	Command  string
	Desc     string
	Callback interface{}
}

var CommandTable []CMD = make([]CMD, 0)

var ErrEmtptyUpdate = errors.New("No valid update data")

func R(options interface{}, command string, desc string, callback interface{}) {
	shellutils.R(options, command, desc, callback)
	// CommandTable = append(CommandTable, CMD{options, command, desc, callback})
}

func InvalidUpdateError() error {
	return ErrEmtptyUpdate
}
