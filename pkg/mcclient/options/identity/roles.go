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

package identity

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type RoleListOptions struct {
	options.BaseListOptions
	OrderByDomain string `help:"order by domain name" choices:"asc|desc"`
}

func (opts *RoleListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(opts)
}

type RoleIdOptions struct {
	ID string `help:"ID or name of role"`
}

func (opts *RoleIdOptions) GetId() string {
	return opts.ID
}

type RoleDetailOptions struct {
	RoleIdOptions
	Domain string `help:"Domain"`
}

func (opts *RoleDetailOptions) Params() (jsonutils.JSONObject, error) {
	ret := jsonutils.NewDict()
	if len(opts.Domain) > 0 {
		ret.Add(jsonutils.NewString(opts.Domain), "domain_id")
	}
	return ret, nil
}

type RoleCreateOptions struct {
	NAME   string `help:"Role name"`
	Domain string `help:"Domain"`
	Desc   string `help:"Description"`

	PublicScope string `help:"public scope" choices:"none|system"`
}

func (opts *RoleCreateOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(opts.NAME), "name")
	if len(opts.Domain) > 0 {
		params.Add(jsonutils.NewString(opts.Domain), "domain_id")
	}
	if len(opts.Desc) > 0 {
		params.Add(jsonutils.NewString(opts.Desc), "description")
	}
	if len(opts.PublicScope) > 0 {
		params.Add(jsonutils.NewString(opts.PublicScope), "public_scope")
	}
	return params, nil
}

type RoleGetPropertyTagValuePairOptions struct {
	RoleListOptions
	options.TagValuePairsOptions
}

func (opts *RoleGetPropertyTagValuePairOptions) Params() (jsonutils.JSONObject, error) {
	params, err := opts.RoleListOptions.Params()
	if err != nil {
		return nil, errors.Wrap(err, "RoleListOptions.Params")
	}
	tagParams, _ := opts.TagValuePairsOptions.Params()
	params.(*jsonutils.JSONDict).Update(tagParams)
	return params, nil
}

type RoleGetPropertyTagValueTreeOptions struct {
	RoleListOptions
	options.TagValueTreeOptions
}

func (opts *RoleGetPropertyTagValueTreeOptions) Params() (jsonutils.JSONObject, error) {
	params, err := opts.RoleListOptions.Params()
	if err != nil {
		return nil, errors.Wrap(err, "RoleListOptions.Params")
	}
	tagParams, _ := opts.TagValueTreeOptions.Params()
	params.(*jsonutils.JSONDict).Update(tagParams)
	return params, nil
}

type RoleGetPropertyDomainTagValuePairOptions struct {
	RoleListOptions
	options.DomainTagValuePairsOptions
}

func (opts *RoleGetPropertyDomainTagValuePairOptions) Params() (jsonutils.JSONObject, error) {
	params, err := opts.RoleListOptions.Params()
	if err != nil {
		return nil, errors.Wrap(err, "RoleListOptions.Params")
	}
	tagParams, _ := opts.DomainTagValuePairsOptions.Params()
	params.(*jsonutils.JSONDict).Update(tagParams)
	return params, nil
}

type RoleGetPropertyDomainTagValueTreeOptions struct {
	RoleListOptions
	options.DomainTagValueTreeOptions
}

func (opts *RoleGetPropertyDomainTagValueTreeOptions) Params() (jsonutils.JSONObject, error) {
	params, err := opts.RoleListOptions.Params()
	if err != nil {
		return nil, errors.Wrap(err, "RoleListOptions.Params")
	}
	tagParams, _ := opts.DomainTagValueTreeOptions.Params()
	params.(*jsonutils.JSONDict).Update(tagParams)
	return params, nil
}
