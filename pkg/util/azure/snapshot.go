package azure

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-04-01/compute"
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SnapshotSku struct {
	Name string
	Tier string
}

type SSnapshot struct {
	region *SRegion

	ID         string
	Name       string
	Location   string
	ManagedBy  string
	Sku        SnapshotSku
	Properties DiskProperties
}

func (self *SSnapshot) GetId() string {
	return self.ID
}

func (self *SSnapshot) GetGlobalId() string {
	return strings.ToLower(self.ID)
}

func (self *SSnapshot) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SSnapshot) GetName() string {
	return self.Name
}

func (self *SSnapshot) GetStatus() string {
	return ""
}

func (self *SSnapshot) IsEmulated() bool {
	return false
}

func (self *SRegion) CreateSnapshot(diskId, snapName, desc string) (*SSnapshot, error) {
	globalId, resourceGroup, snapshotName := pareResourceGroupWithName(snapName, SNAPSHOT_RESOURCE)
	snapClient := compute.NewSnapshotsClientWithBaseURI(self.client.baseUrl, self.SubscriptionID)
	snapClient.Authorizer = self.client.authorizer
	params := compute.Snapshot{
		Name:     &snapshotName,
		Location: &self.Name,
		DiskProperties: &compute.DiskProperties{
			CreationData: &compute.CreationData{
				CreateOption:     compute.Copy,
				SourceResourceID: &diskId,
			},
		},
	}
	self.CreateResourceGroup(resourceGroup)
	if result, err := snapClient.CreateOrUpdate(context.Background(), resourceGroup, snapshotName, params); err != nil {
		return nil, err
	} else if err := result.WaitForCompletion(context.Background(), snapClient.Client); err != nil {
		return nil, err
	}
	return self.GetSnapshotDetail(globalId)
}

func (self *SSnapshot) Delete() error {
	return self.region.DeleteSnapshot(self.ID)
}

func (self *SSnapshot) GetSize() int32 {
	return self.Properties.DiskSizeGB
}

func (self *SRegion) DeleteSnapshot(snapshotId string) error {
	_, resourceGroup, snapshotName := pareResourceGroupWithName(snapshotId, SNAPSHOT_RESOURCE)
	snapClient := compute.NewSnapshotsClientWithBaseURI(self.client.baseUrl, self.SubscriptionID)
	snapClient.Authorizer = self.client.authorizer
	if result, err := snapClient.Delete(context.Background(), resourceGroup, snapshotName); err != nil {
		return err
	} else if err := result.WaitForCompletion(context.Background(), snapClient.Client); err != nil {
		return err
	}
	return nil
}

type AccessURIOutput struct {
	AccessSas string
}

type AccessProperties struct {
	Output AccessURIOutput
}

type AccessURI struct {
	Name       string
	Properties AccessProperties
}

func (self *SRegion) GrantAccessSnapshot(snapshotId string) (string, error) {
	_, resourceGroup, snapshotName := pareResourceGroupWithName(snapshotId, SNAPSHOT_RESOURCE)
	snapClient := compute.NewSnapshotsClientWithBaseURI(self.client.baseUrl, self.SubscriptionID)
	snapClient.Authorizer = self.client.authorizer
	durationInSeconds := int32(3600 * 24)
	params := compute.GrantAccessData{
		Access:            compute.Read,
		DurationInSeconds: &durationInSeconds,
	}
	accessURI := AccessURI{}
	if result, err := snapClient.GrantAccess(context.Background(), resourceGroup, snapshotName, params); err != nil {
		return "", err
	} else if err := result.WaitForCompletion(context.Background(), snapClient.Client); err != nil {
		return "", err
	} else if body, err := ioutil.ReadAll(result.Response().Body); err != nil {
		return "", err
	} else if err := json.Unmarshal(body, &accessURI); err != nil {
		return "", err
	}
	return accessURI.Properties.Output.AccessSas, nil
}

func (self *SSnapshot) Refresh() error {
	if snapshot, err := self.region.GetSnapshotDetail(self.ID); err != nil {
		return err
	} else if err := jsonutils.Update(self, snapshot); err != nil {
		return err
	}
	return nil
}

func (self *SRegion) GetISnapshotById(snapshotId string) (cloudprovider.ICloudSnapshot, error) {
	return self.GetSnapshotDetail(snapshotId)
}

func (self *SRegion) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
	if snapshots, err := self.GetSnapShots(""); err != nil {
		return nil, err
	} else {
		isnapshots := make([]cloudprovider.ICloudSnapshot, len(snapshots))
		for i := 0; i < len(snapshots); i++ {
			isnapshots[i] = &snapshots[i]
		}
		return isnapshots, nil
	}
}

func (self *SSnapshot) GetDiskId() string {
	return self.Properties.CreationData.SourceResourceID
}

func (self *SSnapshot) GetManagerId() string {
	return self.region.client.providerId
}

func (self *SSnapshot) GetRegionId() string {
	return self.region.GetId()
}

func (self *SSnapshot) GetDiskType() string {
	return ""
}
