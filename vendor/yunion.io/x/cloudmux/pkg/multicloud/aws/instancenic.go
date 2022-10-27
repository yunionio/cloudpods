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
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type SInstanceNic struct {
	instance *SInstance

	id      string
	ipAddr  string
	macAddr string

	cloudprovider.DummyICloudNic
}

func (self *SInstanceNic) GetId() string {
	return self.id
}

func (self *SInstanceNic) GetIP() string {
	return self.ipAddr
}

func (self *SInstanceNic) GetMAC() string {
	return self.macAddr
}

func (self *SInstanceNic) InClassicNetwork() bool {
	return false
}

func (self *SInstanceNic) GetDriver() string {
	return "virtio"
}

func (self *SInstanceNic) GetINetworkId() string {
	return self.instance.VpcAttributes.NetworkId
}

func (self *SInstanceNic) getEc2Client() *ec2.EC2 {
	ec2Client, err := self.instance.host.zone.region.getEc2Client()
	if err != nil {
		return nil
	}
	return ec2Client
}

func (self *SInstanceNic) GetSubAddress() ([]string, error) {
	var (
		ec2Client = self.getEc2Client()
		id        = self.GetId()
		input     = &ec2.DescribeNetworkInterfacesInput{
			NetworkInterfaceIds: []*string{
				aws.String(id),
			},
		}
	)
	output, err := ec2Client.DescribeNetworkInterfaces(input)
	if err != nil {
		return nil, errors.Wrapf(err, "aws ec2 DescribeNetworkInterfaces")
	}
	if got := len(output.NetworkInterfaces); got != 1 {
		return nil, errors.Errorf("got aws %d network interface, want 1", got)
	}
	networkInterface := output.NetworkInterfaces[0]
	if got := aws.StringValue(networkInterface.NetworkInterfaceId); got != id {
		return nil, errors.Errorf("got aws network interface %s, want %s", got, id)
	}
	var ipAddrs []string
	for _, privateIp := range networkInterface.PrivateIpAddresses {
		if !aws.BoolValue(privateIp.Primary) {
			ipAddrs = append(ipAddrs, *privateIp.PrivateIpAddress)
		}
	}
	return ipAddrs, nil
}

func (self *SInstanceNic) AssignAddress(ipAddrs []string) error {
	var (
		ec2Client = self.getEc2Client()
		id        = self.GetId()
		input     = &ec2.AssignPrivateIpAddressesInput{
			NetworkInterfaceId: aws.String(id),
			PrivateIpAddresses: aws.StringSlice(ipAddrs),
		}
	)
	_, err := ec2Client.AssignPrivateIpAddresses(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case "PrivateIpAddressLimitExceeded":
				return errors.Wrapf(cloudprovider.ErrAddressCountExceed, "aws ec2 AssignPrivateIpAddresses: %v", err)
			}
		}
		return errors.Wrapf(err, "aws ec2 AssignPrivateIpAddresses")
	}
	return nil
}

func (self *SInstanceNic) UnassignAddress(ipAddrs []string) error {
	var (
		ec2Client = self.getEc2Client()
		id        = self.GetId()
		input     = &ec2.UnassignPrivateIpAddressesInput{
			NetworkInterfaceId: aws.String(id),
			PrivateIpAddresses: aws.StringSlice(ipAddrs),
		}
	)
	_, err := ec2Client.UnassignPrivateIpAddresses(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case "InvalidNetworkInterfaceID.NotFound":
				return nil
			case "InvalidParameterValue":
				msg := aerr.Message()
				// "Some of the specified addresses are not assigned to interface eni-xxxxxxxxxxxxxxxxx"
				if strings.Contains(msg, " addresses are not assigned to interface ") {
					return nil
				}
			}
		}
		return errors.Wrapf(err, "aws ec2 UnassignPrivateIpAddresses")
	}
	return nil
}
