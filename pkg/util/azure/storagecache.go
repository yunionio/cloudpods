package azure

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

const (
	DefaultStorageAccount string = "image"
	DefaultContainer      string = "image-cache"

	DefaultReadBlockSize int64 = 4 * 1024 * 1024
)

type SStoragecache struct {
	region *SRegion

	iimages []cloudprovider.ICloudImage
}

func (self *SStoragecache) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SStoragecache) GetId() string {
	return fmt.Sprintf("%s-%s", self.region.client.providerId, self.region.GetId())
}

func (self *SStoragecache) GetName() string {
	return fmt.Sprintf("%s-%s", self.region.client.providerName, self.region.GetId())
}

func (self *SStoragecache) GetStatus() string {
	return "available"
}

func (self *SStoragecache) Refresh() error {
	return nil
}

func (self *SStoragecache) GetGlobalId() string {
	return fmt.Sprintf("%s-%s", self.region.client.providerId, self.region.GetGlobalId())
}

func (self *SStoragecache) IsEmulated() bool {
	return false
}

func (self *SStoragecache) GetManagerId() string {
	return self.region.client.providerId
}

func (self *SStoragecache) fetchImages() error {
	if images, err := self.region.GetImages(); err != nil {
		return err
	} else {
		self.iimages = make([]cloudprovider.ICloudImage, len(images))
		for i := 0; i < len(images); i++ {
			images[i].storageCache = self
			self.iimages[i] = &images[i]
		}
	}
	return nil
}

func (self *SStoragecache) GetIImages() ([]cloudprovider.ICloudImage, error) {
	if self.iimages == nil {
		if err := self.fetchImages(); err != nil {
			return nil, err
		}
	}
	return self.iimages, nil
}

func (self *SStoragecache) UploadImage(userCred mcclient.TokenCredential, imageId string, osArch, osType, osDist string, extId string, isForce bool) (string, error) {
	if len(extId) > 0 {
		status, _ := self.region.GetImageStatus(extId)
		if status == ImageStatusAvailable && !isForce {
			return extId, nil
		}
	}
	return self.uploadImage(userCred, imageId, osArch, osType, osDist, isForce)
}

func (self *SStoragecache) uploadImage(userCred mcclient.TokenCredential, imageId string, osArch, osType, osDist string, isForce bool) (string, error) {
	s := auth.GetAdminSession(options.Options.Region, "")

	if meta, reader, err := modules.Images.Download(s, imageId); err != nil {
		return "", err
	} else {
		// {"checksum":"d0ab0450979977c6ada8d85066a6e484","container_format":"bare","created_at":"2018-08-10T04:18:07","deleted":"False","disk_format":"vhd","id":"64189033-3ad4-413c-b074-6bf0b6be8508","is_public":"False","min_disk":"0","min_ram":"0","name":"centos-7.3.1611-20180104.vhd","owner":"5124d80475434da8b41fee48d5be94df","properties":{"os_arch":"x86_64","os_distribution":"CentOS","os_type":"Linux","os_version":"7.3.1611-VHD"},"protected":"False","size":"2028505088","status":"active","updated_at":"2018-08-10T04:20:59"}
		log.Infof("meta data %s", meta)

		imageNameOnBlob, _ := meta.GetString("name")
		if !strings.HasSuffix(imageNameOnBlob, ".vhd") {
			imageNameOnBlob = fmt.Sprintf("%s.vhd", imageNameOnBlob)
		}
		tmpFile := fmt.Sprintf("/opt/cloud/workspace/data/glance/image-cache/%s", imageNameOnBlob)
		defer os.Remove(tmpFile)
		f, err := os.Create(tmpFile)
		if err != nil {
			return "", err
		}
		defer f.Close()
		if _, err := io.Copy(f, reader); err != nil {
			return "", err
		}
		storageAccount := fmt.Sprintf("%s%s", self.region.Name, DefaultStorageAccount)

		storage, err := self.region.CreateStorageAccount(storageAccount)
		if err != nil {
			return "", err
		}

		if _, err := self.region.CreateContainer(storage.ID, DefaultContainer); err != nil {
			return "", err
		}

		size, _ := meta.Int("size")

		blobURI, err := self.region.UploadVHD(storage.ID, DefaultContainer, tmpFile)
		if err != nil {
			log.Errorf("uploadContainerFileByPath error: %v", err)
			return "", err
		}

		imageBaseName := imageId
		if imageBaseName[0] >= '0' && imageBaseName[0] <= '9' {
			imageBaseName = fmt.Sprintf("img%s", imageId)
		}
		imageName := imageBaseName
		nameIdx := 1

		// check image name, avoid name conflict
		for {
			if _, err = self.region.GetImageByName(imageName); err != nil {
				if err == cloudprovider.ErrNotFound {
					break
				} else {
					return "", err
				}
			}
			imageName = fmt.Sprintf("%s-%d", imageBaseName, nameIdx)
			nameIdx += 1
		}

		if image, err := self.region.CreateImageByBlob(imageName, osType, blobURI, int32(size>>30)); err != nil {
			return "", err
		} else {
			return image.GetGlobalId(), nil
		}
	}
}

func (self *SStoragecache) CreateIImage(snapshotId, imageName, osType, imageDesc string) (cloudprovider.ICloudImage, error) {
	if image, err := self.region.CreateImage(snapshotId, imageName, osType, imageDesc); err != nil {
		return nil, err
	} else {
		image.storageCache = self
		return image, nil
	}
}

func (self *SStoragecache) DownloadImage(userCred mcclient.TokenCredential, imageId string, extId string) (jsonutils.JSONObject, error) {
	return self.downloadImage(userCred, imageId, extId)
}

func (self *SStoragecache) downloadImage(userCred mcclient.TokenCredential, imageId string, extId string) (jsonutils.JSONObject, error) {
	if image, err := self.region.GetImage(extId); err != nil {
		return nil, err
	} else if snapshotId := image.Properties.StorageProfile.OsDisk.Snapshot.ID; len(snapshotId) == 0 {
		return nil, cloudprovider.ErrNotFound
	} else if uri, err := self.region.GrantAccessSnapshot(snapshotId); err != nil {
		return nil, err
	} else if resp, err := http.Get(uri); err != nil {
		return nil, err
	} else {
		_, _, snapshot := pareResourceGroupWithName(snapshotId, SNAPSHOT_RESOURCE)
		tmpImageFile := fmt.Sprintf("/opt/cloud/workspace/data/glance/image-cache/%s", snapshot)
		defer os.Remove(tmpImageFile)
		if f, err := os.Create(tmpImageFile); err != nil {
			return nil, err
		} else {
			readed, writed, skiped := 0, 0, 0
			data := make([]byte, DefaultReadBlockSize)
			for i := 0; i < int(resp.ContentLength/DefaultReadBlockSize); i++ {
				if _, err := resp.Body.Read(data); err != nil {
					return nil, err
				} else if isEmpty := func(array []byte) bool {
					for i := 0; i < len(array); i++ {
						if array[i] != 0 {
							return false
						}
					}
					return true
				}(data); !isEmpty {
					if _, err := f.Write(data); err != nil {
						return nil, err
					}
					writed += int(DefaultReadBlockSize)
				} else if _, err := f.Seek(DefaultReadBlockSize, os.SEEK_CUR); err != nil {
					return nil, err
				} else {
					skiped += int(DefaultReadBlockSize)
				}
				readed = readed + int(DefaultReadBlockSize)
				log.Debugf("has write %dMb skip %dMb total %dMb", writed>>20, skiped>>20, resp.ContentLength>>20)
			}
			rest := make([]byte, resp.ContentLength%DefaultReadBlockSize)
			if len(rest) > 0 {
				if _, err := resp.Body.Read(rest); err != nil {
					return nil, err
				} else if _, err := f.Write(rest); err != nil {
					return nil, err
				}
			}
			log.Debugf("download complate")
		}

		s := auth.GetAdminSession(options.Options.Region, "")
		params := jsonutils.Marshal(map[string]string{"image_id": imageId, "disk-format": "raw"})
		if file, err := os.Open(tmpImageFile); err != nil {
			return nil, err
		} else if result, err := modules.Images.Upload(s, params, file, resp.ContentLength); err != nil {
			return nil, err
		} else {
			return result, nil
		}

	}
}
