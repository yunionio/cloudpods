package multicloud

import (
	"fmt"

	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SRegion struct{}

func (r *SRegion) CreateSnapshotPolicy(input *cloudprovider.SnapshotPolicyInput) (string, error) {
	return "", fmt.Errorf("CreateSnapshotPolicy not implement")
}

func (r *SRegion) GetISnapshotPolicyById(snapshotPolicyId string) (cloudprovider.ICloudSnapshotPolicy, error) {
	return nil, fmt.Errorf("GetISnapshotPolicyById not implement")
}

func (self *SRegion) GetISnapshotPolicies() ([]cloudprovider.ICloudSnapshotPolicy, error) {
	return nil, fmt.Errorf("GetISnapshotPolicies not implement")
}

func (self *SRegion) DeleteSnapshotPolicy(string) error {
	return fmt.Errorf("DeleteSnapshotPolicy not implement")
}

func (self *SRegion) ApplySnapshotPolicyToDisks(snapshotPolicyId string, diskIds []string) error {
	return fmt.Errorf("ApplySnapshotPolicyToDisks not implement")
}

func (self *SRegion) CancelSnapshotPolicyToDisks(diskIds []string) error {
	return fmt.Errorf("ApplySnapshotPolicyToDisks not implement")
}
