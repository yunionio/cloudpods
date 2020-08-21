package notifyv2

import (
	"yunion.io/x/jsonutils"
	api "yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type TemplateCreateInput struct {
		NAME         string `help:"Name"`
		ContactType  string `help:"Contact type, specifically, setting it to all means all contact type"`
		TemplateType string `help:"Template type"`
		Topic        string `help:"Template topic"`
		Content      string `help:"Template content"`
		Example      string `help:"Example for using this template"`
	}
	R(&TemplateCreateInput{}, "notify-template-create", "Create notify template", func(s *mcclient.ClientSession, args *TemplateCreateInput) error {
		input := api.TemplateCreateInput{
			ContactType:  args.ContactType,
			TemplateType: args.TemplateType,
			Topic:        args.Topic,
			Content:      args.Content,
			Example:      args.Example,
		}
		input.Name = args.NAME
		ret, err := modules.NotifyTemplate.Create(s, jsonutils.Marshal(input))
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
