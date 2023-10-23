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

package apihelper

import (
	"context"
	"fmt"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type GetDBModelsOptions struct {
	modelOptions   *GetModelsOptions
	modelDBManager db.IModelManager
}

func (o GetDBModelsOptions) IncludeOtherCloudEnv() bool {
	return o.modelOptions.InCludeOtherCloudEnv
}

func (o GetDBModelsOptions) GetModelSet() IModelSet {
	return o.modelOptions.ModelSet
}

func (o GetDBModelsOptions) GetMinUpdatedAt() time.Time {
	return o.modelOptions.MinUpdatedAt
}

func (o GetDBModelsOptions) IncludeDetails() bool {
	return o.modelOptions.IncludeDetails
}

func (o GetDBModelsOptions) IncludeEmulated() bool {
	return o.modelOptions.IncludeEmulated
}

func (o GetDBModelsOptions) BatchListSize() int {
	return o.modelOptions.BatchListSize
}

func (o GetDBModelsOptions) InCludeOtherCloudEnv() bool {
	return o.modelOptions.InCludeOtherCloudEnv
}

func GetDBModels(opts *GetDBModelsOptions) error {
	man := opts.modelDBManager
	manKeyPlural := man.KeywordPlural()

	limit := opts.BatchListSize()
	// limit := 5
	listOptions := options.BaseListOptions{
		System:       options.Bool(true),
		Admin:        options.Bool(true),
		Scope:        "system",
		Details:      options.Bool(opts.IncludeDetails()),
		ShowEmulated: options.Bool(opts.IncludeEmulated()),
		OrderBy:      []string{"updated_at"},
		Order:        "asc",
		Limit:        &limit,
	}
	if !opts.InCludeOtherCloudEnv() {
		listOptions.Filter = append(listOptions.Filter,
			"manager_id.isnullorempty()", // len(manager_id) > 0 is for pubcloud objects
			// "external_id.isnullorempty()", // len(external_id) > 0 is for pubcloud objects
		)
		listOptions.CloudEnv = "onpremise"
		// listOptions.Provider = []string{"OneCloud"}
	}
	if inter, ok := opts.GetModelSet().(IModelSetFilter); ok {
		filter := inter.ModelFilter()
		listOptions.Filter = append(listOptions.Filter, filter...)
	}
	params, err := listOptions.Params()
	if err != nil {
		return fmt.Errorf("%s: making list params: %s", manKeyPlural, err)
	}
	if inter, ok := opts.GetModelSet().(IModelListParam); ok {
		filter := inter.ModelParamFilter()
		params.Update(filter)
	}
	//XXX
	//params.Set(api.LBAGENT_QUERY_ORIG_KEY, jsonutils.NewString(api.LBAGENT_QUERY_ORIG_VAL))

	entriesJson := []jsonutils.JSONObject{}
	for {
		log.Debugf("list %s with params: %s", manKeyPlural, params.String())
		var err error
		listResult, err := db.ListItems(man, context.Background(), auth.AdminCredential(), params, nil)
		if err != nil {
			return fmt.Errorf("%s: list failed: %s",
				manKeyPlural, err)
		}
		entriesJson = append(entriesJson, listResult.Data...)
		if listResult.Offset+len(listResult.Data) >= listResult.Total {
			break
		} else {
			offset := listResult.Offset + len(listResult.Data)
			params.Set("offset", jsonutils.NewInt(int64(offset)))
		}
	}
	{
		err := InitializeModelSetFromJSON(opts.GetModelSet(), entriesJson)
		if err != nil {
			return fmt.Errorf("%s: initializing model set failed: %s",
				manKeyPlural, err)
		}
	}

	return nil
}
