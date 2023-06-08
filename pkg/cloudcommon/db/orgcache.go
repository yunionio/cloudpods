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

package db

import (
	"context"

	"yunion.io/x/pkg/errors"

	identityapi "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	identitymodules "yunion.io/x/onecloud/pkg/mcclient/modules/identity"
	identityoptions "yunion.io/x/onecloud/pkg/mcclient/options/identity"
	"yunion.io/x/onecloud/pkg/util/tagutils"
)

func FetchOrganizationTags(ctx context.Context, orgIds []string, orgType identityapi.TOrgType) (tagutils.TTagSetList, error) {
	s := auth.GetAdminSession(ctx, consts.GetRegion())
	opts := identityoptions.OrganizationNodeListOptions{}
	opts.Scope = "system"
	limit := 2048
	opts.Limit = &limit
	opts.Id = orgIds
	opts.OrgType = string(orgType)
	params, err := opts.Params()
	if err != nil {
		return nil, errors.Wrap(err, "Options")
	}
	results, err := identitymodules.OrganizationNodes.List(s, params)
	if err != nil {
		return nil, errors.Wrap(err, "OrganizationNodes.List")
	}
	tagList := tagutils.TTagSetList{}
	for _, orgJson := range results.Data {
		v := struct {
			Tags tagutils.TTagSet `json:"tags"`
		}{}
		err := orgJson.Unmarshal(&v)
		if err != nil {
			return nil, errors.Wrapf(err, "Unmarshal %s", orgJson)
		}
		tagList = tagList.Append(v.Tags)
	}
	return tagList, nil
}
