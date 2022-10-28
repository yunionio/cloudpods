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

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SAppSite struct {
	multicloud.SResourceBase
	AzureTags
	region         *SRegion
	appServicePlan *SAppServicePlan

	Properties SAppSiteProperties
	Id         string
	Name       string
	Kind       string
	Location   string
	Type       string

	stack *string
}

type SAppSiteProperties struct {
	AvailabilityState   string
	DefaultHostName     string
	Enabled             bool
	EnabledHostNames    []string
	HostNameSslStates   []SHostNameSslState
	HostNames           []string
	HttpsOnly           bool
	OutboundIpAddresses string
	ResourceGroup       string
	ServerFarmId        string
	SiteConfig          SSiteConfig
	State               string
}

type SSiteConfig struct {
	// TODO
}

type SHostNameSslState struct {
	HostType   string
	Name       string
	SslState   string
	Thumbprint string
	ToUpdate   bool
	VirtualIP  string
}

func (r *SRegion) GetAppSites() ([]SAppSite, error) {
	result := []SAppSite{}
	resource := "Microsoft.Web/sites"
	err := r.list(resource, url.Values{"api-version": []string{"2019-08-01"}}, &result)
	if err != nil {
		return nil, err
	}
	for i := range result {
		result[i].region = r
	}
	return result, nil
}

func (r *SRegion) GetICloudApps() ([]cloudprovider.ICloudApp, error) {
	ass, err := r.GetAppSites()
	if err != nil {
		return nil, err
	}
	apps := make([]cloudprovider.ICloudApp, 0, len(ass))
	for i := range ass {
		apps = append(apps, &SApp{
			SAppSite: ass[i],
		})
	}
	return apps, nil
}

func (r *SRegion) GetICloudAppById(id string) (cloudprovider.ICloudApp, error) {
	as, err := r.GetAppSite(id)
	if err != nil {
		return nil, err
	}
	return &SApp{
		SAppSite: *as,
	}, nil
}

func (as *SAppSite) GetSlots() ([]SAppSite, error) {
	result := []SAppSite{}
	resource := fmt.Sprintf("subscriptions/%s/resourceGroups/%s/providers/Microsoft.Web/sites/%s/slots", as.region.client.subscriptionId, as.Properties.ResourceGroup, as.Name)
	err := as.region.list(resource, url.Values{"api-version": []string{"2019-08-01"}}, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (r *SRegion) GetAppSite(siteId string) (*SAppSite, error) {
	as := &SAppSite{region: r}
	params := url.Values{"api-version": []string{"2019-08-01"}}
	return as, r.get(siteId, params, as)
}

func (as *SAppSite) GetAppServicePlan() (*SAppServicePlan, error) {
	if as.appServicePlan != nil {
		return as.appServicePlan, nil
	}
	plan, err := as.region.GetAppServicePlanWithCache(as.Properties.ServerFarmId)
	if err != nil {
		return nil, errors.Wrap(err, "unable to get AppServicePlan")
	}
	as.appServicePlan = plan
	return plan, nil
}

type SDeployment struct {
	ID         string
	Name       string
	Kind       string
	Type       string
	Properties SDeploymentProperties
}

type SDeploymentProperties struct {
	Active      bool
	Author      string
	AuthorEmail string
	Deployer    string
	Details     string
	EndTime     string
	Message     string
	StartTime   string
	Status      string
}

func (as *SAppSite) GetDeployments() ([]SDeployment, error) {
	result := []SDeployment{}
	resource := fmt.Sprintf("subscriptions/%s/resourceGroups/%s/providers/Microsoft.Web/sites/%s/deployments", as.region.client._subscriptionId(), as.Properties.ResourceGroup, as.Name)
	err := as.region.client.list(resource, url.Values{"api-version": []string{"2019-08-01"}}, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

type SStringDictionary struct {
	Id         string
	Kind       string
	Name       string
	Type       string
	Properties map[string]string
}

func (as *SAppSite) GetStack() (string, error) {
	if as.stack != nil {
		return *as.stack, nil
	}
	// result := SStringDictionary{}
	resource := fmt.Sprintf("subscriptions/%s/resourceGroups/%s/providers/Microsoft.Web/sites/%s/config/metadata", as.region.client.subscriptionId, as.Properties.ResourceGroup, as.Name)

	params := url.Values{}
	params.Set("api-version", "2019-08-01")
	path := resource + "/list"
	response, err := as.region.client.jsonRequest("POST", path, nil, params, true)
	if err != nil {
		return "", err
	}
	stack, err := response.GetString("properties", "CURRENT_STACK")
	if err != nil {
		return "", errors.Wrapf(err, "unable to parse response correctly, response: %s", response)
	}
	as.stack = &stack
	return stack, nil
}

func (as *SAppSite) GetId() string {
	return as.Id
}

func (as *SAppSite) GetName() string {
	return as.Name
}

func (as *SAppSite) GetGlobalId() string {
	return strings.ToLower(as.Id)
}

func (as *SAppSite) GetStatus() string {
	return "ready"
}

func (as *SAppSite) GetProjectId() string {
	return strings.ToLower(as.Properties.ResourceGroup)
}

type SApp struct {
	SAppSite
}

type SAppEnvironment struct {
	multicloud.SResourceBase
	AzureTags
	SAppSite
}

func (ae *SAppEnvironment) GetInstanceType() (string, error) {
	asp, err := ae.SAppSite.GetAppServicePlan()
	if err != nil {
		return "", err
	}
	return asp.Sku.Name, nil
}

func (ae *SAppEnvironment) GetInstanceNumber() (int, error) {
	asp, err := ae.SAppSite.GetAppServicePlan()
	if err != nil {
		return 0, err
	}
	return asp.Sku.Capacity, nil
}

func (a *SApp) GetEnvironments() ([]cloudprovider.ICloudAppEnvironment, error) {
	sites, err := a.SAppSite.GetSlots()
	if err != nil {
		return nil, err
	}
	aes := make([]cloudprovider.ICloudAppEnvironment, 0, len(sites)+1)
	aes = append(aes, &SAppEnvironment{
		SAppSite: a.SAppSite,
	})
	for i := range sites {
		sites[i].region = a.region
		aes = append(aes, &SAppEnvironment{
			SAppSite: sites[i],
		})
	}
	return aes, nil
}

var techStacks = map[string]string{
	"dotnet":         ".NET",
	"dotnetcore":     ".NET Core",
	"aspdotnet":      "ASP.NET",
	"node":           "Node",
	"python":         "Python",
	"php":            "PHP",
	"ruby":           "Ruby",
	"java":           "Java",
	"javacontainers": "Jave Containers",
}

func (a *SApp) GetTechStack() string {
	if strings.Contains(a.SAppSite.Kind, "container") {
		return "Docker container"
	}
	stack, err := a.GetStack()
	if err != nil {
		log.Errorf("unable to GetStack: %v", err)
	}
	if s, ok := techStacks[stack]; ok {
		return s
	}
	return stack
}

func (a *SApp) GetType() string {
	return "web"
}

func (a *SApp) GetKind() string {
	return a.Kind
}

func (a *SApp) GetOsType() cloudprovider.TOsType {
	if strings.Contains(a.Kind, "linux") {
		return cloudprovider.OsTypeLinux
	}
	return cloudprovider.OsTypeWindows
}

func (self *SApp) SetTags(tags map[string]string, replace bool) error {
	if !replace {
		for k, v := range self.Tags {
			if _, ok := tags[k]; !ok {
				tags[k] = v
			}
		}
	}
	_, err := self.region.client.SetTags(self.Id, tags)
	if err != nil {
		return errors.Wrapf(err, "SetTags")
	}
	return nil
}
