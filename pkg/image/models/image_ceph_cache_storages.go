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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/image/options"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/util/cephutils"
)

func GetRegionCephStorages() (map[string]*computeapi.RbdStorageConf, error) {
	q := struct {
		Scope       string `json:"scope"`
		StorageType string `json:"storage_type"`
	}{
		"system", computeapi.STORAGE_RBD,
	}

	res, err := compute.Storages.List(auth.GetAdminSession(context.Background(), options.Options.Region), jsonutils.Marshal(q))
	if err != nil {
		return nil, errors.Wrap(err, "compute.Storages.List")
	}
	if res.Total <= 0 {
		return nil, nil
	}
	storages := []computeapi.StorageDetails{}
	err = jsonutils.Update(&storages, res.Data)
	if err != nil {
		return nil, errors.Wrap(err, "json parse storage details")
	}
	cephStorages := map[string]*computeapi.RbdStorageConf{}
	for i := range storages {
		if storages[i].StorageType != computeapi.STORAGE_RBD {
			continue
		}
		if jsonutils.QueryBoolean(storages[i].StorageConf, "auto_cache_images", false) {
			conf := new(computeapi.RbdStorageConf)
			if err := storages[i].StorageConf.Unmarshal(conf); err != nil {
				log.Errorf("failed unmarshal storage %s: %s", storages[i].StorageConf, err)
				continue
			}
			cephStorages[storages[i].Id] = conf
		}
	}
	return cephStorages, nil
}

type SCephStorageConf struct {
	StorageIdConf     map[string]*computeapi.RbdStorageConf
	CephFsidStorageId map[string][]string
}

var (
	cephStorages          *SCephStorageConf
	requestedCephStorages bool
)

func GetCephStorages() *SCephStorageConf {
	cephutils.SetCephConfTempDir(options.Options.DefaultImageServiceHomeDir)
	if requestedCephStorages {
		return cephStorages
	}
	cephStorages = getCephStorages()
	requestedCephStorages = true
	return cephStorages
}

func getCephStorages() *SCephStorageConf {
	storagesConf, err := GetRegionCephStorages()
	if err != nil {
		log.Errorf("failed GetCephStorages %s", err)
		return nil
	}
	if len(storagesConf) == 0 {
		log.Infof("No enable auto_cache_image ceph storage found...")
		return nil
	}
	fsidStorageMap := map[string][]string{}
	for id := range storagesConf {
		fsid := getStorageFsid(storagesConf[id])
		if fsid == "" {
			continue
		}
		storages, ok := fsidStorageMap[fsid]
		if !ok {
			storages = make([]string, 0)
		}
		fsidStorageMap[fsid] = append(storages, id)
	}
	return &SCephStorageConf{
		StorageIdConf:     storagesConf,
		CephFsidStorageId: fsidStorageMap,
	}
}

func getStorageFsid(conf *computeapi.RbdStorageConf) string {
	cli, err := cephutils.NewClient(conf.MonHost, conf.Key, conf.Pool, conf.EnableMessengerV2)
	if err != nil {
		log.Errorf("failed new client of ceph storage %s:%s", conf.MonHost, conf.Pool)
		return ""
	}
	defer cli.Close()
	fsid, err := cli.Fsid()
	if err != nil {
		log.Errorf("failed get fsid of ceph storage %s:%s: %s", conf.MonHost, conf.Pool, err)
		return ""
	}
	return fsid
}
