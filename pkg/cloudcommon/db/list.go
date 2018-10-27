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
