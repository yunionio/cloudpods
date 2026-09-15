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
	"context"
	"fmt"
	"strings"
	"sync"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/image"
	"yunion.io/x/onecloud/pkg/image/drivers/container_registries/client"
	"yunion.io/x/onecloud/pkg/mcclient"
)

var containerRegistryDrivers = NewDriverManager("")

type IContainerRegistryDriver interface {
	GetType() api.ContainerRegistryType

	CreateCredential(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *api.ContainerRegistryCreateInput) (string, error)
	GetDockerRegistryClient(url string, config *api.ContainerRegistryConfig) (client.Client, error)

	ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *api.ContainerRegistryCreateInput) (*api.ContainerRegistryCreateInput, error)

	PreparePushImage(ctx context.Context, url string, conf *api.ContainerRegistryConfig, meta *client.ImageMetadata) error
	DownloadImage(ctx context.Context, url string, conf *api.ContainerRegistryConfig, input api.ContainerRegistryDownloadImageInput) (string, error)
}

func RegisterContainerRegistryDriver(driver IContainerRegistryDriver) {
	if err := containerRegistryDrivers.Register(driver, string(driver.GetType())); err != nil {
		panic(fmt.Sprintf("register container registry driver type %q: %v", driver.GetType(), err))
	}
}

func GetContainerRegistryDriver(rType api.ContainerRegistryType) (IContainerRegistryDriver, error) {
	drv, err := containerRegistryDrivers.Get(string(rType))
	if err != nil {
		return nil, errors.Wrapf(err, "get container registry driver by type %q", rType)
	}
	return drv.(IContainerRegistryDriver), nil
}

type DriverManager struct {
	*sync.Map
	keySep string
}

func NewDriverManager(keySep string) *DriverManager {
	if len(keySep) == 0 {
		keySep = "->"
	}
	return &DriverManager{
		Map:    new(sync.Map),
		keySep: keySep,
	}
}

func (m *DriverManager) getIndexKey(keys ...string) string {
	return strings.Join(keys, m.keySep)
}

func (m *DriverManager) Register(drv interface{}, keys ...string) error {
	key := m.getIndexKey(keys...)
	_, ok := m.Load(key)
	if ok {
		return errors.Error(fmt.Sprintf("Driver %s already register", key))
	}
	m.Store(key, drv)
	return nil
}

var ErrDriverNotFound = errors.Error("driver not found")

func (m *DriverManager) Get(keys ...string) (interface{}, error) {
	key := m.getIndexKey(keys...)
	drv, ok := m.Load(key)
	if !ok {
		return nil, errors.Wrapf(ErrDriverNotFound, "Not found driver by key: %v", key)
	}
	return drv, nil
}
