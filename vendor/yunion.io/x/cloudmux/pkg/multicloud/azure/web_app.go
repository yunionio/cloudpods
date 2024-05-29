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
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/cloudmux/pkg/apis"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SAppSite struct {
	multicloud.SResourceBase
	AzureTags
	region *SRegion

	Properties *SAppSiteProperties
	Id         string
	Name       string
	Kind       string
	Location   string
	Type       string
	stack      string
}

type HostNameSslState struct {
	Name     string
	SslState string
	Status   string
}

func (self *HostNameSslState) GetGlobalId() string {
	return self.Name
}

func (self *HostNameSslState) GetName() string {
	return self.Name
}

func (self *HostNameSslState) GetStatus() string {
	return self.Status
}

func (self *HostNameSslState) GetSslState() string {
	return self.SslState
}

type SAppSiteProperties struct {
	DefaultHostName            string
	InboundIpAddress           string
	ServerFarmId               string
	PublicNetworkAccess        string
	VirtualNetworkSubnetId     string
	HostNames                  []string
	HostNameSslStates          []HostNameSslState
	PrivateEndpointConnections []struct {
		Properties struct {
			IpAddresses []string
		}
	}
}

func (r *SRegion) GetAppSites() ([]SAppSite, error) {
	resp, err := r.list_resources("Microsoft.Web/sites", "2019-08-01", nil)
	if err != nil {
		return nil, err
	}
	result := []SAppSite{}
	err = resp.Unmarshal(&result, "value")
	if err != nil {
		return nil, err
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
		ass[i].region = r
		apps = append(apps, &ass[i])
	}
	return apps, nil
}

func (r *SRegion) GetICloudAppById(id string) (cloudprovider.ICloudApp, error) {
	as, err := r.GetAppSite(id)
	if err != nil {
		return nil, err
	}
	return as, nil
}

func (r *SRegion) GetAppSite(siteId string) (*SAppSite, error) {
	resp, err := r.show(siteId, "2023-12-01")
	if err != nil {
		return nil, err
	}
	as := &SAppSite{region: r}
	err = resp.Unmarshal(as)
	if err != nil {
		return nil, err
	}
	return as, nil
}

func (as *SAppSite) GetStack() (string, error) {
	if len(as.stack) > 0 {
		return as.stack, nil
	}
	res := fmt.Sprintf("%s/config/metadata/list", as.Id)
	resp, err := as.region.post_v2(res, "2023-12-01", nil)
	if err != nil {
		return "", nil
	}
	ret := struct {
		Properties struct {
			CurrentStack string
		}
	}{}
	err = resp.Unmarshal(&ret)
	if err != nil {
		return "", nil
	}
	as.stack = ret.Properties.CurrentStack
	return ret.Properties.CurrentStack, nil
}

func (as *SAppSite) GetId() string {
	return as.Id
}

func (as *SAppSite) GetProperties() (*SAppSiteProperties, error) {
	if as.Properties != nil {
		return as.Properties, nil
	}
	app, err := as.region.GetAppSite(as.Id)
	if err != nil {
		return nil, err
	}
	as.Properties = app.Properties
	return as.Properties, nil
}

func (as *SAppSite) GetIpAddress() string {
	ret := []string{}
	properties, err := as.GetProperties()
	if err != nil {
		return ""
	}
	ret = append(ret, properties.InboundIpAddress)
	for _, conn := range properties.PrivateEndpointConnections {
		ret = append(ret, conn.Properties.IpAddresses...)
	}
	return strings.Join(ret, ",")
}

func (as *SAppSite) GetHostname() string {
	properties, err := as.GetProperties()
	if err != nil {
		return ""
	}
	return properties.DefaultHostName
}

func (as *SAppSite) GetPublicNetworkAccess() string {
	properties, err := as.GetProperties()
	if err != nil {
		return ""
	}
	return properties.PublicNetworkAccess
}

func (as *SAppSite) GetServerFarm() string {
	properties, err := as.GetProperties()
	if err != nil {
		return ""
	}
	if len(properties.ServerFarmId) == 0 {
		return ""
	}
	farm, err := as.region.GetServicePlan(properties.ServerFarmId)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%s(%s:%d)", farm.Name, farm.Sku.Name, farm.Sku.Capacity)
}

func (as *SAppSite) GetBackups() ([]cloudprovider.IAppBackup, error) {
	backups, err := as.region.GetAppBackups(as.Id)
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.IAppBackup{}
	for i := range backups {
		backups[i].Type = "manual"
		ret = append(ret, &backups[i])
	}
	snapshots, err := as.region.GetAppSnapshots(as.Id)
	if err != nil {
		return nil, err
	}

	for i := range snapshots {
		snapshots[i].Type = "auto"
		ret = append(ret, &snapshots[i])
	}

	return ret, nil
}

func (as *SAppSite) GetNetworkId() string {
	properties, err := as.GetProperties()
	if err != nil {
		return ""
	}
	return strings.ToLower(properties.VirtualNetworkSubnetId)
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
	return getResourceGroup(as.Id)
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

func (a *SAppSite) GetTechStack() string {
	if strings.Contains(a.Kind, "container") {
		return "Docker container"
	}
	stack, err := a.GetStack()
	if err != nil {
		log.Errorf("unable to GetStack: %v", err)
	}
	if s, ok := techStacks[stack]; ok {
		return s
	}

	res := fmt.Sprintf("%s/config", a.Id)
	resp, err := a.region.list_v2(res, "2023-12-01", nil)
	if err != nil {
		return ""
	}
	value := []struct {
		Properties struct {
			PhpVersion        string
			PythonVersion     string
			NodeVersion       string
			PowerShellVersion string
			JavaVersion       string
		}
	}{}
	resp.Unmarshal(&value, "value")
	for _, v := range value {
		if len(v.Properties.NodeVersion) > 0 {
			return "Node"
		}
		if len(v.Properties.PhpVersion) > 0 {
			return "PHP"
		}
		if len(v.Properties.PowerShellVersion) > 0 {
			return "PowerShell"
		}
		if len(v.Properties.JavaVersion) > 0 {
			return "Java"
		}
	}
	return stack
}

func (a *SAppSite) GetOsType() cloudprovider.TOsType {
	res := fmt.Sprintf("%s/config", a.Id)
	resp, err := a.region.list_v2(res, "2023-12-01", nil)
	if err != nil {
		return cloudprovider.OsTypeLinux
	}
	value := []struct {
		Properties struct {
			LinuxFxVersion   string
			WindowsFxVersion string
		}
	}{}
	resp.Unmarshal(&value, "value")
	for _, v := range value {
		if len(v.Properties.LinuxFxVersion) > 0 {
			return cloudprovider.OsTypeLinux
		}
		if len(v.Properties.WindowsFxVersion) > 0 {
			return cloudprovider.OsTypeWindows
		}
	}
	return cloudprovider.OsTypeWindows
}

func (self *SAppSite) SetTags(tags map[string]string, replace bool) error {
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

func (self *SAppSite) GetDomains() ([]cloudprovider.IAppDomain, error) {
	properties, err := self.GetProperties()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.IAppDomain{}
	for i := range properties.HostNameSslStates {
		domain := properties.HostNameSslStates[i]
		domain.Status = "no_bind"
		if !utils.IsInStringArray(domain.Name, properties.HostNames) {
			continue
		}
		if domain.Name == properties.DefaultHostName || domain.SslState == "SniEnabled" {
			domain.Status = apis.STATUS_AVAILABLE
		}
		ret = append(ret, &domain)
	}
	return ret, nil
}

type SAppServicePlan struct {
	Id   string
	Name string
	Sku  struct {
		Name     string
		Capacity int
	}
}

func (self *SRegion) GetServicePlan(farmId string) (*SAppServicePlan, error) {
	resp, err := self.show(farmId, "2023-12-01")
	if err != nil {
		return nil, err
	}
	ret := &SAppServicePlan{}
	err = resp.Unmarshal(ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}
