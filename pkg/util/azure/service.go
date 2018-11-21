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
