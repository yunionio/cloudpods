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
	"yunion.io/x/jsonutils"
)

type CustomizeListFilterFunc func(item jsonutils.JSONObject) (bool, error)

type CustomizeListFilters struct {
	filters []CustomizeListFilterFunc
}

func NewCustomizeListFilters() *CustomizeListFilters {
	return &CustomizeListFilters{
		filters: []CustomizeListFilterFunc{},
	}
}

func (f *CustomizeListFilters) Append(funcs ...CustomizeListFilterFunc) *CustomizeListFilters {
	f.filters = append(f.filters, funcs...)
	return f
}

func (f CustomizeListFilters) Len() int {
	return len(f.filters)
}

func (f CustomizeListFilters) IsEmpty() bool {
	return f.Len() == 0
}

func (f CustomizeListFilters) DoApply(objs []jsonutils.JSONObject) ([]jsonutils.JSONObject, error) {
	filteredObjs := []jsonutils.JSONObject{}
	for _, obj := range objs {
		ok, err := f.singleApply(obj)
		if err != nil {
			return nil, err
		}
		if ok {
			filteredObjs = append(filteredObjs, obj)
		}
	}
	return filteredObjs, nil
}

func (f CustomizeListFilters) singleApply(obj jsonutils.JSONObject) (bool, error) {
	if f.IsEmpty() {
		return true, nil
	}
	for _, filter := range f.filters {
		ok, err := filter(obj)
		if err != nil {
			return false, err
		}
		if !ok {
			return false, nil
		}
	}
	return true, nil
}
