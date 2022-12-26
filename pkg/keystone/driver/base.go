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

package driver

import (
	"yunion.io/x/pkg/object"

	api "yunion.io/x/onecloud/pkg/apis/identity"
)

var (
	idpDriverClasses = make(map[string]IIdentityBackendClass)
)

func RegisterDriverClass(drv IIdentityBackendClass) {
	idpDriverClasses[drv.Name()] = drv
}

func GetDriverClass(drv string) IIdentityBackendClass {
	if cls, ok := idpDriverClasses[drv]; ok {
		return cls
	}
	return nil
}

func GetDriver(driver string, idpId, idpName, template, targetDomainId string, conf api.TConfigs) (IIdentityBackend, error) {
	drvCls := GetDriverClass(driver)
	if drvCls == nil {
		return nil, ErrNoSuchDriver
	}
	return drvCls.NewDriver(idpId, idpName, template, targetDomainId, conf)
}

type SBaseIdentityDriver struct {
	object.SObject

	Config   api.TConfigs
	IdpId    string
	IdpName  string
	Template string

	TargetDomainId string
	// AutoCreateProject bool
}

func (base *SBaseIdentityDriver) IIdentityBackend() IIdentityBackend {
	return base.GetVirtualObject().(IIdentityBackend)
}

func (base *SBaseIdentityDriver) GetSsoCallbackUri(callbackUrl string) string {
	return callbackUrl
}

func NewBaseIdentityDriver(idpId, idpName, template, targetDomainId string, conf api.TConfigs) (SBaseIdentityDriver, error) {
	drv := SBaseIdentityDriver{}
	drv.IdpId = idpId
	drv.IdpName = idpName
	drv.Template = template
	drv.TargetDomainId = targetDomainId
	// drv.AutoCreateProject = autoCreateProject
	drv.Config = conf
	return drv, nil
}
