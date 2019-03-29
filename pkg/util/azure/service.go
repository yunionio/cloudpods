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
	ResourceType string   `json:"locations,omitempty"`
}

func (self *SRegion) ListServices() ([]SService, error) {
	services := []SService{}
	return services, self.client.List("providers", &services)
}

func (self *SRegion) SerciceShow(serviceType string) (*SService, error) {
	service := SService{}
	return &service, self.client.Get("providers/"+serviceType, []string{}, &service)
}

func (self *SRegion) serviceOperation(resourceType, operation string) error {
	services, err := self.ListServices()
	if err != nil {
		return err
	}
	for _, service := range services {
		if service.Namespace == resourceType {
			_, err := self.client.jsonRequest("POST", fmt.Sprintf("%s/%s", service.ID, operation), "")
			return err
		}
	}
	return fmt.Errorf("failed to find namespace: %s", resourceType)
}

func (self *SRegion) ServiceRegister(resourceType string) error {
	return self.serviceOperation(resourceType, "register")
}

func (self *SRegion) ServiceUnRegister(resourceType string) error {
	return self.serviceOperation(resourceType, "unregister")
}
