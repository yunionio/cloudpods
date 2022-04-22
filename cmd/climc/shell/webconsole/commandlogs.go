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

package webconsole

import (
	"fmt"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/cmd/climc/shell"
	"yunion.io/x/onecloud/cmd/climc/shell/events"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules/webconsole"
)

var (
	R = shell.R
)

func init() {
	R(&events.EventListOptions{}, "webconsole-commandlog", "Show webconsole command logs", func(s *mcclient.ClientSession, args *events.EventListOptions) {
		ret, err := webconsole.CommandLog.List(s, jsonutils.Marshal(args))
		if err != nil {
			fmt.Println(err)
		}
		shell.PrintList(ret, webconsole.CommandLog.GetColumns(s))
	})
}
