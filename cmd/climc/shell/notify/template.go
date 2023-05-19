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

package notify

import (
	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/notify"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type TemplateCreateInput struct {
		ContactType  string `help:"Contact type, specifically, setting it to all means all contact type"`
		TemplateType string `help:"Template type"`
		Topic        string `help:"Template topic"`
		Content      string `help:"Template content"`
		Example      string `help:"Example for using this template"`
		Lang         string `help:"Language of Template"`
	}
	R(&TemplateCreateInput{}, "notify-template-create", "Create notify template", func(s *mcclient.ClientSession, args *TemplateCreateInput) error {
		input := api.TemplateCreateInput{
			ContactType:  args.ContactType,
			TemplateType: args.TemplateType,
			Topic:        args.Topic,
			Content:      args.Content,
			Example:      args.Example,
			Lang:         args.Lang,
		}
		ret, err := modules.NotifyTemplate.Create(s, jsonutils.Marshal(input))
		if err != nil {
			return err
		}
		printObject(ret)
		return nil
	})
	type TemplateSaveInput struct {
		ContactType string `help:"contact type" positional:"true"`
		Force       bool   `help:"Whether to force the update of existing templates"`

		TemplateType string `help:"Template type"`
		Topic        string `help:"Template topic"`
		Content      string `help:"Template content"`
		Example      string `help:"Example for using this template"`
		Lang         string `help:"Language of Template"`
	}
	R(&TemplateSaveInput{}, "notify-template-save", "Save notify templates", func(s *mcclient.ClientSession, args *TemplateSaveInput) error {
		templates := []api.TemplateCreateInput{
			{
				ContactType:  args.ContactType,
				TemplateType: args.TemplateType,
				Topic:        args.Topic,
				Content:      args.Content,
				Example:      args.Example,
				Lang:         args.Lang,
			},
		}
		input := api.TemplateManagerSaveInput{
			ContactType: args.ContactType,
			Templates:   templates,
			Force:       args.Force,
		}
		ret, err := modules.NotifyTemplate.PerformClassAction(s, "save", jsonutils.Marshal(input))
		if err != nil {
			return err
		}
		printObject(ret)
		return nil
	})
	type TemplateListInput struct {
		options.BaseListOptions

		ContactType  string `help:"Contact type"`
		TemplateType string `help:"Template type"`
		Topic        string `help:"Topic"`
		Lang         string `help:"Lang"`
	}
	R(&TemplateListInput{}, "notify-template-list", "List notify template", func(s *mcclient.ClientSession, args *TemplateListInput) error {
		params, err := options.ListStructToParams(args)
		if err != nil {
			return err
		}
		list, err := modules.NotifyTemplate.List(s, params)
		if err != nil {
			return err
		}
		printList(list, modules.NotifyTemplate.GetColumns(s))
		return nil
	})
	type TemplateInput struct {
		ID string `help:"id or name of template"`
	}
	R(&TemplateInput{}, "notify-template-get", "Get notify template", func(s *mcclient.ClientSession, args *TemplateInput) error {
		ret, err := modules.NotifyTemplate.Get(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(ret)
		return nil
	})
	type TemplateUpdateInput struct {
		ID      string `help:"id or name of template"`
		Content string `help:"Template content"`
		Example string `help:"Example for using this template"`
	}
	R(&TemplateUpdateInput{}, "notify-template-update", "Update notify template", func(s *mcclient.ClientSession, args *TemplateUpdateInput) error {
		input := api.TemplateUpdateInput{
			Content: args.Content,
			Example: args.Example,
		}
		ret, err := modules.NotifyTemplate.Update(s, args.ID, jsonutils.Marshal(input))
		if err != nil {
			return err
		}
		printObject(ret)
		return nil
	})
	R(&TemplateInput{}, "notify-template-delete", "Delete notify template", func(s *mcclient.ClientSession, args *TemplateInput) error {
		ret, err := modules.NotifyTemplate.Delete(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(ret)
		return nil
	})
}
