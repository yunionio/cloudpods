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
	"fmt"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SKubeCluster struct {
	multicloud.SResourceBase
	AwsTags
	region *SRegion

	Name               string
	Arn                string  `json:"arn"`
	CreatedAt          float64 `json:"createdAt"`
	Version            string  `json:"version"`
	Endpoint           string  `json:"endpoint"`
	RoleArn            string  `json:"roleArn"`
	ResourcesVpcConfig struct {
		SubnetIds              []string `json:"subnetIds"`
		SecurityGroupIds       []string `json:"securityGroupIds"`
		ClusterSecurityGroupId string   `json:"clusterSecurityGroupId"`
		VpcId                  string   `json:"vpcId"`
		EndpointPublicAccess   bool     `json:"endpointPublicAccess"`
		EndpointPrivateAccess  bool     `json:"endpointPrivateAccess"`
		PublicAccessCidrs      []string `json:"publicAccessCidrs"`
	} `json:"resourcesVpcConfig"`
	KubernetesNetworkConfig struct {
		ServiceIpv4CIDR string `json:"serviceIpv4Cidr"`
		ServiceIpv6CIDR string `json:"serviceIpv6Cidr"`
		IPFamily        string `json:"ipFamily"`
	} `json:"kubernetesNetworkConfig"`
	Logging struct {
		ClusterLogging []ClusterLogging `json:"clusterLogging"`
	} `json:"logging"`
	Identity             string `json:"identity"`
	Status               string `json:"status"`
	CertificateAuthority struct {
		Data string
	} `json:"certificateAuthority"`
	ClientRequestToken string `json:"clientRequestToken"`
	PlatformVersion    string `json:"platformVersion"`
	EncryptionConfig   string `json:"encryptionConfig"`
	ConnectorConfig    string `json:"connectorConfig"`
	Id                 string `json:"id"`
	Health             string `json:"health"`
}

type ClusterLogging struct {
	Types   []string `json:"types"`
	Enabled bool     `json:"enabled"`
}

func (self *SKubeCluster) GetName() string {
	return self.Name
}

func (self *SKubeCluster) GetId() string {
	return self.Name
}

func (self *SKubeCluster) GetGlobalId() string {
	return self.GetId()
}

func (self *SKubeCluster) GetEnabled() bool {
	return true
}

func (self *SKubeCluster) GetStatus() string {
	if len(self.Status) == 0 {
		self.Refresh()
	}
	switch self.Status {
	case "ACTIVE":
		return api.KUBE_CLUSTER_STATUS_RUNNING
	case "DELETING":
		return api.KUBE_CLUSTER_STATUS_DELETING
	default:
		return strings.ToLower(self.Status)
	}
}

func (self *SKubeCluster) GetKubeConfig(private bool, expireMinutes int) (*cloudprovider.SKubeconfig, error) {
	if len(self.CertificateAuthority.Data) == 0 {
		self.Refresh()
	}
	eksId := fmt.Sprintf("%s:%s:cluster/%s", self.region.RegionId, self.region.client.ownerId, self.Name)
	config := fmt.Sprintf(`apiVersion: v1
clusters:
- cluster:
    server: %s 
    certificate-authority-data: %s 
  name: arn:aws:eks:%s
contexts:
- context:
    cluster: arn:aws:eks:%s
    user: arn:aws:eks:%s
  name: arn:aws:eks:%s
current-context: arn:aws:eks:%s
kind: Config
preferences: {}
users:
- name: arn:aws:eks:%s
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1beta1
      command: aws-iam-authenticator
      args:
        - "token"
        - "-i"
        - "%s"`, self.Endpoint, self.CertificateAuthority.Data, eksId, eksId, eksId, eksId, eksId, eksId, self.Name)
	return &cloudprovider.SKubeconfig{
		Config: config,
	}, nil
}

func (self *SKubeCluster) GetIKubeNodePools() ([]cloudprovider.ICloudKubeNodePool, error) {
	ret := []cloudprovider.ICloudKubeNodePool{}
	nextToken := ""
	for {
		part, nextToken, err := self.region.GetNodegroups(self.Name, nextToken)
		if err != nil {
			return nil, errors.Wrapf(err, "GetNodegroups")
		}
		for i := range part {
			ret = append(ret, &part[i])
		}
		if len(nextToken) == 0 {
			break
		}
	}
	return ret, nil
}

func (self *SKubeCluster) GetIKubeNodes() ([]cloudprovider.ICloudKubeNode, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SKubeCluster) Delete(isRetain bool) error {
	return self.region.DeleteKubeCluster(self.Name)
}

func (self *SKubeCluster) GetCreatedAt() time.Time {
	return time.Unix(int64(self.CreatedAt), 0)
}

func (self *SKubeCluster) GetVpcId() string {
	if len(self.ResourcesVpcConfig.VpcId) == 0 {
		self.Refresh()
	}
	return self.ResourcesVpcConfig.VpcId
}

func (self *SKubeCluster) GetVersion() string {
	if len(self.Version) == 0 {
		self.Refresh()
	}
	return self.Version
}

func (self *SKubeCluster) GetNetworkIds() []string {
	if len(self.ResourcesVpcConfig.SubnetIds) == 0 {
		self.Refresh()
	}
	return self.ResourcesVpcConfig.SubnetIds
}

func (self *SKubeCluster) Refresh() error {
	cluster, err := self.region.GetKubeCluster(self.Name)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, cluster)
}

func (self *SRegion) GetKubeClusters(nextToken string) ([]SKubeCluster, string, error) {
	ret := struct {
		Clusters  []string
		NextToken string
	}{}
	params := map[string]interface{}{
		"include": "all",
	}
	if len(nextToken) > 0 {
		params["nextToken"] = nextToken
	}
	result := []SKubeCluster{}
	err := self.eksRequest("ListClusters", "/clusters", params, &ret)
	if err != nil {
		return nil, "", errors.Wrapf(err, "ListClusters")
	}
	for i := range ret.Clusters {
		result = append(result, SKubeCluster{
			region: self,
			Name:   ret.Clusters[i],
		})
	}
	return result, ret.NextToken, nil
}

func (self *SRegion) GetKubeCluster(name string) (*SKubeCluster, error) {
	params := map[string]interface{}{
		"name": name,
	}
	ret := struct {
		Cluster SKubeCluster
	}{}
	err := self.eksRequest("DescribeCluster", "/clusters/{name}", params, &ret)
	if err != nil {
		return nil, errors.Wrapf(err, "DescribeCluster")
	}
	ret.Cluster.region = self
	return &ret.Cluster, nil
}

func (self *SRegion) DeleteKubeCluster(name string) error {
	params := map[string]interface{}{
		"name": name,
	}
	ret := struct {
	}{}
	return self.eksRequest("DeleteCluster", "/clusters/{name}", params, &ret)
}

func (self *SRegion) GetICloudKubeClusters() ([]cloudprovider.ICloudKubeCluster, error) {
	ret := []cloudprovider.ICloudKubeCluster{}
	nextToken := ""
	for {
		part, nextToken, err := self.GetKubeClusters(nextToken)
		if err != nil {
			return nil, errors.Wrapf(err, "GetKubeClusters")
		}
		for i := range part {
			part[i].region = self
			ret = append(ret, &part[i])
		}
		if len(nextToken) == 0 {
			break
		}
	}
	return ret, nil
}

func (self *SRegion) GetICloudKubeClusterById(id string) (cloudprovider.ICloudKubeCluster, error) {
	cluster, err := self.GetKubeCluster(id)
	if err != nil {
		return nil, err
	}
	return cluster, nil
}

func (self *SRegion) CreateIKubeCluster(opts *cloudprovider.KubeClusterCreateOptions) (cloudprovider.ICloudKubeCluster, error) {
	cluster, err := self.CreateKubeCluster(opts)
	if err != nil {
		return nil, err
	}
	return cluster, nil
}

func (self *SRegion) CreateKubeCluster(opts *cloudprovider.KubeClusterCreateOptions) (*SKubeCluster, error) {
	if !opts.PrivateAccess && !opts.PublicAccess { // avoid occur 'Private and public endpoint access cannot be false' error
		opts.PrivateAccess = true
	}
	params := map[string]interface{}{
		"name":               opts.NAME,
		"clientRequestToken": utils.GenRequestId(20),
		"resourcesVpcConfig": map[string]interface{}{
			"endpointPrivateAccess": opts.PrivateAccess,
			"endpointPublicAccess":  opts.PublicAccess,
			"subnetIds":             opts.NetworkIds,
		},
		"tags": opts.Tags,
	}
	if len(opts.RoleName) == 0 {
		opts.RoleName = "eksClusterRole"
	}
	role, err := func() (*SRole, error) {
		role, err := self.client.GetRole(opts.RoleName)
		if err != nil {
			if errors.Cause(err) == cloudprovider.ErrNotFound {
				params := map[string]string{
					"RoleName":                 opts.RoleName,
					"Description":              opts.Desc,
					"AssumeRolePolicyDocument": k8sRole,
				}
				role := struct {
					Role SRole
				}{}
				err := self.client.iamRequest("CreateRole", params, &role)
				if err != nil {
					return nil, errors.Wrapf(err, "CreateRole")
				}
				role.Role.client = self.client
				return &role.Role, nil
			}
			return nil, errors.Wrapf(err, "GetRole(%s)", opts.RoleName)
		}
		return role, nil
	}()
	if err != nil {
		return nil, err
	}

	err = self.client.AttachRolePolicy(opts.RoleName, self.client.getIamArn("AmazonEKSClusterPolicy"))
	if err != nil {
		return nil, errors.Wrapf(err, "AttachRolePolicy")
	}

	params["roleArn"] = role.Arn
	if len(opts.ServiceCIDR) > 0 {
		params["kubernetesNetworkConfig"] = map[string]interface{}{
			"ipFamily":        "ipv4",
			"serviceIpv4Cidr": opts.ServiceCIDR,
		}
	}
	if len(opts.Version) > 0 {
		params["version"] = opts.Version
	}
	ret := struct {
		Cluster SKubeCluster
	}{}
	err = self.eksRequest("CreateCluster", "/clusters", params, &ret)
	if err != nil {
		return nil, err
	}
	ret.Cluster.region = self
	return &ret.Cluster, nil
}

func (self *SRegion) CreateNodegroup(cluster string, opts *cloudprovider.KubeNodePoolCreateOptions) (*SNodeGroup, error) {
	params := map[string]interface{}{
		"nodegroupName":      opts.NAME,
		"clientRequestToken": utils.GenRequestId(20),
		"diskSize":           opts.RootDiskSizeGb,
		"instanceTypes":      opts.InstanceTypes,
		"tags":               opts.Tags,
		"scalingConfig": map[string]interface{}{
			"desiredSize": opts.DesiredInstanceCount,
			"maxSize":     opts.MaxInstanceCount,
			"minSize":     opts.MinInstanceCount,
		},
		"subnets": opts.NetworkIds,
	}
	if len(opts.PublicKey) > 0 {
		keypairName, err := self.SyncKeypair(opts.PublicKey)
		if err != nil {
			return nil, errors.Wrapf(err, "syncKeypair")
		}
		params["remoteAccess"] = map[string]string{
			"ec2SshKey": keypairName,
		}
	}
	roleName := "AmazonEKSNodeRole"
	role, err := func() (*SRole, error) {
		role, err := self.client.GetRole(roleName)
		if err != nil {
			if errors.Cause(err) == cloudprovider.ErrNotFound {
				params := map[string]string{
					"RoleName":                 roleName,
					"Description":              opts.Desc,
					"AssumeRolePolicyDocument": nodeRole,
				}
				role := struct {
					Role SRole
				}{}
				err := self.client.iamRequest("CreateRole", params, &role)
				if err != nil {
					return nil, errors.Wrapf(err, "CreateRole")
				}
				role.Role.client = self.client
				return &role.Role, nil
			}
			return nil, errors.Wrapf(err, "GetRole(%s)", roleName)
		}
		return role, nil
	}()
	if err != nil {
		return nil, errors.Wrapf(err, "Create role")
	}

	for _, policy := range []string{"AmazonEKSWorkerNodePolicy", "AmazonEC2ContainerRegistryReadOnly"} {
		err = self.client.AttachRolePolicy(roleName, self.client.getIamArn(policy))
		if err != nil {
			return nil, errors.Wrapf(err, "AttachRolePolicy %s", policy)
		}
	}

	params["nodeRole"] = role.Arn

	ret := struct {
		Nodegroup SNodeGroup
	}{}
	err = self.eksRequest("CreateNodegroup", fmt.Sprintf("/clusters/%s/node-groups", cluster), params, &ret)
	if err != nil {
		return nil, err
	}
	ret.Nodegroup.region = self
	return &ret.Nodegroup, nil
}

func (self *SKubeCluster) CreateIKubeNodePool(opts *cloudprovider.KubeNodePoolCreateOptions) (cloudprovider.ICloudKubeNodePool, error) {
	nodegroup, err := self.region.CreateNodegroup(self.Name, opts)
	if err != nil {
		return nil, errors.Wrapf(err, "CreateNodegroup")
	}
	return nodegroup, nil
}

func (self *SKubeCluster) GetDescription() string {
	return ""
}
