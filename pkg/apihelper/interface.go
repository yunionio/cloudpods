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
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	mcclient "yunion.io/x/onecloud/pkg/mcclient"
	mcclient_modulebase "yunion.io/x/onecloud/pkg/mcclient/modulebase"
)

type ModelSetsUpdateResult struct {
	Correct bool
	Changed bool
}

type IModelSets interface {
	NewEmpty() IModelSets
	ModelSetList() []IModelSet
	ApplyUpdates(IModelSets) ModelSetsUpdateResult
	Copy() IModelSets
	CopyJoined() IModelSets
}

// IAPIMapModelSets is implemented by the model sets that can be fetched from
// the apimap service instead of from the compute APIs.
type IAPIMapModelSets interface {
	IModelSets

	// APIMapTimestamp returns the version of the model sets currently served
	// by the apimap service, without fetching the payload.
	APIMapTimestamp(s *mcclient.ClientSession) (int64, error)

	// FetchFromAPIMap fetches the model sets from the apimap service.  The
	// returned timestamp is the version of the returned payload, which is
	// what the caller has to compare against on the next round.
	FetchFromAPIMap(s *mcclient.ClientSession) (IModelSets, int64, error)
}

type IModelSet interface {
	ModelManager() mcclient_modulebase.IBaseManager
	NewModel() db.IModel
	AddModel(db.IModel)
	Copy() IModelSet
	IncludeDetails() bool
}

type IModelSetEmulatedIncluder interface {
	IncludeEmulated() bool
}

type IModelSetFilter interface {
	ModelFilter() []string
}

type IModelListParam interface {
	ModelParamFilter() jsonutils.JSONObject
}

type IModelListSetParams interface {
	SetModelListParams(params *jsonutils.JSONDict) *jsonutils.JSONDict
}

// SyncModelSets refreshes mssOld with the model sets listed from the compute
// service APIs.  Its counterpart, for the model sets served by the apimap
// service, is APIHelper's own sync path, which also has to keep track of the
// version that the model sets reflect.
func SyncModelSets(mssOld IModelSets, s *mcclient.ClientSession, opt *Options) (ModelSetsUpdateResult, error) {
	mssNews, err := syncModelSets(mssOld, s, opt)
	if err != nil {
		return ModelSetsUpdateResult{}, errors.Wrap(err, "syncModelSets")
	}
	r := mssOld.ApplyUpdates(mssNews)
	return r, nil
}

func syncModelSets(mssOld IModelSets, s *mcclient.ClientSession, opt *Options) (IModelSets, error) {
	// mss := mssOld.ModelSetList()
	mssNews := mssOld.NewEmpty()
	for _, msNew := range mssNews.ModelSetList() {
		var (
			// minUpdatedAt    = ModelSetMaxUpdatedAt(mss[i])
			includeEmulated = false
		)
		if optProvider, ok := msNew.(IModelSetEmulatedIncluder); ok {
			includeEmulated = optProvider.IncludeEmulated()
		}
		err := GetModels(&GetModelsOptions{
			ClientSession: s,
			ModelManager:  msNew.ModelManager(),
			// MinUpdatedAt:  minUpdatedAt,
			ModelSet:      msNew,
			BatchListSize: opt.ListBatchSize,

			IncludeDetails:       msNew.IncludeDetails(),
			IncludeEmulated:      includeEmulated,
			InCludeOtherCloudEnv: opt.IncludeOtherCloudEnv,
		})
		if err != nil {
			return nil, errors.Wrap(err, "GetModels")
		}
	}
	return mssNews, nil
}
