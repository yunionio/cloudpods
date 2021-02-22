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

package yunionagent

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type NoticeListOptions struct {
	options.BaseListOptions

	VisibleScope string `help:"visiable scope" choices:"system|domain"`
}

func (n *NoticeListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(n)
}

type NoticeCreateOptions struct {
	TITLE   string `help:"The notice title" json:"title"`
	CONTENT string `help:"The notice content" json:"content"`
}

func (n *NoticeCreateOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(n), nil
}

type NoticeOptions struct {
	ID string `help:"ID of notice to update"`
}

func (n *NoticeOptions) GetId() string {
	return n.ID
}

func (n *NoticeOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type SnoticeUpdateOptions struct {
	Title   string `help:"The notice title"`
	Content string `help:"The notice content"`
}

type NoticesUpdateOptions struct {
	NoticeOptions
	SnoticeUpdateOptions
}

func (n *NoticesUpdateOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(n.SnoticeUpdateOptions), nil
}
