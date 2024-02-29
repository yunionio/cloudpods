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

package cephfs

import (
	"fmt"
	"net/url"
	"strings"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/pkg/errors"
)

type SCephFsDir struct {
	multicloud.SVirtualResourceBase
	multicloud.SBillingBase
	multicloud.STagBase
	client *SCephFSClient

	Name   string
	Path   string
	Parent string
	Quotas struct {
		MaxFiles int64
		MaxBytes int64
	}
}

func (dir *SCephFsDir) GetId() string {
	return dir.Path
}

func (dir *SCephFsDir) GetGlobalId() string {
	return dir.Path
}

func (dir *SCephFsDir) GetName() string {
	return dir.Name
}

func (dir *SCephFsDir) GetStatus() string {
	return api.NAS_STATUS_AVAILABLE
}

func (dir *SCephFsDir) GetFileSystemType() string {
	return "standard"
}

func (dir *SCephFsDir) GetStorageType() string {
	return "capacity"
}

func (dir *SCephFsDir) GetProtocol() string {
	return "CephFS"
}

func (dir *SCephFsDir) GetCapacityGb() int64 {
	return dir.Quotas.MaxBytes / 1024 / 1024 / 1024
}

func (dir *SCephFsDir) GetUsedCapacityGb() int64 {
	return 0
}

func (dir *SCephFsDir) GetMountTargetCountLimit() int {
	return 0
}

func (dir *SCephFsDir) GetZoneId() string {
	return ""
}

func (dir *SCephFsDir) GetMountTargets() ([]cloudprovider.ICloudMountTarget, error) {
	return []cloudprovider.ICloudMountTarget{}, nil
}

func (dir *SCephFsDir) CreateMountTarget(opts *cloudprovider.SMountTargetCreateOptions) (cloudprovider.ICloudMountTarget, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (dir *SCephFsDir) Delete() error {
	return dir.client.DeleteDir(dir.client.fsId, dir.Path)
}

func (cli *SCephFSClient) GetCephDirs(fsId string) ([]SCephFsDir, error) {
	res := fmt.Sprintf("cephfs/%s/ls_dir", fsId)
	params := url.Values{}
	resp, err := cli.list(res, params)
	if err != nil {
		return nil, err
	}
	ret := []SCephFsDir{}
	err = resp.Unmarshal(&ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (cli *SCephFSClient) GetICloudFileSystems() ([]cloudprovider.ICloudFileSystem, error) {
	dirs, err := cli.GetCephDirs(cli.fsId)
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudFileSystem{}
	for i := range dirs {
		dirs[i].client = cli
		ret = append(ret, &dirs[i])
	}
	return ret, nil
}

func (cli *SCephFSClient) GetICloudFileSystemById(id string) (cloudprovider.ICloudFileSystem, error) {
	dirs, err := cli.GetCephDirs(cli.fsId)
	if err != nil {
		return nil, err
	}
	for i := range dirs {
		dirs[i].client = cli
		if dirs[i].GetGlobalId() == id {
			return &dirs[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
}

func (cli *SCephFSClient) CreateDir(fsId, path string) error {
	res := fmt.Sprintf("cephfs/%s/tree", fsId)
	_, err := cli.post(res, map[string]interface{}{
		"path": fmt.Sprintf("/%s", strings.TrimPrefix(path, "/")),
	})
	return err
}

func (cli *SCephFSClient) DeleteDir(fsId, path string) error {
	res := fmt.Sprintf("cephfs/%s/tree", fsId)
	_, err := cli.delete(res, map[string]interface{}{
		"path": fmt.Sprintf("/%s", strings.TrimPrefix(path, "/")),
	})
	return err
}

func (cli *SCephFSClient) SetDirQuota(fsId, path string, maxBytes int64) error {
	res := fmt.Sprintf("cephfs/%s/quota", fsId)
	_, err := cli.put(res, map[string]interface{}{
		"path":      fmt.Sprintf("/%s", strings.TrimPrefix(path, "/")),
		"max_bytes": maxBytes,
	})
	return err
}

func (cli *SCephFSClient) CreateICloudFileSystem(opts *cloudprovider.FileSystemCraeteOptions) (cloudprovider.ICloudFileSystem, error) {
	err := cli.CreateDir(cli.fsId, opts.Name)
	if err != nil {
		return nil, err
	}
	cli.SetDirQuota(cli.fsId, opts.Name, opts.Capacity*1024*1024*1024)
	return cli.GetICloudFileSystemById("/" + opts.Name)
}
