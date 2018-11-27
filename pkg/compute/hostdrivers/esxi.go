package hostdrivers

import (
	"context"
	"fmt"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/httputils"
)

type SESXiHostDriver struct {
	SManagedVirtualizationHostDriver
}

func init() {
	driver := SESXiHostDriver{}
	models.RegisterHostDriver(&driver)
}

func (self *SESXiHostDriver) GetHostType() string {
	return models.HOST_TYPE_ESXI
}

func (self *SESXiHostDriver) CheckAndSetCacheImage(ctx context.Context, host *models.SHost, storageCache *models.SStoragecache, task taskman.ITask) error {
	params := task.GetParams()
	imageId, err := params.GetString("image_id")
	if err != nil {
		return err
	}
	isForce := jsonutils.QueryBoolean(params, "is_force", false)
	obj, err := models.CachedimageManager.FetchById(imageId)
	if err != nil {
		return err
	}
	cacheImage := obj.(*models.SCachedimage)
	srcHostCacheImage, err := cacheImage.ChooseSourceStoragecacheInRange(models.HOST_TYPE_ESXI, []string{host.Id},
		[]interface{}{host.GetZone(), host.GetCloudprovider()})
	if err != nil {
		return err
	}

	type contentStruct struct {
		ImageId        string
		HostId         string
		HostIp         string
		SrcHostIp      string
		SrcPath        string
		SrcDatastore   models.SVCenterAccessInfo
		Datastore      models.SVCenterAccessInfo
		Format         string
		IsForce        bool
		StoragecacheId string
	}

	content := contentStruct{}
	content.ImageId = imageId
	content.HostId = host.Id
	content.HostIp = host.AccessIp
	content.Format = cacheImage.GetFormat()

	storage := host.GetStorageByFilePath(storageCache.Path)
	accessInfo, err := host.GetCloudaccount().GetVCenterAccessInfo(storage.ExternalId)
	if err != nil {
		return err
	}
	content.Datastore = accessInfo

	if srcHostCacheImage != nil {
		err = srcHostCacheImage.AddDownloadRefcount()
		if err != nil {
			return err
		}
		srcHost, err := srcHostCacheImage.GetHost()
		if err != nil {
			return err
		}
		content.SrcHostIp = srcHost.AccessIp
		content.SrcPath = srcHostCacheImage.Path
		srcStorage := srcHost.GetStorageByFilePath(srcHostCacheImage.Path)
		accessInfo, err := srcHost.GetCloudaccount().GetVCenterAccessInfo(srcStorage.ExternalId)
		if err != nil {
			return err
		}
		content.SrcDatastore = accessInfo
	}

	agent, err := host.GetEsxiAgentHost()
	if err != nil {
		log.Errorf("find ESXi agent fail: %s", err)
		return err
	}

	if agent == nil {
		return fmt.Errorf("fail to find valid ESXi agent")
	}

	url := fmt.Sprintf("%s/disks/image_cache", agent.ManagerUri)

	if isForce {
		content.IsForce = true
	}
	content.StoragecacheId = storageCache.Id

	body := jsonutils.NewDict()
	body.Add(jsonutils.Marshal(&content), "disk")

	header := task.GetTaskRequestHeader()

	_, _, err = httputils.JSONRequest(httputils.GetDefaultClient(), ctx, "POST", url, header, body, false)
	if err != nil {
		return err
	}
	return nil
}

func (self *SESXiHostDriver) RequestAllocateDiskOnStorage(ctx context.Context, host *models.SHost, storage *models.SStorage, disk *models.SDisk, task taskman.ITask, content *jsonutils.JSONDict) error {
	agent, err := host.GetEsxiAgentHost()
	if err != nil {
		log.Errorf("find ESXi agent fail: %s", err)
		return err
	}

	if agent == nil {
		return fmt.Errorf("fail to find valid ESXi agent")
	}

	type specStruct struct {
		Datastore models.SVCenterAccessInfo
		HostIp    string
		Format    string
	}

	spec := specStruct{}
	spec.HostIp = host.AccessIp
	spec.Format = "vmdk"

	accessInfo, err := host.GetCloudaccount().GetVCenterAccessInfo(storage.ExternalId)
	if err != nil {
		return err
	}
	spec.Datastore = accessInfo

	body := jsonutils.NewDict()
	body.Add(jsonutils.Marshal(&spec), "disk")

	url := fmt.Sprintf("/disks/agent/create/%s", disk.Id)

	header := task.GetTaskRequestHeader()

	_, err = agent.Request(task.GetUserCred(), "POST", url, header, body)
	return err
}

func (self *SESXiHostDriver) RequestPrepareSaveDiskOnHost(ctx context.Context, host *models.SHost, disk *models.SDisk, imageId string, task taskman.ITask) error {
	agent, err := host.GetEsxiAgentHost()
	if err != nil {
		log.Errorf("find ESXi agent fail: %s", err)
		return err
	}

	if agent == nil {
		return fmt.Errorf("fail to find valid ESXi agent")
	}

	guests := disk.GetGuests()
	if len(guests) == 0 {
		return fmt.Errorf("No VM associate with this disk")
	}

	if len(guests) > 1 {
		return fmt.Errorf("The disk is attached to multiple guests")
	}

	guest := guests[0]

	if guest.HostId != host.Id {
		return fmt.Errorf("The only guest is not on the host????")
	}

	type specStruct struct {
		Vm      models.SVCenterAccessInfo
		Disk    models.SVCenterAccessInfo
		HostIp  string
		ImageId string
	}

	spec := specStruct{}
	spec.HostIp = host.AccessIp
	spec.ImageId = imageId

	account := host.GetCloudaccount()
	accessInfo, err := account.GetVCenterAccessInfo(guest.ExternalId)
	if err != nil {
		return err
	}
	spec.Vm = accessInfo

	accessInfo, err = account.GetVCenterAccessInfo(disk.ExternalId)
	if err != nil {
		return err
	}
	spec.Disk = accessInfo

	body := jsonutils.NewDict()
	body.Add(jsonutils.Marshal(&spec), "disk")

	url := fmt.Sprintf("/disks/agent/save-prepare/%s", disk.Id)
	header := task.GetTaskRequestHeader()

	_, err = agent.Request(task.GetUserCred(), "POST", url, header, body)
	return err
}

func (self *SESXiHostDriver) RequestSaveUploadImageOnHost(ctx context.Context, host *models.SHost, disk *models.SDisk, imageId string, task taskman.ITask, data jsonutils.JSONObject) error {

	imagePath, _ := data.GetString("backup")
	if len(imagePath) == 0 {
		return fmt.Errorf("missing parameter backup")
	}
	agentId, _ := data.GetString("agent_id")
	if len(agentId) == 0 {
		return fmt.Errorf("missing parameter agent_id")
	}

	agent := models.HostManager.FetchHostById(agentId)
	if agent == nil {
		return fmt.Errorf("cannot find host with id %s", agentId)
	}

	storage := disk.GetStorage()

	type specStruct struct {
		ImagePath      string
		ImageId        string
		StorageId      string
		StoragecacheId string
		Compress       bool `json:",allowempty"`
	}

	spec := specStruct{}
	spec.ImageId = imageId
	spec.ImagePath = imagePath
	spec.StorageId = storage.Id
	spec.StoragecacheId = storage.StoragecacheId
	spec.Compress = false

	body := jsonutils.NewDict()
	body.Add(jsonutils.Marshal(&spec), "disk")

	url := "/disks/agent/upload"

	header := task.GetTaskRequestHeader()

	_, err := agent.Request(task.GetUserCred(), "POST", url, header, body)
	return err
}
