package shell

import (
	"github.com/yunionio/jsonutils"

	"github.com/yunionio/onecloud/pkg/mcclient"
	"github.com/yunionio/onecloud/pkg/mcclient/modules"
)

func init() {

	/**
	 * 操作用户的通信地址（如果用户的通信地址不存在则进行添加；如果已存在则进行修改；如果设置空则进行删除。）
	 */
	type ContactsUpdateOptions struct {
		UID         string `help:"The user you wanna add contact to (Keystone User ID)"`
		CONTACTTYPE string `help:"The contact type email|mobile" choices:"email|mobile"`
		CONTACT     string `help:"The contacts details mobile number or email address, if set it the empty str means delete"`
		Status      string `help:"Enabled or disabled contact status" choices:"enable|disable"`
	}
	R(&ContactsUpdateOptions{}, "contact-update", "Create, delete or update contact for user", func(s *mcclient.ClientSession, args *ContactsUpdateOptions) error {
		arr := jsonutils.NewArray()
		tmpObj := jsonutils.NewDict()
		tmpObj.Add(jsonutils.NewString(args.CONTACTTYPE), "contact_type")
		tmpObj.Add(jsonutils.NewString(args.CONTACT), "contact")
		if len(args.Status) > 0 {
			if args.Status == "disable" {
				tmpObj.Add(jsonutils.JSONFalse, "enabled")
			} else {
				tmpObj.Add(jsonutils.JSONTrue, "enabled")
			}
		}

		arr.Add(tmpObj)

		params := jsonutils.NewDict()
		params.Add(arr, "contacts")

		contact, err := modules.Contacts.PerformAction(s, args.UID, "update-contact", params)

		if err != nil {
			return err
		}

		printObject(contact)
		return nil
	})

	type ContactsDeleteOptions struct {
		UID         string `help:"The user you wanna add contact to (Keystone User ID)"`
		CONTACTTYPE string `help:"The contact type email|mobile" choices:"email|mobile"`
	}
	R(&ContactsDeleteOptions{}, "contact-delete", "Delete contact for user", func(s *mcclient.ClientSession, args *ContactsDeleteOptions) error {
		arr := jsonutils.NewArray()
		tmpObj := jsonutils.NewDict()
		tmpObj.Add(jsonutils.NewString(args.CONTACTTYPE), "contact_type")
		tmpObj.Add(jsonutils.NewString(""), "contact")
		arr.Add(tmpObj)
		params := jsonutils.NewDict()
		params.Add(arr, "contacts")
		contact, err := modules.Contacts.PerformAction(s, args.UID, "update-contact", params)
		if err != nil {
			return err
		}
		printObject(contact)
		return nil
	})

	/**
	 * 获得所有用户的所有通信地址列表
	 */
	type ContactsListOptions struct {
		BaseListOptions
	}
	R(&ContactsListOptions{}, "contact-list", "List all contacts for all users", func(s *mcclient.ClientSession, args *ContactsListOptions) error {
		params := FetchPagingParams(args.BaseListOptions)

		result, err := modules.Contacts.List(s, params)
		if err != nil {
			return err
		}

		printList(result, modules.Contacts.GetColumns(s))
		return nil
	})

	/**
	 * 获得一个用户全部通信地址
	 */
	type ContactsListForUserOptions struct {
		BaseListOptions
		UID string `help:"The user you wanna find contact from (Keystone User ID)"`
	}
	R(&ContactsListForUserOptions{}, "contact-show", "List all contacts for the users", func(s *mcclient.ClientSession, args *ContactsListForUserOptions) error {
		params := FetchPagingParams(args.BaseListOptions)

		result, err := modules.Contacts.Get(s, args.UID, params)
		if err != nil {
			return err
		}

		contactsStr, err := result.GetString("details")
		if err != nil {
			return err
		}

		contactsJson, err := jsonutils.ParseString(contactsStr)
		if err != nil {
			return err
		}

		contacts, err := contactsJson.GetArray()
		if err != nil {
			return err
		}

		printList(&modules.ListResult{Data: contacts}, nil)
		return nil
	})

	/**
	 * 触发验证通信地址操作
	 */
	type ContactsVerifyOptions struct {
		UID          string `help:"The user you wanna verify contact for (Keystone User ID)"`
		CONTACT_TYPE string `help:"The contact type email|mobile"`
		CONTACT      string `help:"The contacts details mobile number or email address"`
	}
	R(&ContactsVerifyOptions{}, "contact-verify-trigger", "Trigger contact verify", func(s *mcclient.ClientSession, args *ContactsVerifyOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.CONTACT_TYPE), "contact_type")
		params.Add(jsonutils.NewString(args.CONTACT), "contact")
		/*
		   if len(args.Email) > 0 {
		       params.Add(jsonutils.NewString(args.Email), "email")
		   }
		   if len(args.Mobile) > 0 {
		       params.Add(jsonutils.NewString(args.Mobile), "mobile")
		   }
		*/

		_, err := modules.Contacts.PerformAction(s, args.UID, "verify", params)
		if err != nil {
			return err
		}
		return nil
	})

	type ContactsBatchDeleteOptions struct {
		UIDS []string `help:"All user'id you wanna to delete contacts (Keystone User ID)"`
	}
	R(&ContactsBatchDeleteOptions{}, "contact-delete", "Delete all contacts for the user", func(s *mcclient.ClientSession, args *ContactsBatchDeleteOptions) error {
		arr := jsonutils.NewArray()
		for _, f := range args.UIDS {
			arr.Add(jsonutils.NewString(f))
		}

		params := jsonutils.NewDict()
		params.Add(arr, "contacts")

		contact, err := modules.Contacts.DoBatchDeleteContacts(s, params)

		if err != nil {
			return err
		}

		printObject(contact)
		return nil
	})
}
