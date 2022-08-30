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
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type ServerGetPropertyTagValuePairOptions struct {
	ServerListOptions
	options.TagValuePairsOptions
}

func (opts *ServerGetPropertyTagValuePairOptions) Params() (jsonutils.JSONObject, error) {
	params, err := opts.ServerListOptions.Params()
	if err != nil {
		return nil, errors.Wrap(err, "ProjectListOptions.Params")
	}
	tagParams, _ := opts.TagValuePairsOptions.Params()
	params.(*jsonutils.JSONDict).Update(tagParams)
	return params, nil
}

type ServerGetPropertyTagValueTreeOptions struct {
	ServerListOptions
	options.TagValueTreeOptions
}

func (opts *ServerGetPropertyTagValueTreeOptions) Params() (jsonutils.JSONObject, error) {
	params, err := opts.ServerListOptions.Params()
	if err != nil {
		return nil, errors.Wrap(err, "ProjectListOptions.Params")
	}
	tagParams, _ := opts.TagValueTreeOptions.Params()
	params.(*jsonutils.JSONDict).Update(tagParams)
	return params, nil
}

type ServerGetPropertyProjectTagValuePairOptions struct {
	ServerListOptions
	options.ProjectTagValuePairsOptions
}

func (opts *ServerGetPropertyProjectTagValuePairOptions) Params() (jsonutils.JSONObject, error) {
	params, err := opts.ServerListOptions.Params()
	if err != nil {
		return nil, errors.Wrap(err, "ProjectListOptions.Params")
	}
	tagParams, _ := opts.ProjectTagValuePairsOptions.Params()
	params.(*jsonutils.JSONDict).Update(tagParams)
	return params, nil
}

type ServerGetPropertyProjectTagValueTreeOptions struct {
	ServerListOptions
	options.ProjectTagValueTreeOptions
}

func (opts *ServerGetPropertyProjectTagValueTreeOptions) Params() (jsonutils.JSONObject, error) {
	params, err := opts.ServerListOptions.Params()
	if err != nil {
		return nil, errors.Wrap(err, "ProjectListOptions.Params")
	}
	tagParams, _ := opts.ProjectTagValueTreeOptions.Params()
	params.(*jsonutils.JSONDict).Update(tagParams)
	return params, nil
}

type ServerGetPropertyDomainTagValuePairOptions struct {
	ServerListOptions
	options.DomainTagValuePairsOptions
}

func (opts *ServerGetPropertyDomainTagValuePairOptions) Params() (jsonutils.JSONObject, error) {
	params, err := opts.ServerListOptions.Params()
	if err != nil {
		return nil, errors.Wrap(err, "ProjectListOptions.Params")
	}
	tagParams, _ := opts.DomainTagValuePairsOptions.Params()
	params.(*jsonutils.JSONDict).Update(tagParams)
	return params, nil
}

type ServerGetPropertyDomainTagValueTreeOptions struct {
	ServerListOptions
	options.DomainTagValueTreeOptions
}

func (opts *ServerGetPropertyDomainTagValueTreeOptions) Params() (jsonutils.JSONObject, error) {
	params, err := opts.ServerListOptions.Params()
	if err != nil {
		return nil, errors.Wrap(err, "ProjectListOptions.Params")
	}
	tagParams, _ := opts.DomainTagValueTreeOptions.Params()
	params.(*jsonutils.JSONDict).Update(tagParams)
	return params, nil
}
