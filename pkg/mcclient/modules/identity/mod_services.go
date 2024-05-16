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

	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

type ServiceManagerV3 struct {
	modulebase.ResourceManager
}

func (m *ServiceManagerV3) GetConfig(s *mcclient.ClientSession, serviceType string) (jsonutils.JSONObject, error) {
	serviceType = s.GetServiceName(serviceType)
	resp, err := m.List(s, jsonutils.Marshal(map[string]string{"type": serviceType}))
	if err != nil {
		return nil, errors.Wrap(err, "list")
	}
	if len(resp.Data) == 0 {
		return nil, httperrors.NewResourceNotFoundError2("service", serviceType)
	}
	serviceId, _ := resp.Data[0].GetString("id")
	conf, err := m.GetSpecific(s, serviceId, "config", nil)
	if err != nil {
		return nil, errors.Wrap(err, "GetSpecific")
	}
	return conf, nil
}

var (
	Services   modulebase.ResourceManager
	ServicesV3 ServiceManagerV3
)

func init() {
	Services = modules.NewIdentityManager("OS-KSADM:service",
		"OS-KSADM:services",
		[]string{},
		[]string{"ID", "Name", "Type", "Description"})

	modules.Register(&Services)

	ServicesV3 = ServiceManagerV3{
		ResourceManager: modules.NewIdentityV3Manager("service",
			"services",
			[]string{},
			[]string{"ID", "Name", "Type", "Description"}),
	}

	modules.Register(&ServicesV3)
}
