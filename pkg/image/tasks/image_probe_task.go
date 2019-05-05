package tasks

import (
	"context"
	"fmt"
	"os"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/image"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/hostman/guestfs/fsdriver"
	"yunion.io/x/onecloud/pkg/hostman/storageman"
	"yunion.io/x/onecloud/pkg/image/models"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
)

type ImageProbeTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(ImageProbeTask{})
}

func (self *ImageProbeTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	image := obj.(*models.SImage)
	self.StartImageProbe(ctx, image)
}

func (self *ImageProbeTask) StartImageProbe(ctx context.Context, image *models.SImage) {
	var (
		probeSucc bool = true
		errMsg    string
	)

	// probe image info
	func() {
		diskPath := image.GetPath("")
		kvmDisk := storageman.NewKVMGuestDisk(diskPath)
		defer kvmDisk.Disconnect()
		if !kvmDisk.Connect() {
			probeSucc = false
			errMsg = "Disk connector failed to connect image"
			return
		}

		rootfs := kvmDisk.MountKvmRootfs()
		if rootfs == nil {
			probeSucc = false
			errMsg = "Failed mounting rootfs for kvm disk"
			return
		}
		defer kvmDisk.UmountKvmRootfs(rootfs)

		imageInfo := self.getImageInfo(kvmDisk, rootfs)
		self.updateImageInfo(ctx, image, imageInfo)
	}()

	// if failed, set image unavailable, because size and chksum changed
	if err := self.updataeImageSizeAndChksum(ctx, image); err != nil {
		image.SetStatus(self.UserCred, api.IMAGE_STATUS_KILLED,
			fmt.Sprintf("Image update failed after probe %s", err))
		probeSucc = false
		errMsg = err.Error()
	}

	if probeSucc {
		self.TaskComplete(ctx, image)
	} else {
		self.TaskFailed(ctx, image, errMsg)
	}

}

func (self *ImageProbeTask) updataeImageSizeAndChksum(
	ctx context.Context, image *models.SImage,
) error {
	imagePath := image.GetPath("")
	fp, err := os.Open(imagePath)
	if err != nil {
		return err
	}
	defer fp.Close()

	stat, err := fp.Stat()
	if err != nil {
		return err
	}

	chksum, err := fileutils2.MD5(imagePath)
	if err != nil {
		return err
	}

	fastchksum, err := fileutils2.FastCheckSum(imagePath)
	if err != nil {
		return err
	}

	_, err = db.Update(image, func() error {
		image.Size = stat.Size()
		image.Checksum = chksum
		image.FastHash = fastchksum
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func (self *ImageProbeTask) updateImageInfo(ctx context.Context, image *models.SImage, imageInfo *sImageInfo) {
	if len(imageInfo.OsType) > 0 {
		models.ImagePropertyManager.SaveProperty(ctx, self.UserCred, image.Id, api.IMAGE_OS_TYPE, imageInfo.OsType)
	}
	if imageInfo.IsUEFISupport {
		models.ImagePropertyManager.SaveProperty(ctx, self.UserCred, image.Id, api.IMAGE_UEFI_SUPPORT, "true")
	}
	if imageInfo.IsLVMPartition {
		models.ImagePropertyManager.SaveProperty(ctx, self.UserCred, image.Id, api.IMAGE_IS_LVM_PARTITION, "true")
	}
	models.ImagePropertyManager.SaveProperties(ctx, self.UserCred, image.Id, jsonutils.Marshal(imageInfo.osInfo))
}

type sImageInfo struct {
	osInfo *fsdriver.SReleaseInfo

	OsType         string
	IsUEFISupport  bool
	IsLVMPartition bool
}

func (self *ImageProbeTask) getImageInfo(kvmDisk *storageman.SKVMGuestDisk, rootfs fsdriver.IRootFsDriver) *sImageInfo {
	partition := rootfs.GetPartition()
	return &sImageInfo{
		osInfo:         rootfs.GetReleaseInfo(partition),
		OsType:         rootfs.GetOs(),
		IsUEFISupport:  kvmDisk.DetectIsUEFISupport(rootfs),
		IsLVMPartition: kvmDisk.IsLVMPartition(),
	}
}

func (self *ImageProbeTask) TaskFailed(ctx context.Context, image *models.SImage, reason string) {
	log.Errorln(reason)
	self.SetStageFailed(ctx, reason)
}

func (self *ImageProbeTask) TaskComplete(ctx context.Context, image *models.SImage) {
	log.Infof("Image Probe Complete ...")
	if jsonutils.QueryBoolean(self.Params, "do_convert", false) {
		self.SetStage("OnConvertComplete", nil)
		image.StartImageConvertTask(ctx, self.UserCred, self.GetId())
	} else {
		self.SetStageComplete(ctx, nil)
	}
}

func (self *ImageProbeTask) OnConvertComplete(ctx context.Context, image *models.SImage, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}
