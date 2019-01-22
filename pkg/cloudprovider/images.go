package cloudprovider

import (
	"time"
)

const (
	IMAGE_STATUS_ACTIVE  = "active"
	IMAGE_STATUS_QUEUED  = "queued"
	IMAGE_STATUS_SAVING  = "saving"
	IMAGE_STATUS_KILLED  = "killed"
	IMAGE_STATUS_DELETED = "deleted"

	CachedImageTypeSystem     = "system"
	CachedImageTypeCustomized = "customized"
	CachedImageTypeShared     = "shared"
	CachedImageTypeMarket     = "market"
)

type SImage struct {
	Checksum string
	// ContainerFormat string
	CreatedAt  time.Time
	Deleted    bool
	DiskFormat string
	Id         string
	IsPublic   bool
	MinDisk    int
	MinRam     int
	Name       string
	Owner      string
	Properties map[string]string
	Protected  bool
	Size       int64
	Status     string
	// UpdatedAt       time.Time
}

func CloudImage2Image(image ICloudImage) SImage {
	return SImage{
		CreatedAt:  image.GetCreateTime(),
		Deleted:    false,
		DiskFormat: image.GetImageFormat(),
		Id:         image.GetId(),
		IsPublic:   image.GetImageType() != CachedImageTypeCustomized,
		MinDisk:    image.GetMinOsDiskSizeGb(),
		MinRam:     0,
		Name:       image.GetName(),
		Properties: map[string]string{
			"os_type":         image.GetOsType(),
			"os_distribution": image.GetOsDist(),
			"os_version":      image.GetOsVersion(),
			"os_arch":         image.GetOsArch(),
		},
		Protected: true,
		Size:      image.GetSize(),
		Status:    image.GetImageStatus(),
	}
}
