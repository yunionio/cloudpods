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
	"path/filepath"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudcommon/etcd"
)

const (
	EtcdInformerPrefix     = "/onecloud/informer"
	EtcdInformerClientsKey = "@clients"

	EventTypeCreate = "CREATE"
	EventTypeUpdate = "UPDATE"
	EventTypeDelete = "DELETE"
)

type TEventType string

type modelObject struct {
	EventType TEventType          `json:"event_type"`
	Object    *jsonutils.JSONDict `json:"object"`
	OldObject *jsonutils.JSONDict `json:"old_object"`
}

func (obj modelObject) ToKey() string {
	return jsonutils.Marshal(obj).String()
}

func newModelObjectFromValue(val []byte) (*modelObject, error) {
	jObj, err := jsonutils.Parse(val)
	if err != nil {
		return nil, errors.Wrapf(err, "parse key %s", val)
	}
	ret := new(modelObject)
	if err := jObj.Unmarshal(ret); err != nil {
		return nil, errors.Wrap(err, "unmarshal to model object")
	}
	return ret, nil
}

type EtcdBackend struct {
	client   *etcd.SEtcdClient
	leaseTTL int64
}

func newEtcdBackend(opt *etcd.SEtcdOptions, onKeepaliveFailure func()) (*EtcdBackend, error) {
	opt.EtcdNamspace = EtcdInformerPrefix
	be := new(EtcdBackend)
	be.leaseTTL = int64(opt.EtcdLeaseExpireSeconds)
	if onKeepaliveFailure == nil {
		onKeepaliveFailure = be.onKeepaliveFailure
	}
	cli, err := etcd.NewEtcdClient(opt, onKeepaliveFailure)
	if err != nil {
		return nil, errors.Wrap(err, "new etcd client")
	}
	be.client = cli
	return be, nil
}

func NewEtcdBackend(opt *etcd.SEtcdOptions, onKeepaliveFailure func()) (*EtcdBackend, error) {
	be, err := newEtcdBackend(opt, onKeepaliveFailure)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	be.initClientResources(ctx)
	be.StartClientWatch(ctx)
	return be, nil
}

func (b *EtcdBackend) initClientResources(ctx context.Context) error {
	pairs, err := b.client.List(ctx, "/")
	if err != nil {
		return errors.Wrap(err, "list all client resources")
	}
	for _, pair := range pairs {
		keywordPlural, err := b.getClientWatchResource([]byte(pair.Key))
		if err == nil && len(keywordPlural) != 0 {
			AddWatchedResources(keywordPlural)
		}
	}
	return nil
}

func (b *EtcdBackend) getObjectKey(obj *ModelObject) string {
	if obj.IsJoint {
		return fmt.Sprintf("/%s/%s/%s", obj.KeywordPlural, obj.MasterId, obj.SlaveId)
	}
	return fmt.Sprintf("/%s/%s", obj.KeywordPlural, obj.Id)
}

func (b *EtcdBackend) getValue(eventType TEventType, obj *ModelObject) string {
	modelObject := modelObject{
		EventType: eventType,
		Object:    obj.Object,
	}
	return modelObject.ToKey()
}

func (b *EtcdBackend) getUpdateValue(obj *ModelObject, oldObj *jsonutils.JSONDict) string {
	modelObject := modelObject{
		EventType: EventTypeUpdate,
		Object:    obj.Object,
		OldObject: oldObj,
	}
	return modelObject.ToKey()
}

func (b *EtcdBackend) GetType() string {
	return "etcd"
}

func (b *EtcdBackend) Create(ctx context.Context, obj *ModelObject) error {
	key := b.getObjectKey(obj)
	val := b.getValue(EventTypeCreate, obj)
	return b.put(ctx, key, val)
}

func (b *EtcdBackend) Update(ctx context.Context, obj *ModelObject, oldObj *jsonutils.JSONDict) error {
	key := b.getObjectKey(obj)
	val := b.getUpdateValue(obj, oldObj)
	return b.put(ctx, key, val)
}

func (b *EtcdBackend) Delete(ctx context.Context, obj *ModelObject) error {
	key := b.getObjectKey(obj)
	val := b.getValue(EventTypeDelete, obj)
	err := b.put(ctx, key, val)
	return err
}

func (b *EtcdBackend) put(ctx context.Context, key, val string) error {
	return b.PutWithLease(ctx, key, val, b.leaseTTL)
}

func (b *EtcdBackend) PutSession(ctx context.Context, key, val string) error {
	return b.client.PutSession(ctx, key, val)
}

func (b *EtcdBackend) PutWithLease(ctx context.Context, key, val string, ttlSeconds int64) error {
	return b.client.PutWithLease(ctx, key, val, ttlSeconds)
}

func (b *EtcdBackend) onKeepaliveFailure() {
	if err := b.client.RestartSession(); err != nil {
		log.Errorf("restart etcd session error: %v", err)
		return
	}
	b.StartClientWatch(context.Background())
}

func (b *EtcdBackend) StartClientWatch(ctx context.Context) {
	b.client.Unwatch("/")
	b.client.Watch(ctx, "/", b.onClientResourceCreate, b.onClientResourceUpdate, b.onClientResourceDelete)
}

func (b *EtcdBackend) isClientsKey(key []byte) bool {
	return strings.Contains(string(key), EtcdInformerClientsKey)
}

func (b *EtcdBackend) getClientWatchResource(key []byte) (string, error) {
	// key is like: /servers/@clients/default-climc-5d4c8d49f6-p6l68
	keyPath := string(key)
	parts := strings.Split(keyPath, "/")
	if len(parts) != 4 {
		return "", errors.Errorf("invalid client resource key: %v", parts)
	}
	if parts[2] != EtcdInformerClientsKey {
		return "", errors.Errorf("key %s not contains %s", keyPath, EtcdInformerClientsKey)
	}
	return parts[1], nil
}

func (b *EtcdBackend) onClientResourceAdd(key []byte) {
	if !b.isClientsKey(key) {
		return
	}
	keywordPlural, err := b.getClientWatchResource(key)
	if err != nil {
		log.Errorf("get client watch resource error: %v", err)
		return
	}
	AddWatchedResources(keywordPlural)
}

func (b *EtcdBackend) onClientResourceCreate(ctx context.Context, key, value []byte) {
	b.onClientResourceAdd(key)
}

func (b *EtcdBackend) onClientResourceUpdate(ctx context.Context, key, oldvalue, value []byte) {
	b.onClientResourceAdd(key)
}

func (b *EtcdBackend) shouldDeleteWatchedResource(ctx context.Context, keywordPlural string) bool {
	// clientsKey is like: /servers/@clients
	clientsKey := fmt.Sprintf("/%s/%s", keywordPlural, EtcdInformerClientsKey)
	pairs, err := b.client.List(ctx, clientsKey)
	if err != nil {
		log.Errorf("list clientsKey %s error: %v", clientsKey, err)
		return false
	}
	return len(pairs) == 0
}

func (b *EtcdBackend) onClientResourceDelete(ctx context.Context, key []byte) {
	if !b.isClientsKey(key) {
		return
	}
	keywordPlural, err := b.getClientWatchResource(key)
	if err != nil {
		log.Errorf("get client watch resource keywordPlural error: %v", err)
		return
	}
	if b.shouldDeleteWatchedResource(ctx, keywordPlural) {
		DeleteWatchedResources(keywordPlural)
	}
}

func (b *EtcdBackend) getWatchKey(key string) string {
	return filepath.Join("/", key)
}
