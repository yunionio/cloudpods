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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudcommon/etcd"
)

const (
	EtcdInformerPrefix = "/onecloud/informer"

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

func NewEtcdBackend(opt *etcd.SEtcdOptions, onKeepaliveFailure func()) (*EtcdBackend, error) {
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
	return b.client.PutWithLease(ctx, key, val, b.leaseTTL)
}

func (b *EtcdBackend) onKeepaliveFailure() {
	if err := b.client.RestartSession(); err != nil {
		log.Errorf("restart etcd session error: %v", err)
	}
}

func (b *EtcdBackend) getWatchKey(key string) string {
	return filepath.Join("/", key)
}

func (b *EtcdBackend) Watch(ctx context.Context, key string, handler ResourceEventHandler) error {
	return b.client.Watch(ctx, b.getWatchKey(key), b.onCreate(handler), b.onModify(handler))
}

func (b *EtcdBackend) Unwatch(key string) {
	b.client.Unwatch(b.getWatchKey(key))
}

func (b *EtcdBackend) onCreate(handler ResourceEventHandler) etcd.TEtcdCreateEventFunc {
	return func(key, value []byte) {
		b.processEvent(handler, string(key), value)
	}
}

func (b *EtcdBackend) onModify(handler ResourceEventHandler) etcd.TEtcdModifyEventFunc {
	return func(key, _, value []byte) {
		// not care about oldvalue, so ignore it
		b.processEvent(handler, string(key), value)
	}
}

func (b *EtcdBackend) processEvent(handler ResourceEventHandler, key string, value []byte) {
	if len(value) == 0 {
		// object already deleted by lease out of ttl
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
		log.Errorf("Invalid event type: %s, mObj: %#v", eType, mObj)
	}
}
