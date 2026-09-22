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

package image

import (
	"strings"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

// fetchContainerImageEnvs parses env strings in the format of key=value
func fetchContainerImageEnvs(envStrs []string) []*apis.ContainerKeyValue {
	envs := make([]*apis.ContainerKeyValue, 0, len(envStrs))
	for _, env := range envStrs {
		pos := strings.Index(env, "=")
		if pos > 0 {
			key := strings.TrimSpace(env[:pos])
			val := strings.TrimSpace(env[pos+1:])
			envs = append(envs, &apis.ContainerKeyValue{Key: key, Value: val})
		}
	}
	return envs
}

type ContainerRegistryListOptions struct {
	options.BaseListOptions
	Type string `help:"Container registry type" json:"type" choices:"harbor|common|custom"`
	Url  string `help:"Filter by url" json:"url"`
}

func (o *ContainerRegistryListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(o)
}

type ContainerRegistryIdOptions struct {
	ID string `help:"ID or name of the registry" json:"-"`
}

func (o *ContainerRegistryIdOptions) GetId() string {
	return o.ID
}

func (o *ContainerRegistryIdOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type ContainerRegistryCreateOptions struct {
	apis.SharableVirtualResourceCreateInput
	TYPE     string `help:"Repository type" choices:"common|harbor|custom" json:"type"`
	URL      string `help:"Repository url" json:"url"`
	Username string `help:"Username" json:"-"`
	Password string `help:"Password" json:"-"`
}

func (o *ContainerRegistryCreateOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.Marshal(o).(*jsonutils.JSONDict)
	params.Remove("username")
	params.Remove("password")
	if o.Username != "" || o.Password != "" {
		common := jsonutils.NewDict()
		common.Set("username", jsonutils.NewString(o.Username))
		common.Set("password", jsonutils.NewString(o.Password))
		config := jsonutils.NewDict()
		switch o.TYPE {
		case "harbor":
			config.Set("harbor", common)
		case "custom":
			config.Set("custom", common)
		default:
			config.Set("common", common)
		}
		config.Set("type", jsonutils.NewString(o.TYPE))
		params.Set("config", config)
	}
	return params, nil
}

type ContainerRegistryGetImagesOptions struct {
	ContainerRegistryIdOptions
	Details        bool   `help:"Include tag details" json:"details"`
	RepositoryName string `help:"Filter by repository name" json:"repository_name"`
}

func (o *ContainerRegistryGetImagesOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(o), nil
}

type ContainerRegistryGetImageTagsOptions struct {
	ContainerRegistryIdOptions
	REPOSITORY string `help:"image repository, e.g. 'yunion/region'"`
}

func (o *ContainerRegistryGetImageTagsOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(map[string]interface{}{
		"repository": o.REPOSITORY,
	}), nil
}

type ContainerRegistryImportOptions struct {
	Force bool `help:"Force re-import even if URL exists" json:"force"`
}

func (o *ContainerRegistryImportOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(o), nil
}

type ContainerImageListOptions struct {
	options.BaseListOptions
	ImageName  string `json:"image_name" help:"filter by image name"`
	ImageLabel string `json:"image_label" help:"filter by image label"`
	RegistryId string `json:"registry_id" help:"filter by registry id"`
}

func (o *ContainerImageListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(o)
}

type ContainerImageIdOptions struct {
	ID string `help:"ID or name of container image"`
}

func (o *ContainerImageIdOptions) GetId() string {
	return o.ID
}

func (o *ContainerImageIdOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type ContainerImageCreateOptions struct {
	apis.SharableVirtualResourceCreateInput
	IMAGE_NAME   string `json:"image_name"`
	IMAGE_LABEL  string `json:"image_label"`
	CredentialId string `json:"credential_id"`
	RegistryId   string `json:"registry_id"`

	Command []string `help:"Default startup command used when a container does not specify one, repeatable" json:"-"`
	Args    []string `help:"Default startup args used when a container does not specify one, repeatable" json:"-"`
	Env     []string `help:"Default env in the format of key=value, e.g. FOO=bar, repeatable" json:"-"`
}

func (o *ContainerImageCreateOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.Marshal(o).(*jsonutils.JSONDict)
	if len(o.Command) > 0 {
		params.Set("command", jsonutils.Marshal(o.Command))
	}
	if len(o.Args) > 0 {
		params.Set("args", jsonutils.Marshal(o.Args))
	}
	if len(o.Env) > 0 {
		params.Set("envs", jsonutils.Marshal(fetchContainerImageEnvs(o.Env)))
	}
	return params, nil
}

type ContainerImageUpdateOptions struct {
	apis.SharableVirtualResourceBaseUpdateInput
	ID           string
	ImageName    string `json:"image_name"`
	ImageLabel   string `json:"image_label"`
	CredentialId string `json:"credential_id"`
	RegistryId   string `json:"registry_id"`

	Command []string `help:"Replace the default startup command, repeatable" json:"-"`
	Args    []string `help:"Replace the default startup args, repeatable" json:"-"`
	Env     []string `help:"Replace the default env in the format of key=value, repeatable" json:"-"`
}

func (o *ContainerImageUpdateOptions) GetId() string {
	return o.ID
}

func (o *ContainerImageUpdateOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.Marshal(o).(*jsonutils.JSONDict)
	if len(o.Command) > 0 {
		params.Set("command", jsonutils.Marshal(o.Command))
	}
	if len(o.Args) > 0 {
		params.Set("args", jsonutils.Marshal(o.Args))
	}
	if len(o.Env) > 0 {
		params.Set("envs", jsonutils.Marshal(fetchContainerImageEnvs(o.Env)))
	}
	return params, nil
}
