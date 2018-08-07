package esxi

import (
	"github.com/vmware/govmomi/vim25/mo"

	"github.com/yunionio/jsonutils"
	"github.com/yunionio/onecloud/pkg/cloudprovider"
	"github.com/yunionio/onecloud/pkg/compute/models"
)

var DATASTORE_PROPS = []string {"name", "parent"}


type SDatastore struct {
	SManagedObject
}

func NewDatastore(manager *SESXiClient, ds *mo.Datastore, dc *SDatacenter) *SDatastore {
	return &SDatastore{SManagedObject: newManagedObject(manager, ds, dc)}
}

func (self *SDatastore) getDatastore() *mo.Datastore {
	return self.object.(*mo.Datastore)
}

func (self *SDatastore) GetGlobalId() string {
	return ""
}

func (self *SDatastore) GetStatus() string {
	if self.getDatastore().Summary.Accessible {
		return models.STORAGE_ONLINE
	} else {
		return models.STORAGE_OFFLINE
	}
}

func (self *SDatastore) Refresh() error {
	return cloudprovider.ErrNotImplemented
}

func (self *SDatastore) IsEmulated() bool {
	return false
}

func (self *SDatastore) getVolumeId() string {
	return self.getDatastore().Summary.Type
}

func (self *SDatastore) GetIStoragecache() cloudprovider.ICloudStoragecache {
	return nil
}

func (self *SDatastore) GetIZone() cloudprovider.ICloudZone {
	return nil
}

func (self *SDatastore) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SDatastore) GetStorageType() string {
	return self.getDatastore().Summary.Type
}

func (self *SDatastore) GetMediumType() string {
	return ""
}

func (self *SDatastore) GetCapacityMB() int {
	return 0
}

func (self *SDatastore) GetStorageConf() jsonutils.JSONObject {
	conf := jsonutils.NewDict()
	return conf
}

func (self *SDatastore) GetEnabled() bool {
	return true
}

func (self *SDatastore) GetManagerId() string {
	return self.manager.providerId
}

func (self *SDatastore) CreateIDisk(name string, sizeGb int, desc string) (cloudprovider.ICloudDisk, error) {
	return nil, cloudprovider.ErrNotImplemented
}