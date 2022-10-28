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

package azure

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type SServices struct {
	Value []SService `json:"value,omitempty"`
}

type SService struct {
	ID                string         `json:"id,omitempty"`
	Namespace         string         `json:"namespace,omitempty"`
	RegistrationState string         `json:"registrationState,omitempty"`
	ResourceTypes     []ResourceType `json:"resourceTypes,omitempty"`
}

type ResourceType struct {
	ApiVersions  []string `json:"apiVersions,omitempty"`
	Capabilities string   `json:"capabilities,omitempty"`
	Locations    []string `json:"locations,omitempty"`
	ResourceType string   `json:"resourceType,omitempty"`
}

func (self *SAzureClient) ListServices() ([]SService, error) {
	services := []SService{}
	return services, self.list("providers", url.Values{}, &services)
}

func (self *SAzureClient) GetSercice(serviceType string) (*SService, error) {
	service := SService{}
	return &service, self.get("providers/"+serviceType, url.Values{}, &service)
}

func (self *SAzureClient) serviceOperation(serviceType, operation string) error {
	resource := fmt.Sprintf("subscriptions/%s/providers/%s", self.subscriptionId, serviceType)
	_, err := self.perform(resource, operation, nil)
	return err
}

func (self *SAzureClient) waitServiceStatus(serviceType, status string) error {
	return cloudprovider.Wait(time.Second*10, time.Minute*5, func() (bool, error) {
		services, err := self.ListServices()
		if err != nil {
			return false, errors.Wrapf(err, "ListServices")
		}
		for _, service := range services {
			if strings.ToLower(service.Namespace) == strings.ToLower(serviceType) {
				if service.RegistrationState == status {
					return true, nil
				}
				log.Debugf("service %s status: %s expect %s", serviceType, service.RegistrationState, status)
			}
		}
		return false, nil
	})
}

func (self *SAzureClient) ServiceRegister(serviceType string) error {
	err := self.serviceOperation(serviceType, "register")
	if err != nil {
		return errors.Wrapf(err, "serviceOperation(%s)", "register")
	}
	return self.waitServiceStatus(serviceType, "Registered")
}

func (self *SAzureClient) ServiceUnRegister(serviceType string) error {
	err := self.serviceOperation(serviceType, "unregister")
	if err != nil {
		return errors.Wrapf(err, "serviceOperation(%s)", "unregister")
	}
	return self.waitServiceStatus(serviceType, "NotRegistered")
}
