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

package compute

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/printutils"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

type MetadataManager struct {
	modulebase.ResourceManager
}

var (
	ComputeMetadatas  MetadataManager
	IdentityMetadatas MetadataManager
	ImageMetadatas    MetadataManager
)

func init() {
	ComputeMetadatas = MetadataManager{modules.NewComputeManager("metadata", "metadatas",
		[]string{"id", "key", "value"},
		[]string{})}
	// !!! Register computer metadata ONLY !!! QIUJIAN
	modules.RegisterCompute(&ComputeMetadatas)

	IdentityMetadatas = MetadataManager{modules.NewIdentityV3Manager("metadata", "metadatas",
		[]string{"id", "key", "value"},
		[]string{})}
	ImageMetadatas = MetadataManager{modules.NewImageManager("metadata", "metadatas",
		[]string{"id", "key", "value"},
		[]string{})}
}

func (this *MetadataManager) getModule(session *mcclient.ClientSession, params jsonutils.JSONObject) (modulebase.Manager, error) {
	service := "compute"
	if params.Contains("service") {
		service, _ = params.GetString("service")
	} else {
		// 若参数有resources，可根据资源类型自动判断服务类型
		resources := []string{}
		if params.Contains("resources") { // yunionapi
			err := params.Unmarshal(&resources, "resources")
			if err != nil {
				return nil, httperrors.NewInputParameterError("invalid resources format")
			}
		} else if params.Contains("resources.0") { // climc
			resource, _ := params.GetString("resources.0")
			if len(resource) > 0 {
				resources = append(resources, resource)
			}
		}
		if len(resources) >= 1 {
			resource := resources[0]
			find := false
			keyStrings := []string{resource, resource + "s", resource + "ies"}
			mods, _ := modulebase.GetRegisterdModules()
			for _, keyString := range keyStrings {
				if utils.IsInStringArray(keyString, mods) {
					mod, err := modulebase.GetModule(session, keyString)
					if err == nil {
						service = mod.ServiceType()
						find = true
						break
					}
				}
			}
			if !find {
				return nil, fmt.Errorf("No such module %s", resource)
			}
		}
	}

	_, err := session.GetServiceURL(service, "")
	if err != nil {
		return nil, httperrors.NewNotFoundError("service %s not found error: %v", service, err)
	}

	return &modulebase.ResourceManager{
		BaseManager: *modulebase.NewBaseManager(service, "", "", []string{}, []string{}),
		Keyword:     "metadata", KeywordPlural: "metadatas",
	}, nil
}

func (this *MetadataManager) List(session *mcclient.ClientSession, params jsonutils.JSONObject) (*printutils.ListResult, error) {
	mod, err := this.getModule(session, params)
	if err != nil {
		return nil, err
	}
	return mod.List(session, params)
}

func (this *MetadataManager) Get(session *mcclient.ClientSession, id string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	mod, err := this.getModule(session, params)
	if err != nil {
		return nil, err
	}
	return mod.Get(session, id, params)
}
