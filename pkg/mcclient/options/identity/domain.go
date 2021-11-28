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

type DomainListOptions struct {
	options.BaseListOptions
	IdpId       string `help:"filter by idp_id"`
	IdpEntityId string `help:"filter by idp_entity_id"`
}

func (opts *DomainListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(opts)
}

type DomainGetPropertyTagValuePairOptions struct {
	DomainListOptions
	options.TagValuePairsOptions
}

func (opts *DomainGetPropertyTagValuePairOptions) Params() (jsonutils.JSONObject, error) {
	params, err := opts.DomainListOptions.Params()
	if err != nil {
		return nil, errors.Wrap(err, "DomainListOptions.Params")
	}
	tagParams, _ := opts.TagValuePairsOptions.Params()
	params.(*jsonutils.JSONDict).Update(tagParams)
	return params, nil
}

type DomainGetPropertyTagValueTreeOptions struct {
	DomainListOptions
	options.TagValueTreeOptions
}

func (opts *DomainGetPropertyTagValueTreeOptions) Params() (jsonutils.JSONObject, error) {
	params, err := opts.DomainListOptions.Params()
	if err != nil {
		return nil, errors.Wrap(err, "DomainListOptions.Params")
	}
	tagParams, _ := opts.TagValueTreeOptions.Params()
	params.(*jsonutils.JSONDict).Update(tagParams)
	return params, nil
}
