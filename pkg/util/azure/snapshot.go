package azure

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-04-01/compute"
	"yunion.io/x/jsonutils"
)

type SnapshotSku struct {
	Name string
	Tier string
}

type SSnapshot struct {
	disk *SDisk

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
	globalId, _, _ := pareResourceGroupWithName(self.ID, SNAPSHOT_RESOURCE)
	return globalId
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
	if result, err := snapClient.CreateOrUpdate(context.Background(), resourceGroup, snapshotName, params); err != nil {
		return nil, err
	} else if err := result.WaitForCompletion(context.Background(), snapClient.Client); err != nil {
		return nil, err
	}
	return self.GetSnapshotDetail(globalId)
}

func (self *SSnapshot) Delete() error {
	if self.disk == nil {
		return fmt.Errorf("not init disk for snapshot %s", self.Name)
	}
	return self.disk.storage.zone.region.DeleteSnapshot(self.ID)
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
	snapClient.RequestInspector = LogRequest()
	snapClient.ResponseInspector = LogResponse()
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
	if snapshot, err := self.disk.GetSnapshotDetail(self.ID); err != nil {
		return err
	} else if err := jsonutils.Update(self, snapshot); err != nil {
		return err
	}
	return nil
}
