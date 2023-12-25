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

package identity

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	identity_api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

type SServiceManager struct {
	modulebase.ResourceManager
}

var (
	Services   modulebase.ResourceManager
	ServicesV3 SServiceManager
)

func (manager *SServiceManager) GetConfig(s *mcclient.ClientSession, typeStr string) (*jsonutils.JSONDict, error) {
	opts := struct {
		Type  string
		Scope string
	}{
		Type:  s.GetServiceName(typeStr),
		Scope: "system",
	}
	list, err := manager.List(s, jsonutils.Marshal(opts))
	if err != nil {
		return nil, errors.Wrap(err, "List")
	}
	if len(list.Data) == 0 {
		return nil, errors.Wrapf(httperrors.ErrNotFound, "type %s not found", typeStr)
	}
	srv := identity_api.SService{}
	err = list.Data[0].Unmarshal(&srv)
	if err != nil {
		return nil, errors.Wrap(err, "Unmarshal service")
	}

	configObj, err := manager.GetSpecific(s, srv.Id, "config", nil)
	if err != nil {
		return nil, errors.Wrap(err, "GetSpecific config")
	}
	if configObj.Contains("config") {
		configObj, _ = configObj.Get("config")
	}
	return configObj.(*jsonutils.JSONDict), nil
}

func init() {
	Services = modules.NewIdentityManager("OS-KSADM:service",
		"OS-KSADM:services",
		[]string{},
		[]string{"ID", "Name", "Type", "Description"})

	modules.Register(&Services)

	ServicesV3 = SServiceManager{
		ResourceManager: modules.NewIdentityV3Manager("service",
			"services",
			[]string{},
			[]string{"ID", "Name", "Type", "Description"}),
	}
	modules.Register(&ServicesV3)
}
