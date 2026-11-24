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

package dashboardcodegen

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/stringutils"

	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	ycmodules "yunion.io/x/onecloud/pkg/mcclient/modules/yunionconf"
)

// ValidateOnly re-runs static validation for the validate API.
func ValidateOnly(input api.DashboardPanelValidateInput) (*api.DashboardPanelValidateOutput, error) {
	result, err := ValidateSource(input.Source, input.ApiAllowlist)
	if err != nil {
		return &api.DashboardPanelValidateOutput{
			Ok:    false,
			Error: err.Error(),
		}, nil
	}
	return &api.DashboardPanelValidateOutput{
		Ok:            true,
		Entry:         result.Entry,
		ContentSha256: result.ContentSha256,
		ApiAllowlist:  result.ApiAllowlist,
		Warnings:      result.Warnings,
	}, nil
}

// Save re-validates source and writes a user-namespace yunionconf parameter.
func Save(ctx context.Context, userCred mcclient.TokenCredential, input api.DashboardPanelSaveInput) (*api.DashboardPanelSaveOutput, error) {
	if userCred == nil {
		return nil, errors.Error("unauthorized")
	}
	scope, err := NormalizeScope(input.Scope)
	if err != nil {
		return nil, err
	}
	if utf8.RuneCountInString(strings.TrimSpace(input.Title)) > api.DashboardPanelMaxTitleRunes {
		return nil, errors.Errorf("title exceeds %d characters", api.DashboardPanelMaxTitleRunes)
	}
	if utf8.RuneCountInString(input.Prompt) > api.DashboardPanelMaxPromptRunes {
		return nil, errors.Errorf("prompt exceeds %d characters", api.DashboardPanelMaxPromptRunes)
	}

	result, err := ValidateSource(input.Source, input.ApiAllowlist)
	if err != nil {
		return nil, errors.Wrap(err, "validate source")
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		name = ScriptParameterName(stringutils.UUID4())
	} else {
		name = ScriptParameterName(name)
	}

	value := api.DashboardCustomScriptValue{
		Version:       api.DashboardPanelScriptVersion,
		Title:         strings.TrimSpace(input.Title),
		Language:      "javascript",
		Entry:         result.Entry,
		Source:        result.Source,
		ContentSha256: result.ContentSha256,
		ApiAllowlist:  result.ApiAllowlist,
		Prompt:        TruncateRunes(strings.TrimSpace(input.Prompt), 512),
		Scope:         scope,
		ValidatedAt:   time.Now().UTC().Format(time.RFC3339),
	}

	session := auth.GetSession(ctx, userCred, "")
	if session == nil {
		return nil, errors.Error("user session is nil")
	}

	paramId, err := upsertScriptParameter(session, name, value)
	if err != nil {
		return nil, err
	}

	out := &api.DashboardPanelSaveOutput{
		Name:          name,
		ParameterId:   paramId,
		Entry:         result.Entry,
		ContentSha256: result.ContentSha256,
		ApiAllowlist:  result.ApiAllowlist,
		Warnings:      result.Warnings,
	}

	panelParam := strings.TrimSpace(input.PanelParameter)
	if panelParam != "" {
		itemId := strings.TrimSpace(input.ItemId)
		if itemId == "" {
			itemId = "dashboard-item-" + stringutils.UUID4()
		}
		if err := mergePanelLayout(session, panelParam, itemId, name, value.Title, input.Layout); err != nil {
			return nil, errors.Wrap(err, "merge panel layout")
		}
		out.PanelParameter = panelParam
		out.ItemId = itemId
	}

	log.Infof("dashboard custom script saved user=%s name=%s sha256=%s prompt=%q",
		userCred.GetUserId(), name, result.ContentSha256, value.Prompt)

	return out, nil
}

func upsertScriptParameter(session *mcclient.ClientSession, name string, value api.DashboardCustomScriptValue) (string, error) {
	existing, err := ycmodules.Parameters.Get(session, name, nil)
	if err == nil && existing != nil {
		params := jsonutils.NewDict()
		params.Set("value", jsonutils.Marshal(value))
		updated, uerr := ycmodules.Parameters.Update(session, name, params)
		if uerr != nil {
			return "", errors.Wrapf(uerr, "update parameter %s", name)
		}
		id, _ := updated.GetString("id")
		return id, nil
	}

	create := jsonutils.NewDict()
	create.Set("name", jsonutils.NewString(name))
	create.Set("value", jsonutils.Marshal(value))
	created, cerr := ycmodules.Parameters.Create(session, create)
	if cerr != nil {
		return "", errors.Wrapf(cerr, "create parameter %s", name)
	}
	id, _ := created.GetString("id")
	return id, nil
}

func mergePanelLayout(session *mcclient.ClientSession, panelName, itemId, scriptParam, title string, layout *api.DashboardPanelLayoutHint) error {
	panel, err := ycmodules.Parameters.Get(session, panelName, nil)
	if err != nil {
		return errors.Wrapf(err, "get panel parameter %s", panelName)
	}
	valueObj, _ := panel.Get("value")
	valueDict, ok := valueObj.(*jsonutils.JSONDict)
	if !ok || valueDict == nil {
		valueDict = jsonutils.NewDict()
	}

	x, y, w, h, minH := 0, 0, 4, 4, 2
	if layout != nil {
		x, y, w, h = layout.X, layout.Y, layout.W, layout.H
		if layout.MinH > 0 {
			minH = layout.MinH
		}
	}
	if w <= 0 {
		w = 4
	}
	if h <= 0 {
		h = 4
	}

	item := jsonutils.NewDict()
	layoutObj := jsonutils.NewDict()
	layoutObj.Set("x", jsonutils.NewInt(int64(x)))
	layoutObj.Set("y", jsonutils.NewInt(int64(y)))
	layoutObj.Set("w", jsonutils.NewInt(int64(w)))
	layoutObj.Set("h", jsonutils.NewInt(int64(h)))
	layoutObj.Set("minH", jsonutils.NewInt(int64(minH)))
	layoutObj.Set("component", jsonutils.NewString(api.DashboardPanelComponentName))
	item.Set("layout", layoutObj)

	params := jsonutils.NewDict()
	params.Set("script_parameter", jsonutils.NewString(scriptParam))
	if title != "" {
		params.Set("title", jsonutils.NewString(title))
	}
	item.Set("params", params)
	valueDict.Set(itemId, item)

	update := jsonutils.NewDict()
	update.Set("value", valueDict)
	_, err = ycmodules.Parameters.Update(session, panelName, update)
	if err != nil {
		return errors.Wrapf(err, "update panel parameter %s", panelName)
	}
	return nil
}
