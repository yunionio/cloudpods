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

type IDBModelSets interface {
	IModelSets
	FetchFromAPIMap(s *mcclient.ClientSession) (IModelSets, error)
}

type IModelSet interface {
	ModelManager() mcclient_modulebase.IBaseManager
	NewModel() db.IModel
	AddModel(db.IModel)
	Copy() IModelSet
}

type IDBModelSet interface {
	IModelSet
	DBModelManager() db.IModelManager
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

func SyncModelSets(mssOld IModelSets, s *mcclient.ClientSession, opt *Options) (ModelSetsUpdateResult, error) {
	var (
		mssNews IModelSets
		err     error
	)
	if mssDB, ok := mssOld.(IDBModelSets); ok && !opt.FetchFromComputeService {
		mssNews, err = mssDB.FetchFromAPIMap(s)
		if err != nil {
			return ModelSetsUpdateResult{}, errors.Wrap(err, "FetchFromAPIMap")
		}
	} else {
		mssNews, err = syncModelSets(mssOld, s, opt)
		if err != nil {
			return ModelSetsUpdateResult{}, errors.Wrap(err, "syncModelSets")
		}
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

			IncludeDetails:       opt.IncludeDetails,
			IncludeEmulated:      includeEmulated,
			InCludeOtherCloudEnv: opt.IncludeOtherCloudEnv,
		})
		if err != nil {
			return nil, errors.Wrap(err, "GetModels")
		}
	}
	return mssNews, nil
}

func SyncDBModelSets(mssOld IModelSets, s *mcclient.ClientSession, opt *Options) (r ModelSetsUpdateResult, err error) {
	mssNews := mssOld.NewEmpty()
	for _, msNew := range mssNews.ModelSetList() {
		var (
			includeEmulated = false
		)
		if optProvider, ok := msNew.(IModelSetEmulatedIncluder); ok {
			includeEmulated = optProvider.IncludeEmulated()
		}
		msi, ok := msNew.(IDBModelSet)
		opts := &GetModelsOptions{
			ClientSession: s,
			ModelManager:  msNew.ModelManager(),
			ModelSet:      msNew,
			BatchListSize: opt.ListBatchSize,

			IncludeDetails:       opt.IncludeDetails,
			IncludeEmulated:      includeEmulated,
			InCludeOtherCloudEnv: opt.IncludeOtherCloudEnv,
		}
		if ok {
			dbOpts := &GetDBModelsOptions{
				modelOptions:   opts,
				modelDBManager: msi.DBModelManager(),
			}
			err = GetDBModels(dbOpts)
			if err != nil {
				return
			}
		} else {
			err = GetModels(opts)
			if err != nil {
				return
			}
		}
	}
	r = mssOld.ApplyUpdates(mssNews)
	return r, nil
}
