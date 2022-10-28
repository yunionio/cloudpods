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
	"net/url"
	"strings"

	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SManagedSQLServer struct {
	region *SRegion
	multicloud.SDBInstanceBase
	AzureTags

	Location string `json:"location"`
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Sku      struct {
		Name     string `json:"name"`
		Tier     string `json:"tier"`
		Capacity int    `json:"capacity"`
		Family   string `json:"family"`
	} `json:"sku"`
	Properties struct {
		Fullyqualifieddomainname   string `json:"fullyQualifiedDomainName"`
		Administratorlogin         string `json:"administratorLogin"`
		Subnetid                   string `json:"subnetId"`
		State                      string `json:"state"`
		Provisioningstate          string `json:"provisioningState"`
		Vcores                     int    `json:"vCores"`
		Storagesizeingb            int    `json:"storageSizeInGB"`
		Licensetype                string `json:"licenseType"`
		Collation                  string `json:"collation"`
		Publicdataendpointenabled  bool   `json:"publicDataEndpointEnabled"`
		Proxyoverride              string `json:"proxyOverride"`
		Minimaltlsversion          string `json:"minimalTlsVersion"`
		Dnszone                    string `json:"dnsZone"`
		Maintenanceconfigurationid string `json:"maintenanceConfigurationId"`
		Storageaccounttype         string `json:"storageAccountType"`
	} `json:"properties"`
}

func (self *SRegion) ListManagedSQLServer() ([]SManagedSQLServer, error) {
	rds := []SManagedSQLServer{}
	err := self.list("Microsoft.Sql/managedInstances", url.Values{}, &rds)
	if err != nil {
		return nil, errors.Wrapf(err, "list")
	}
	return rds, nil
}

func (self *SRegion) GetManagedSQLServer(id string) (*SManagedSQLServer, error) {
	rds := &SManagedSQLServer{region: self}
	return rds, self.get(id, url.Values{}, rds)
}

func (self *SManagedSQLServer) GetDiskSizeGB() int {
	return self.Properties.Storagesizeingb
}

func (self *SManagedSQLServer) GetEngine() string {
	return api.DBINSTANCE_TYPE_SQLSERVER
}

func (self *SManagedSQLServer) GetEngineVersion() string {
	return "latest"
}

func (self *SManagedSQLServer) GetGlobalId() string {
	return strings.ToLower(self.ID)
}

func (self *SManagedSQLServer) GetCategory() string {
	return self.Sku.Family
}

func (self *SManagedSQLServer) GetProjectId() string {
	return getResourceGroup(self.ID)
}

func (self *SManagedSQLServer) GetIVpcId() string {
	if len(self.Properties.Subnetid) > 0 {
		info := strings.Split(self.Properties.Subnetid, "/")
		if len(info) > 2 {
			return strings.Join(info[:len(info)-2], "/")
		}
	}
	return ""
}

func (self *SManagedSQLServer) GetId() string {
	return self.ID
}

func (self *SManagedSQLServer) GetInstanceType() string {
	return self.Sku.Name
}

func (self *SManagedSQLServer) GetMaintainTime() string {
	return ""
}

func (self *SManagedSQLServer) GetName() string {
	return self.Name
}

func (self *SManagedSQLServer) GetPort() int {
	return 1433
}

func (self *SManagedSQLServer) GetStatus() string {
	switch self.Properties.State {
	case "Succeeded", "Running", "Ready":
		return api.DBINSTANCE_RUNNING
	case "Creating", "Created":
		return api.DBINSTANCE_DEPLOYING
	case "Deleted", "Deleting":
		return api.DBINSTANCE_BACKUP_DELETING
	case "Failed":
		return api.DBINSTANCE_CREATE_FAILED
	default:
		return strings.ToLower(self.Properties.State)
	}
}

func (self *SManagedSQLServer) GetStorageType() string {
	return self.Properties.Storageaccounttype
}

func (self *SManagedSQLServer) GetVcpuCount() int {
	return self.Properties.Vcores
}

func (self *SManagedSQLServer) GetVmemSizeMB() int {
	switch self.Sku.Family {
	case "Gen5":
		return int(float32(self.Properties.Vcores) * 5.1 * 1024)
	case "Gen4":
		return self.Properties.Vcores * 7 * 1024
	}
	return 0
}

func (self *SManagedSQLServer) GetZone1Id() string {
	return ""
}

func (self *SManagedSQLServer) GetZone2Id() string {
	return ""
}

func (self *SManagedSQLServer) GetZone3Id() string {
	return ""
}

func (self *SManagedSQLServer) GetConnectionStr() string {
	return self.Properties.Fullyqualifieddomainname
}
