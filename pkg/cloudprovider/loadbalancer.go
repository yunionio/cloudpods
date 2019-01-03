package cloudprovider

import "yunion.io/x/onecloud/pkg/util/billing"

type SLoadbalancer struct {
	Name             string
	ZoneID           string
	VpcID            string
	NetworkID        string
	Address          string
	AddressType      string
	LoadbalancerSpec string
	ChargeType       string
	Bandwidth        int
	billingCycle     *billing.SBillingCycle
}
