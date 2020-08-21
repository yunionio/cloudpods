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

package notifyv2

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type ReceiverListOptions struct {
		options.BaseListOptions
		UId                 string `help:"user id in keystone"`
		UName               string `help:"user name in keystone"`
		EnabledContactType  string `help:"enabled contact type"`
		VerifiedContactType string `help:"verified contact type"`
	}
	R(&ReceiverListOptions{}, "notify-receiver-list", "List notify receiver", func(s *mcclient.ClientSession, args *ReceiverListOptions) error {
		params, err := options.ListStructToParams(args)
		if err != nil {
			return err
		}
		result, err := modules.NotifyReceiver.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.NotifyReceiver.GetColumns(s))
		return nil
	})
	type ReceiverCreateOptions struct {
		UID                 string   `help:"user id in keystone"`
		Email               string   `help:"email of receiver"`
		Mobile              string   `help:"mobile of receiver"`
		EnabledContactTypes []string `help:"enabled contact type"`
	}
	R(&ReceiverCreateOptions{}, "notify-receiver-create", "Create notify receiver", func(s *mcclient.ClientSession, args *ReceiverCreateOptions) error {
		params := jsonutils.Marshal(args).(*jsonutils.JSONDict)
		receiver, err := modules.NotifyReceiver.Create(s, params)
		if err != nil {
			return err
		}
		printObject(receiver)
		return nil
	})
	type ReceiverUpdateInput struct {
		ID                 string   `help:"Id or Name of receiver"`
		Email              string   `help:"email of receiver"`
		Mobile             string   `help:"mobile of receiver"`
		EnabledContactType []string `help:"enabled contact type"`
	}
	R(&ReceiverUpdateInput{}, "notify-receiver-update", "Update notify receiver", func(s *mcclient.ClientSession, args *ReceiverUpdateInput) error {
		params := jsonutils.NewDict()
		if len(args.Email) > 0 {
			params.Set("email", jsonutils.NewString(args.Email))
		}
		if len(args.Mobile) > 0 {
			params.Set("mobile", jsonutils.NewString(args.Mobile))
		}
		if len(args.EnabledContactType) > 0 {
			params.Set("enabled_contact_types", jsonutils.NewStringArray(args.EnabledContactType))
		}
		ret, err := modules.NotifyReceiver.Update(s, args.ID, params)
		if err != nil {
			return err
		}
		printObject(ret)
		return nil
	})
	type ReceiverOptions struct {
		ID string `help:"Id or Name of receiver"`
	}
	R(&ReceiverOptions{}, "notify-receiver-show", "Show notify receiver", func(s *mcclient.ClientSession, args *ReceiverOptions) error {
		ret, err := modules.NotifyReceiver.Get(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(ret)
		return nil
	})
	R(&ReceiverOptions{}, "notify-receiver-delete", "Delete notify receiver", func(s *mcclient.ClientSession, args *ReceiverOptions) error {
		receiver, err := modules.NotifyReceiver.Delete(s, args.ID, jsonutils.NewDict())
		if err != nil {
			return err
		}
		printObject(receiver)
		return nil
	})
	R(&ReceiverOptions{}, "notify-receiver-enable", "Enable notify receiver", func(s *mcclient.ClientSession, args *ReceiverOptions) error {
		ret, err := modules.NotifyReceiver.PerformAction(s, args.ID, "enable", jsonutils.NewDict())
		if err != nil {
			return err
		}
		printObject(ret)
		return nil
	})
	R(&ReceiverOptions{}, "notify-receiver-disable", "Disable notify receiver", func(s *mcclient.ClientSession, args *ReceiverOptions) error {
		ret, err := modules.NotifyReceiver.PerformAction(s, args.ID, "disable", jsonutils.NewDict())
		if err != nil {
			return err
		}
		printObject(ret)
		return nil
	})
	type ReceiverTriggerVerifyInput struct {
		ID          string `help:"Id or Name of receiver"`
		ContactType string `help:"Contact type to trigger verify" choices:"email|mobile"`
	}
	R(&ReceiverTriggerVerifyInput{}, "notify-receiver-trigger-verify", "Trigger verification for receiver about some contact", func(s *mcclient.ClientSession, args *ReceiverTriggerVerifyInput) error {
		params := jsonutils.NewDict()
		params.Set("contact_type", jsonutils.NewString(args.ContactType))
		ret, err := modules.NotifyReceiver.PerformAction(s, args.ID, "trigger-verify", params)
		if err != nil {
			return err
		}
		printObject(ret)
		return nil
	})
	type ReceiverVerifyInput struct {
		ID          string `help:"Id or Name of receiver"`
		ContactType string `help:"Contact type to trigger verify" choices:"email|mobile"`
		Token       string `help:"Token from verify message sent to you"`
	}
	R(&ReceiverVerifyInput{}, "notify-receiver-verify", "Verify receiver about some contact type", func(s *mcclient.ClientSession, args *ReceiverVerifyInput) error {
		params := jsonutils.NewDict()
		params.Set("contact_type", jsonutils.NewString(args.ContactType))
		params.Set("token", jsonutils.NewString(args.Token))
		ret, err := modules.NotifyReceiver.PerformAction(s, args.ID, "verify", params)
		if err != nil {
			return err
		}
		printObject(ret)
		return nil
	})
	R(&ReceiverOptions{}, "notify-receiver-enable", "Enable receiver", func(s *mcclient.ClientSession, args *ReceiverOptions) error {
		ret, err := modules.NotifyReceiver.PerformAction(s, args.ID, "enable", nil)
		if err != nil {
			return err
		}
		printObject(ret)
		return nil
	})
	R(&ReceiverOptions{}, "notify-receiver-disable", "Disable receiver", func(s *mcclient.ClientSession, args *ReceiverOptions) error {
		ret, err := modules.NotifyReceiver.PerformAction(s, args.ID, "disable", nil)
		if err != nil {
			return err
		}
		printObject(ret)
		return nil
	})
}
