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
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type NotifyTemplateUpdateOptions struct {
		CONTACTTYPE  string `help:"the contanct type, such as 'email', 'mobile'"`
		Topic        string `help:"the topic of temlate, such as 'VERIFY', 'ALARM'"`
		TemplateType string `help:"the type of template" choices:"content|title|remote"`
		Content      string `help:"the content of template"`
	}
	R(&NotifyTemplateUpdateOptions{}, "notify-template-update", "Create, update contact for user", func(s *mcclient.ClientSession,
		args *NotifyTemplateUpdateOptions) error {
		arr := jsonutils.NewArray()
		tmpObj := jsonutils.NewDict()
		tmpObj.Add(jsonutils.NewString(args.Topic), "topic")
		tmpObj.Add(jsonutils.NewString(args.TemplateType), "template_type")
		tmpObj.Add(jsonutils.NewString(args.Content), "content")
		arr.Add(tmpObj)

		params := jsonutils.NewDict()
		params.Add(arr, "notifytemplates")

		contact, err := modules.NotifyTemplates.PerformAction(s, args.CONTACTTYPE, "update-template", params)

		if err != nil {
			return err
		}

		printObject(contact)
		return nil
	})

	type NotifyTemplateDeleteOptions struct {
		CONTACTTYPE string `help:"the contanct type, such as 'email', 'mobile'"`
		Topic       string `help:"the topic of temlate, such as 'VERIFY', 'ALARM'"`
	}

	R(&NotifyTemplateDeleteOptions{}, "notify-template-delete", "delete notify template",
		func(s *mcclient.ClientSession, args *NotifyTemplateDeleteOptions) error {

			tmpObj := jsonutils.NewDict()
			tmpObj.Add(jsonutils.NewString(args.CONTACTTYPE), "contact_type")
			tmpObj.Add(jsonutils.NewString(args.Topic), "topic")
			_, err := modules.NotifyTemplates.DeleteContents(s, tmpObj)
			if err != nil {
				return err
			}
			return nil
		})

	type NotifyTemplateListOptions struct {
		options.BaseListOptions
	}

	R(&NotifyTemplateListOptions{}, "notify-template-list", "List all notify template",
		func(s *mcclient.ClientSession, args *NotifyTemplateListOptions) error {

			params, err := args.BaseListOptions.Params()
			if err != nil {
				return err
			}
			result, err := modules.NotifyTemplates.List(s, params)
			if err != nil {
				return err
			}
			printList(result, modules.NotifyTemplates.GetColumns(s))
			return nil
		})
}
