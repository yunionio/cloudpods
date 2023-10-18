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

package schedtag

import (
	"fmt"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/sets"

	"yunion.io/x/onecloud/pkg/compute/models"
)

var globalManager IManager

type IManager interface {
	GetSessionManager(sessionId string) (ISessionManager, error)
	FreeSessionManager(sessionId string)
}

type ISessionManager interface {
	GetSessionId() string
	GetAllSchedtags(resType string) ([]ISchedtag, error)
	GetEnabledDynamicSchedtagsByResource(resType string) ([]IDynamicschedtag, error)
	GetCandidateSchedtags(resType, id string) []ISchedtag
}

type iSchedtagManager interface {
	getAllSchedtags(resType string) ([]ISchedtag, error)
	getEnabledDynamicSchedtagsByResource(resType string) ([]IDynamicschedtag, error)
	getCandidateSchedtags(resType, id string) []ISchedtag
}

type manager struct {
	sessions iCache
}

func newManager() IManager {
	return &manager{
		sessions: newCache(),
	}
}

func (m *manager) GetSessionManager(sessionId string) (ISessionManager, error) {
	obj, err := m.sessions.get(sessionId, func() (interface{}, error) {
		ret, err := newSessionManager(sessionId)
		if err != nil {
			return nil, errors.Wrapf(err, "newSessionManager by %q", sessionId)
		}
		return ret, nil
	})
	if err != nil {
		return nil, err
	}
	return obj.(ISessionManager), nil
}

func (m *manager) FreeSessionManager(sessionId string) {
	m.sessions.delete(sessionId)
}

type sessionManager struct {
	sessionId   string
	schedtagMan iSchedtagManager
}

func newSessionManager(sId string) (ISessionManager, error) {
	sm, err := newSchedtagManager()
	if err != nil {
		return nil, errors.Wrap(err, "newSchedtagManager")
	}
	return &sessionManager{
		sessionId:   sId,
		schedtagMan: sm,
	}, nil
}

func (m *sessionManager) GetSessionId() string {
	return m.sessionId
}

func (m *sessionManager) GetSchedtagManager() iSchedtagManager {
	return m.schedtagMan
}

func (m *sessionManager) GetAllSchedtags(resType string) ([]ISchedtag, error) {
	return m.schedtagMan.getAllSchedtags(resType)
}

func (m *sessionManager) GetEnabledDynamicSchedtagsByResource(resType string) ([]IDynamicschedtag, error) {
	return m.schedtagMan.getEnabledDynamicSchedtagsByResource(resType)
}

func (m *sessionManager) GetCandidateSchedtags(resType, id string) []ISchedtag {
	return m.schedtagMan.getCandidateSchedtags(resType, id)
}

type schedtagManager struct {
	schedtagsByType        iCache
	dynamicSchedtagsByType iCache

	resourceSchedtags iCache
	allSchedtags      iCache
	resourceTypes     sets.String
}

func newSchedtagManagerWithoutInit() *schedtagManager {
	m := &schedtagManager{
		schedtagsByType:        newCache(),
		dynamicSchedtagsByType: newCache(),
		resourceSchedtags:      newCache(),
		allSchedtags:           newCache(),
		resourceTypes:          sets.NewString(),
	}
	return m
}

func newSchedtagManager() (iSchedtagManager, error) {
	m := newSchedtagManagerWithoutInit()
	if err := m.initAllSchedtags(); err != nil {
		return nil, errors.Wrap(err, "initAllSchedtags")
	}
	return m, nil
}

func (as *schedtagManager) initAllSchedtags() error {
	// init all schedtags
	tags := make([]models.SSchedtag, 0)
	if err := models.SchedtagManager.Query().All(&tags); err != nil {
		return errors.Wrap(err, "fetch all schedtags from db")
	}
	iTags := make(map[string][]ISchedtag, 0)
	for i := range tags {
		tag := tags[i]
		obj := newSchedtag(&tags[i])

		as.allSchedtags.set(obj.GetId(), obj)

		resType := tag.ResourceType
		objs, ok := iTags[resType]
		if ok {
			objs = append(objs, obj)
			iTags[resType] = objs
		} else {
			iTags[resType] = []ISchedtag{obj}
		}
	}
	for resType := range iTags {
		as.resourceTypes.Insert(resType)
		tags := iTags[resType]
		as.schedtagsByType.set(resType, tags)
		// initResourceSchedtags
		if err := as.initResourceSchedtags(resType, tags); err != nil {
			return errors.Wrapf(err, "initResourceSchedtags(%q)", resType)
		}
	}

	// // initDynamicschedtags
	// if err := as.initDynamicschedtags(iTags); err != nil {
	// 	return errors.Wrapf(err, "initDynamicschedtags")
	// }
	return nil
}

func (as *schedtagManager) getAllSchedtags(resType string) ([]ISchedtag, error) {
	data, ok := as.schedtagsByType.rawGet(resType)
	if !ok {
		return nil, nil
	}
	return data.([]ISchedtag), nil
}

func (as *schedtagManager) getResourceSchedtagKey(resType, id string) string {
	return fmt.Sprintf("%s/%s", resType, id)
}

func (as *schedtagManager) initResourceSchedtags(resType string, tags []ISchedtag) error {
	resTags := make(map[string][]ISchedtag)

	for i := range tags {
		t := tags[i]
		objs, err := t.getResources()
		if err != nil {
			return errors.Wrapf(err, "get schedtag %s(%s) resources", t.GetName(), t.GetId())
		}
		for _, obj := range objs {
			key := as.getResourceSchedtagKey(resType, obj.GetId())
			tmpTags, ok := resTags[key]
			if !ok {
				resTags[key] = []ISchedtag{t}
			} else {
				tmpTags = append(tmpTags, t)
				resTags[key] = tmpTags
			}
		}
	}

	for key, tags := range resTags {
		as.resourceSchedtags.set(key, tags)
	}

	return nil
}

func (as *schedtagManager) getResourceTypes() []string {
	return as.resourceTypes.List()
}

func (as *schedtagManager) initDynamicschedtags() error {
	records := make(map[string][]IDynamicschedtag)
	for _, resType := range as.getResourceTypes() {
		dTags := models.DynamicschedtagManager.GetEnabledDynamicSchedtagsByResource(resType)
		for i := range dTags {
			dTag := dTags[i]
			// find schedtag of dynamicschedtag
			sTag, ok := as.allSchedtags.rawGet(dTag.SchedtagId)
			if !ok {
				continue
			}
			obj := newDynamicschedtag(&dTag, sTag.(ISchedtag))
			objs, ok := records[resType]
			if ok {
				objs = append(objs, obj)
				records[resType] = objs
			} else {
				records[resType] = []IDynamicschedtag{obj}
			}
		}
	}

	for resType := range records {
		as.dynamicSchedtagsByType.set(resType, records[resType])
	}

	return nil
}

func (as *schedtagManager) getCandidateSchedtags(resType, id string) []ISchedtag {
	ret, ok := as.resourceSchedtags.rawGet(as.getResourceSchedtagKey(resType, id))
	if !ok {
		return nil
	}
	return ret.([]ISchedtag)
}

func (as *schedtagManager) getEnabledDynamicSchedtagsByResource(resType string) ([]IDynamicschedtag, error) {
	// ret, ok := as.dynamicSchedtagsByType.rawGet(resType)
	// if !ok {
	// 	return nil, nil
	// }
	// return ret.([]IDynamicschedtag), nil
	data, err := as.dynamicSchedtagsByType.get(resType, func() (interface{}, error) {
		dynamicTags := models.DynamicschedtagManager.GetEnabledDynamicSchedtagsByResource(resType)
		ret := make([]IDynamicschedtag, len(dynamicTags))
		for i := range dynamicTags {
			dTag := &dynamicTags[i]
			sTag, ok := as.allSchedtags.rawGet(dTag.SchedtagId)
			if !ok {
				return nil, errors.Errorf("not found schedtag %q by dynamicschedtag %q", dTag.SchedtagId, dTag.GetName())
			}
			ret[i] = newDynamicschedtag(dTag, sTag.(ISchedtag))
		}
		return ret, nil
	})
	if err != nil {
		return nil, errors.Wrapf(err, "get dynamicschedtags by %q", resType)
	}
	return data.([]IDynamicschedtag), nil
}

type ISchedtag interface {
	GetName() string
	GetId() string
	GetDefaultStrategy() string

	getResources() ([]models.IModelWithSchedtag, error)
}

type schedtag struct {
	*models.SSchedtag
}

func newSchedtag(obj *models.SSchedtag) ISchedtag {
	return &schedtag{
		SSchedtag: obj,
	}
}

func (st *schedtag) GetDefaultStrategy() string {
	return st.SSchedtag.DefaultStrategy
}

func (st *schedtag) getResources() ([]models.IModelWithSchedtag, error) {
	return st.SSchedtag.GetResources()
}

type dynamicschedtag struct {
	*models.SDynamicschedtag

	schedtag ISchedtag
}

type IDynamicschedtag interface {
	GetSchedtag() ISchedtag
	GetCondition() string
	GetId() string
	GetName() string
}

func newDynamicschedtag(obj *models.SDynamicschedtag, sTag ISchedtag) IDynamicschedtag {
	return &dynamicschedtag{
		SDynamicschedtag: obj,
		schedtag:         sTag,
	}
}

func (ds *dynamicschedtag) GetSchedtag() ISchedtag {
	return ds.schedtag
}

func (ds *dynamicschedtag) GetCondition() string {
	return ds.SDynamicschedtag.Condition
}

func init() {
	globalManager = newManager()
}

func getSessionManager(sessionId string) (ISessionManager, error) {
	if globalManager == nil {
		panic("global schedtag manager is not init")
	}
	return globalManager.GetSessionManager(sessionId)
}

func freeSessionManager(sessionId string) {
	globalManager.FreeSessionManager(sessionId)
}
