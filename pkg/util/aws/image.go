package aws

import (
	"time"
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type ImageStatusType string

const (
	ImageStatusCreating     ImageStatusType = "Creating"
	ImageStatusAvailable    ImageStatusType = "Available"
	ImageStatusUnAvailable  ImageStatusType = "UnAvailable"
	ImageStatusCreateFailed ImageStatusType = "CreateFailed"
)

type SImage struct {
	storageCache *SStoragecache

	Architecture         string
	CreationTime         time.Time
	Description          string
	ImageId              string
	ImageName            string
	OSName               string
	OSType               string
	IsSupportCloudinit   bool
	IsSupportIoOptimized bool
	Platform             string
	Size                 int
	Status               ImageStatusType
	Usage                string
}

func (self *SImage) GetId() string {
	panic("implement me")
}

func (self *SImage) GetName() string {
	panic("implement me")
}

func (self *SImage) GetGlobalId() string {
	panic("implement me")
}

func (self *SImage) GetStatus() string {
	panic("implement me")
}

func (self *SImage) Refresh() error {
	panic("implement me")
}

func (self *SImage) IsEmulated() bool {
	panic("implement me")
}

func (self *SImage) GetMetadata() *jsonutils.JSONDict {
	panic("implement me")
}

func (self *SImage) Delete() error {
	panic("implement me")
}

func (self *SImage) GetIStoragecache() cloudprovider.ICloudStoragecache {
	panic("implement me")
}
