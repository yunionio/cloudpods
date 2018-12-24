package qcloud

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	coslib "github.com/nelsonken/cos-go-sdk-v5/cos"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
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

func (self *SStoragecache) CreateIImage(snapshoutId, imageName, osType, imageDesc string) (cloudprovider.ICloudImage, error) {
	// if imageId, err := self.region.createIImage(snapshoutId, imageName, imageDesc); err != nil {
	// 	return nil, err
	// } else if image, err := self.region.GetImage(imageId); err != nil {
	// 	return nil, err
	// } else {
	// 	image.storageCache = self
	// 	iimage := make([]cloudprovider.ICloudImage, 1)
	// 	iimage[0] = image
	// 	if err := cloudprovider.WaitStatus(iimage[0], compute.IMAGE_STATUS_ACTIVE, 15*time.Second, 3600*time.Second); err != nil {
	// 		return nil, err
	// 	}
	// 	return iimage[0], nil
	// }
	return nil, nil
}

func (self *SStoragecache) DownloadImage(userCred mcclient.TokenCredential, imageId string, extId string, path string) (jsonutils.JSONObject, error) {
	//return self.downloadImage(userCred, imageId, extId, path)
	return nil, nil
}

func (self *SStoragecache) fetchImages() error {
	images := make([]SImage, 0)
	for {
		parts, total, err := self.region.GetImages("", "PRIVATE_IMAGE", nil, "", len(images), 50)
		if err != nil {
			return err
		}
		images = append(images, parts...)
		if len(images) >= total {
			break
		}
	}
	self.iimages = make([]cloudprovider.ICloudImage, len(images))
	for i := 0; i < len(images); i += 1 {
		images[i].storageCache = self
		self.iimages[i] = &images[i]
	}
	return nil
}

func (self *SStoragecache) GetIImages() ([]cloudprovider.ICloudImage, error) {
	if self.iimages == nil {
		err := self.fetchImages()
		if err != nil {
			return nil, err
		}
	}
	return self.iimages, nil
}

func (self *SStoragecache) GetIImageById(extId string) (cloudprovider.ICloudImage, error) {
	parts, _, err := self.region.GetImages("", "PRIVATE_IMAGE", []string{extId}, "", 0, 1)
	if err != nil {
		return nil, err
	}
	if len(parts) == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	parts[1].storageCache = self
	return &parts[0], nil
}

func (self *SStoragecache) GetPath() string {
	return ""
}

func (self *SStoragecache) UploadImage(ctx context.Context, userCred mcclient.TokenCredential, imageId string, osArch, osType, osDist, osVersion string, extId string, isForce bool) (string, error) {
	if len(extId) > 0 {
		log.Debugf("UploadImage: Image external ID exists %s", extId)

		status, err := self.region.GetImageStatus(extId)
		if err != nil {
			log.Errorf("GetImageStatus error %s", err)
		}
		if status == ImageStatusAvailable && !isForce {
			return extId, nil
		}
	} else {
		log.Debugf("UploadImage: no external ID")
	}
	return self.uploadImage(ctx, userCred, imageId, osArch, osType, osDist, osVersion, isForce)
}

func (self *SRegion) getCosUrl(bucket, object string) string {
	//signature := cosauth.NewSignature(self.client.AppID, bucket, self.client.SecretID, time.Now().Add(time.Minute*30).String(), time.Now().String(), "yunion", object).SignOnce(self.client.SecretKey)
	return fmt.Sprintf("http://%s-%s.cos.%s.myqcloud.com/%s", bucket, self.client.AppID, self.Region, object)
}

func (self *SStoragecache) uploadImage(ctx context.Context, userCred mcclient.TokenCredential, imageId string, osArch, osType, osDist, osVersion string, isForce bool) (string, error) {
	// first upload image to oss
	s := auth.GetAdminSession(ctx, options.Options.Region, "")

	meta, reader, err := modules.Images.Download(s, imageId)
	if err != nil {
		return "", err
	}

	tmpFile := fmt.Sprintf("%s/%s", options.Options.TempPath, imageId)
	defer os.Remove(tmpFile)
	f, err := os.Create(tmpFile)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := io.Copy(f, reader); err != nil {
		return "", err
	}

	log.Infof("meta data %s", meta)
	cos, err := self.region.GetCosClient()
	if err != nil {
		log.Errorf("GetOssClient err %s", err)
		return "", err
	}
	bucketName := strings.ToLower(fmt.Sprintf("imgcache-%s", self.region.GetId()))
	err = cos.BucketExists(context.Background(), bucketName)
	if err != nil {
		log.Debugf("Bucket %s not exists, to create ...", bucketName)
		err := cos.CreateBucket(context.Background(), bucketName, &coslib.AccessControl{ACL: "public-read"})
		if err != nil {
			log.Errorf("Create bucket error %s", err)
			return "", err
		}
	} else {
		log.Debugf("Bucket %s exists", bucketName)
	}
	log.Debugf("To upload image to bucket %s ...", bucketName)
	err = cos.Bucket(bucketName).UploadObjectBySlice(context.Background(), imageId, tmpFile, 3, map[string]string{})
	if err != nil {
		log.Errorf("UploadObject error %s %s", imageId, err)
		return "", err
	}

	defer cos.Bucket(bucketName).DeleteObject(context.Background(), imageId)

	// 腾讯云镜像名称需要小于20个字符
	imageBaseName := imageId[:10]
	if imageBaseName[0] >= '0' && imageBaseName[0] <= '9' {
		imageBaseName = fmt.Sprintf("img%s", imageId[:10])
	}
	imageName := imageBaseName
	nameIdx := 1

	// check image name, avoid name conflict
	for {
		_, err = self.region.GetImageByName(imageName)
		if err != nil {
			if err == cloudprovider.ErrNotFound {
				break
			} else {
				return "", err
			}
		}
		imageName = fmt.Sprintf("%s-%d", imageBaseName, nameIdx)
		nameIdx++
	}

	log.Debugf("Import image %s", imageName)
	if image, err := self.region.ImportImage(imageName, osArch, osDist, osVersion, self.region.getCosUrl(bucketName, imageId)); err != nil {
		return "", err
	} else if cloudprovider.WaitStatus(image, string(ImageStatusAvailable), 15*time.Second, 3600*time.Second); err != nil {
		return "", err
	} else {
		return image.ImageId, nil
	}
}
