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

package devtool

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type ScriptListOptions struct {
	options.BaseListOptions
}

func (so *ScriptListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(so)
}

type ScriptOptions struct {
	ID string `help:"id or name of script"`
}

func (so *ScriptOptions) GetId() string {
	return so.ID
}

func (so *ScriptOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type SscriptApplyOptions struct {
	SERVERID []string `help:"server id" json:"server_id"`
}

type ScriptApplyOptions struct {
	ScriptOptions
	SscriptApplyOptions
}

func (so *ScriptApplyOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(so.SscriptApplyOptions), nil
}

type SscriptBatchApplyOptions struct {
	SERVERIDS []string `help:"server id list" json:"server_ids"`
}

type ScriptBatchApplyOptions struct {
	ScriptOptions
	SscriptBatchApplyOptions
}

func (so *ScriptBatchApplyOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(so.SscriptBatchApplyOptions), nil
}

type ScriptApplyRecordListOptions struct {
	options.BaseListOptions
	ScriptId      string
	ScriptApplyId string
	ServerId      string
}

func (so *ScriptApplyRecordListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(so)
}
