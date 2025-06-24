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

package compute

import (
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type CDNDomainListOptions struct {
	options.BaseListOptions
}

func (opts *CDNDomainListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(opts)
}

type CDNDomainCreateOptions struct {
	DOMAIN       string `help:"Domain Name"`
	MANAGER      string `help:"Cloudprovider Id"`
	AREA         string `help:"Area" choices:"mainland|overseas|global"`
	SERVICE_TYPE string `help:"Service Type" choices:"web|download|media"`
	ORIGIN_TYPE  string `help:"Origin Type" choices:"domain|ip|bucket|third_party"`
	ORIGIN       string `help:"Origin Addr"`
	ServerName   string `help:"Cdn Server Name"`
}

func (opts *CDNDomainCreateOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(opts.DOMAIN), "name")
	params.Add(jsonutils.NewString(opts.MANAGER), "cloudprovider_id")
	params.Add(jsonutils.NewString(opts.AREA), "area")
	params.Add(jsonutils.NewString(opts.SERVICE_TYPE), "service_type")
	params.Add(jsonutils.Marshal([]interface{}{
		map[string]string{
			"origin":      opts.ORIGIN,
			"type":        opts.ORIGIN_TYPE,
			"server_name": opts.ServerName,
		},
	}), "origins")
	return params, nil
}

type CDNDomainUpdateOptions struct {
	options.BaseIdOptions
	Description string
	Delete      string `help:"Lock or not lock cdn domain" choices:"enable|disable"`
}

func (opts *CDNDomainUpdateOptions) Params() (jsonutils.JSONObject, error) {
	params, err := options.StructToParams(opts)
	if err != nil {
		return nil, err
	}
	if len(opts.Delete) > 0 {
		if opts.Delete == "disable" {
			params.Add(jsonutils.JSONTrue, "disable_delete")
		} else {
			params.Add(jsonutils.JSONFalse, "disable_delete")
		}
	}
	return params, nil
}

type CDNClearCacheOptions struct {
	options.BaseIdOptions
	cloudprovider.CacheClearOptions
}

func (opts *CDNClearCacheOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts.CacheClearOptions), nil
}

type CDNChangeConfigOptions struct {
	options.BaseIdOptions
	cloudprovider.CacheConfig
}

func (opts *CDNChangeConfigOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts.CacheConfig), nil
}

type CDNDeleteCustomHostnameOptions struct {
	options.BaseIdOptions
	CustomHostnameId string
}

func (opts *CDNDeleteCustomHostnameOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(map[string]string{"id": opts.CustomHostnameId}), nil
}

type CDNCustomHostnameOptions struct {
	options.BaseIdOptions
}
