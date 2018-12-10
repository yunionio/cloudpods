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
