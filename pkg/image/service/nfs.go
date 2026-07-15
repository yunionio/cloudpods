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

package service

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	imageapi "yunion.io/x/onecloud/pkg/apis/image"
	"yunion.io/x/onecloud/pkg/image/options"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

func initNFS() error {
	if options.Options.StorageDriver != imageapi.IMAGE_STORAGE_DRIVER_NFS {
		return nil
	}
	if len(options.Options.NfsStorageId) == 0 {
		return fmt.Errorf("nfs_storage_id is required when storage_driver is nfs")
	}

	storage, err := getNFSStorage(options.Options.NfsStorageId)
	if err != nil {
		return errors.Wrapf(err, "get nfs storage %s", options.Options.NfsStorageId)
	}
	if storage.StorageType != computeapi.STORAGE_NFS {
		return fmt.Errorf("storage %s is %s, not nfs", options.Options.NfsStorageId, storage.StorageType)
	}
	if storage.StorageConf == nil {
		return fmt.Errorf("storage %s has empty storage_conf", options.Options.NfsStorageId)
	}

	host, err := storage.StorageConf.GetString("nfs_host")
	if err != nil {
		return errors.Wrapf(err, "storage %s missing nfs_host", options.Options.NfsStorageId)
	}
	sharedDir, err := storage.StorageConf.GetString("nfs_shared_dir")
	if err != nil {
		return errors.Wrapf(err, "storage %s missing nfs_shared_dir", options.Options.NfsStorageId)
	}

	return mountNFS(host, sharedDir, options.Options.NfsMountPoint, options.Options.NfsMountOptions)
}

func getNFSStorage(storageId string) (*computeapi.StorageDetails, error) {
	params := jsonutils.NewDict()
	params.Set("details", jsonutils.JSONTrue)
	obj, err := compute.Storages.Get(auth.GetAdminSession(context.Background(), options.Options.Region), storageId, params)
	if err != nil {
		return nil, err
	}
	storage := new(computeapi.StorageDetails)
	if err := obj.Unmarshal(storage); err != nil {
		return nil, errors.Wrap(err, "unmarshal storage details")
	}
	return storage, nil
}

func mountNFS(host, sharedDir, mountPoint, mountOptions string) error {
	if err := os.MkdirAll(mountPoint, 0755); err != nil {
		return errors.Wrapf(err, "mkdir %s", mountPoint)
	}
	if err := os.MkdirAll(path.Join(mountPoint, imageapi.NfsSubDirName), 0755); err != nil {
		return errors.Wrapf(err, "mkdir %s", mountPoint)
	}
	source := fmt.Sprintf("%s:%s", host, sharedDir)
	if out, err := procutils.NewRemoteCommandAsFarAsPossible("mountpoint", mountPoint).Output(); err == nil {
		log.Infof("%s has already been mounted as glance nfs store", mountPoint)
		return nil
	} else {
		log.Infof("%s is not mounted yet: %s", mountPoint, strings.TrimSpace(string(out)))
	}

	args := []string{"-t", "nfs"}
	if len(mountOptions) > 0 {
		args = append(args, "-o", mountOptions)
	}
	args = append(args, source, mountPoint)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	out, err := procutils.NewRemoteCommandContextAsFarAsPossible(ctx, "mount", args...).Output()
	if err != nil {
		return errors.Wrapf(err, "mount %s to %s failed: %s", source, mountPoint, out)
	}
	log.Infof("mounted nfs %s to glance filesystem store %s", source, mountPoint)

	return nil
}
