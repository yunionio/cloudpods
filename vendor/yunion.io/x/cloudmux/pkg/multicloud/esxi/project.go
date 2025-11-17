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

package esxi

import (
	"context"

	"github.com/vmware/govmomi/object"
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/pkg/errors"
)

type SProject struct {
	multicloud.SProjectBase
	multicloud.STagBase

	client *SESXiClient
	folder *object.Folder

	Name string
	Id   string
}

func (project *SProject) GetId() string {
	return project.Id
}

func (project *SProject) GetGlobalId() string {
	return project.Id
}

func (project *SProject) GetName() string {
	return project.Name
}

func (project *SProject) GetStatus() string {
	return api.EXTERNAL_PROJECT_STATUS_AVAILABLE
}

func (cli *SESXiClient) listAllFolders(ctx context.Context, folder *object.Folder, prefix string) ([]SProject, error) {
	ret := []SProject{}
	name, err := folder.ObjectName(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "ObjectName")
	}
	if len(prefix) > 0 {
		name = prefix + name
	}
	ret = append(ret, SProject{
		folder: folder,
		Id:     folder.Reference().Value,
		Name:   name,
	})

	children, err := folder.Children(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "Children")
	}

	for _, child := range children {
		if subFolder, ok := child.(*object.Folder); ok {
			folders, err := cli.listAllFolders(ctx, subFolder, name+"|")
			if err != nil {
				return nil, errors.Wrap(err, "listAllFolders")
			}
			ret = append(ret, folders...)
		}
	}
	return ret, nil
}

func (dc *SDatacenter) GetVMFolders() ([]SProject, error) {
	vmFolders, err := dc.getObjectDatacenter().Folders(dc.manager.context)
	if err != nil {
		return nil, errors.Wrap(err, "Folders")
	}
	ret, err := dc.manager.listAllFolders(dc.manager.context, vmFolders.VmFolder, "")
	if err != nil {
		return nil, errors.Wrap(err, "listAllFolders")
	}
	return ret, nil
}

func (dc *SDatacenter) GetFolder(folderId string) (*object.Folder, error) {
	vmFolders, err := dc.getObjectDatacenter().Folders(dc.manager.context)
	if err != nil {
		return nil, errors.Wrap(err, "Folders")
	}
	ret, err := dc.manager.listAllFolders(dc.manager.context, vmFolders.VmFolder, "")
	if err != nil {
		return nil, errors.Wrap(err, "listAllFolders")
	}
	for i := range ret {
		if ret[i].Id == folderId {
			return ret[i].folder, nil
		}
	}
	return vmFolders.VmFolder, nil
}

func (cli *SESXiClient) GetVMFolders() ([]SProject, error) {
	dcs, err := cli.GetDatacenters()
	if err != nil {
		return nil, errors.Wrap(err, "GetDatacenters")
	}
	ret := make([]SProject, 0)
	for i := range dcs {
		folders, err := dcs[i].GetVMFolders()
		if err != nil {
			return nil, errors.Wrap(err, "GetVMFolders")
		}
		ret = append(ret, folders...)
	}
	return ret, nil
}

func (cli *SESXiClient) GetIProjects() ([]cloudprovider.ICloudProject, error) {
	projects, err := cli.GetVMFolders()
	if err != nil {
		return nil, errors.Wrap(err, "GetVMFolders")
	}
	ret := make([]cloudprovider.ICloudProject, 0)
	for i := range projects {
		projects[i].client = cli
		ret = append(ret, &projects[i])
	}
	return ret, nil
}
