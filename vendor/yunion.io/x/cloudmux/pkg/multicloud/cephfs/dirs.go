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
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
)

type SCephFsDir struct {
	multicloud.SNasBase
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

func (dir *SCephFsDir) Refresh() error {
	dirs, err := dir.client.GetCephDirs(dir.client.fsId)
	if err != nil {
		return err
	}
	for i := range dirs {
		if dirs[i].GetGlobalId() == dir.GetGlobalId() {
			return jsonutils.Update(dir, &dirs[i])
		}
	}
	return errors.Wrapf(cloudprovider.ErrNotFound, dir.Path)
}

func (dir *SCephFsDir) SetQuota(input *cloudprovider.SFileSystemSetQuotaInput) error {
	return dir.client.SetQuota(dir.client.fsId, dir.Path, input.MaxGb, input.MaxFiles)
}

func (cli *SCephFSClient) SetQuota(fsId, path string, maxGb, maxFiles int64) error {
	path = fmt.Sprintf("/%s", strings.TrimPrefix(path, "/"))
	res := fmt.Sprintf("cephfs/%s/quota?path=%s", fsId, path)
	params := map[string]interface{}{}
	if maxFiles > 0 {
		params["max_files"] = fmt.Sprintf("%d", maxFiles)
	}
	if maxGb > 0 {
		params["max_bytes"] = fmt.Sprintf("%d", maxGb*1024*1024*1024)
	}
	_, err := cli.put(res, params)
	return err
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
	for i := range ret {
		ret[i].client = cli
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
	res := fmt.Sprintf("cephfs/%s/tree?path=%s", fsId, fmt.Sprintf("/%s", strings.TrimPrefix(path, "/")))
	_, err := cli.delete(res, map[string]interface{}{})
	return err
}

func (cli *SCephFSClient) CreateICloudFileSystem(opts *cloudprovider.FileSystemCraeteOptions) (cloudprovider.ICloudFileSystem, error) {
	err := cli.CreateDir(cli.fsId, opts.Name)
	if err != nil {
		return nil, err
	}
	cli.SetQuota(cli.fsId, opts.Name, opts.Capacity, 0)
	return cli.GetICloudFileSystemById("/" + opts.Name)
}
