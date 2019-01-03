package huawei

import (
	"fmt"
	"net"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

// 华为云的子网有点特殊。子网在整个region可用。
type SWire struct {
	region *SRegion
	vpc    *SVpc

	inetworks []cloudprovider.ICloudNetwork
}

func (self *SWire) GetId() string {
	return fmt.Sprintf("%s-%s", self.vpc.GetId(), self.region.GetId())
}

func (self *SWire) GetName() string {
	return self.GetId()
}

func (self *SWire) GetGlobalId() string {
	return fmt.Sprintf("%s-%s", self.vpc.GetGlobalId(), self.region.GetGlobalId())
}

func (self *SWire) GetStatus() string {
	return "available"
}

func (self *SWire) Refresh() error {
	return nil
}

func (self *SWire) IsEmulated() bool {
	return true
}

func (self *SWire) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SWire) GetIVpc() cloudprovider.ICloudVpc {
	return self.vpc
}

func (self *SWire) GetIZone() cloudprovider.ICloudZone {
	return nil
}

func (self *SWire) GetINetworks() ([]cloudprovider.ICloudNetwork, error) {
	if self.inetworks == nil {
		err := self.vpc.fetchNetworks()
		if err != nil {
			return nil, err
		}
	}
	return self.inetworks, nil
}

func (self *SWire) GetBandwidth() int {
	return 10000
}

func (self *SWire) GetINetworkById(netid string) (cloudprovider.ICloudNetwork, error) {
	networks, err := self.GetINetworks()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(networks); i += 1 {
		if networks[i].GetGlobalId() == netid {
			return networks[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

/*
华为云子网可用区，类似一个zone标签。即使指定了zone子网在整个region依然是可用。
通过华为web控制台创建子网需要指定可用区。这里是不指定的。
*/
func (self *SWire) CreateINetwork(name string, cidr string, desc string) (cloudprovider.ICloudNetwork, error) {
	networkId, err := self.region.createNetwork(self.vpc.GetId(), name, cidr, desc)
	if err != nil {
		log.Errorf("createNetwork error %s", err)
		return nil, err
	}

	var network *SNetwork
	err = cloudprovider.WaitCreated(5*time.Second, 60*time.Second, func() bool {
		self.inetworks = nil
		network = self.getNetworkById(networkId)
		if network == nil {
			return false
		} else {
			return true
		}
	})

	if err != nil {
		log.Errorf("cannot find network after create????")
		return nil, err
	}

	network.wire = self
	return network, nil
}

func (self *SWire) addNetwork(network *SNetwork) {
	if self.inetworks == nil {
		self.inetworks = make([]cloudprovider.ICloudNetwork, 0)
	}
	find := false
	for i := 0; i < len(self.inetworks); i += 1 {
		if self.inetworks[i].GetId() == network.ID {
			find = true
			break
		}
	}
	if !find {
		self.inetworks = append(self.inetworks, network)
	}
}

func (self *SWire) getNetworkById(networkId string) *SNetwork {
	networks, err := self.GetINetworks()
	if err != nil {
		return nil
	}
	log.Debugf("search for networks %d", len(networks))
	for i := 0; i < len(networks); i += 1 {
		log.Debugf("search %s", networks[i].GetName())
		network := networks[i]
		if network.GetId() == networkId {
			return network.(*SNetwork)
		}
	}
	return nil
}

func getDefaultGateWay(cidr string) (string, error) {
	ip, _, err := net.ParseCIDR(cidr)
	if err != nil {
		return "", err
	}

	ipv4 := ip.To4()
	if ipv4 == nil || len(ip.String()) == net.IPv6len {
		return "", fmt.Errorf("ipv6 is not supported currently")
	}

	if ipv4[3] != 0 {
		return "", fmt.Errorf("the last byte of ip address must be zero. e.g 192.168.0.0/16")
	}

	ipv4[3] = 1
	return ipv4.String(), nil
}

// https://support.huaweicloud.com/api-vpc/zh-cn_topic_0020090590.html
// cidr 掩码长度不能大于28
func (self *SRegion) createNetwork(vpcId string, name string, cidr string, desc string) (string, error) {
	gateway, err := getDefaultGateWay(cidr)
	if err != nil {
		return "", err
	}

	params := jsonutils.NewDict()
	subnetObj := jsonutils.NewDict()
	subnetObj.Add(jsonutils.NewString(name), "name")
	subnetObj.Add(jsonutils.NewString(vpcId), "vpc_id")
	subnetObj.Add(jsonutils.NewString(cidr), "cidr")
	subnetObj.Add(jsonutils.NewString(gateway), "gateway_ip")
	params.Add(subnetObj, "subnet")

	subnet := SNetwork{}
	err = DoCreate(self.ecsClient.Subnets.Create, params, &subnet)
	return subnet.ID, err
}
