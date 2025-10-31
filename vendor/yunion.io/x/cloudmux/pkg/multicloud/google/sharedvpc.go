package google

import (
	"fmt"
	"strings"
	"time"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/pkg/errors"
)

type SSharedGlobalNetwork struct {
	GoogleTags

	client *SGoogleClient

	Network     string
	Subnetworks []SXpnNetwork
}

func (network *SSharedGlobalNetwork) GetId() string {
	return getGlobalId(network.Network)
}

func (network *SSharedGlobalNetwork) GetGlobalId() string {
	return network.GetId()
}

func (network *SSharedGlobalNetwork) GetName() string {
	info := strings.Split(network.GetId(), "/")
	if len(info) == 6 {
		return fmt.Sprintf("%s(%s)", info[5], info[1])
	}
	return network.GetId()
}

func (network *SSharedGlobalNetwork) GetStatus() string {
	return api.GLOBAL_VPC_STATUS_AVAILABLE
}

func (network *SSharedGlobalNetwork) IsEmulated() bool {
	return true
}

func (network *SSharedGlobalNetwork) Refresh() error {
	return nil
}

func (network *SSharedGlobalNetwork) GetCreatedAt() time.Time {
	return time.Time{}
}

func (network *SSharedGlobalNetwork) GetDescription() string {
	return ""
}

func (network *SSharedGlobalNetwork) GetISecurityGroups() ([]cloudprovider.ICloudSecurityGroup, error) {
	return []cloudprovider.ICloudSecurityGroup{}, nil
}

func (network *SSharedGlobalNetwork) CreateISecurityGroup(opts *cloudprovider.SecurityGroupCreateInput) (cloudprovider.ICloudSecurityGroup, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (network *SSharedGlobalNetwork) Delete() error {
	return cloudprovider.ErrNotSupported
}

func (client *SGoogleClient) GetSharedGlobalNetworks() ([]SSharedGlobalNetwork, error) {
	xhosts, err := client.GetXpnHosts()
	if err != nil {
		if e, ok := err.(*gError); ok && e.ErrorInfo.Code == 400 {
			return []SSharedGlobalNetwork{}, nil
		}
		return nil, errors.Wrapf(err, "GetXpnHosts")
	}
	ret := []SSharedGlobalNetwork{}
	networkMap := map[string][]SXpnNetwork{}
	for _, xhost := range xhosts {
		resources, err := client.GetXpnResources(xhost.Name)
		if err != nil {
			return nil, errors.Wrapf(err, "GetXpnResources(%s)", xhost.Name)
		}
		for _, resource := range resources {
			if strings.EqualFold(resource.Type, "project") && resource.Id == client.projectId {
				networks, err := client.GetXpnNetworks(xhost.Name)
				if err != nil {
					return nil, errors.Wrapf(err, "GetXpnNetworks(%s)", xhost.Name)
				}
				for i := range networks {
					_, ok := networkMap[networks[i].Network]
					if !ok {
						networkMap[networks[i].Network] = []SXpnNetwork{}
					}
					networkMap[networks[i].Network] = append(networkMap[networks[i].Network], networks[i])
				}
			}
		}
	}
	for network, subnetworks := range networkMap {
		ret = append(ret, SSharedGlobalNetwork{
			client:      client,
			Network:     network,
			Subnetworks: subnetworks,
		})
	}
	return ret, nil
}
