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

	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SPrivateendpointconnection struct {
	ID         string `json:"id"`
	Properties struct {
		Provisioningstate string `json:"provisioningState"`
		Privateendpoint   struct {
			ID string `json:"id"`
		} `json:"privateEndpoint"`
		Privatelinkserviceconnectionstate struct {
			Status          string `json:"status"`
			Description     string `json:"description"`
			Actionsrequired string `json:"actionsRequired"`
		} `json:"privateLinkServiceConnectionState"`
	} `json:"properties"`
}

type SSQLServer struct {
	region *SRegion
	multicloud.SDBInstanceBase
	multicloud.AzureTags

	dbs []SSQLServerDatabase

	Kind       string `json:"kind"`
	Properties struct {
		Administratorlogin         string                       `json:"administratorLogin"`
		Version                    string                       `json:"version"`
		State                      string                       `json:"state"`
		Fullyqualifieddomainname   string                       `json:"fullyQualifiedDomainName"`
		Privateendpointconnections []SPrivateendpointconnection `json:"privateEndpointConnections"`
		Publicnetworkaccess        string                       `json:"publicNetworkAccess"`
	} `json:"properties"`
	Location string `json:"location"`
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
}

func (self *SRegion) ListSQLServer() ([]SSQLServer, error) {
	rds := []SSQLServer{}
	err := self.list("Microsoft.Sql/servers", url.Values{}, &rds)
	if err != nil {
		return nil, errors.Wrapf(err, "list")
	}
	return rds, nil
}

func (self *SRegion) GetSQLServer(id string) (*SSQLServer, error) {
	rds := &SSQLServer{region: self}
	return rds, self.get(id, url.Values{}, rds)
}

func (self *SSQLServer) GetDiskSizeGB() int {
	dbs, err := self.fetchDatabase()
	if err != nil {
		return 0
	}
	sizeMb := 0
	for _, db := range dbs {
		sizeMb += db.GetDiskSizeMb()
	}
	return sizeMb / 1024
}

func (self *SSQLServer) GetEngine() string {
	return api.DBINSTANCE_TYPE_SQLSERVER
}

func (self *SSQLServer) GetEngineVersion() string {
	return self.Properties.Version
}

func (self *SSQLServer) GetGlobalId() string {
	return strings.ToLower(self.ID)
}

func (self *SSQLServer) GetProjectId() string {
	return getResourceGroup(self.ID)
}

func (self *SSQLServer) GetCategory() string {
	return api.AZURE_DBINSTANCE_CATEGORY_BASIC
}

func (self *SSQLServer) GetIVpcId() string {
	return ""
}

func (self *SSQLServer) GetId() string {
	return self.ID
}

func (self *SSQLServer) GetInstanceType() string {
	return ""
}

func (self *SSQLServer) GetMaintainTime() string {
	return ""
}

func (self *SSQLServer) GetName() string {
	return self.Name
}

func (self *SSQLServer) GetPort() int {
	return 1433
}

func (self *SSQLServer) GetStatus() string {
	switch self.Properties.State {
	case "Ready":
		return api.DBINSTANCE_RUNNING
	default:
		return self.Properties.State
	}
}

func (self *SSQLServer) GetStorageType() string {
	return api.AZURE_DBINSTANCE_STORAGE_TYPE_DEFAULT
}

func (self *SSQLServer) fetchDatabase() ([]SSQLServerDatabase, error) {
	if len(self.dbs) > 0 {
		return self.dbs, nil
	}
	var err error
	self.dbs, err = self.region.GetSQLServerDatabases(self.ID)
	return self.dbs, err
}

func (self *SSQLServer) GetVcpuCount() int {
	dbs, err := self.fetchDatabase()
	if err != nil {
		return 0
	}
	vcpu := 0
	for _, db := range dbs {
		vcpu += db.GetVcpuCount()
	}
	return vcpu
}

func (self *SSQLServer) GetVmemSizeMB() int {
	dbs, err := self.fetchDatabase()
	if err != nil {
		return 0
	}
	mem := 0
	for _, db := range dbs {
		mem += db.GetVmemSizeMb()
	}
	return mem
}

func (self *SSQLServer) GetSysTags() map[string]string {
	dtu := self.GetDTU()
	if dtu > 0 {
		return map[string]string{"DTU": fmt.Sprintf("%d", dtu)}
	}
	return nil
}

func (self *SSQLServer) GetDTU() int {
	dbs, err := self.fetchDatabase()
	if err != nil {
		return 0
	}
	dtu := 0
	for _, db := range dbs {
		dtu += db.GetDTU()
	}
	return dtu
}

func (self *SSQLServer) GetZone1Id() string {
	return ""
}

func (self *SSQLServer) GetZone2Id() string {
	return ""
}

func (self *SSQLServer) GetZone3Id() string {
	return ""
}

func (self *SSQLServer) GetConnectionStr() string {
	return self.Properties.Fullyqualifieddomainname
}
