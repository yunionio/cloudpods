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
	"strings"

	"yunion.io/x/jsonutils"
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
	Metadatas MetadataManager
)

func init() {
	Metadatas = MetadataManager{modules.NewComputeManager("metadata", "metadatas",
		[]string{"id", "key", "value"},
		[]string{})}
	modules.RegisterCompute(&Metadatas)
}

func (this *MetadataManager) getModule(session *mcclient.ClientSession, params jsonutils.JSONObject) (modulebase.Manager, error) {
	service, version := "compute", ""
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
			keyString := resource + "s"
			if strings.HasSuffix(resource, "y") && resource != NatGateways.GetKeyword() {
				keyString = resource[:len(resource)-1] + "ies"
			}
			find := false
			mods, _ := modulebase.GetRegisterdModules()
			for _versin, mds := range mods {
				if utils.IsInStringArray(keyString, mds) {
					version = _versin
					session.SetApiVersion(version)
					mod, err := modulebase.GetModule(session, keyString)
					if err != nil {
						return nil, err
					}
					service = mod.ServiceType()
					find = true
					break
				}
			}
			if !find {
				return nil, fmt.Errorf("No such module %s", keyString)
			}
		}
	}

	switch service {
	case "identity":
		version = "v3"
	case "compute":
		version = "v2"
	default:
		version = "v1"
	}

	session.SetApiVersion(version)
	_, err := session.GetServiceURL(service, "")
	if err != nil {
		return nil, httperrors.NewNotFoundError("service %s not found error: %v", service, err)
	}

	return &modulebase.ResourceManager{
		BaseManager: *modulebase.NewBaseManager(service, "", "", []string{}, []string{}),
		Keyword:     "metadata", KeywordPlural: "metadatas",
	}, nil
}

func (this *MetadataManager) List(session *mcclient.ClientSession, params jsonutils.JSONObject) (*modulebase.ListResult, error) {
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
