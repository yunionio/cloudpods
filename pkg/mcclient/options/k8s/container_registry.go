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

package k8s

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type RegistryListOptions struct {
	options.BaseListOptions
	Type string `help:"Container registry type" json:"type" choices:"harbor"`
}

func (o *RegistryListOptions) Params() (jsonutils.JSONObject, error) {
	params, err := o.BaseListOptions.Params()
	if err != nil {
		return nil, err
	}
	if o.Type != "" {
		params.Add(jsonutils.NewString(o.Type), "type")
	}
	return params, nil
}

type RegistryGetOptions struct {
	NAME string `help:"ID or name of the repo" json:"name"`
}

func (o *RegistryGetOptions) GetId() string {
	return o.NAME
}

func (o *RegistryGetOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type RegistryCreateCommonOptions struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type RegistryCreateHarborOptions struct {
	RegistryCreateCommonOptions
}

type RegistryCreateConfigOptions struct {
	Common RegistryCreateHarborOptions `json:"common"`
	Harbor RegistryCreateHarborOptions `json:"harbor"`
}

type RegistryCreateOptions struct {
	RegistryGetOptions
	TYPE   string                      `help:"Repository type" choices:"common|harbor" json:"type"`
	URL    string                      `help:"Repository url" json:"url"`
	Config RegistryCreateConfigOptions `json:"config"`
}

func (o RegistryCreateOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(o), nil
}

type RegistryGetImagesOptions struct {
	RegistryGetOptions
}

func (o RegistryGetImagesOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type RegistryGetImageTagsOptions struct {
	RegistryGetOptions
	REPOSITORY string `help:"image repository, e.g. 'yunion/region'"`
}

func (o RegistryGetImageTagsOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(map[string]interface{}{
		"repository": o.REPOSITORY,
	}), nil
}

type RegistryPublicOptions struct {
	ID             []string `help:"ID or name of image" json:"-"`
	Scope          string   `help:"sharing scope" choices:"system|domain|project" json:"scope"`
	SharedProjects []string `help:"Share to projects" json:"shared_projects"`
	SharedDomains  []string `help:"Share to domains" json:"shared_domains"`
}

func (o RegistryPublicOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(o), nil
}

func (o RegistryPublicOptions) GetIds() []string {
	return o.ID
}
