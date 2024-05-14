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

type TopicUpdateOptions struct {
	ID          string
	AdvanceDays []int
}

func (opts *TopicUpdateOptions) Params() (jsonutils.JSONObject, error) {
	d := jsonutils.NewDict()
	d.Set("advance_days", jsonutils.Marshal(opts.AdvanceDays))
	return d, nil
}

func (so *TopicUpdateOptions) GetId() string {
	return so.ID
}

type STopicAddActionInput struct {
	ID        string
	ACTION_ID string
}

func (opt *STopicAddActionInput) GetId() string {
	return opt.ID
}

func (rl *STopicAddActionInput) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(rl)
}

type STopicAddResourceInput struct {
	ID          string
	RESOURCE_ID string
}

func (opt *STopicAddResourceInput) GetId() string {
	return opt.ID
}

func (rl *STopicAddResourceInput) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(rl)
}
