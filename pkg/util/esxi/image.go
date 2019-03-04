package esxi

import (
	"context"
	"path"
	"strings"
	"time"

	"github.com/vmware/govmomi/object"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type SImage struct {
	cache    *SDatastoreImageCache
	filename string
	size     int64
	createAt time.Time
}

func (self *SImage) GetMinRamSizeMb() int {
	return 0
}

func (self *SImage) getDatacenter() *object.Datacenter {
	return self.cache.datastore.datacenter.getDcObj()
}

func (self *SImage) getFullFilename() string {
	return self.cache.datastore.getPathString(self.filename)
}

func (self *SImage) GetId() string {
	idstr := path.Base(self.filename)
	if strings.HasSuffix(idstr, ".vmdk") {
		idstr = idstr[:len(idstr)-5]
	}
	return strings.ToLower(idstr)
}

func (self *SImage) GetName() string {
	return self.GetId()
}

func (self *SImage) GetGlobalId() string {
	return self.GetId()
}

func (self *SImage) GetStatus() string {
	dm := object.NewVirtualDiskManager(self.cache.datastore.manager.client.Client)
	ctx := context.Background()
	_, err := dm.QueryVirtualDiskInfo(ctx, self.getFullFilename(), self.getDatacenter(), true)
	if err != nil {
		return models.CACHED_IMAGE_STATUS_CACHE_FAILED
	}
	return models.CACHED_IMAGE_STATUS_READY
}

func (self *SImage) GetImageStatus() string {
	status := self.GetStatus()
	if status == models.CACHED_IMAGE_STATUS_READY {
		return cloudprovider.IMAGE_STATUS_ACTIVE
	}
	return cloudprovider.IMAGE_STATUS_DELETED
}

func (self *SImage) Refresh() error {
	return nil
}

func (self *SImage) IsEmulated() bool {
	return false
}

func (self *SImage) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SImage) Delete(ctx context.Context) error {
	return self.cache.datastore.DeleteVmdk(ctx, self.filename)
}

func (self *SImage) GetIStoragecache() cloudprovider.ICloudStoragecache {
	return self.cache
}

func (self *SImage) GetImageType() string {
	return cloudprovider.CachedImageTypeCustomized
}

func (self *SImage) GetSize() int64 {
	return self.size
}

func (self *SImage) GetOsType() string {
	return ""
}

func (self *SImage) GetOsDist() string {
	return ""
}

func (self *SImage) GetOsVersion() string {
	return ""
}

func (self *SImage) GetOsArch() string {
	return ""
}

func (self *SImage) GetMinOsDiskSizeGb() int {
	return int(self.GetSize() / 1024 / 1024 / 1024)
}

func (self *SImage) GetImageFormat() string {
	return "vmdk"
}

func (self *SImage) GetCreateTime() time.Time {
	return self.createAt
}
