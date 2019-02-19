package huawei

import (
	"context"
	"fmt"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/huawei/obs"
	"yunion.io/x/onecloud/pkg/util/qemuimg"
)

type SStoragecache struct {
	region *SRegion

	iimages []cloudprovider.ICloudImage
}

func GetBucketName(regionId string, imageId string) string {
	return fmt.Sprintf("imgcache-%s-%s", strings.ToLower(regionId), imageId)
}

func (self *SStoragecache) fetchImages() error {
	imagesGold, err := self.region.GetImages("", ImageOwnerPublic, "", EnvFusionCompute)
	if err != nil {
		return err
	}

	imagesSelf, err := self.region.GetImages("", ImageOwnerSelf, "", EnvFusionCompute)
	if err != nil {
		return err
	}

	self.iimages = make([]cloudprovider.ICloudImage, len(imagesGold)+len(imagesSelf))
	for i := range imagesGold {
		imagesGold[i].storageCache = self
		self.iimages[i] = &imagesGold[i]
	}
	for i := range imagesSelf {
		imagesSelf[i].storageCache = self
		self.iimages[i+len(imagesGold)] = &imagesSelf[i]
	}
	return nil
}

func (self *SStoragecache) GetId() string {
	return fmt.Sprintf("%s-%s", self.region.client.providerId, self.region.GetId())
}

func (self *SStoragecache) GetName() string {
	return fmt.Sprintf("%s-%s", self.region.client.providerName, self.region.GetId())
}

func (self *SStoragecache) GetGlobalId() string {
	return fmt.Sprintf("%s-%s", self.region.client.providerId, self.region.GetGlobalId())
}

func (self *SStoragecache) GetStatus() string {
	return "available"
}

func (self *SStoragecache) Refresh() error {
	return nil
}

func (self *SStoragecache) IsEmulated() bool {
	return false
}

func (self *SStoragecache) GetMetadata() *jsonutils.JSONDict {
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
	image, err := self.region.GetImage(extId)
	image.storageCache = self
	return &image, err
}

func (self *SStoragecache) GetPath() string {
	return ""
}

func (self *SStoragecache) GetManagerId() string {
	return self.region.client.providerId
}

// 目前支持使用vhd、zvhd、vmdk、qcow2、raw、zvhd2、vhdx、qcow、vdi或qed格式镜像文件创建私有镜像。
// 快速通道功能可快速完成镜像制作，但镜像文件需转换为raw或zvhd2格式并完成镜像优化。
// https://support.huaweicloud.com/api-ims/zh-cn_topic_0083905788.html
func (self *SStoragecache) CreateIImage(snapshotId, imageName, osType, imageDesc string) (cloudprovider.ICloudImage, error) {
	if imageId, err := self.region.createIImage(snapshotId, imageName, imageDesc); err != nil {
		return nil, err
	} else if image, err := self.region.GetImage(imageId); err != nil {
		return nil, err
	} else {
		image.storageCache = self
		iimage := make([]cloudprovider.ICloudImage, 1)
		iimage[0] = &image
		if err := cloudprovider.WaitStatus(iimage[0], "avaliable", 15*time.Second, 3600*time.Second); err != nil {
			return nil, err
		}
		return iimage[0], nil
	}
}

func (self *SStoragecache) DownloadImage(userCred mcclient.TokenCredential, imageId string, extId string, path string) (jsonutils.JSONObject, error) {
	return self.downloadImage(userCred, imageId, extId)
}

func (self *SStoragecache) downloadImage(userCred mcclient.TokenCredential, imageId string, extId string) (jsonutils.JSONObject, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SStoragecache) UploadImage(ctx context.Context, userCred mcclient.TokenCredential, imageId string, osArch, osType, osDist, osVersion string, extId string, isForce bool) (string, error) {
	if len(extId) > 0 {
		log.Debugf("UploadImage: Image external ID exists %s", extId)

		image, err := self.region.GetImage(extId)
		if err != nil {
			log.Errorf("GetImageStatus error %s", err)
		}
		if image.Status == ImageStatusActive && !isForce {
			return extId, nil
		}
	} else {
		log.Debugf("UploadImage: no external ID")
	}

	return self.uploadImage(ctx, userCred, imageId, osArch, osType, osDist, osVersion, isForce)
}

func (self *SStoragecache) uploadImage(ctx context.Context, userCred mcclient.TokenCredential, imageId string, osArch, osType, osDist string, osVersion string, isForce bool) (string, error) {
	bucketName := GetBucketName(self.region.GetId(), imageId)
	obsClient, err := self.region.getOBSClient()
	if err != nil {
		return "", err
	}

	// create bucket
	input := &obs.CreateBucketInput{}
	input.Bucket = bucketName
	input.Location = self.region.GetId()
	_, err = obsClient.CreateBucket(input)
	if err != nil {
		return "", err
	}
	defer obsClient.DeleteBucket(bucketName)

	// upload to huawei cloud
	s := auth.GetAdminSession(ctx, options.Options.Region, "")
	meta, reader, err := modules.Images.Download(s, imageId, string(qemuimg.VMDK), false)
	if err != nil {
		return "", err
	}
	log.Debugf("Images meta data %s", meta)
	_image, err := modules.Images.Get(s, imageId, nil)
	if err != nil {
		return "", err
	}

	minDiskGB, _ := _image.Int("min_disk")
	if minDiskGB <= 0 {
		minDiskGB = 40
	}
	// upload to huawei cloud
	obj := &obs.PutObjectInput{}
	obj.Bucket = bucketName
	obj.Key = imageId
	obj.Body = reader

	_, err = obsClient.PutObject(obj)
	if err != nil {
		return "", err
	}

	objDelete := &obs.DeleteObjectInput{}
	objDelete.Bucket = bucketName
	objDelete.Key = imageId
	defer obsClient.DeleteObject(objDelete) // remove object

	// check image name, avoid name conflict
	imageBaseName := imageId
	if imageBaseName[0] >= '0' && imageBaseName[0] <= '9' {
		imageBaseName = fmt.Sprintf("img%s", imageId)
	}
	imageName := imageBaseName
	nameIdx := 1

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
		nameIdx += 1
		log.Debugf("uploadImage Match remote name %s", imageName)
	}

	jobId, err := self.region.ImportImageJob(imageName, osDist, osVersion, osArch, bucketName, imageId, minDiskGB)

	if err != nil {
		log.Errorf("ImportImage error %s %s %s %s", jobId, imageId, bucketName, err)
		return "", err
	}

	// timeout: 1hour = 3600 seconds
	serviceType := self.region.ecsClient.Images.ServiceType()
	err = self.region.waitTaskStatus(serviceType, jobId, TASK_SUCCESS, 15*time.Second, 3600*time.Second)
	if err != nil {
		log.Errorf("waitTaskStatus %s", err)
		return "", err
	}

	// https://support.huaweicloud.com/api-ims/zh-cn_topic_0022473688.html
	return self.region.GetTaskEntityID(serviceType, jobId, "image_id")
}

func (self *SRegion) getStoragecache() *SStoragecache {
	if self.storageCache == nil {
		self.storageCache = &SStoragecache{region: self}
	}
	return self.storageCache
}

type SJob struct {
	Status     string            `json:"status"`
	Entities   map[string]string `json:"entities"`
	JobID      string            `json:"job_id"`
	JobType    string            `json:"job_type"`
	BeginTime  string            `json:"begin_time"`
	EndTime    string            `json:"end_time"`
	ErrorCode  string            `json:"error_code"`
	FailReason string            `json:"fail_reason"`
}

// https://support.huaweicloud.com/api-ims/zh-cn_topic_0020092109.html
func (self *SRegion) createIImage(snapshotId, imageName, imageDesc string) (string, error) {
	snapshot, err := self.GetSnapshotById(snapshotId)
	if err != nil {
		return "", err
	}

	disk, err := self.GetDisk(snapshot.VolumeID)
	if err != nil {
		return "", err
	}

	if disk.GetDiskType() != models.DISK_TYPE_SYS {
		return "", fmt.Errorf("disk type err, expected disk type %s", models.DISK_TYPE_SYS)
	}

	if len(disk.Attachments) == 0 {
		return "", fmt.Errorf("disk is not attached.")
	}

	imageObj := jsonutils.NewDict()
	imageObj.Add(jsonutils.NewString(disk.Attachments[0].ServerID), "instance_id")
	imageObj.Add(jsonutils.NewString(imageName), "name")
	imageObj.Add(jsonutils.NewString(imageDesc), "description")

	ret, err := self.ecsClient.Images.PerformAction2("action", "", imageObj, "")
	if err != nil {
		return "", err
	}

	job := SJob{}
	jobId, err := ret.GetString("job_id")
	querys := map[string]string{"service_type": self.ecsClient.Images.ServiceType()}
	err = DoGet(self.ecsClient.Jobs.Get, jobId, querys, &job)
	if err != nil {
		return "", err
	}

	if job.Status == "SUCCESS" {
		imageId, exists := job.Entities["image_id"]
		if exists {
			return imageId, nil
		} else {
			return "", fmt.Errorf("image id not found in create image job %s", job.JobID)
		}
	} else {
		return "", fmt.Errorf("create image failed, %s", job.FailReason)
	}

}

func (self *SRegion) GetIStoragecaches() ([]cloudprovider.ICloudStoragecache, error) {
	storageCache := self.getStoragecache()
	return []cloudprovider.ICloudStoragecache{storageCache}, nil
}

func (self *SRegion) GetIStoragecacheById(idstr string) (cloudprovider.ICloudStoragecache, error) {
	storageCache := self.getStoragecache()
	if storageCache.GetGlobalId() == idstr {
		return storageCache, nil
	}
	return nil, cloudprovider.ErrNotFound
}
