package esxi

import (
	"context"
	"path"

	"github.com/vmware/govmomi/object"

	"strings"
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SImage struct {
	cache    *SDatastoreImageCache
	filename string
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
	return path.Base(self.filename)
}

func (self *SImage) GetGlobalId() string {
	return self.GetId()
}

func (self *SImage) GetStatus() string {
	dm := object.NewVirtualDiskManager(self.cache.datastore.manager.client.Client)
	ctx := context.Background()
	_, err := dm.QueryVirtualDiskInfo(ctx, self.getFullFilename(), self.getDatacenter(), true)
	if err != nil {
		return "saving"
	}
	return "active"
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
