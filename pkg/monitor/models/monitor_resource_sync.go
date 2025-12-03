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

package models

import (
	"fmt"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis/monitor"
)

var (
	resourceSyncMap     map[string]IResourceSync
	guestResourceSync   IResourceSync
	hostResourceSync    IResourceSync
	redisResourceSync   IResourceSync
	rdsResourceSync     IResourceSync
	ossResourceSync     IResourceSync
	accountResourceSync IResourceSync
	storageResourceSync IResourceSync
)

func RegistryResourceSync(sync IResourceSync) error {
	if resourceSyncMap == nil {
		resourceSyncMap = make(map[string]IResourceSync)
	}
	if _, ok := resourceSyncMap[sync.SyncType()]; ok {
		return errors.Errorf(fmt.Sprintf("syncType:%s has registered", sync.SyncType()))
	}
	resourceSyncMap[sync.SyncType()] = sync
	return nil
}

func GetResourceSyncByType(syncType string) IResourceSync {
	if resourceSyncMap == nil {
		resourceSyncMap = make(map[string]IResourceSync)
	}
	return resourceSyncMap[syncType]
}

func GetResourceSyncMap() map[string]IResourceSync {
	if resourceSyncMap == nil {
		resourceSyncMap = make(map[string]IResourceSync)
	}
	return resourceSyncMap
}

type SyncObject struct {
	sync IResourceSync
}

type IResourceSync interface {
	SyncType() string
}

type GuestResourceSync struct {
	SyncObject
}

func NewGuestResourceSync() IResourceSync {
	if guestResourceSync == nil {
		sync := new(GuestResourceSync)
		obj := newSyncObj(sync)
		sync.SyncObject = obj
		guestResourceSync = sync
	}

	return guestResourceSync
}

func (g *GuestResourceSync) SyncType() string {
	return monitor.METRIC_RES_TYPE_GUEST
}

type HostResourceSync struct {
	SyncObject
}

func (self *HostResourceSync) SyncType() string {
	return monitor.METRIC_RES_TYPE_HOST
}

func NewHostResourceSync() IResourceSync {
	if hostResourceSync == nil {
		sync := new(HostResourceSync)
		obj := newSyncObj(sync)
		sync.SyncObject = obj
		hostResourceSync = sync
	}
	return hostResourceSync
}

type RedisResourceSync struct {
	SyncObject
}

func NewRedisResourceSync() IResourceSync {
	if redisResourceSync == nil {
		sync := new(RedisResourceSync)
		obj := newSyncObj(sync)
		sync.SyncObject = obj
		redisResourceSync = sync
	}

	return redisResourceSync
}

func (g *RedisResourceSync) SyncType() string {
	return monitor.METRIC_RES_TYPE_REDIS
}

type RdsResourceSync struct {
	SyncObject
}

func NewRdsResourceSync() IResourceSync {
	if rdsResourceSync == nil {
		sync := new(RdsResourceSync)
		obj := newSyncObj(sync)
		sync.SyncObject = obj
		rdsResourceSync = sync
	}

	return rdsResourceSync
}

func (g *RdsResourceSync) SyncType() string {
	return monitor.METRIC_RES_TYPE_RDS
}

type OssResourceSync struct {
	SyncObject
}

func NewOssResourceSync() IResourceSync {
	if ossResourceSync == nil {
		sync := new(OssResourceSync)
		obj := newSyncObj(sync)
		sync.SyncObject = obj
		ossResourceSync = sync
	}

	return ossResourceSync
}

func (g *OssResourceSync) SyncType() string {
	return monitor.METRIC_RES_TYPE_OSS
}

type AccountResourceSync struct {
	SyncObject
}

func NewAccountResourceSync() IResourceSync {
	if accountResourceSync == nil {
		sync := new(AccountResourceSync)
		obj := newSyncObj(sync)
		sync.SyncObject = obj
		accountResourceSync = sync
	}

	return accountResourceSync
}

func (g *AccountResourceSync) SyncType() string {
	return monitor.METRIC_RES_TYPE_CLOUDACCOUNT
}

type StorageResourceSync struct {
	SyncObject
}

func NewStorageResourceSync() IResourceSync {
	if storageResourceSync == nil {
		sync := new(StorageResourceSync)
		obj := newSyncObj(sync)
		sync.SyncObject = obj
		storageResourceSync = sync
	}

	return storageResourceSync
}

func (g *StorageResourceSync) SyncType() string {
	return monitor.METRIC_RES_TYPE_STORAGE
}

func newSyncObj(sync IResourceSync) SyncObject {
	return SyncObject{sync: sync}
}
