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

package misc

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type NoticesListOptions struct {
		options.BaseListOptions
	}

	R(&NoticesListOptions{}, "notice-list", "list notices", func(s *mcclient.ClientSession, args *NoticesListOptions) error {
		params, err := options.ListStructToParams(args)
		if err != nil {
			return err
		}

		result, err := modules.Notice.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.Notice.GetColumns(s))
		return nil
	})

	type NoticesCreateOptions struct {
		TITLE   string `help:"The notice title"`
		CONTENT string `help:"The notice content"`
	}

	R(&NoticesCreateOptions{}, "notice-create", "create a notice", func(s *mcclient.ClientSession, args *NoticesCreateOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.TITLE), "title")
		params.Add(jsonutils.NewString(args.CONTENT), "content")

		notice, err := modules.Notice.Create(s, params)
		if err != nil {
			return err
		}
		printObject(notice)
		return nil
	})

	type NoticesUpdateOptions struct {
		ID      string `help:"ID of notice to update"`
		Title   string `help:"The notice title"`
		Content string `help:"The notice content"`
	}

	R(&NoticesUpdateOptions{}, "notice-update", "update notice", func(s *mcclient.ClientSession, args *NoticesUpdateOptions) error {
		params := jsonutils.NewDict()
		if len(args.Title) > 0 {
			params.Add(jsonutils.NewString(args.Title), "title")
		}

		if len(args.Content) > 0 {
			params.Add(jsonutils.NewString(args.Content), "content")
		}

		notice, err := modules.Notice.Update(s, args.ID, params)
		if err != nil {
			return err
		}
		printObject(notice)
		return nil
	})

	type NoticesDeleteOptions struct {
		ID string `help:"ID of notice to update"`
	}

	R(&NoticesDeleteOptions{}, "notice-delete", "delete notice", func(s *mcclient.ClientSession, args *NoticesDeleteOptions) error {
		notice, err := modules.Notice.Delete(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(notice)
		return nil
	})
}
