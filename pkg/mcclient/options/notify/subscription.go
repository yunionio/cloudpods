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

// Copyright 2019 Yunion
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

	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type TopicListOptions struct {
	options.BaseListOptions
}

func (opts *TopicListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(opts)
}

type TopicOptions struct {
	ID string `help:"Id or Name of topic"`
}

func (so *TopicOptions) GetId() string {
	return so.ID
}

func (so *TopicOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type SubscriberCreateOptions struct {
	TopicId               string   `positional:"true"`
	ResourceScope         string   `positional:"true" choices:"system|domain|project"`
	ResourceAttributionId string   `help:"project id or domain id of resource"`
	Type                  string   `positional:"true" choices:"receiver|robot|role"`
	Receivers             []string `help:"required if type is 'receiver'"`
	Role                  string   `help:"required if type is 'role'"`
	RoleScope             string   `help:"required if type is 'role'"`
	Robot                 string   `help:"required if type is 'robot'"`
	Scope                 string   `positional:"true"`
}

func (sc *SubscriberCreateOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(sc), nil
}

type SubscriberListOptions struct {
	options.BaseListOptions
	TopicId       string
	ResourceScope string `choices:"system|domain|project"`
	Type          string `choices:"receiver|robot|role"`
	SCOPE         string `choices:"system|domain"`
}

func (sl *SubscriberListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(sl)
}

type SubscriberOptions struct {
	ID string
}

func (s *SubscriberOptions) GetId() string {
	return s.ID
}

func (s *SubscriberOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type SubscriberChangeOptions struct {
	SubscriberOptions
	Receivers []string
	Role      string
	RoleScope string
	Robot     string
}

func (ssr *SubscriberChangeOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.Marshal(ssr)
	params.(*jsonutils.JSONDict).Remove("id")
	return params, nil
}
