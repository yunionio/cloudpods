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

package events

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules/cloudevent"
	"yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/mcclient/modules/identity"
	"yunion.io/x/onecloud/pkg/mcclient/modules/image"
	"yunion.io/x/onecloud/pkg/mcclient/modules/logger"
)

func init() {
	type EventSplitableOptions struct {
		Service string `help:"service" choices:"compute|identity|image|log|cloudevent" default:"compute"`
	}
	R(&EventSplitableOptions{}, "logs-splitable", "Show splitable info of event table", func(s *mcclient.ClientSession, args *EventSplitableOptions) error {
		var results jsonutils.JSONObject
		var err error
		switch args.Service {
		case "identity":
			results, err = identity.IdentityLogs.Get(s, "splitable", nil)
		case "image":
			results, err = image.ImageLogs.Get(s, "splitable", nil)
		case "log":
			results, err = logger.Actions.Get(s, "splitable", nil)
		case "cloudevent":
			results, err = cloudevent.Cloudevents.Get(s, "splitable", nil)
		default:
			results, err = compute.Logs.Get(s, "splitable", nil)
		}
		if err != nil {
			return err
		}
		tables, err := results.GetArray()
		if err != nil {
			return err
		}
		listResult := &modulebase.ListResult{
			Data: tables,
		}
		printList(listResult, nil)
		return nil
	})
	R(&EventSplitableOptions{}, "logs-purge", "Purge obsolete splitable of event table", func(s *mcclient.ClientSession, args *EventSplitableOptions) error {
		var results jsonutils.JSONObject
		var err error
		switch args.Service {
		case "identity":
			results, err = identity.IdentityLogs.PerformClassAction(s, "purge-splitable", nil)
		case "image":
			results, err = image.ImageLogs.PerformClassAction(s, "purge-splitable", nil)
		case "log":
			results, err = logger.Actions.PerformClassAction(s, "purge-splitable", nil)
		case "cloudevent":
			results, err = cloudevent.Cloudevents.PerformClassAction(s, "purge-splitable", nil)
		default:
			results, err = compute.Logs.PerformClassAction(s, "purge-splitable", nil)
		}
		if err != nil {
			return err
		}
		printObject(results)
		return nil
	})
}
