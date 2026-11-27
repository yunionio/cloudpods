// Copyright 2026 Yunion
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

package llm

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
)

type DashboardPanelGenerateOptions struct {
	Prompt string `help:"natural language requirement for the custom panel"`
	Scope  string `help:"optional dashboard scope; omit to follow frontend view ctx.scope" choices:"system|domain|project"`
	Title  string `help:"optional panel title"`
}

func (o *DashboardPanelGenerateOptions) Params() (jsonutils.JSONObject, error) {
	if o.Prompt == "" {
		return nil, errors.Error("prompt is required")
	}
	params := jsonutils.NewDict()
	params.Set("prompt", jsonutils.NewString(o.Prompt))
	if o.Scope != "" {
		params.Set("scope", jsonutils.NewString(o.Scope))
	}
	if o.Title != "" {
		params.Set("title", jsonutils.NewString(o.Title))
	}
	return params, nil
}

type DashboardPanelValidateOptions struct {
	Source string `help:"javascript source to validate"`
}

func (o *DashboardPanelValidateOptions) Params() (jsonutils.JSONObject, error) {
	if o.Source == "" {
		return nil, errors.Error("source is required")
	}
	params := jsonutils.NewDict()
	params.Set("source", jsonutils.NewString(o.Source))
	return params, nil
}

type DashboardPanelSaveOptions struct {
	Source         string `help:"javascript source to save"`
	Scope          string `help:"optional dashboard scope; omit to follow frontend view ctx.scope" choices:"system|domain|project"`
	Title          string `help:"optional panel title"`
	Prompt         string `help:"original natural language prompt (audit summary)"`
	Name           string `help:"optional existing parameter name to update"`
	PanelParameter string `help:"optional overview panel parameter name to merge CustomJsPanel into"`
	ItemId         string `help:"optional layout item id when merging into a panel"`
}

func (o *DashboardPanelSaveOptions) Params() (jsonutils.JSONObject, error) {
	if o.Source == "" {
		return nil, errors.Error("source is required")
	}
	params := jsonutils.NewDict()
	params.Set("source", jsonutils.NewString(o.Source))
	if o.Scope != "" {
		params.Set("scope", jsonutils.NewString(o.Scope))
	}
	if o.Title != "" {
		params.Set("title", jsonutils.NewString(o.Title))
	}
	if o.Prompt != "" {
		params.Set("prompt", jsonutils.NewString(o.Prompt))
	}
	if o.Name != "" {
		params.Set("name", jsonutils.NewString(o.Name))
	}
	if o.PanelParameter != "" {
		params.Set("panel_parameter", jsonutils.NewString(o.PanelParameter))
	}
	if o.ItemId != "" {
		params.Set("item_id", jsonutils.NewString(o.ItemId))
	}
	return params, nil
}
