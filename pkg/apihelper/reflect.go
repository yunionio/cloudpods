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
	"fmt"
	"reflect"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/printutils"
	"yunion.io/x/pkg/util/timeutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	mcclient "yunion.io/x/onecloud/pkg/mcclient"
	mcclient_modulebase "yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

// A hack to workaround the IsZero() in timeutils.Utcify.  This depends on the
// fact that database time has a resolution of 1-second
var PseudoZeroTime = time.Time{}.Add(time.Nanosecond)

type GetModelsOptions struct {
	ClientSession *mcclient.ClientSession
	ModelManager  mcclient_modulebase.IBaseManager
	ModelSet      IModelSet

	BatchListSize        int
	MinUpdatedAt         time.Time
	IncludeDetails       bool
	IncludeEmulated      bool
	InCludeOtherCloudEnv bool
}

func GetModels(opts *GetModelsOptions) error {
	man := opts.ModelManager
	manKeyPlural := man.KeyString()

	minUpdatedAt := opts.MinUpdatedAt
	minUpdatedAtFilter := func(time time.Time) string {
		// TODO add GE
		tstr := timeutils.MysqlTime(time)
		return fmt.Sprintf("updated_at.ge('%s')", tstr)
	}
	setNextListParams := func(params *jsonutils.JSONDict, lastUpdatedAt time.Time, lastResult *printutils.ListResult) (time.Time, error) {
		// NOTE: the updated_at field has second-level resolution.
		// If they all have the same date...
		var max time.Time
		nmax := 0
		n := len(lastResult.Data)

		// find out the max updated_at date in the result set, and how
		// many in the set has this date
		for i := n - 1; i >= 0; i-- {
			j := lastResult.Data[i]
			updatedAt, err := j.GetTime("updated_at")
			if err != nil {
				log.Warningf("%s: updated_at field: %s, %s",
					manKeyPlural, err, j.String())
				continue
			}
			if max.IsZero() {
				max = updatedAt
			}
			if max.Equal(updatedAt) {
				nmax += 1
			}
		}
		// error if we do not have valid date
		if max.IsZero() {
			return time.Time{}, fmt.Errorf("%s: cannot find next updated_at after '%q'",
				manKeyPlural, lastUpdatedAt)
		}

		var newTime time.Time
		var newOffset int
		// if not all updated_at date are the same, then we can
		// continue to the next age.
		if nmax < n || (!max.Equal(lastUpdatedAt) && !max.Equal(PseudoZeroTime)) {
			newTime = max
			newOffset = nmax
		} else {
			newTime = lastUpdatedAt
			newOffset = lastResult.Offset + n
		}
		params.Set("filter.0", jsonutils.NewString(minUpdatedAtFilter(newTime)))
		params.Set("offset", jsonutils.NewInt(int64(newOffset)))
		return newTime, nil
	}

	listOptions := options.BaseListOptions{
		System:       options.Bool(true),
		Admin:        options.Bool(true),
		Scope:        "system",
		Details:      options.Bool(opts.IncludeDetails),
		ShowEmulated: options.Bool(opts.IncludeEmulated),
		Filter: []string{
			minUpdatedAtFilter(minUpdatedAt), // order matters, filter.0
		},
		OrderBy: []string{"updated_at"},
		Order:   "asc",
		Limit:   options.Int(opts.BatchListSize),
		Offset:  options.Int(0),
	}
	if !opts.InCludeOtherCloudEnv {
		listOptions.Filter = append(listOptions.Filter,
			"manager_id.isnullorempty()", // len(manager_id) > 0 is for pubcloud objects
			// "external_id.isnullorempty()", // len(external_id) > 0 is for pubcloud objects
		)
		listOptions.CloudEnv = "onpremise"
		// listOptions.Provider = []string{"OneCloud"}
	}
	if inter, ok := opts.ModelSet.(IModelSetFilter); ok {
		filter := inter.ModelFilter()
		listOptions.Filter = append(listOptions.Filter, filter...)
	}
	if !minUpdatedAt.Equal(PseudoZeroTime) {
		// Only fetching pending deletes when we are doing incremental fetch
		listOptions.PendingDeleteAll = options.Bool(true)
		listOptions.DeleteAll = options.Bool(true)
	}
	params, err := listOptions.Params()
	if err != nil {
		return fmt.Errorf("%s: making list params: %s", manKeyPlural, err)
	}
	if inter, ok := opts.ModelSet.(IModelListParam); ok {
		filter := inter.ModelParamFilter()
		params.Update(filter)
	}
	if inter, ok := opts.ModelSet.(IModelListSetParams); ok {
		params = inter.SetModelListParams(params)
	}
	//XXX
	//params.Set(api.LBAGENT_QUERY_ORIG_KEY, jsonutils.NewString(api.LBAGENT_QUERY_ORIG_VAL))

	entriesJson := []jsonutils.JSONObject{}
	for {
		var err error
		listResult, err := opts.ModelManager.List(opts.ClientSession, params)
		if err != nil {
			log.Errorf("%s: list failed with updated_at.gt('%s'): %s", manKeyPlural, minUpdatedAt, err)
			return errors.Wrapf(err, "%s list failed with params: %s", manKeyPlural, params.QueryString())
		}
		entriesJson = append(entriesJson, listResult.Data...)
		if listResult.Offset+len(listResult.Data) >= listResult.Total {
			break
		}
		minUpdatedAt, err = setNextListParams(params, minUpdatedAt, listResult)
		if err != nil {
			return fmt.Errorf("%s: %s", manKeyPlural, err)
		}
	}
	{
		err := InitializeModelSetFromJSON(opts.ModelSet, entriesJson)
		if err != nil {
			return fmt.Errorf("%s: initializing model set failed: %s",
				manKeyPlural, err)
		}
	}
	return nil
}

func InitializeModelSetFromJSON(set IModelSet, entriesJson []jsonutils.JSONObject) error {
	setRv := reflect.ValueOf(set)
	for _, kRv := range setRv.MapKeys() {
		zRv := reflect.Value{}
		setRv.SetMapIndex(kRv, zRv)
	}
	manKeyPlural := set.ModelManager().KeyString()
	for _, entryJson := range entriesJson {
		m := set.NewModel()
		var err error
		err = entryJson.Unmarshal(m)
		if err != nil {
			return fmt.Errorf("%s: unmarshal: %v: %s", manKeyPlural, err, entryJson.String())
		}
		{
			keyRv := reflect.ValueOf(m.GetId())
			oldMRv := setRv.MapIndex(keyRv)
			if oldMRv.IsValid() {
				// check version
				oldM := oldMRv.Interface().(db.IModel)
				oldVersion := oldM.GetUpdateVersion()
				version := m.GetUpdateVersion()
				if oldVersion > version {
					oldUpdatedAt := oldM.GetUpdatedAt()
					updatedAt := m.GetUpdatedAt()
					log.Warningf("prefer loadbalancer with update_version %d(%s) to %d(%s)",
						oldVersion, oldUpdatedAt, version, updatedAt)
					return nil
				}
			}
		}
		set.AddModel(m)
	}
	return nil
}

func ModelSetMaxUpdatedAt(set IModelSet) time.Time {
	r := PseudoZeroTime
	setRv := reflect.ValueOf(set)
	for _, kRv := range setRv.MapKeys() {
		mRv := setRv.MapIndex(kRv)
		m := mRv.Interface().(db.IModel)
		updatedAt := m.GetUpdatedAt()
		if r.Before(updatedAt) {
			r = updatedAt
		}
	}
	return r
}

type ModelSetUpdateResult struct {
	Changed      bool
	MaxUpdatedAt time.Time
}

// ModelSetApplyUpdates applies bSet to aSet.
//
//   - PendingDeleted in bSet are removed from aSet
//   - Newer models in bSet are updated in aSet
func ModelSetApplyUpdates(aSet, bSet IModelSet) *ModelSetUpdateResult {
	r := &ModelSetUpdateResult{
		Changed: false,
	}
	{
		a := ModelSetMaxUpdatedAt(aSet)
		b := ModelSetMaxUpdatedAt(bSet)
		if b.After(a) {
			r.MaxUpdatedAt = b
		} else {
			r.MaxUpdatedAt = a
		}
	}
	aSetRv := reflect.ValueOf(aSet)
	bSetRv := reflect.ValueOf(bSet)
	for _, kRv := range bSetRv.MapKeys() {
		bMRv := bSetRv.MapIndex(kRv)
		b := bMRv.Interface()
		bM := b.(db.IModel)
		bGone := bM.GetDeleted()
		if !bGone {
			bVM, ok := b.(db.IPendingDeletable)
			if ok {
				bGone = bVM.GetPendingDeleted()
			}
		}
		aMRv := aSetRv.MapIndex(kRv)
		if aMRv.IsValid() {
			aM := aMRv.Interface().(db.IModel)
			if bGone {
				// oops, deleted
				aSetRv.SetMapIndex(kRv, reflect.Value{})
				r.Changed = true
				continue
			}
			if aM.GetUpdateVersion() < bM.GetUpdateVersion() {
				// oops, updated
				aSetRv.SetMapIndex(kRv, bMRv)
				r.Changed = true
				continue
			}
		} else {
			if bGone {
				// hmm, gone before even knowning
				continue
			}
			// oops, new member
			aSetRv.SetMapIndex(kRv, bMRv)
			r.Changed = true
		}
	}
	return r
}
