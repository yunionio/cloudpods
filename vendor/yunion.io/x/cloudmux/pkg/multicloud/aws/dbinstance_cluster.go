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

package aws

import (
	"context"
	"time"

	billing "yunion.io/x/cloudmux/pkg/apis/billing"
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/pkg/errors"
)

type SDBInstanceCluster struct {
	multicloud.SDBInstanceBase
	AwsTags

	region *SRegion

	CrossAccountClone                      bool                          `xml:"CrossAccountClone"`
	AllocatedStorage                       int                           `xml:"AllocatedStorage"`
	IOOptimizedNextAllowedModificationTime time.Time                     `xml:"IOOptimizedNextAllowedModificationTime"`
	AssociatedRoles                        []string                      `xml:"AssociatedRoles"`
	DBActivityStreamStatus                 string                        `xml:"DBActivityStreamStatus"`
	HttpEndpointEnabled                    bool                          `xml:"HttpEndpointEnabled"`
	DBClusterIdentifier                    string                        `xml:"DBClusterIdentifier"`
	Status                                 string                        `xml:"Status"`
	PreferredBackupWindow                  string                        `xml:"PreferredBackupWindow"`
	DeletionProtection                     bool                          `xml:"DeletionProtection"`
	Endpoint                               string                        `xml:"Endpoint"`
	ReaderEndpoint                         string                        `xml:"ReaderEndpoint"`
	EarliestRestorableTime                 time.Time                     `xml:"EarliestRestorableTime"`
	ClusterCreateTime                      time.Time                     `xml:"ClusterCreateTime"`
	ActivityStreamStatus                   string                        `xml:"ActivityStreamStatus"`
	DBSubnetGroup                          string                        `xml:"DBSubnetGroup"`
	VpcSecurityGroups                      []SVpcSecurityGroupMembership `xml:"VpcSecurityGroups>VpcSecurityGroupMembership"`
	PreferredMaintenanceWindow             string                        `xml:"PreferredMaintenanceWindow"`
	DBClusterParameterGroup                string                        `xml:"DBClusterParameterGroup"`
	AutoMinorVersionUpgrade                bool                          `xml:"AutoMinorVersionUpgrade"`
	DBClusterArn                           string                        `xml:"DBClusterArn"`
	AvailabilityZones                      []string                      `xml:"AvailabilityZones>AvailabilityZone"`
	ReadReplicaIdentifiers                 []string                      `xml:"ReadReplicaIdentifiers"`
	DocdbAnalyticsEnabled                  bool                          `xml:"DocdbAnalyticsEnabled"`
	EngineVersion                          string                        `xml:"EngineVersion"`
	MasterUsername                         string                        `xml:"MasterUsername"`
	DBClusterMembers                       []struct {
		DBInstanceIdentifier          string `xml:"DBInstanceIdentifier"`
		DBClusterParameterGroupStatus string `xml:"DBClusterParameterGroupStatus"`
		PromotionTier                 int    `xml:"PromotionTier"`
		IsClusterWriter               bool   `xml:"IsClusterWriter"`
	} `xml:"DBClusterMembers>DBClusterMember"`
	Port                             int       `xml:"Port"`
	BackupRetentionPeriod            int       `xml:"BackupRetentionPeriod"`
	KmsKeyId                         string    `xml:"KmsKeyId"`
	DbClusterResourceId              string    `xml:"DbClusterResourceId"`
	LatestRestorableTime             time.Time `xml:"LatestRestorableTime"`
	EngineMode                       string    `xml:"EngineMode"`
	Engine                           string    `xml:"Engine"`
	IAMDatabaseAuthenticationEnabled bool      `xml:"IAMDatabaseAuthenticationEnabled"`
	NetworkType                      string    `xml:"NetworkType"`
	MultiAZ                          bool      `xml:"MultiAZ"`
	DomainMemberships                []string  `xml:"DomainMemberships"`
	StorageEncrypted                 bool      `xml:"StorageEncrypted"`
	HostedZoneId                     string    `xml:"HostedZoneId"`
	StorageType                      string    `xml:"StorageType"`
	CopyTagsToSnapshot               bool      `xml:"CopyTagsToSnapshot"`
}

func (rds *SDBInstanceCluster) GetName() string {
	return rds.DBClusterIdentifier
}

func (rds *SDBInstanceCluster) GetId() string {
	return rds.DBClusterIdentifier
}

func (rds *SDBInstanceCluster) GetGlobalId() string {
	return rds.GetId()
}

func (rds *SDBInstanceCluster) GetStatus() string {
	switch rds.Status {
	case "creating", "backing-up":
		return api.DBINSTANCE_DEPLOYING
	case "available":
		return api.DBINSTANCE_RUNNING
	case "deleting":
		return api.DBINSTANCE_DELETING
	case "rebooting":
		return api.DBINSTANCE_REBOOTING
	default:
		return api.DBINSTANCE_UNKNOWN
	}
}

func (rds *SDBInstanceCluster) GetBillingType() string {
	return billing.BILLING_TYPE_POSTPAID
}

func (rds *SDBInstanceCluster) Reboot() error {
	return cloudprovider.ErrNotImplemented
}

func (rds *SDBInstanceCluster) GetSecurityGroupIds() ([]string, error) {
	ids := []string{}
	for _, v := range rds.VpcSecurityGroups {
		ids = append(ids, v.VpcSecurityGroupId)
	}
	return ids, nil
}

func (rds *SDBInstanceCluster) SetSecurityGroups(ids []string) error {
	return cloudprovider.ErrNotSupported
}

func (rds *SDBInstanceCluster) GetPort() int {
	return rds.Port
}

func (rds *SDBInstanceCluster) GetEngine() string {
	return rds.Engine
}

func (rds *SDBInstanceCluster) GetEngineVersion() string {
	return rds.EngineVersion
}

func (rds *SDBInstanceCluster) GetInstanceType() string {
	return ""
}

func (rds *SDBInstanceCluster) GetVcpuCount() int {
	return 0
}

func (rds *SDBInstanceCluster) GetDiskSizeGB() int {
	return 0
}

func (rds *SDBInstanceCluster) GetDiskSizeUsedMB() int {
	return 0
}

func (rds *SDBInstanceCluster) GetCategory() string {
	return api.AWS_DBINSTANCE_CATEGORY_CLUSTER
}

func (rds *SDBInstanceCluster) GetStorageType() string {
	return "standard"
}

func (rds *SDBInstanceCluster) GetVmemSizeMB() int {
	return 0
}

func (rds *SDBInstanceCluster) GetMaintainTime() string {
	return ""
}

func (rds *SDBInstanceCluster) GetConnectionStr() string {
	return ""
}

func (rds *SDBInstanceCluster) GetInternalConnectionStr() string {
	return ""
}

func (rds *SDBInstanceCluster) GetZone1Id() string {
	for i, zone := range rds.AvailabilityZones {
		if i == 0 {
			return zone
		}
	}
	return ""
}

func (rds *SDBInstanceCluster) GetZone2Id() string {
	for i, zone := range rds.AvailabilityZones {
		if i == 1 {
			return zone
		}
	}
	return ""
}

func (rds *SDBInstanceCluster) GetZone3Id() string {
	for i, zone := range rds.AvailabilityZones {
		if i == 2 {
			return zone
		}
	}
	return ""
}

func (rds *SDBInstanceCluster) GetIVpcId() string {
	return ""
}

func (rds *SDBInstanceCluster) GetIops() int {
	return 0
}

func (rds *SDBInstanceCluster) GetDBNetworks() ([]cloudprovider.SDBInstanceNetwork, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (rds *SDBInstanceCluster) GetIDBInstanceParameters() ([]cloudprovider.ICloudDBInstanceParameter, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (rds *SDBInstanceCluster) GetIDBInstanceDatabases() ([]cloudprovider.ICloudDBInstanceDatabase, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (rds *SDBInstanceCluster) GetIDBInstanceAccounts() ([]cloudprovider.ICloudDBInstanceAccount, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (rds *SDBInstanceCluster) GetIDBInstanceBackups() ([]cloudprovider.ICloudDBInstanceBackup, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (rds *SDBInstanceCluster) ChangeConfig(ctx context.Context, config *cloudprovider.SManagedDBInstanceChangeConfig) error {
	return cloudprovider.ErrNotSupported
}

func (rds *SDBInstanceCluster) OpenPublicConnection() error {
	return cloudprovider.ErrNotSupported
}

func (rds *SDBInstanceCluster) ClosePublicConnection() error {
	return cloudprovider.ErrNotSupported
}

func (rds *SDBInstanceCluster) CreateDatabase(conf *cloudprovider.SDBInstanceDatabaseCreateConfig) error {
	return cloudprovider.ErrNotSupported
}

func (rds *SDBInstanceCluster) CreateAccount(conf *cloudprovider.SDBInstanceAccountCreateConfig) error {
	return cloudprovider.ErrNotSupported
}

func (rds *SDBInstanceCluster) CreateIBackup(conf *cloudprovider.SDBInstanceBackupCreateConfig) (string, error) {
	return "", cloudprovider.ErrNotSupported
}

func (rds *SDBInstanceCluster) RecoveryFromBackup(conf *cloudprovider.SDBInstanceRecoveryConfig) error {
	return cloudprovider.ErrNotSupported
}

func (rds *SDBInstanceCluster) Update(ctx context.Context, input cloudprovider.SDBInstanceUpdateOptions) error {
	return cloudprovider.ErrNotSupported
}

func (rds *SDBInstanceCluster) Delete() error {
	return cloudprovider.ErrNotSupported
}

func (region *SRegion) GetDBInstanceClusters(id string) ([]SDBInstanceCluster, error) {
	ret := []SDBInstanceCluster{}
	params := map[string]string{}
	if len(id) > 0 {
		params["DBClusterIdentifier"] = id
	}
	for {
		part := struct {
			DBClusters []SDBInstanceCluster `xml:"DBClusters>DBCluster"`
			Marker     string               `xml:"Marker"`
		}{}
		err := region.rdsRequest("DescribeDBClusters", params, &part)
		if err != nil {
			return nil, errors.Wrap(err, "DescribeDBClusters")
		}
		ret = append(ret, part.DBClusters...)
		if len(part.DBClusters) == 0 || len(part.Marker) == 0 {
			break
		}
		params["Marker"] = part.Marker
	}

	return ret, nil
}

func (region *SRegion) GetDBInstanceCluster(id string) (*SDBInstanceCluster, error) {
	clusters, err := region.GetDBInstanceClusters(id)
	if err != nil {
		return nil, err
	}
	for i := range clusters {
		if clusters[i].DBClusterIdentifier == id {
			clusters[i].region = region
			return &clusters[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
}
