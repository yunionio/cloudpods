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

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type ProjectListOptions struct {
	options.BaseListOptions

	UserId  string `help:"filter by user id"`
	GroupId string `help:"filter by group id"`
	IdpId   string `help:"filter by idp id"`

	OrderByDomain string `help:"order by domain name" choices:"asc|desc"`
}

func (opts *ProjectListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(opts)
}

type ProjectGetPropertyTagValuePairOptions struct {
	ProjectListOptions
	options.TagValuePairsOptions
}

func (opts *ProjectGetPropertyTagValuePairOptions) Params() (jsonutils.JSONObject, error) {
	params, err := opts.ProjectListOptions.Params()
	if err != nil {
		return nil, errors.Wrap(err, "ProjectListOptions.Params")
	}
	tagParams, _ := opts.TagValuePairsOptions.Params()
	params.(*jsonutils.JSONDict).Update(tagParams)
	return params, nil
}

type ProjectGetPropertyTagValueTreeOptions struct {
	ProjectListOptions
	options.TagValueTreeOptions
}

func (opts *ProjectGetPropertyTagValueTreeOptions) Params() (jsonutils.JSONObject, error) {
	params, err := opts.ProjectListOptions.Params()
	if err != nil {
		return nil, errors.Wrap(err, "ProjectListOptions.Params")
	}
	tagParams, _ := opts.TagValueTreeOptions.Params()
	params.(*jsonutils.JSONDict).Update(tagParams)
	return params, nil
}

type ProjectGetPropertyDomainTagValuePairOptions struct {
	ProjectListOptions
	options.DomainTagValuePairsOptions
}

func (opts *ProjectGetPropertyDomainTagValuePairOptions) Params() (jsonutils.JSONObject, error) {
	params, err := opts.ProjectListOptions.Params()
	if err != nil {
		return nil, errors.Wrap(err, "ProjectListOptions.Params")
	}
	tagParams, _ := opts.DomainTagValuePairsOptions.Params()
	params.(*jsonutils.JSONDict).Update(tagParams)
	return params, nil
}

type ProjectGetPropertyDomainTagValueTreeOptions struct {
	ProjectListOptions
	options.DomainTagValueTreeOptions
}

func (opts *ProjectGetPropertyDomainTagValueTreeOptions) Params() (jsonutils.JSONObject, error) {
	params, err := opts.ProjectListOptions.Params()
	if err != nil {
		return nil, errors.Wrap(err, "ProjectListOptions.Params")
	}
	tagParams, _ := opts.DomainTagValueTreeOptions.Params()
	params.(*jsonutils.JSONDict).Update(tagParams)
	return params, nil
}

type ProjectIdOptions struct {
	ID string `help:"ID or name of project" json:"-"`
}

func (opts *ProjectIdOptions) GetId() string {
	return opts.ID
}

type ProjectSetAdminOptions struct {
	ProjectIdOptions

	api.SProjectSetAdminInput
}

func (opts *ProjectSetAdminOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}
