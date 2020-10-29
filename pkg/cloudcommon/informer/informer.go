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
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
)

const (
	ErrBackendNotInit = errors.Error("InformerBackend not init")
)

var (
	defaultBackend IInformerBackend
)

type IInformerBackend interface {
	GetType() string
	Create(ctx context.Context, obj *ModelObject) error
	Update(ctx context.Context, obj *ModelObject, oldObj *jsonutils.JSONDict) error
	Delete(ctx context.Context, obj *ModelObject) error
}

func Init(be IInformerBackend) {
	if defaultBackend != nil {
		log.Fatalf("informer backend %q already init", be.GetType())
	}
	defaultBackend = be
}

func Set(be IInformerBackend) {
	defaultBackend = be
}

func GetDefaultBackend() IInformerBackend {
	if defaultBackend == nil {
		log.V(10).Warningf("default informer backend is not init")
	}
	return defaultBackend
}

func IsInit() bool {
	return defaultBackend != nil
}

type ModelObject struct {
	Object        *jsonutils.JSONDict
	KeywordPlural string
	Id            string
	IsJoint       bool
	MasterId      string
	SlaveId       string
}

func NewModel(obj interface{}, keywordPlural, id string) *ModelObject {
	return &ModelObject{
		Object:        jsonutils.Marshal(obj).(*jsonutils.JSONDict),
		KeywordPlural: keywordPlural,
		Id:            id,
	}
}

func NewJointModel(obj interface{}, keywordPlural, masterId, slaveId string) *ModelObject {
	model := NewModel(obj, keywordPlural, "")
	model.IsJoint = true
	model.MasterId = masterId
	model.SlaveId = slaveId
	return model
}

func isResourceWatched(keywordPlural string) bool {
	return GetWatchResources().Has(keywordPlural)
}

func Create(ctx context.Context, obj *ModelObject) error {
	if !isResourceWatched(obj.KeywordPlural) {
		return nil
	}
	return run(ctx, func(ctx context.Context, be IInformerBackend) error {
		return be.Create(ctx, obj)
	})
}

func Update(ctx context.Context, obj *ModelObject, oldObj *jsonutils.JSONDict) error {
	if !isResourceWatched(obj.KeywordPlural) {
		return nil
	}
	return run(ctx, func(ctx context.Context, be IInformerBackend) error {
		return be.Update(ctx, obj, oldObj)
	})
}

func Delete(ctx context.Context, obj *ModelObject) error {
	if !isResourceWatched(obj.KeywordPlural) {
		return nil
	}
	return run(ctx, func(ctx context.Context, be IInformerBackend) error {
		return be.Delete(ctx, obj)
	})
}
