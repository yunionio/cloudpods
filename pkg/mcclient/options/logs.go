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

package options

import "yunion.io/x/jsonutils"

type EventSplitableOptions struct {
	ID      string `choices:"splitable|splitable-export"`
	Service string `help:"service" choices:"compute|identity|image|log|cloudevent|monitor|notify" default:"compute"`
	Table   string `help:"when id is splitable-export table must be input"`
}

func (self *EventSplitableOptions) GetId() string {
	return self.ID
}

func (self *EventSplitableOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(self), nil
}

type EventPurgeSplitableOptions struct {
	Service string `help:"service" choices:"compute|identity|image|log|cloudevent|monitor|notify" default:"compute"`
	Tables  []string
}

func (self *EventPurgeSplitableOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(self), nil
}
