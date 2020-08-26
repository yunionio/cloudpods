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

package informer

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudcommon/etcd"
	"yunion.io/x/onecloud/pkg/cloudcommon/informer"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SWatchManager struct {
	client        *mcclient.Client
	region        string
	interfaceType string
	watchBackend  informer.IWatcher
}

func NewWatchManagerBySession(session *mcclient.ClientSession) (*SWatchManager, error) {
	return NewWatchManager(session.GetClient(), session.GetToken(), session.GetRegion(), session.GetEndpointType())
}

func NewWatchManager(client *mcclient.Client, token mcclient.TokenCredential, region, interfaceType string) (*SWatchManager, error) {
	endpoint, err := client.GetCommonEtcdEndpoint(token, region, interfaceType)
	if err != nil {
		return nil, errors.Wrap(err, "get common etcd endpoint")
	}
	tlsCfg, err := client.GetCommonEtcdTLSConfig(endpoint)
	if err != nil {
		return nil, errors.Wrap(err, "get common etcd tls config")
	}
	opt := &etcd.SEtcdOptions{
		EtcdEndpoint:              []string{endpoint.Url},
		EtcdTimeoutSeconds:        5,
		EtcdRequestTimeoutSeconds: 10,
		EtcdLeaseExpireSeconds:    5,
	}
	if tlsCfg != nil {
		tlsCfg.InsecureSkipVerify = true
		opt.EtcdEnabldSsl = true
		opt.TLSConfig = tlsCfg
	}
	be, err := informer.NewEtcdBackendForClient(opt)
	if err != nil {
		return nil, errors.Wrap(err, "new etcd informer backend")
	}
	man := &SWatchManager{
		client:        client,
		region:        region,
		interfaceType: interfaceType,
		watchBackend:  be,
	}
	return man, nil
}

type IResourceManager interface {
	KeyString() string
	GetKeyword() string
}

type EventHandler interface {
	OnAdd(obj *jsonutils.JSONDict)
	OnUpdate(oldObj, newObj *jsonutils.JSONDict)
	OnDelete(obj *jsonutils.JSONDict)
}

type IWatcher interface {
	AddEventHandler(ctx context.Context, handler EventHandler) error
}

type sWatcher struct {
	manager         *SWatchManager
	resourceManager IResourceManager
	ctx             context.Context
	eventHandler    EventHandler
}

func (man *SWatchManager) For(resMan IResourceManager) IWatcher {
	watcher := &sWatcher{
		manager:         man,
		resourceManager: resMan,
	}
	return watcher
}

func (w *sWatcher) AddEventHandler(ctx context.Context, handler EventHandler) error {
	w.ctx = ctx
	w.eventHandler = w.wrapEventHandler(handler)
	return w.manager.watch(w.ctx, w.resourceManager, w.eventHandler)
}

func (man *SWatchManager) watch(ctx context.Context, resMan IResourceManager, handler informer.ResourceEventHandler) error {
	return man.watchBackend.Watch(ctx, resMan.KeyString(), handler)
}

func (w *sWatcher) wrapEventHandler(handler EventHandler) informer.ResourceEventHandler {
	return &wrapEventHandler{handler}
}

type wrapEventHandler struct {
	handler EventHandler
}

func (h *wrapEventHandler) OnAdd(obj *jsonutils.JSONDict) {
	h.handler.OnAdd(obj)
}

func (h *wrapEventHandler) OnUpdate(oldObj, newObj *jsonutils.JSONDict) {
	h.handler.OnUpdate(oldObj, newObj)
}

func (h *wrapEventHandler) OnDelete(obj *jsonutils.JSONDict) {
	h.handler.OnDelete(obj)
}

type EventHandlerFuncs struct {
	AddFunc    func(obj *jsonutils.JSONDict)
	UpdateFunc func(oldObj, newObj *jsonutils.JSONDict)
	DeleteFunc func(obj *jsonutils.JSONDict)
}

func (r EventHandlerFuncs) OnAdd(obj *jsonutils.JSONDict) {
	if r.AddFunc != nil {
		r.AddFunc(obj)
	}
}

func (r EventHandlerFuncs) OnUpdate(oldObj, newObj *jsonutils.JSONDict) {
	if r.UpdateFunc != nil {
		r.UpdateFunc(oldObj, newObj)
	}
}

func (r EventHandlerFuncs) OnDelete(obj *jsonutils.JSONDict) {
	if r.DeleteFunc != nil {
		r.DeleteFunc(obj)
	}
}

type FilteringEventHandler struct {
	FilterFunc func(obj *jsonutils.JSONDict) bool
	Handler    EventHandler
}

func (r FilteringEventHandler) OnAdd(obj *jsonutils.JSONDict) {
	if !r.FilterFunc(obj) {
		return
	}
	r.Handler.OnAdd(obj)
}

func (r FilteringEventHandler) OnUpdate(oldObj, newObj *jsonutils.JSONDict) {
	newer := r.FilterFunc(newObj)
	older := r.FilterFunc(oldObj)
	switch {
	case newer && older:
		r.Handler.OnUpdate(oldObj, newObj)
	case newer && !older:
		r.Handler.OnAdd(newObj)
	case !newer && older:
		r.Handler.OnDelete(oldObj)
	default:
		// do nothing
	}
}

func (r FilteringEventHandler) OnDelete(obj *jsonutils.JSONDict) {
	if !r.FilterFunc(obj) {
		return
	}
	r.Handler.OnDelete(obj)
}
