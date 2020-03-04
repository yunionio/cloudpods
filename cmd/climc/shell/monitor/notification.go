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
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules/monitor"
	options "yunion.io/x/onecloud/pkg/mcclient/options/monitor"
)

func init() {
	nN := cmdN("notification")
	R(&options.NotificationListOptions{}, nN("list"), "List all alert notification",
		func(s *mcclient.ClientSession, args *options.NotificationListOptions) error {
			params, err := args.Params()
			if err != nil {
				return err
			}
			ret, err := monitor.Notifications.List(s, params)
			if err != nil {
				return err
			}
			printList(ret, monitor.Notifications.GetColumns(s))
			return nil
		})

	R(&options.NotificationDingDingCreateOptions{}, nN("create-dingding"),
		"Create dingding alert notification",
		func(s *mcclient.ClientSession, args *options.NotificationDingDingCreateOptions) error {
			params, err := args.Params()
			if err != nil {
				return err
			}
			ret, err := monitor.Notifications.Create(s, params.JSON(params))
			if err != nil {
				return err
			}
			printObject(ret)
			return nil
		})

	R(&options.NotificationFeishuCreateOptions{}, nN("create-feishu"),
		"Create feishu alert notification",
		func(s *mcclient.ClientSession, args *options.NotificationFeishuCreateOptions) error {
			params, err := args.Params()
			if err != nil {
				return err
			}
			ret, err := monitor.Notifications.Create(s, params.JSON(params))
			if err != nil {
				return err
			}
			printObject(ret)
			return nil
		})

	R(&options.NotificationShowOptions{}, nN("show"), "Show alert notification",
		func(s *mcclient.ClientSession, args *options.NotificationShowOptions) error {
			ret, err := monitor.Notifications.Get(s, args.ID, nil)
			if err != nil {
				return err
			}
			printObject(ret)
			return nil
		})

	R(&options.NotificationUpdateOptions{}, nN("update"), "Update alert notification",
		func(s *mcclient.ClientSession, args *options.NotificationUpdateOptions) error {
			params, err := args.Params()
			if err != nil {
				return err
			}
			ret, err := monitor.Notifications.Update(s, args.ID, params.JSON(params))
			if err != nil {
				return err
			}
			printObject(ret)
			return nil
		})

	R(&options.NotificationDeleteOptions{}, nN("delete"), "Show delete notification",
		func(s *mcclient.ClientSession, args *options.NotificationDeleteOptions) error {
			ret := monitor.Notifications.BatchDelete(s, args.ID, nil)
			printBatchResults(ret, monitor.Notifications.GetColumns(s))
			return nil
		})
}
