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
	"time"

	"yunion.io/x/onecloud/pkg/multicloud/huawei"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type EventListOptions struct {
		Start time.Time
		End   time.Time
	}
	shellutils.R(&EventListOptions{}, "event-list", "List events", func(cli *huawei.SRegion, args *EventListOptions) error {
		events, err := cli.GetEvents(args.Start, args.End)
		if err != nil {
			return err
		}
		printList(events, 0, 0, 0, []string{})
		return nil
	})
}
