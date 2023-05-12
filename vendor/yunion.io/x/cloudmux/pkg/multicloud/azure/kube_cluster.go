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
	"encoding/base64"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SKubeCluster struct {
	multicloud.SResourceBase
	AzureTags
	region *SRegion

	Id         string `json:"id"`
	Location   string `json:"location"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	Properties struct {
		ProvisioningState string `json:"provisioningState"`
		PowerState        struct {
			Code string `json:"code"`
		} `json:"powerState"`
		KubernetesVersion       string          `json:"kubernetesVersion"`
		DNSPrefix               string          `json:"dnsPrefix"`
		Fqdn                    string          `json:"fqdn"`
		AzurePortalFQDN         string          `json:"azurePortalFQDN"`
		AgentPoolProfiles       []SKubeNodePool `json:"agentPoolProfiles"`
		ServicePrincipalProfile struct {
			ClientId string `json:"clientId"`
		} `json:"servicePrincipalProfile"`
		AddonProfiles struct {
			Azurepolicy struct {
				Enabled bool   `json:"enabled"`
				Config  string `json:"config"`
			} `json:"azurepolicy"`
			HTTPApplicationRouting struct {
				Enabled bool `json:"enabled"`
				Config  struct {
					HTTPApplicationRoutingZoneName string `json:"HTTPApplicationRoutingZoneName"`
				} `json:"config"`
			} `json:"httpApplicationRouting"`
			OmsAgent struct {
				Enabled bool `json:"enabled"`
				Config  struct {
					LogAnalyticsWorkspaceResourceId string `json:"logAnalyticsWorkspaceResourceId"`
				} `json:"config"`
				Identity struct {
					ResourceId string `json:"resourceId"`
					ClientId   string `json:"clientId"`
					ObjectId   string `json:"objectId"`
				} `json:"identity"`
			} `json:"omsAgent"`
		} `json:"addonProfiles"`
		NodeResourceGroup string `json:"nodeResourceGroup"`
		EnableRBAC        bool   `json:"enableRBAC"`
		NetworkProfile    struct {
			NetworkPlugin       string `json:"networkPlugin"`
			LoadBalancerSku     string `json:"loadBalancerSku"`
			LoadBalancerProfile struct {
				ManagedOutboundIPs struct {
					Count int `json:"count"`
				} `json:"managedOutboundIPs"`
				EffectiveOutboundIPs []struct {
					Id string `json:"id"`
				} `json:"effectiveOutboundIPs"`
			} `json:"loadBalancerProfile"`
			PodCidr          string `json:"podCidr"`
			ServiceCidr      string `json:"serviceCidr"`
			DNSServiceIP     string `json:"dnsServiceIP"`
			DockerBridgeCidr string `json:"dockerBridgeCidr"`
			OutboundType     string `json:"outboundType"`
		} `json:"networkProfile"`
		MaxAgentPools          int `json:"maxAgentPools"`
		APIServerAccessProfile struct {
			EnablePrivateCluster bool `json:"enablePrivateCluster"`
		} `json:"apiServerAccessProfile"`
		IdentityProfile struct {
			Kubeletidentity struct {
				ResourceId string `json:"resourceId"`
				ClientId   string `json:"clientId"`
				ObjectId   string `json:"objectId"`
			} `json:"kubeletidentity"`
		} `json:"identityProfile"`
	} `json:"properties"`
	Identity struct {
		Type        string `json:"type"`
		PrincipalId string `json:"principalId"`
		TenantId    string `json:"tenantId"`
	} `json:"identity"`
	Sku struct {
		Name string `json:"name"`
		Tier string `json:"tier"`
	} `json:"sku"`
}

func (self *SKubeCluster) GetName() string {
	return self.Name
}

func (self *SKubeCluster) GetId() string {
	return self.Id
}

func (self *SKubeCluster) GetGlobalId() string {
	return strings.ToLower(self.Id)
}

func (self *SKubeCluster) Refresh() error {
	cluster, err := self.region.GetKubeCluster(self.Id)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, cluster)
}

func (self *SKubeCluster) GetStatus() string {
	return strings.ToLower(self.Properties.PowerState.Code)
}

func (self *SKubeCluster) Delete(isRetain bool) error {
	return self.region.Delete(self.Id)
}

func (self *SKubeCluster) GetEnabled() bool {
	return true
}

func (self *SKubeCluster) GetIKubeNodes() ([]cloudprovider.ICloudKubeNode, error) {
	return []cloudprovider.ICloudKubeNode{}, nil
}

func (self *SKubeCluster) GetIKubeNodePools() ([]cloudprovider.ICloudKubeNodePool, error) {
	ret := []cloudprovider.ICloudKubeNodePool{}
	for i := range self.Properties.AgentPoolProfiles {
		self.Properties.AgentPoolProfiles[i].cluster = self
		ret = append(ret, &self.Properties.AgentPoolProfiles[i])
	}
	return ret, nil
}

func (self *SKubeCluster) GetKubeConfig(private bool, expireMinute int) (*cloudprovider.SKubeconfig, error) {
	return self.region.GetKubeConfig(self.Id)
}

func (self *SKubeCluster) GetVersion() string {
	return self.Properties.KubernetesVersion
}

func (self *SKubeCluster) GetVpcId() string {
	return ""
}

func (self *SKubeCluster) GetNetworkIds() []string {
	return []string{}
}

func (self *SRegion) GetICloudKubeClusters() ([]cloudprovider.ICloudKubeCluster, error) {
	clusters, err := self.GetKubeClusters()
	if err != nil {
		return nil, errors.Wrapf(err, "GetKubClusters")
	}
	ret := []cloudprovider.ICloudKubeCluster{}
	for i := range clusters {
		clusters[i].region = self
		ret = append(ret, &clusters[i])
	}
	return ret, nil
}

func (self *SRegion) GetICloudKubeClusterById(id string) (cloudprovider.ICloudKubeCluster, error) {
	cluster, err := self.GetKubeCluster(id)
	if err != nil {
		return nil, errors.Wrapf(err, "GetKubeCluster(%s)", id)
	}
	return cluster, nil
}

func (self *SRegion) GetKubeCluster(id string) (*SKubeCluster, error) {
	ret := &SKubeCluster{region: self}
	return ret, self.get(id, nil, ret)
}

func (self *SRegion) GetKubeClusters() ([]SKubeCluster, error) {
	clusters := []SKubeCluster{}
	return clusters, self.list("Microsoft.ContainerService/managedClusters", nil, &clusters)
}

func (self *SRegion) GetKubeConfig(id string) (*cloudprovider.SKubeconfig, error) {
	resp, err := self.perform(id, "listClusterAdminCredential", nil)
	if err != nil {
		return nil, errors.Wrapf(err, "listClusterAdminCredential")
	}
	ret := struct {
		Kubeconfigs []struct {
			Name  string
			Value string
		}
	}{}
	err = resp.Unmarshal(&ret)
	if err != nil {
		return nil, errors.Wrapf(err, "resp.Unmarshal")
	}
	if len(ret.Kubeconfigs) == 0 {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "empty kubeconfig")
	}
	result := &cloudprovider.SKubeconfig{}
	config, err := base64.StdEncoding.DecodeString(ret.Kubeconfigs[0].Value)
	if err != nil {
		return nil, errors.Wrapf(err, "base64.decode")
	}
	result.Config = string(config)
	return result, err
}

func (self *SKubeCluster) CreateIKubeNodePool(opts *cloudprovider.KubeNodePoolCreateOptions) (cloudprovider.ICloudKubeNodePool, error) {
	return nil, cloudprovider.ErrNotImplemented
}
