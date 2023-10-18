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
	"context"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/wait"

	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/informer"
	"yunion.io/x/onecloud/pkg/mcclient/modules/compute"
)

var (
	globalDataManager *dataManager
)

func Start(ctx context.Context, refreshInterval time.Duration) error {
	var err error
	globalDataManager, err = newDataManager(refreshInterval)
	if err != nil {
		panic(errors.Wrap(err, "newDataManager"))
	}
	globalDataManager.startWatch(ctx)
	globalDataManager.sync()
	return nil
}

func GetAllSchedtags(resType string) ([]ISchedtag, error) {
	return globalDataManager.tagMan.getAllSchedtags(resType)
}

func GetEnabledDynamicSchedtagsByResource(resType string) ([]IDynamicschedtag, error) {
	ret, ok := globalDataManager.tagMan.dynamicSchedtagsByType.rawGet(resType)
	if !ok {
		return nil, nil
	}
	return ret.([]IDynamicschedtag), nil
}

func GetCandidateSchedtags(resType, id string) []ISchedtag {
	return globalDataManager.tagMan.getCandidateSchedtags(resType, id)
}

type dataManager struct {
	tagMan *schedtagManager

	refreshInterval time.Duration
}

func newDataManager(refreshInterval time.Duration) (*dataManager, error) {
	man := &dataManager{
		refreshInterval: refreshInterval,
	}
	return man, nil
}

func (m *dataManager) startWatch(ctx context.Context) {
	s := auth.GetAdminSession(ctx, consts.GetRegion())
	informer.NewWatchManagerBySessionBg(s, func(man *informer.SWatchManager) error {
		for _, res := range []informer.IResourceManager{
			compute.Schedtags,
			compute.Schedtaghosts,
			compute.Schedtagstorages,
			compute.Schedtagnetworks,
			compute.Schedtagcloudproviders,
			compute.Schedtagcloudregions,
			compute.Schedtagzones,
		} {
			if err := man.For(res).AddEventHandler(ctx, newEventHandler(res, m)); err != nil {
				return errors.Wrapf(err, "watch resource %s", res.KeyString())
			}
		}
		return nil
	})
}

type eventHandler struct {
	res     informer.IResourceManager
	dataMan *dataManager
}

func newEventHandler(res informer.IResourceManager, dataMan *dataManager) informer.EventHandler {
	return &eventHandler{
		res:     res,
		dataMan: dataMan,
	}
}

func (e eventHandler) keyword() string {
	return e.res.GetKeyword()
}

func (e eventHandler) OnAdd(obj *jsonutils.JSONDict) {
	log.Infof("%s [CREATED]: \n%s", e.keyword(), obj.String())
	e.dataMan.syncOnce()
}

func (e eventHandler) OnUpdate(oldObj, newObj *jsonutils.JSONDict) {
	log.Infof("%s [UPDATED]: \n[NEW]: %s\n[OLD]: %s", e.keyword(), newObj.String(), oldObj.String())
	e.dataMan.syncOnce()
}

func (e eventHandler) OnDelete(obj *jsonutils.JSONDict) {
	log.Infof("%s [DELETED]: \n%s", e.keyword(), obj.String())
	e.dataMan.syncOnce()
}

func (m *dataManager) sync() {
	wait.Forever(m.syncOnce, m.refreshInterval)
}

func (m *dataManager) syncOnce() {
	log.Infof("Schedtag data start sync")
	startTime := time.Now()

	if err := func() error {
		m.tagMan = newSchedtagManagerWithoutInit()
		if err := m.tagMan.initAllSchedtags(); err != nil {
			return errors.Wrap(err, "initAllSchedtags")
		}
		if err := m.tagMan.initDynamicschedtags(); err != nil {
			return errors.Wrap(err, "initResourceSchedtags")
		}
		return nil
	}(); err != nil {
		log.Errorf("Schedtag sync data error: %v", err)
		return
	}

	log.Infof("Schedtag finish sync, elapsed %s", time.Since(startTime))
}
