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

package watcher

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis"
	identity_api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/syncman"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/informer"
)

type SInformerSyncManager struct {
	syncman.SSyncManager
	resourceManager informer.IResourceManager
	done            bool
}

func (manager *SInformerSyncManager) OnAdd(obj *jsonutils.JSONDict) {
	log.Infof("[CREATED]: \n%s", obj.String())
	if manager.NeedSync(obj) {
		manager.SyncOnce()
	}
}

func (manager *SInformerSyncManager) OnUpdate(oldObj, newObj *jsonutils.JSONDict) {
	log.Infof("[UPDATED]: \n[NEW]: %s\n[OLD]: %s", newObj.String(), oldObj.String())
	if manager.NeedSync(oldObj) || manager.NeedSync(newObj) {
		manager.SyncOnce()
	}
}

func (manager *SInformerSyncManager) OnDelete(obj *jsonutils.JSONDict) {
	log.Infof("[DELETED]: \n%s", obj.String())
	if manager.NeedSync(obj) {
		manager.SyncOnce()
	}
}

func (manager *SInformerSyncManager) OnServiceCatalogChange(catalog mcclient.IServiceCatalog) {
	if manager.done {
		return
	}
	url, _ := mcclient.CatalogGetServiceURL(catalog, apis.SERVICE_TYPE_ETCD, consts.GetRegion(), "", identity_api.EndpointInterfaceInternal)
	if len(url) == 0 {
		log.Debugf("[%s] OnServiceCatalogChange: no etcd internal url found, retry", manager.Name())
		return
	}
	err := manager.startWatcher()
	if err != nil {
		log.Errorf("[%s] watching resource errror %s", manager.Name(), err)
		return
	}
	manager.done = true
}

func (manager *SInformerSyncManager) StartWatching(resMan informer.IResourceManager) {
	manager.resourceManager = resMan
	auth.RegisterCatalogListener(manager)
}

func (manager *SInformerSyncManager) startWatcher() error {
	log.Infof("[%s] Start resource informer watcher for %s", manager.Name(), manager.resourceManager.GetKeyword())
	ctx := context.Background()
	s := auth.GetAdminSession(ctx, consts.GetRegion())
	informer.NewWatchManagerBySessionBg(s, func(watchMan *informer.SWatchManager) error {
		if err := watchMan.For(manager.resourceManager).AddEventHandler(ctx, manager); err != nil {
			return errors.Wrapf(err, "watch resource %s", manager.resourceManager.GetKeyword())
		}
		return nil
	})
	return nil
}
