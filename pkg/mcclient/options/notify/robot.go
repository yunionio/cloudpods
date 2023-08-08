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
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"

	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type RobotListOptions struct {
	options.BaseListOptions
	Lang    string
	Type    string `choices:"feishu|dingtalk|workwx|webhook"`
	Enabled *bool
}

func (rl *RobotListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(rl)
}

type RobotCreateOptions struct {
	NAME        string
	Type        string `choices:"feishu|dingtalk|workwx|webhook"`
	Address     string
	Lang        string
	Header      string
	Body        string
	MsgKey      string
	UseTemplate bool `help:"just for webhook"`
}

func (rc *RobotCreateOptions) Params() (jsonutils.JSONObject, error) {
	dict := jsonutils.NewDict()
	jsonutils.Update(&dict, rc)
	if len(rc.Header) > 0 {
		header, err := jsonutils.Parse([]byte(rc.Header))
		if err != nil {
			return nil, errors.Wrap(err, "parse header")
		}
		dict.Set("header", header)
	}

	if len(rc.Body) > 0 {
		body, err := jsonutils.Parse([]byte(rc.Body))
		if err != nil {
			return nil, errors.Wrap(err, "parse body")
		}
		dict.Set("body", body)
	}

	dict.Set("use_template", jsonutils.Marshal(rc.UseTemplate))
	return dict, nil
}

type RobotOptions struct {
	ID string `help:"Id or Name of robot"`
}

func (r *RobotOptions) GetId() string {
	return r.ID
}

func (r *RobotOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type RobotUpdateOptions struct {
	RobotOptions
	SrobotUpdateOptions
}

type SrobotUpdateOptions struct {
	Address     string
	Lang        string
	Header      *string
	Body        *string
	MsgKey      string
	UseTemplate tristate.TriState
}

func (ru *RobotUpdateOptions) Params() (jsonutils.JSONObject, error) {
	dict := jsonutils.NewDict()
	jsonutils.Update(&dict, ru)
	if ru.Header != nil {
		header, err := jsonutils.Parse([]byte(*ru.Header))
		if err != nil {
			return nil, errors.Wrap(err, "parse header")
		}
		dict.Set("header", header)
	}
	if ru.Body != nil {
		body, err := jsonutils.Parse([]byte(*ru.Body))
		if err != nil {
			return nil, errors.Wrap(err, "parse body")
		}
		dict.Set("body", body)
	}
	if ru.UseTemplate.IsFalse() {
		dict.Set("use_template", jsonutils.JSONFalse)
	} else if ru.UseTemplate.IsTrue() {
		dict.Set("use_template", jsonutils.JSONTrue)
	}
	return dict, nil
}
