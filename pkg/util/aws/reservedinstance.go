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
	"github.com/aws/aws-sdk-go/service/ec2"

	"yunion.io/x/log"
)

func (self *SRegion) GetReservedInstance() error {
	params := &ec2.DescribeReservedInstancesInput{}
	res, err := self.ec2Client.DescribeReservedInstances(params)
	if err != nil {
		log.Errorf("DescribeReservedInstances fail %s", err)
		return err
	}
	log.Debugf("%#v", res)
	return nil
}

type SReservedHostOffering struct {
	Duration       int
	HourlyPrice    float64
	InstanceFamily string
	OfferingId     string
	PaymentOption  string
	UpfrontPrice   float64
}

func (self *SRegion) GetReservedHostOfferings() error {
	res, err := self.ec2Client.DescribeHostReservationOfferings(nil)
	if err != nil {
		log.Errorf("DescribeHostReservationOfferings fail %s", err)
		return err
	}
	log.Debugf("%#v", res)
	return nil
}
