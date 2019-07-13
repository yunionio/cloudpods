package aws

import "time"

type SGroup struct {
	GroupId   string `xml:"groupId"`
	GroupName string `xml:"groupName"`
}

type SPrivateIpAddress struct {
	PrivateIpAddress string `xml:"privateIpAddress"`
	PrivateDnsName   string `xml:"privateDnsName"`
	Primary          bool   `xml:"primary"`
}

type SAttachment struct {
	AttachmentId        string    `xml:"attachmentId"`
	InstanceOwnerId     string    `xml:"instanceOwnerId"`
	DeviceIndex         int       `xml:"deviceIndex"`
	Status              string    `xml:"status"`
	AttachTime          time.Time `xml:"attachTime"`
	DeleteOnTermination bool      `xml:"deleteOnTermination"`
}

type SNetworkInterface struct {
	NetworkInterfaceId    string              `xml:"networkInterfaceId"`
	SubnetId              string              `xml:"subnetId"`
	VpcId                 string              `xml:"vpcId"`
	AvailabilityZone      string              `xml:"availabilityZone"`
	Description           string              `xml:"description"`
	OwnerId               string              `xml:"ownerId"`
	RequesterId           string              `xml:"requesterId"`
	RequesterManaged      bool                `xml:"requesterManaged"`
	Status                string              `xml:"status"`
	MacAddress            string              `xml:"macAddress"`
	PrivateIpAddress      string              `xml:"privateIpAddress"`
	PrivateDnsName        string              `xml:"privateDnsName"`
	SourceDestCheck       bool                `xml:"sourceDestCheck"`
	GroupSet              []SGroup            `xml:"groupSet>item"`
	Attachment            SAttachment         `xml:"attachment"`
	PrivateIpAddressesSet []SPrivateIpAddress `xml:"privateIpAddressesSet>item"`
	InterfaceType         string              `xml:"interfaceType"`
}

type SNetworkInterfaces struct {
	NetworkInterface []SNetworkInterface `xml:"networkInterfaceSet>item"`
}

func (region *SRegion) GetNetworkInterfaces() ([]SNetworkInterface, error) {
	params := map[string]string{}
	interfaces := SNetworkInterfaces{}
	err := region.ec2Request("DescribeNetworkInterfaces", params, &interfaces)
	if err != nil {
		return nil, err
	}
	return interfaces.NetworkInterface, nil
}
