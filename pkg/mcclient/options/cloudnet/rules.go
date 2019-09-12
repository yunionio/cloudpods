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

package cloudnet

import (
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type RuleCreateOptions struct {
	NAME string

	Router string `required:"true"`

	MatchSrcNet    string
	MatchDestNet   string
	MatchProto     string
	MatchSrcPort   int
	MatchDestPort  int
	MatchInIfname  string
	MatchOutIfname string

	Action        string
	ActionOptions string
}

type RuleGetOptions struct {
	ID string `json:"-"`
}

type RuleUpdateOptions struct {
	ID   string `json:"-"`
	Name string

	MatchSrcNet    string
	MatchDestNet   string
	MatchProto     string
	MatchSrcPort   int `json:",omitzero"`
	MatchDestPort  int `json:",omitzero"`
	MatchInIfname  string
	MatchOutIfname string

	Action        string
	ActionOptions string
}

type RuleDeleteOptions struct {
	ID string `json:"-"`
}

type RuleListOptions struct {
	options.BaseListOptions

	Router string

	MatchSrcNet    string
	MatchDestNet   string
	MatchProto     string
	MatchSrcPort   int `json:",omitzero"`
	MatchDestPort  int `json:",omitzero"`
	MatchInIfname  string
	MatchOutIfname string

	Action        string
	ActionOptions string
}
