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
	"fmt"
	"os"
	"path/filepath"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudcommon/etcd"
)

type IWatcher interface {
	Watch(ctx context.Context, key string, handler ResourceEventHandler) error
	Unwatch(key string)
}

type ResourceEventHandler interface {
	OnAdd(obj *jsonutils.JSONDict)
	OnUpdate(oldObj, newObj *jsonutils.JSONDict)
	OnDelete(obj *jsonutils.JSONDict)
}

type EtcdBackendForClient struct {
	*EtcdBackend
	clientResources map[string]ResourceEventHandler
}

func NewEtcdBackendForClient(opt *etcd.SEtcdOptions) (*EtcdBackendForClient, error) {
	bec := &EtcdBackendForClient{
		clientResources: make(map[string]ResourceEventHandler, 0),
	}
	be, err := newEtcdBackend(opt, bec.onKeepaliveFailure)
	if err != nil {
		return nil, err
	}
	bec.EtcdBackend = be
	bec.StartClientWatch(context.Background())
	return bec, nil
}

func (b *EtcdBackendForClient) onKeepaliveFailure() {
	if err := b.client.RestartSession(); err != nil {
		log.Errorf("restart etcd session error: %v", err)
		return
	}
	b.StartClientWatch(context.Background())
}

func (b *EtcdBackendForClient) StartClientWatch(ctx context.Context) {
	wf := func(key string) string {
		return filepath.Join(EtcdInformerPrefix, key)
	}
	for key, handler := range b.clientResources {
		// if etcd pod deleted, should unwatch then rewatch resource
		b.Unwatch(key)
		if err := b.Watch(ctx, key, handler); err != nil {
			log.Errorf("start watch client resource %s error: %v", key, err)
			continue
		}
		log.Infof("%s rewatched", wf(key))
	}
	b.client.Unwatch("/")
	if err := b.client.Watch(ctx, "/", b.onClientResourceCreate, b.onClientResourceUpdate, b.onClientResourceDelete); err != nil {
		log.Errorf("start watch %s error: %v", wf("/"), err)
	} else {
		log.Infof("%s watched", wf("/"))
	}
}

func (b *EtcdBackend) getClientRegisterKey(resKey string) (string, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return "", errors.Wrap(err, "get hostname")
	}
	clientKey := fmt.Sprintf("/%s/%s/%s", resKey, EtcdInformerClientsKey, hostname)
	return clientKey, nil
}

func (b *EtcdBackendForClient) registerClientResource(ctx context.Context, key string) error {
	clientKey, err := b.getClientRegisterKey(key)
	if err != nil {
		return err
	}
	if err := b.PutSession(ctx, clientKey, "ok"); err != nil {
		return err
	}
	return nil
}

func (b *EtcdBackendForClient) Watch(ctx context.Context, key string, handler ResourceEventHandler) error {
	b.clientResources[key] = handler
	if err := b.registerClientResource(ctx, key); err != nil {
		return errors.Wrapf(err, "register watch client resource %s", key)
	}
	return b.client.Watch(ctx, b.getWatchKey(key), b.onCreate(ctx, handler), b.onModify(ctx, handler), nil)
}

func (b *EtcdBackendForClient) Unwatch(key string) {
	delete(b.clientResources, key)
	b.client.Unwatch(b.getWatchKey(key))
}

func (b *EtcdBackendForClient) onCreate(ctx context.Context, handler ResourceEventHandler) etcd.TEtcdCreateEventFunc {
	return func(ctx context.Context, key, value []byte) {
		b.processEvent(handler, string(key), value)
	}
}

func (b *EtcdBackendForClient) onModify(ctx context.Context, handler ResourceEventHandler) etcd.TEtcdModifyEventFunc {
	return func(ctx context.Context, key, _, value []byte) {
		// not care about oldvalue, so ignore it
		b.processEvent(handler, string(key), value)
	}
}

func (b *EtcdBackendForClient) processEvent(handler ResourceEventHandler, key string, value []byte) {
	if len(value) == 0 {
		// object already deleted by lease out of ttl
		return
	}
	if b.isClientsKey([]byte(key)) {
		return
	}
	mObj, err := newModelObjectFromValue(value)
	if err != nil {
		log.Errorf("new %s model objecd from value error: %v", key, err)
		return
	}
	eType := mObj.EventType
	switch eType {
	case EventTypeCreate:
		handler.OnAdd(mObj.Object)
	case EventTypeUpdate:
		handler.OnUpdate(mObj.OldObject, mObj.Object)
	case EventTypeDelete:
		handler.OnDelete(mObj.Object)
	default:
		log.Errorf("Invalid key %s, event type: %s, mObj: %#v", key, eType, mObj)
	}
}

func (b *EtcdBackendForClient) onClientResourceAdd(key []byte) {
	// do nothing
	// self registered key has store in b.clientResources
}

func (b *EtcdBackendForClient) onClientResourceCreate(ctx context.Context, key, value []byte) {
	b.onClientResourceAdd(key)
}

func (b *EtcdBackendForClient) onClientResourceUpdate(ctx context.Context, key, oldvalue, value []byte) {
	b.onClientResourceAdd(key)
}

func (b *EtcdBackendForClient) onClientResourceDelete(ctx context.Context, key []byte) {
	if !b.isClientsKey(key) {
		return
	}
	keywordPlural, err := b.getClientWatchResource(key)
	if err != nil {
		log.Errorf("get client watch resource keywordPlural error: %v", err)
		return
	}
	_, ok := b.clientResources[keywordPlural]
	if !ok {
		return
	}
	if err := b.registerClientResource(ctx, keywordPlural); err != nil {
		log.Errorf("rewatch client resource %s error: %v", keywordPlural, err)
		return
	}
}
