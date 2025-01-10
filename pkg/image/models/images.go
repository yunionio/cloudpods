// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package models

import (
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/pinyinutils"
	"yunion.io/x/pkg/util/qemuimgfmt"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/util/streamutils"
	"yunion.io/x/pkg/util/timeutils"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/image"
	noapi "yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/quotas"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
	deployapi "yunion.io/x/onecloud/pkg/hostman/hostdeployer/apis"
	"yunion.io/x/onecloud/pkg/hostman/hostdeployer/deployclient"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/image/options"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	identity_modules "yunion.io/x/onecloud/pkg/mcclient/modules/identity"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/image"
	"yunion.io/x/onecloud/pkg/mcclient/modules/notify"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/procutils"
	"yunion.io/x/onecloud/pkg/util/qemuimg"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

const (
	LocalFilePrefix = api.LocalFilePrefix
)

type SImageManager struct {
	db.SSharableVirtualResourceBaseManager
	db.SMultiArchResourceBaseManager
	db.SEncryptedResourceManager
}

var ImageManager *SImageManager

var imgStreamingWorkerMan *appsrv.SWorkerManager

func init() {
	ImageManager = &SImageManager{
		SSharableVirtualResourceBaseManager: db.NewSharableVirtualResourceBaseManager(
			SImage{},
			"images",
			"image",
			"images",
		),
	}
	ImageManager.SetVirtualObject(ImageManager)

}

func InitImageStreamWorkers() {
	imgStreamingWorkerMan = appsrv.NewWorkerManager("image_streaming_worker", options.Options.ImageStreamWorkerCount, 1024, true)
}

/*
+------------------+--------------+------+-----+---------+-------+
| Field            | Type         | Null | Key | Default | Extra |
+------------------+--------------+------+-----+---------+-------+
| id               | varchar(36)  | NO   | PRI | NULL    |       |
| name             | varchar(255) | YES  |     | NULL    |       |
| size             | bigint(20)   | YES  |     | NULL    |       |
| status           | varchar(30)  | NO   |     | NULL    |       |
| is_public        | tinyint(1)   | NO   | MUL | NULL    |       |
| location         | text         | YES  |     | NULL    |       |
| created_at       | datetime     | NO   |     | NULL    |       |
| updated_at       | datetime     | YES  |     | NULL    |       |
| deleted_at       | datetime     | YES  |     | NULL    |       |
| deleted          | tinyint(1)   | NO   | MUL | NULL    |       |
| parent_id        | varchar(36)  | YES  |     | NULL    |       |
| disk_format      | varchar(20)  | YES  |     | NULL    |       |
| container_format | varchar(20)  | YES  |     | NULL    |       |
| checksum         | varchar(32)  | YES  |     | NULL    |       |
| owner            | varchar(255) | YES  |     | NULL    |       |
| min_disk         | int(11)      | NO   |     | NULL    |       |
| min_ram          | int(11)      | NO   |     | NULL    |       |
| protected        | tinyint(1)   | YES  |     | NULL    |       |
| description      | varchar(256) | YES  |     | NULL    |       |
+------------------+--------------+------+-----+---------+-------+
*/
type SImage struct {
	db.SSharableVirtualResourceBase
	db.SMultiArchResourceBase
	db.SEncryptedResource

	// 镜像大小, 单位Byte
	Size int64 `nullable:"true" list:"user" create:"optional"`
	// 存储地址
	Location string `nullable:"true" list:"user"`

	// 镜像格式
	DiskFormat string `width:"20" charset:"ascii" nullable:"true" list:"user" create:"optional" default:"raw"`
	// 校验和
	Checksum string `width:"32" charset:"ascii" nullable:"true" get:"user" list:"user"`
	FastHash string `width:"32" charset:"ascii" nullable:"true" get:"user"`
	// 用户Id
	Owner string `width:"255" charset:"ascii" nullable:"true" get:"user"`
	// 最小系统盘要求
	MinDiskMB int32 `name:"min_disk" nullable:"false" default:"0" list:"user" create:"optional" update:"user"`
	// 最小内存要求
	MinRamMB int32 `name:"min_ram" nullable:"false" default:"0" list:"user" create:"optional" update:"user"`

	// 是否有删除保护
	Protected tristate.TriState `default:"true" list:"user" get:"user" create:"optional" update:"user"`
	// 是否是标准镜像
	IsStandard tristate.TriState `default:"false" list:"user" get:"user" create:"admin_optional" update:"user"`
	// 是否是主机镜像
	IsGuestImage tristate.TriState `default:"false" create:"optional" list:"user"`
	// 是否是数据盘镜像
	IsData tristate.TriState `default:"false" create:"optional" list:"user" update:"user"`

	// image copy from url, save origin checksum before probe
	// 从镜像时长导入的镜像校验和
	OssChecksum string `width:"32" charset:"ascii" nullable:"true" get:"user" list:"user"`

	// 加密状态, "",encrypting,encrypted
	EncryptStatus string `width:"16" charset:"ascii" nullable:"true" get:"user" list:"user"`
}

func (manager *SImageManager) CustomizeHandlerInfo(info *appsrv.SHandlerInfo) {
	manager.SSharableVirtualResourceBaseManager.CustomizeHandlerInfo(info)

	switch info.GetName(nil) {
	case "get_details", "create", "update":
		info.SetProcessTimeout(time.Hour * 4).SetWorkerManager(imgStreamingWorkerMan)
	}
}

func (manager *SImageManager) FetchCreateHeaderData(ctx context.Context, header http.Header) (jsonutils.JSONObject, error) {
	data := modules.FetchImageMeta(header)
	return data, nil
}

func (manager *SImageManager) FetchUpdateHeaderData(ctx context.Context, header http.Header) (jsonutils.JSONObject, error) {
	return modules.FetchImageMeta(header), nil
}

func (manager *SImageManager) InitializeData() error {
	// set cloudregion ID
	images := make([]SImage, 0)
	q := manager.Query().IsNullOrEmpty("tenant_id")
	err := db.FetchModelObjects(manager, q, &images)
	if err != nil {
		return err
	}
	for i := 0; i < len(images); i += 1 {
		if len(images[i].ProjectId) == 0 {
			db.Update(&images[i], func() error {
				images[i].ProjectId = images[i].Owner
				return nil
			})
		}
	}
	return nil
}

func (manager *SImageManager) GetPropertyDetail(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	appParams := appsrv.AppContextGetParams(ctx)
	appParams.OverrideResponseBodyWrapper = true

	queryDict := query.(*jsonutils.JSONDict)
	queryDict.Add(jsonutils.JSONTrue, "details")

	items, err := db.ListItems(manager, ctx, userCred, queryDict, nil)
	if err != nil {
		log.Errorf("Fail to list items: %s", err)
		return nil, httperrors.NewGeneralError(err)
	}
	return modulebase.ListResult2JSONWithKey(items, manager.KeywordPlural()), nil
}

func (manager *SImageManager) IsCustomizedGetDetailsBody() bool {
	return true
}

func (self *SImage) CustomizedGetDetailsBody(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	filePath := self.Location
	status := self.Status

	if self.IsGuestImage.IsFalse() {
		formatStr := jsonutils.GetAnyString(query, []string{"format", "disk_format"})
		if len(formatStr) > 0 && formatStr != api.IMAGE_DISK_FORMAT_TGZ {
			subimg := ImageSubformatManager.FetchSubImage(self.Id, formatStr)
			if subimg != nil {
				if strings.HasPrefix(subimg.Location, api.LocalFilePrefix) {
					isTorrent := jsonutils.QueryBoolean(query, "torrent", false)
					if !isTorrent {
						filePath = subimg.Location
						status = subimg.Status
					} else {
						filePath = subimg.getLocalTorrentLocation()
						status = subimg.TorrentStatus
					}
				} else {
					filePath = subimg.Location
					status = subimg.Status
				}
			} else {
				return nil, httperrors.NewNotFoundError("format %s not found", formatStr)
			}
		}
	}

	if status != api.IMAGE_STATUS_ACTIVE {
		return nil, httperrors.NewInvalidStatusError("cannot download in status %s", status)
	}

	if filePath == "" {
		return nil, httperrors.NewInvalidStatusError("empty file path")
	}

	size, rc, err := GetImage(ctx, filePath)
	if err != nil {
		return nil, errors.Wrap(err, "get image")
	}
	defer rc.Close()

	appParams := appsrv.AppContextGetParams(ctx)
	appParams.Response.Header().Set("Content-Length", strconv.FormatInt(size, 10))

	_, err = streamutils.StreamPipe(rc, appParams.Response, false, nil)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}

	return nil, nil
}

func (self *SImage) getMoreDetails(out api.ImageDetails) api.ImageDetails {
	properties, err := ImagePropertyManager.GetProperties(self.Id)
	if err != nil {
		log.Errorf("ImagePropertyManager.GetProperties fail %s", err)
	}
	out.Properties = properties

	if self.PendingDeleted {
		pendingDeletedAt := self.PendingDeletedAt.Add(time.Second * time.Duration(options.Options.PendingDeleteExpireSeconds))
		out.AutoDeleteAt = pendingDeletedAt
	}

	var ossChksum = self.OssChecksum
	if len(self.OssChecksum) == 0 {
		ossChksum = self.Checksum
	}
	out.OssChecksum = ossChksum
	out.DisableDelete = self.Protected.Bool()
	return out
}

func (manager *SImageManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.ImageDetails {
	rows := make([]api.ImageDetails, len(objs))

	virtRows := manager.SSharableVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	encRows := manager.SEncryptedResourceManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)

	for i := range rows {
		image := objs[i].(*SImage)
		rows[i] = api.ImageDetails{
			SharableVirtualResourceDetails: virtRows[i],
			EncryptedResourceDetails:       encRows[i],
		}
		rows[i] = image.getMoreDetails(rows[i])
	}

	return rows
}

func (self *SImage) GetExtraDetailsHeaders(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) map[string]string {
	headers := make(map[string]string)

	details := ImageManager.FetchCustomizeColumns(ctx, userCred, query, []interface{}{self}, nil, false)
	extra := jsonutils.Marshal(details[0]).(*jsonutils.JSONDict)
	for _, k := range extra.SortedKeys() {
		if k == "properties" {
			continue
		}
		val, _ := extra.GetString(k)
		if len(val) > 0 {
			headers[fmt.Sprintf("%s%s", modules.IMAGE_META, k)] = val
		}
	}

	jsonDict := jsonutils.Marshal(self).(*jsonutils.JSONDict)
	fields, _ := db.GetDetailFields(self.GetModelManager(), userCred)
	for _, k := range jsonDict.SortedKeys() {
		if utils.IsInStringArray(k, fields) {
			val, _ := jsonDict.GetString(k)
			if len(val) > 0 {
				headers[fmt.Sprintf("%s%s", modules.IMAGE_META, k)] = val
			}
		}
	}

	formatStr := jsonutils.GetAnyString(query, []string{"format", "disk_format"})
	if len(formatStr) > 0 {
		subimg := ImageSubformatManager.FetchSubImage(self.Id, formatStr)
		if subimg != nil {
			headers[fmt.Sprintf("%s%s", modules.IMAGE_META, "disk_format")] = formatStr
			isTorrent := jsonutils.QueryBoolean(query, "torrent", false)
			if !isTorrent {
				headers[fmt.Sprintf("%s%s", modules.IMAGE_META, "status")] = subimg.Status
				headers[fmt.Sprintf("%s%s", modules.IMAGE_META, "size")] = fmt.Sprintf("%d", subimg.Size)
				headers[fmt.Sprintf("%s%s", modules.IMAGE_META, "checksum")] = subimg.Checksum
			} else {
				headers[fmt.Sprintf("%s%s", modules.IMAGE_META, "status")] = subimg.TorrentStatus
				headers[fmt.Sprintf("%s%s", modules.IMAGE_META, "size")] = fmt.Sprintf("%d", subimg.TorrentSize)
				headers[fmt.Sprintf("%s%s", modules.IMAGE_META, "checksum")] = subimg.TorrentChecksum
			}
		}
	}

	// none of subimage business
	var ossChksum = self.OssChecksum
	if len(self.OssChecksum) == 0 {
		ossChksum = self.Checksum
	}
	headers[fmt.Sprintf("%s%s", modules.IMAGE_META, "oss_checksum")] = ossChksum

	properties, _ := ImagePropertyManager.GetProperties(self.Id)
	if len(properties) > 0 {
		for k, v := range properties {
			headers[fmt.Sprintf("%s%s", modules.IMAGE_META_PROPERTY, k)] = v
		}
	}

	if self.PendingDeleted {
		pendingDeletedAt := self.PendingDeletedAt.Add(time.Second * time.Duration(options.Options.PendingDeleteExpireSeconds))
		headers[fmt.Sprintf("%s%s", modules.IMAGE_META, "auto_delete_at")] = timeutils.FullIsoTime(pendingDeletedAt)
	}

	return headers
}

func (manager *SImageManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input api.ImageCreateInput,
) (api.ImageCreateInput, error) {
	var err error
	input.SharableVirtualResourceCreateInput, err = manager.SSharableVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.SharableVirtualResourceCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "SSharableVirtualResourceBaseManager.ValidateCreateData")
	}
	input.EncryptedResourceCreateInput, err = manager.SEncryptedResourceManager.ValidateCreateData(ctx, userCred, ownerId, query, input.EncryptedResourceCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "SEncryptedResourceManager.ValidateCreateData")
	}

	// If this image is the part of guest image (contains "guest_image_id"),
	// we do not need to check and set pending quota
	// because that pending quota has been checked and set in SGuestImage.ValidateCreateData
	if input.IsGuestImage == nil || !*input.IsGuestImage {
		pendingUsage := SQuota{Image: 1}
		keys := imageCreateInput2QuotaKeys(input.DiskFormat, ownerId)
		pendingUsage.SetKeys(keys)
		if err := quotas.CheckSetPendingQuota(ctx, userCred, &pendingUsage); err != nil {
			return input, httperrors.NewOutOfQuotaError("%s", err)
		}
	}

	return input, nil
}

func (self *SImage) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	err := self.SSharableVirtualResourceBase.CustomizeCreate(ctx, userCred, ownerId, query, data)
	if err != nil {
		return err
	}
	self.Status = api.IMAGE_STATUS_QUEUED
	self.Owner = self.ProjectId
	err = self.SEncryptedResource.CustomizeCreate(ctx, userCred, ownerId, data, "image-"+pinyinutils.Text2Pinyin(self.Name))
	if err != nil {
		return errors.Wrap(err, "SEncryptedResource.CustomizeCreate")
	}
	return nil
}

func (self *SImage) GetLocalPath(format string) string {
	path := filepath.Join(options.Options.FilesystemStoreDatadir, self.Id)
	if len(format) > 0 {
		path = fmt.Sprintf("%s.%s", path, format)
	}
	return path
}

func (self *SImage) GetPath(format string) string {
	path := filepath.Join(options.Options.FilesystemStoreDatadir, self.Id)
	if options.Options.StorageDriver == api.IMAGE_STORAGE_DRIVER_S3 {
		path = filepath.Join(options.Options.S3MountPoint, self.Id)
	}
	if len(format) > 0 {
		path = fmt.Sprintf("%s.%s", path, format)
	}
	return path
}

func (self *SImage) unprotectImage() {
	db.Update(self, func() error {
		self.Protected = tristate.False
		return nil
	})
}

func (self *SImage) OnJointFailed(ctx context.Context, userCred mcclient.TokenCredential) {
	log.Errorf("create joint of image and guest image failed")
	self.SetStatus(ctx, userCred, api.IMAGE_STATUS_KILLED, "")
	self.unprotectImage()
}

func (self *SImage) OnSaveFailed(ctx context.Context, userCred mcclient.TokenCredential, msg jsonutils.JSONObject) {
	self.saveFailed(userCred, msg)
	logclient.AddActionLogWithContext(ctx, self, logclient.ACT_IMAGE_SAVE, msg, userCred, false)
}

func (self *SImage) OnSaveTaskFailed(task taskman.ITask, userCred mcclient.TokenCredential, msg jsonutils.JSONObject) {
	self.saveFailed(userCred, msg)
	logclient.AddActionLogWithStartable(task, self, logclient.ACT_IMAGE_SAVE, msg, userCred, false)
}

func (self *SImage) OnSaveSuccess(ctx context.Context, userCred mcclient.TokenCredential, msg string) {
	self.SetStatus(ctx, userCred, api.IMAGE_STATUS_SAVED, "save success")
	self.saveSuccess(userCred, msg)
	logclient.AddActionLogWithContext(ctx, self, logclient.ACT_IMAGE_SAVE, msg, userCred, true)
}

func (self *SImage) OnSaveTaskSuccess(task taskman.ITask, userCred mcclient.TokenCredential, msg string) {
	self.SetStatus(context.Background(), userCred, api.IMAGE_STATUS_SAVED, "save success")
	self.saveSuccess(userCred, msg)
	logclient.AddActionLogWithStartable(task, self, logclient.ACT_IMAGE_SAVE, msg, userCred, true)
}

func (self *SImage) saveSuccess(userCred mcclient.TokenCredential, msg string) {
	// do not set this status, until image converting complete
	// self.SetStatus(ctx,userCred, api.IMAGE_STATUS_ACTIVE, msg)
	db.OpsLog.LogEvent(self, db.ACT_SAVE, msg, userCred)
}

func (self *SImage) saveFailed(userCred mcclient.TokenCredential, msg jsonutils.JSONObject) {
	log.Errorf("saveFailed: %s", msg.String())
	self.SetStatus(context.Background(), userCred, api.IMAGE_STATUS_KILLED, msg.String())
	self.unprotectImage()
	db.OpsLog.LogEvent(self, db.ACT_SAVE_FAIL, msg, userCred)
}

func (self *SImage) saveImageFromStream(localPath string, reader io.Reader, totalSize int64, calChecksum bool) (*streamutils.SStreamProperty, error) {
	fp, err := os.Create(localPath)
	if err != nil {
		return nil, err
	}
	defer fp.Close()
	lastSaveTime := time.Now()
	return streamutils.StreamPipe(reader, fp, calChecksum, func(saved int64) {
		now := time.Now()
		if now.Sub(lastSaveTime) > 5*time.Second {
			self.saveSize(saved, totalSize)
			lastSaveTime = now
		}
	})
}

func (self *SImage) saveSize(newSize, totalSize int64) error {
	_, err := db.Update(self, func() error {
		self.Size = newSize
		if totalSize > 0 {
			self.Progress = float32(float64(newSize) / float64(totalSize) * 100.0)
		}
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "Update size")
	}
	return nil
}

// Image always do probe and customize after save from stream
func (self *SImage) SaveImageFromStream(reader io.Reader, totalSize int64, calChecksum bool) error {
	localPath := self.GetLocalPath("")

	err := func() error {
		sp, err := self.saveImageFromStream(localPath, reader, totalSize, calChecksum)
		if err != nil {
			return errors.Wrapf(err, "saveImageFromStream")
		}

		virtualSizeBytes := int64(0)
		format := ""
		img, err := qemuimg.NewQemuImage(localPath)
		if err != nil {
			return errors.Wrapf(err, "NewQemuImage %s", localPath)
		}
		format = string(img.Format)
		virtualSizeBytes = img.SizeBytes

		var fastChksum string
		if calChecksum {
			fastChksum, err = fileutils2.FastCheckSum(localPath)
			if err != nil {
				return errors.Wrapf(err, "FastCheckSum %s", localPath)
			}
		}

		_, err = db.Update(self, func() error {
			self.Size = sp.Size
			if calChecksum {
				self.Checksum = sp.CheckSum
				self.FastHash = fastChksum
			}
			self.Location = fmt.Sprintf("%s%s", LocalFilePrefix, localPath)
			if len(format) > 0 {
				self.DiskFormat = format
			}
			if virtualSizeBytes > 0 {
				self.MinDiskMB = int32(math.Ceil(float64(virtualSizeBytes) / 1024 / 1024))
			}
			return nil
		})
		if err != nil {
			return errors.Wrapf(err, "db.Update")
		}

		return nil
	}()
	if err != nil {
		if fileutils2.IsFile(localPath) {
			if e := os.Remove(localPath); e != nil {
				log.Errorf("remove failed file %s error: %v", localPath, err)
			}
		}
	}

	return err
}

func (self *SImage) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SSharableVirtualResourceBase.PostCreate(ctx, userCred, ownerId, query, data)

	// if SImage belong to a guest image, pending quota will not be set.
	if self.IsGuestImage.IsFalse() {
		pendingUsage := SQuota{Image: 1}
		inputDiskFormat, _ := data.GetString("disk_format")
		keys := imageCreateInput2QuotaKeys(inputDiskFormat, ownerId)
		pendingUsage.SetKeys(keys)
		cancelUsage := SQuota{Image: 1}
		keys = self.GetQuotaKeys()
		cancelUsage.SetKeys(keys)
		quotas.CancelPendingUsage(ctx, userCred, &pendingUsage, &cancelUsage, true)
	}

	detectedProperties, err := ImagePropertyManager.GetProperties(self.Id)
	if err == nil {
		if osArch := detectedProperties[api.IMAGE_OS_ARCH]; strings.Contains(osArch, "aarch") {
			props, _ := data.Get("properties")
			if props != nil {
				dict := props.(*jsonutils.JSONDict)
				dict.Set(api.IMAGE_OS_ARCH, jsonutils.NewString(osArch))
			}
			db.Update(self, func() error {
				self.OsArch = apis.OS_ARCH_AARCH64
				return nil
			})
		}
	}

	if data.Contains("properties") {
		// update properties
		props, _ := data.Get("properties")
		err := ImagePropertyManager.SaveProperties(ctx, userCred, self.Id, props)
		if err != nil {
			log.Warningf("save properties error %s", err)
		}
	}

	appParams := appsrv.AppContextGetParams(ctx)
	if appParams.Request.ContentLength > 0 {
		db.OpsLog.LogEvent(self, db.ACT_SAVING, "create upload", userCred)
		self.SetStatus(ctx, userCred, api.IMAGE_STATUS_SAVING, "create upload")

		err := self.SaveImageFromStream(appParams.Request.Body, appParams.Request.ContentLength, false)
		if err != nil {
			self.OnSaveFailed(ctx, userCred, jsonutils.NewString(fmt.Sprintf("create upload fail %s", err)))
			return
		}

		self.OnSaveSuccess(ctx, userCred, "create upload success")
		self.StartImagePipeline(ctx, userCred, false)
	} else {
		copyFrom := appParams.Request.Header.Get(modules.IMAGE_META_COPY_FROM)
		compress := appParams.Request.Header.Get(modules.IMAGE_META_COMPRESS_FORMAT)
		if len(copyFrom) > 0 {
			self.startImageCopyFromUrlTask(ctx, userCred, copyFrom, compress, "")
		}
	}
}

// After image probe and customization, image size and checksum changed
// will recalculate checksum in the end
func (self *SImage) StartImagePipeline(
	ctx context.Context, userCred mcclient.TokenCredential, skipProbe bool,
) error {
	data := jsonutils.NewDict()
	if skipProbe {
		data.Set("skip_probe", jsonutils.JSONTrue)
	}
	task, err := taskman.TaskManager.NewTask(
		ctx, "ImagePipelineTask", self, userCred, data, "", "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (img *SImage) ValidateUpdateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.ImageUpdateInput,
) (api.ImageUpdateInput, error) {
	if img.Status != api.IMAGE_STATUS_QUEUED {
		if !img.CanUpdate(input) {
			return input, httperrors.NewForbiddenError("image is the part of guest imgae")
		}
		appParams := appsrv.AppContextGetParams(ctx)
		if appParams != nil && appParams.Request.ContentLength > 0 {
			return input, httperrors.NewInvalidStatusError("cannot upload in status %s", img.Status)
		}
		if input.MinDiskMB != nil && *input.MinDiskMB > 0 && img.DiskFormat != string(qemuimgfmt.ISO) {
			img, err := qemuimg.NewQemuImage(img.GetLocalLocation())
			if err != nil {
				return input, errors.Wrap(err, "open image")
			}
			virtualSizeMB := img.SizeBytes / 1024 / 1024
			if virtualSizeMB > 0 && *input.MinDiskMB < int32(virtualSizeMB) {
				return input, httperrors.NewBadRequestError("min disk size must >= %v", virtualSizeMB)
			}
		}
	} else {
		appParams := appsrv.AppContextGetParams(ctx)
		if appParams != nil {
			// always probe
			// if self.IsData.IsTrue() {
			// 	isProbe = false
			// }
			if appParams.Request.ContentLength > 0 {
				// upload image
				img.SetStatus(ctx, userCred, api.IMAGE_STATUS_SAVING, "update start upload")
				// If isProbe is true calculating checksum is not necessary wheng saving from stream,
				// otherwise, it is needed.

				err := img.SaveImageFromStream(appParams.Request.Body, appParams.Request.ContentLength, img.IsData.IsFalse())
				if err != nil {
					img.OnSaveFailed(ctx, userCred, jsonutils.NewString(fmt.Sprintf("update upload failed %s", err)))
					return input, httperrors.NewGeneralError(err)
				}
				img.OnSaveSuccess(ctx, userCred, "update upload success")
				// data.Remove("status")
				// For guest image, DoConvertAfterProbe is not necessary.
				img.StartImagePipeline(ctx, userCred, false)
			} else {
				copyFrom := appParams.Request.Header.Get(modules.IMAGE_META_COPY_FROM)
				compress := appParams.Request.Header.Get(modules.IMAGE_META_COMPRESS_FORMAT)
				if len(copyFrom) > 0 {
					err := img.startImageCopyFromUrlTask(ctx, userCred, copyFrom, compress, "")
					if err != nil {
						img.OnSaveFailed(ctx, userCred, jsonutils.NewString(fmt.Sprintf("update copy from url failed %s", err)))
						return input, httperrors.NewGeneralError(err)
					}
				}
			}
		}
	}
	var err error
	input.SharableVirtualResourceBaseUpdateInput, err = img.SSharableVirtualResourceBase.ValidateUpdateData(ctx, userCred, query, input.SharableVirtualResourceBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "SSharableVirtualResourceBase.ValidateUpdateData")
	}
	return input, nil
}

func (self *SImage) PreUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SSharableVirtualResourceBase.PreUpdate(ctx, userCred, query, data)
}

func (self *SImage) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SSharableVirtualResourceBase.PostUpdate(ctx, userCred, query, data)

	if data.Contains("properties") {
		// update properties
		props, _ := data.Get("properties")
		err := ImagePropertyManager.SaveProperties(ctx, userCred, self.Id, props)
		if err != nil {
			log.Errorf("save properties error %s", err)
		}
	}
}

func (self *SImage) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	overridePendingDelete := false
	purge := false
	if query != nil {
		overridePendingDelete = jsonutils.QueryBoolean(query, "override_pending_delete", false)
		purge = jsonutils.QueryBoolean(query, "purge", false)
	}
	if (overridePendingDelete || purge) && !db.IsAdminAllowDelete(ctx, userCred, self) {
		return false
	}
	return self.IsOwner(userCred) || db.IsAdminAllowDelete(ctx, userCred, self)
}

func (img *SImage) ValidateDeleteCondition(ctx context.Context, info jsonutils.JSONObject) error {
	if img.Protected.IsTrue() {
		return httperrors.NewForbiddenError("image is protected")
	}
	if img.IsStandard.IsTrue() {
		return httperrors.NewForbiddenError("image is standard")
	}
	guestImgCnt, err := img.getGuestImageCount()
	if err != nil {
		return errors.Wrap(err, "getGuestImageCount")
	}
	if img.IsGuestImage.IsTrue() || guestImgCnt > 0 {
		return httperrors.NewForbiddenError("image is the part of guest image")
	}
	// if self.IsShared() {
	// 	return httperrors.NewForbiddenError("image is shared")
	// }
	return img.SSharableVirtualResourceBase.ValidateDeleteCondition(ctx, nil)
}

func (self *SImage) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	log.Infof("image delete do nothing")
	return nil
}

func (self *SImage) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return self.SSharableVirtualResourceBase.Delete(ctx, userCred)
}

func (self *SImage) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	overridePendingDelete := false
	purge := false
	if query != nil {
		overridePendingDelete = jsonutils.QueryBoolean(query, "override_pending_delete", false)
		purge = jsonutils.QueryBoolean(query, "purge", false)
	}
	if utils.IsInStringArray(self.Status, []string{
		api.IMAGE_STATUS_KILLED,
		api.IMAGE_STATUS_QUEUED,
	}) {
		overridePendingDelete = true
	}
	return self.startDeleteImageTask(ctx, userCred, "", purge, overridePendingDelete)
}

func (self *SImage) startDeleteImageTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string, isPurge bool, overridePendingDelete bool) error {
	params := jsonutils.NewDict()
	if isPurge {
		params.Add(jsonutils.JSONTrue, "purge")
	}
	if overridePendingDelete {
		params.Add(jsonutils.JSONTrue, "override_pending_delete")
	}
	params.Add(jsonutils.NewString(self.Status), "image_status")

	self.SetStatus(ctx, userCred, api.IMAGE_STATUS_DEACTIVATED, "")

	task, err := taskman.TaskManager.NewTask(ctx, "ImageDeleteTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SImage) startImageCopyFromUrlTask(ctx context.Context, userCred mcclient.TokenCredential, copyFrom, compress string, parentTaskId string) error {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(copyFrom), "copy_from")
	params.Add(jsonutils.NewString(compress), "compress_format")

	msg := fmt.Sprintf("copy from url %s", copyFrom)
	if len(compress) > 0 {
		msg += " " + compress
	}
	self.SetStatus(ctx, userCred, api.IMAGE_STATUS_SAVING, msg)
	db.OpsLog.LogEvent(self, db.ACT_SAVING, msg, userCred)

	task, err := taskman.TaskManager.NewTask(ctx, "ImageCopyFromUrlTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SImage) StartImageCheckTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "ImageCheckTask", self, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SImage) StartPutImageTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "PutImageTask", self, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return err
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SImage) PerformCancelDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if self.PendingDeleted && !self.Deleted {
		err := self.DoCancelPendingDelete(ctx, userCred)
		if err != nil {
			return nil, errors.Wrap(err, "DoCancelPendingDelete")
		}
		self.RecoverUsages(ctx, userCred)
	}
	return nil, nil
}

func (manager *SImageManager) getExpiredPendingDeleteDisks() []SImage {
	deadline := time.Now().Add(time.Duration(options.Options.PendingDeleteExpireSeconds*-1) * time.Second)

	q := manager.Query()
	// those images part of guest image will be clean in GuestImageManager.CleanPendingDeleteImages
	q = q.IsTrue("pending_deleted").LT("pending_deleted_at",
		deadline).Limit(options.Options.PendingDeleteMaxCleanBatchSize).IsFalse("is_guest_image")

	disks := make([]SImage, 0)
	err := db.FetchModelObjects(ImageManager, q, &disks)
	if err != nil {
		log.Errorf("fetch disks error %s", err)
		return nil
	}

	return disks
}

func (manager *SImageManager) CleanPendingDeleteImages(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	disks := manager.getExpiredPendingDeleteDisks()
	if disks == nil {
		return
	}
	for i := 0; i < len(disks); i += 1 {
		// clean pendingdelete so that overridePendingDelete is true
		disks[i].startDeleteImageTask(ctx, userCred, "", false, true)
	}
}

func (self *SImage) DoPendingDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	err := self.SSharableVirtualResourceBase.DoPendingDelete(ctx, userCred)
	if err != nil {
		return err
	}
	_, err = db.Update(self, func() error {
		self.Status = api.IMAGE_STATUS_PENDING_DELETE
		return nil
	})
	return err
}

func (self *SImage) DoCancelPendingDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	err := self.SSharableVirtualResourceBase.DoCancelPendingDelete(ctx, userCred)
	if err != nil {
		return err
	}
	_, err = db.Update(self, func() error {
		self.Status = api.IMAGE_STATUS_ACTIVE
		return nil
	})
	return err
}

type SImageUsage struct {
	Count int64
	Size  int64
}

func (manager *SImageManager) count(ctx context.Context, scope rbacscope.TRbacScope, ownerId mcclient.IIdentityProvider, status string, isISO tristate.TriState, pendingDelete bool, guestImage tristate.TriState, policyResult rbacutils.SPolicyResult) map[string]SImageUsage {
	sq := manager.Query("id")
	sq = db.ObjectIdQueryWithPolicyResult(ctx, sq, manager, policyResult)
	switch scope {
	case rbacscope.ScopeSystem:
		// do nothing
	case rbacscope.ScopeDomain:
		sq = sq.Equals("domain_id", ownerId.GetProjectDomainId())
	case rbacscope.ScopeProject:
		sq = sq.Equals("tenant_id", ownerId.GetProjectId())
	}
	// exclude GuestImage!!!
	if guestImage.IsTrue() {
		sq = sq.IsTrue("is_guest_image")
	} else if guestImage.IsFalse() {
		sq = sq.IsFalse("is_guest_image")
	}
	if len(status) > 0 {
		sq = sq.Equals("status", status)
	}
	sq = sq.NotEquals("status", api.IMAGE_STATUS_KILLED)
	if pendingDelete {
		sq = sq.IsTrue("pending_deleted")
	} else {
		sq = sq.IsFalse("pending_deleted")
	}
	if isISO.IsTrue() {
		sq = sq.Equals("disk_format", string(qemuimgfmt.ISO))
	} else if isISO.IsFalse() {
		sq = sq.NotEquals("disk_format", string(qemuimgfmt.ISO))
	}
	cnt, _ := sq.CountWithError()

	subimages := ImageSubformatManager.Query().SubQuery()
	q := subimages.Query(subimages.Field("format"),
		sqlchemy.COUNT("count"),
		sqlchemy.SUM("size", subimages.Field("size")))
	q = q.In("image_id", sq.SubQuery())
	q = q.GroupBy(subimages.Field("format"))
	type sFormatImageUsage struct {
		Format string
		Count  int64
		Size   int64
	}
	var usages []sFormatImageUsage
	err := q.All(&usages)
	if err != nil {
		log.Errorf("query usage fail %s", err)
		return nil
	}
	ret := make(map[string]SImageUsage)
	totalSize := int64(0)
	for _, u := range usages {
		ret[u.Format] = SImageUsage{Count: u.Count, Size: u.Size}
		totalSize += u.Size
	}
	ret["total"] = SImageUsage{Count: int64(cnt), Size: totalSize}
	return ret
}

func expandUsageCount(usages map[string]int64, prefix, imgType, state string, count map[string]SImageUsage) {
	for k, u := range count {
		key := []string{}
		if len(prefix) > 0 {
			key = append(key, prefix)
		}
		key = append(key, imgType)
		if len(state) > 0 {
			key = append(key, state)
		}
		key = append(key, k)
		countKey := strings.Join(append(key, "count"), ".")
		sizeKey := strings.Join(append(key, "size"), ".")
		usages[countKey] = u.Count
		usages[sizeKey] = u.Size
	}
}

func (manager *SImageManager) Usage(ctx context.Context, scope rbacscope.TRbacScope, ownerId mcclient.IIdentityProvider, prefix string, policyResult rbacutils.SPolicyResult) map[string]int64 {
	usages := make(map[string]int64)
	count := manager.count(ctx, scope, ownerId, api.IMAGE_STATUS_ACTIVE, tristate.False, false, tristate.False, policyResult)
	expandUsageCount(usages, prefix, "img", "", count)
	count = manager.count(ctx, scope, ownerId, api.IMAGE_STATUS_ACTIVE, tristate.True, false, tristate.False, policyResult)
	expandUsageCount(usages, prefix, string(qemuimgfmt.ISO), "", count)
	count = manager.count(ctx, scope, ownerId, api.IMAGE_STATUS_ACTIVE, tristate.None, false, tristate.False, policyResult)
	expandUsageCount(usages, prefix, "imgiso", "", count)
	count = manager.count(ctx, scope, ownerId, "", tristate.False, true, tristate.False, policyResult)
	expandUsageCount(usages, prefix, "img", "pending_delete", count)
	count = manager.count(ctx, scope, ownerId, "", tristate.True, true, tristate.False, policyResult)
	expandUsageCount(usages, prefix, string(qemuimgfmt.ISO), "pending_delete", count)
	count = manager.count(ctx, scope, ownerId, "", tristate.None, true, tristate.False, policyResult)
	expandUsageCount(usages, prefix, "imgiso", "pending_delete", count)
	return usages
}

func (self *SImage) GetImageType() api.TImageType {
	if self.DiskFormat == string(qemuimgfmt.ISO) {
		return api.ImageTypeISO
	} else if self.DiskFormat == api.IMAGE_DISK_FORMAT_TGZ {
		return api.ImageTypeTarGzip
	} else {
		return api.ImageTypeTemplate
	}
}

func (self *SImage) newSubformat(ctx context.Context, format qemuimgfmt.TImageFormat, migrate bool) error {
	subformat := &SImageSubformat{}
	subformat.SetModelManager(ImageSubformatManager, subformat)

	subformat.ImageId = self.Id
	subformat.Format = string(format)

	if migrate {
		subformat.Size = self.Size
		subformat.Checksum = self.Checksum
		subformat.FastHash = self.FastHash
		// saved successfully
		subformat.Status = api.IMAGE_STATUS_ACTIVE
		subformat.Location = self.Location
	} else {
		subformat.Status = api.IMAGE_STATUS_QUEUED
	}

	subformat.TorrentStatus = api.IMAGE_STATUS_QUEUED

	err := ImageSubformatManager.TableSpec().Insert(ctx, subformat)
	if err != nil {
		log.Errorf("fail to make subformat %s: %s", format, err)
		return err
	}
	return nil
}

func (self *SImage) migrateSubImage(ctx context.Context) error {
	log.Debugf("migrateSubImage")
	if !qemuimgfmt.IsSupportedImageFormat(self.DiskFormat) {
		log.Warningf("Unsupported image format %s, no need to migrate", self.DiskFormat)
		return nil
	}

	subimg := ImageSubformatManager.FetchSubImage(self.Id, self.DiskFormat)
	if subimg != nil {
		return nil
	}

	imgInst, err := self.getQemuImage()
	if err != nil {
		return errors.Wrap(err, "getQemuImage")
	}
	if self.GetImageType() != api.ImageTypeISO && imgInst.IsSparse() && utils.IsInStringArray(self.DiskFormat, options.Options.TargetImageFormats) {
		// need to convert again
		return self.newSubformat(ctx, qemuimgfmt.String2ImageFormat(self.DiskFormat), false)
	} else {
		localPath := self.GetLocalLocation()
		if !strings.HasSuffix(localPath, fmt.Sprintf(".%s", self.DiskFormat)) {
			newLocalpath := fmt.Sprintf("%s.%s", localPath, self.DiskFormat)
			out, err := procutils.NewCommand("mv", "-f", localPath, newLocalpath).Output()
			if err != nil {
				return errors.Wrapf(err, "rename file failed %s", out)
			}
			_, err = db.Update(self, func() error {
				self.Location = self.GetNewLocation(newLocalpath)
				return nil
			})
			if err != nil {
				return err
			}
		}
		return self.newSubformat(ctx, qemuimgfmt.String2ImageFormat(self.DiskFormat), true)
	}
}

func (img *SImage) isEncrypted() bool {
	return img.DiskFormat == string(qemuimgfmt.QCOW2) && len(img.EncryptKeyId) > 0
}

func (self *SImage) makeSubImages(ctx context.Context) error {
	if self.GetImageType() == api.ImageTypeISO || self.GetImageType() == api.ImageTypeTarGzip {
		// do not convert iso
		return nil
	}
	if self.isEncrypted() {
		// do not convert encrypted qcow2
		return nil
	}
	log.Debugf("[MakeSubImages] convert image to %#v", options.Options.TargetImageFormats)
	for _, format := range options.Options.TargetImageFormats {
		if !qemuimgfmt.IsSupportedImageFormat(format) {
			continue
		}
		if format != self.DiskFormat {
			// need to create a record
			subformat := ImageSubformatManager.FetchSubImage(self.Id, format)
			if subformat == nil {
				err := self.newSubformat(ctx, qemuimgfmt.String2ImageFormat(format), false)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (self *SImage) doConvertAllSubformats() error {
	subimgs := ImageSubformatManager.GetAllSubImages(self.Id)
	for i := 0; i < len(subimgs); i += 1 {
		if subimgs[i].Status == api.IMAGE_STATUS_ACTIVE {
			continue
		}
		if !utils.IsInStringArray(subimgs[i].Format, options.Options.TargetImageFormats) {
			// cleanup
			continue
		}
		if self.DiskFormat == subimgs[i].Format {
			continue
		}
		err := subimgs[i].doConvert(self)
		if err != nil {
			return errors.Wrap(err, "")
		}
	}
	return nil
}

func (self *SImage) GetLocalLocation() string {
	if strings.HasPrefix(self.Location, api.LocalFilePrefix) {
		return self.Location[len(api.LocalFilePrefix):]
	} else if strings.HasPrefix(self.Location, api.S3Prefix) {
		return path.Join(options.Options.S3MountPoint, self.Location[len(api.S3Prefix):])
	} else {
		return ""
	}
}

func (self *SImage) GetPrefix() string {
	if strings.HasPrefix(self.Location, api.LocalFilePrefix) {
		return api.LocalFilePrefix
	} else if strings.HasPrefix(self.Location, api.S3Prefix) {
		return api.S3Prefix
	} else {
		return api.LocalFilePrefix
	}
}

func (self *SImage) GetNewLocation(newLocalPath string) string {
	if strings.HasPrefix(self.Location, api.S3Prefix) {
		return api.S3Prefix + path.Base(newLocalPath)
	} else {
		return api.LocalFilePrefix + newLocalPath
	}
}

func (self *SImage) getQemuImage() (*qemuimg.SQemuImage, error) {
	return qemuimg.NewQemuImageWithIOLevel(self.GetLocalLocation(), qemuimg.IONiceIdle)
}

func (self *SImage) StopTorrents() {
	subimgs := ImageSubformatManager.GetAllSubImages(self.Id)
	for i := 0; i < len(subimgs); i += 1 {
		subimgs[i].StopTorrent()
	}
}

func (self *SImage) seedTorrents() {
	subimgs := ImageSubformatManager.GetAllSubImages(self.Id)
	for i := 0; i < len(subimgs); i += 1 {
		subimgs[i].seedTorrent(self.Id)
	}
}

func (self *SImage) RemoveFile() error {
	filePath := self.GetLocalLocation()
	if len(filePath) == 0 {
		filePath = self.GetPath("")
	}
	if len(filePath) > 0 && fileutils2.IsFile(filePath) {
		return os.Remove(filePath)
	}
	return nil
}

func (self *SImage) Remove(ctx context.Context, userCred mcclient.TokenCredential) error {
	subimgs := ImageSubformatManager.GetAllSubImages(self.Id)
	for i := 0; i < len(subimgs); i += 1 {
		err := subimgs[i].cleanup(ctx, userCred)
		if err != nil {
			return errors.Wrapf(err, "remove subimg %s", subimgs[i].GetName())
		}
	}

	// 考虑镜像下载中断情况
	if len(self.Location) == 0 || strings.HasPrefix(self.Location, LocalFilePrefix) {
		return self.RemoveFile()
	} else {
		return RemoveImage(ctx, self.Location)
	}

}

func (manager *SImageManager) getAllAliveImages() []SImage {
	images := make([]SImage, 0)
	q := manager.Query().NotIn("status", api.ImageDeadStatus)
	err := db.FetchModelObjects(manager, q, &images)
	if err != nil {
		log.Errorf("fail to query active images %s", err)
		return nil
	}
	return images
}

func CheckImages() {
	ctx := context.WithValue(context.TODO(), "checkimage", 1)
	images := ImageManager.getAllAliveImages()
	for i := 0; i < len(images); i += 1 {
		log.Debugf("convert image subformats %s", images[i].Name)
		images[i].StartImageCheckTask(ctx, auth.AdminCredential(), "")
	}
}

func (self *SImage) GetDetailsSubformats(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	subimgs := ImageSubformatManager.GetAllSubImages(self.Id)
	ret := make([]SImageSubformatDetails, len(subimgs))
	for i := 0; i < len(subimgs); i += 1 {
		ret[i] = subimgs[i].GetDetails()
	}
	return jsonutils.Marshal(ret), nil
}

// 磁盘镜像列表
func (manager *SImageManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.ImageListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SSharableVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, query.SharableVirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SSharableVirtualResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SMultiArchResourceBaseManager.ListItemFilter(ctx, q, userCred, query.MultiArchResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SMultiArchResourceBaseManager.ListItemFilter")
	}
	if len(query.DiskFormats) > 0 {
		q = q.In("disk_format", query.DiskFormats)
	}
	if len(query.SubFormats) > 0 {
		sq := ImageSubformatManager.Query().SubQuery()
		q = q.Join(sq, sqlchemy.Equals(sq.Field("image_id"), q.Field("id"))).Filter(sqlchemy.In(sq.Field("format"), query.SubFormats))
	}
	if query.Uefi != nil && *query.Uefi {
		imagePropertyQ := ImagePropertyManager.Query().
			Equals("name", api.IMAGE_UEFI_SUPPORT).Equals("value", "true").SubQuery()
		q = q.Join(imagePropertyQ, sqlchemy.Equals(q.Field("id"), imagePropertyQ.Field("image_id")))
	}
	if query.IsStandard != nil {
		if *query.IsStandard {
			q = q.IsTrue("is_standard")
		} else {
			q = q.IsFalse("is_standard")
		}
	}
	if query.Protected != nil {
		if *query.Protected {
			q = q.IsTrue("protected")
		} else {
			q = q.IsFalse("protected")
		}
	}
	if query.IsGuestImage != nil {
		if *query.IsGuestImage {
			q = q.IsTrue("is_guest_image")
		} else {
			q = q.IsFalse("is_guest_image")
		}
	}
	if query.IsData != nil {
		if *query.IsData {
			q = q.IsTrue("is_data")
		} else {
			q = q.IsFalse("is_data")
		}
	}

	propFilter := func(nameKeys []string, vals []string, presiceMatch bool) {
		if len(vals) == 0 {
			return
		}
		propQ := ImagePropertyManager.Query().In("name", nameKeys)
		conds := make([]sqlchemy.ICondition, 0)
		for _, val := range vals {
			field := propQ.Field("value")
			if presiceMatch {
				conds = append(conds, sqlchemy.Equals(field, val))
			} else {
				conds = append(conds,
					sqlchemy.Like(field, val),
					sqlchemy.Contains(field, val))
			}
		}
		propQ.Filter(sqlchemy.OR(conds...))
		propSq := propQ.SubQuery()
		q = q.Join(propSq, sqlchemy.Equals(q.Field("id"), propSq.Field("image_id"))).Distinct()
	}
	propFilter([]string{api.IMAGE_OS_ARCH}, query.OsArchs, query.OsArchPreciseMatch)
	propFilter([]string{api.IMAGE_OS_TYPE}, query.OsTypes, query.OsTypePreciseMatch)
	propFilter([]string{api.IMAGE_OS_DISTRO, "distro"}, query.Distributions, query.DistributionPreciseMatch)

	return q, nil
}

func (manager *SImageManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.ImageListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SSharableVirtualResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.SharableVirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SSharableVirtualResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (manager *SImageManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SSharableVirtualResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

type sUnactiveReason int

const (
	FileNoExists sUnactiveReason = iota
	FileSizeMismatch
	FileChecksumMismatch
	Others
)

func isActive(localPath string, size int64, chksum string, fastHash string, useFastHash bool, noChecksum bool) (bool, sUnactiveReason) {
	if len(localPath) == 0 || !fileutils2.Exists(localPath) {
		log.Errorf("invalid file: %s", localPath)
		return false, FileNoExists
	}
	if size != fileutils2.FileSize(localPath) {
		log.Errorf("size mistmatch: %s", localPath)
		return false, FileSizeMismatch
	}
	if len(chksum) == 0 || len(fastHash) == 0 || noChecksum {
		return true, Others
	}
	if useFastHash && len(fastHash) > 0 {
		fhash, err := fileutils2.FastCheckSum(localPath)
		if err != nil {
			log.Errorf("IsActive fastChecksum fail %s for %s", err, localPath)
			return false, Others
		}
		if fastHash != fhash {
			log.Errorf("IsActive fastChecksum mismatch for %s", localPath)
			return false, FileChecksumMismatch
		}
	} else {
		md5sum, err := fileutils2.MD5(localPath)
		if err != nil {
			log.Errorf("IsActive md5 fail %s for %s", err, localPath)
			return false, Others
		}
		if chksum != md5sum {
			log.Errorf("IsActive checksum mismatch: %s", localPath)
			return false, FileChecksumMismatch
		}
	}
	return true, Others
}

func (self *SImage) IsIso() bool {
	return self.DiskFormat == string(api.ImageTypeISO)
}

func (self *SImage) isActive(useFast bool, noChecksum bool) bool {
	active, reason := isActive(self.GetLocalLocation(), self.Size, self.Checksum, self.FastHash, useFast, noChecksum)
	if active || reason != FileChecksumMismatch {
		return active
	}
	data := jsonutils.NewDict()
	data.Set("name", jsonutils.NewString(self.Name))
	notifyclient.SystemExceptionNotifyWithResult(context.TODO(), noapi.ActionChecksumTest, noapi.TOPIC_RESOURCE_IMAGE, noapi.ResultFailed, data)
	return false
}

func (self *SImage) DoCheckStatus(ctx context.Context, userCred mcclient.TokenCredential, useFast bool) {
	if utils.IsInStringArray(self.Status, api.ImageDeadStatus) {
		return
	}
	if IsCheckStatusEnabled(self) {
		if self.isActive(useFast, true) {
			if self.Status != api.IMAGE_STATUS_ACTIVE {
				self.SetStatus(ctx, userCred, api.IMAGE_STATUS_ACTIVE, "check active")
			}
			if len(self.FastHash) == 0 {
				fastHash, err := fileutils2.FastCheckSum(self.GetLocalLocation())
				if err != nil {
					log.Errorf("DoCheckStatus fileutils2.FastChecksum fail %s", err)
				} else {
					_, err := db.Update(self, func() error {
						self.FastHash = fastHash
						return nil
					})
					if err != nil {
						log.Errorf("DoCheckStatus save FastHash fail %s", err)
					}
				}
			}
			img, err := qemuimg.NewQemuImage(self.GetLocalLocation())
			if err == nil {
				format := string(img.Format)
				virtualSizeMB := int32(img.SizeBytes / 1024 / 1024)
				if (len(format) > 0 && self.DiskFormat != format) || (virtualSizeMB > 0 && self.MinDiskMB != virtualSizeMB) {
					db.Update(self, func() error {
						if len(format) > 0 {
							self.DiskFormat = format
						}
						if virtualSizeMB > 0 && self.MinDiskMB < virtualSizeMB {
							self.MinDiskMB = virtualSizeMB
						}
						return nil
					})
				}
			} else {
				log.Warningf("fail to check image size of %s(%s)", self.Id, self.Name)
			}
		} else {
			if self.Status != api.IMAGE_STATUS_QUEUED {
				self.SetStatus(ctx, userCred, api.IMAGE_STATUS_QUEUED, "check inactive")
			}
		}
	}

	if self.Status == api.IMAGE_STATUS_ACTIVE {
		self.StartImagePipeline(ctx, userCred, true)
	}
}

func (self *SImage) PerformMarkStandard(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data jsonutils.JSONObject,
) (jsonutils.JSONObject, error) {
	if self.IsGuestImage.IsTrue() {
		return nil, errors.Wrap(httperrors.ErrForbidden, "cannot mark standard to a guest image")
	}
	isStandard := jsonutils.QueryBoolean(data, "is_standard", false)
	if !self.IsStandard.IsTrue() && isStandard {
		input := apis.PerformPublicProjectInput{}
		input.Scope = "system"
		_, err := self.PerformPublic(ctx, userCred, query, input)
		if err != nil {
			return nil, errors.Wrap(err, "PerformPublic")
		}
		diff, err := db.Update(self, func() error {
			self.IsStandard = tristate.True
			return nil
		})
		if err != nil {
			return nil, httperrors.NewGeneralError(err)
		}
		db.OpsLog.LogEvent(self, db.ACT_UPDATE, diff, userCred)
	} else if self.IsStandard.IsTrue() && !isStandard {
		diff, err := db.Update(self, func() error {
			self.IsStandard = tristate.False
			return nil
		})
		if err != nil {
			return nil, httperrors.NewGeneralError(err)
		}
		db.OpsLog.LogEvent(self, db.ACT_UPDATE, diff, userCred)
	}
	return nil, nil
}

func (self *SImage) PerformUpdateTorrentStatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	formatStr, _ := query.GetString("format")
	if len(formatStr) == 0 {
		return nil, httperrors.NewMissingParameterError("format")
	}
	subimg := ImageSubformatManager.FetchSubImage(self.Id, formatStr)
	if subimg == nil {
		return nil, httperrors.NewResourceNotFoundError("format %s not found", formatStr)
	}
	subimg.SetStatusSeeding(true)
	return nil, nil
}

func (self *SImage) CanUpdate(input api.ImageUpdateInput) bool {
	dict := jsonutils.Marshal(input).(*jsonutils.JSONDict)
	// Only allow update description for now when Image is part of guest image
	return self.IsGuestImage.IsFalse() || (dict.Length() == 1 && dict.Contains("description"))
}

func (img *SImage) GetQuotaKeys() quotas.IQuotaKeys {
	keys := SImageQuotaKeys{}
	keys.SBaseProjectQuotaKeys = quotas.OwnerIdProjectQuotaKeys(rbacscope.ScopeProject, img.GetOwnerId())
	if img.GetImageType() == api.ImageTypeISO {
		keys.Type = string(api.ImageTypeISO)
	} else {
		keys.Type = string(api.ImageTypeTemplate)
	}
	return keys
}

func imageCreateInput2QuotaKeys(format string, ownerId mcclient.IIdentityProvider) quotas.IQuotaKeys {
	keys := SImageQuotaKeys{}
	keys.SBaseProjectQuotaKeys = quotas.OwnerIdProjectQuotaKeys(rbacscope.ScopeProject, ownerId)
	if format == string(api.ImageTypeISO) {
		keys.Type = string(api.ImageTypeISO)
	} else if len(format) > 0 {
		keys.Type = string(api.ImageTypeTemplate)
	}
	return keys
}

func (img *SImage) GetUsages() []db.IUsage {
	if img.PendingDeleted || img.Deleted {
		return nil
	}
	usage := SQuota{Image: 1}
	keys := img.GetQuotaKeys()
	usage.SetKeys(keys)
	return []db.IUsage{
		&usage,
	}
}

func (img *SImage) PerformUpdateStatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.ImageUpdateStatusInput) (jsonutils.JSONObject, error) {
	if !utils.IsInStringArray(input.Status, api.ImageDeadStatus) {
		return nil, httperrors.NewBadRequestError("can't udpate image to status %s, must in %v", input.Status, api.ImageDeadStatus)
	}
	_, err := db.Update(img, func() error {
		img.Status = input.Status
		return nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "udpate image status")
	}
	db.OpsLog.LogEvent(img, db.ACT_UPDATE_STATUS, input.Reason, userCred)
	logclient.AddSimpleActionLog(img, logclient.ACT_UPDATE_STATUS, input.Reason, userCred, true)
	return nil, nil
}

func (img *SImage) PerformSetClassMetadata(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformSetClassMetadataInput) (jsonutils.JSONObject, error) {
	ret, err := img.SStandaloneAnonResourceBase.PerformSetClassMetadata(ctx, userCred, query, input)
	if err != nil {
		return ret, err
	}
	task, err := taskman.TaskManager.NewTask(ctx, "ImageSyncClassMetadataTask", img, userCred, nil, "", "", nil)
	if err != nil {
		return nil, err
	} else {
		task.ScheduleRun(nil)
	}
	return nil, nil
}

func (img *SImage) PerformPublic(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformPublicProjectInput) (jsonutils.JSONObject, error) {
	if img.IsGuestImage.IsTrue() {
		return nil, errors.Wrap(httperrors.ErrForbidden, "cannot perform public for guest image")
	}
	if img.EncryptStatus != api.IMAGE_ENCRYPT_STATUS_UNENCRYPTED {
		return nil, errors.Wrap(httperrors.ErrForbidden, "cannot perform public for encrypted image")
	}
	return img.performPublic(ctx, userCred, query, input)
}

func (img *SImage) performPublic(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformPublicProjectInput) (jsonutils.JSONObject, error) {
	if img.IsStandard.IsTrue() {
		return nil, errors.Wrap(httperrors.ErrForbidden, "cannot perform public for standard image")
	}
	return img.SSharableVirtualResourceBase.PerformPublic(ctx, userCred, query, input)
}

func (img *SImage) performPrivate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformPrivateInput) (jsonutils.JSONObject, error) {
	if img.IsStandard.IsTrue() {
		return nil, errors.Wrap(httperrors.ErrForbidden, "cannot perform private for standard image")
	}
	return img.SSharableVirtualResourceBase.PerformPrivate(ctx, userCred, query, input)
}

func (img *SImage) PerformPrivate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformPrivateInput) (jsonutils.JSONObject, error) {
	if img.IsGuestImage.IsTrue() {
		return nil, errors.Wrap(httperrors.ErrForbidden, "cannot perform private for guest image")
	}
	return img.performPrivate(ctx, userCred, query, input)
}

func (img *SImage) PerformProbe(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.PerformProbeInput) (jsonutils.JSONObject, error) {
	if img.Status != api.IMAGE_STATUS_ACTIVE && img.Status != api.IMAGE_STATUS_SAVED {
		return nil, httperrors.NewInvalidStatusError("cannot probe in status %s", img.Status)
	}
	img.SetStatus(ctx, userCred, api.IMAGE_STATUS_PROBING, "perform probe")
	err := img.StartImagePipeline(ctx, userCred, false)
	if err != nil {
		return nil, errors.Wrap(err, "ImageProbeAndCustomization")
	}
	return nil, nil
}

func (img *SImage) PerformChangeOwner(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformChangeProjectOwnerInput) (jsonutils.JSONObject, error) {
	ret, err := img.SVirtualResourceBase.PerformChangeOwner(ctx, userCred, query, input)
	if err != nil {
		return nil, err
	}
	task, err := taskman.TaskManager.NewTask(ctx, "ImageSyncClassMetadataTask", img, userCred, nil, "", "", nil)
	if err != nil {
		return nil, err
	} else {
		task.ScheduleRun(nil)
	}
	return ret, nil
}

func UpdateImageConfigTargetImageFormats(ctx context.Context, userCred mcclient.TokenCredential) error {
	s := auth.GetSession(ctx, userCred, options.Options.Region)
	serviceId, err := common_options.GetServiceIdByType(s, api.SERVICE_TYPE, "")
	if err != nil {
		return errors.Wrap(err, "get service id")
	}

	defConf, err := common_options.GetServiceConfig(s, serviceId)
	if err != nil {
		return errors.Wrap(err, "GetServiceConfig")
	}
	targetFormats := make([]string, 0)
	err = defConf.Unmarshal(&targetFormats, "target_image_formats")
	if err != nil {
		return errors.Wrap(err, "get target_image_formats")
	}
	if !utils.IsInStringArray(string(qemuimgfmt.VMDK), targetFormats) {
		targetFormats = append(targetFormats, string(qemuimgfmt.VMDK))
	}
	defConfDict := defConf.(*jsonutils.JSONDict)
	defConfDict.Set("target_image_formats", jsonutils.NewStringArray(targetFormats))
	nconf := jsonutils.NewDict()
	nconf.Add(defConfDict, "config", "default")
	_, err = identity_modules.ServicesV3.PerformAction(s, serviceId, "config", nconf)
	if err != nil {
		return errors.Wrap(err, "fail to save config")
	}
	return nil
}

func (m *SImageManager) PerformVmwareAccountAdded(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input apis.PerformChangeProjectOwnerInput) (jsonutils.JSONObject, error) {
	log.Infof("perform vmware account added")

	if !utils.IsInStringArray(string(qemuimgfmt.VMDK), options.Options.TargetImageFormats) {
		if err := UpdateImageConfigTargetImageFormats(ctx, userCred); err != nil {
			log.Errorf("failed update target_image_formats %s", err)
		} else {
			options.Options.TargetImageFormats = append(options.Options.TargetImageFormats, string(qemuimgfmt.VMDK))
		}
	}
	return nil, nil
}

/*func (image *SImage) getRealPath() string {
	diskPath := image.GetPath("")
	if !fileutils2.Exists(diskPath) {
		diskPath = image.GetPath(image.DiskFormat)
		if !fileutils2.Exists(diskPath) {
			return ""
		}
	}
	return diskPath
}*/

func (image *SImage) doProbeImageInfo(ctx context.Context, userCred mcclient.TokenCredential) (bool, error) {
	if image.IsIso() {
		// no need to probe
		return false, nil
	}
	if image.IsData.IsTrue() {
		// no need to probe
		return false, nil
	}
	diskPath := image.GetLocalLocation()
	if len(diskPath) == 0 {
		return false, errors.Wrap(httperrors.ErrNotFound, "disk file not found")
	}
	if deployclient.GetDeployClient() == nil {
		return false, fmt.Errorf("deploy client not init")
	}
	diskInfo := &deployapi.DiskInfo{
		Path: diskPath,
	}
	if image.IsEncrypted() {
		key, err := image.GetEncryptInfo(ctx, userCred)
		if err != nil {
			return false, errors.Wrap(err, "GetEncryptInfo")
		}
		diskInfo.EncryptPassword = key.Key
		diskInfo.EncryptAlg = string(key.Alg)
	}
	imageInfo, err := deployclient.GetDeployClient().ProbeImageInfo(ctx, &deployapi.ProbeImageInfoPramas{DiskInfo: diskInfo})
	if err != nil {
		return false, errors.Wrap(err, "ProbeImageInfo")
	}
	log.Infof("image probe info: %s", jsonutils.Marshal(imageInfo))
	err = image.updateImageInfo(ctx, userCred, imageInfo)
	if err != nil {
		return false, errors.Wrap(err, "updateImageInfo")
	}
	return true, nil
}

func (image *SImage) updateImageInfo(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	imageInfo *deployapi.ImageInfo,
) error {
	if imageInfo.OsInfo == nil || gotypes.IsNil(imageInfo.OsInfo) {
		log.Warningln("imageInfo.OsInfo is empty!")
		return nil
	}

	db.Update(image, func() error {
		image.OsArch = imageInfo.OsInfo.Arch
		return nil
	})

	imageProperties := jsonutils.Marshal(imageInfo.OsInfo).(*jsonutils.JSONDict)

	imageProperties.Set(api.IMAGE_OS_ARCH, jsonutils.NewString(imageInfo.OsInfo.Arch))
	imageProperties.Set("os_version", jsonutils.NewString(imageInfo.OsInfo.Version))
	imageProperties.Set("os_distribution", jsonutils.NewString(imageInfo.OsInfo.Distro))
	imageProperties.Set("os_language", jsonutils.NewString(imageInfo.OsInfo.Language))

	imageProperties.Set(api.IMAGE_OS_TYPE, jsonutils.NewString(imageInfo.OsType))
	imageProperties.Set(api.IMAGE_PARTITION_TYPE, jsonutils.NewString(imageInfo.PhysicalPartitionType))
	if imageInfo.IsUefiSupport {
		imageProperties.Set(api.IMAGE_UEFI_SUPPORT, jsonutils.JSONTrue)
	} else {
		imageProperties.Set(api.IMAGE_UEFI_SUPPORT, jsonutils.JSONFalse)
	}
	if imageInfo.IsLvmPartition {
		imageProperties.Set(api.IMAGE_IS_LVM_PARTITION, jsonutils.JSONTrue)
	} else {
		imageProperties.Set(api.IMAGE_IS_LVM_PARTITION, jsonutils.JSONFalse)
	}
	if imageInfo.IsReadonly {
		imageProperties.Set(api.IMAGE_IS_READONLY, jsonutils.JSONTrue)
	} else {
		imageProperties.Set(api.IMAGE_IS_READONLY, jsonutils.JSONFalse)
	}
	if imageInfo.IsInstalledCloudInit {
		imageProperties.Set(api.IMAGE_INSTALLED_CLOUDINIT, jsonutils.JSONTrue)
	} else {
		imageProperties.Set(api.IMAGE_INSTALLED_CLOUDINIT, jsonutils.JSONFalse)
	}
	return ImagePropertyManager.SaveProperties(ctx, userCred, image.Id, imageProperties)
}

func (image *SImage) updateChecksum() error {
	imagePath := image.GetLocalLocation()
	if len(imagePath) == 0 {
		return errors.Wrapf(httperrors.ErrNotFound, "image file %s not found", image.Location)
	}
	fp, err := os.Open(imagePath)
	if err != nil {
		return err
	}
	defer fp.Close()

	stat, err := fp.Stat()
	if err != nil {
		return errors.Wrapf(err, "stat %s", imagePath)
	}

	chksum, err := fileutils2.MD5(imagePath)
	if err != nil {
		return errors.Wrapf(err, "md5 %s", imagePath)
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
		return errors.Wrap(err, "Update image")
	}

	// also update the corresponding subformats
	subimg := ImageSubformatManager.FetchSubImage(image.Id, image.DiskFormat)
	if subimg != nil {
		_, err = db.Update(subimg, func() error {
			subimg.Size = stat.Size()
			subimg.Checksum = chksum
			subimg.FastHash = fastchksum
			return nil
		})
		if err != nil {
			return errors.Wrap(err, "Update subformat")
		}
	}

	return nil
}

func (image *SImage) isLocal() bool {
	return strings.HasPrefix(image.Location, LocalFilePrefix)
}

func (image *SImage) doUploadPermanentStorage(ctx context.Context, userCred mcclient.TokenCredential) (bool, error) {
	uploaded := false
	if image.isLocal() {
		imagePath := image.GetLocalLocation()
		image.SetStatus(ctx, userCred, api.IMAGE_STATUS_SAVING, "save image to specific storage")
		storage := GetStorage()
		location, err := storage.SaveImage(ctx, imagePath, nil)
		if err != nil {
			log.Errorf("Failed save image to specific storage %s", err)
			errStr := fmt.Sprintf("save image to storage %s: %v", storage.Type(), err)
			image.SetStatus(ctx, userCred, api.IMAGE_STATUS_SAVE_FAIL, errStr)
			return false, errors.Wrapf(err, "save image to storage %s", storage.Type())
		}
		if location != image.Location {
			uploaded = true
			// save success! to update the location
			_, err = db.Update(image, func() error {
				image.Location = location
				return nil
			})
			if err != nil {
				log.Errorf("failed update image location %s", err)
				return false, errors.Wrap(err, "update image location")
			}
			// update location success, remove local copy
			if err = procutils.NewCommand("rm", "-f", imagePath).Run(); err != nil {
				log.Errorf("failed remove file %s: %s", imagePath, err)
			}
		}
		image.SetStatus(ctx, userCred, api.IMAGE_STATUS_ACTIVE, "save image to specific storage complete")
	}

	subimgs := ImageSubformatManager.GetAllSubImages(image.Id)
	for i := 0; i < len(subimgs); i++ {
		if !subimgs[i].isLocal() {
			continue
		}
		if subimgs[i].Format == image.DiskFormat {
			_, err := db.Update(&subimgs[i], func() error {
				subimgs[i].Location = image.Location
				subimgs[i].Status = api.IMAGE_STATUS_ACTIVE
				return nil
			})
			if err != nil {
				log.Errorf("failed update subimg %s", err)
			}
		} else {
			imagePath := subimgs[i].GetLocalLocation()
			storage := GetStorage()
			location, err := GetStorage().SaveImage(ctx, imagePath, nil)
			if err != nil {
				log.Errorf("Failed save image to sepcific storage %s", err)
				subimgs[i].SetStatus(api.IMAGE_STATUS_SAVE_FAIL)
				return false, errors.Wrapf(err, "save sub image %s to storage %s", subimgs[i].Format, storage.Type())
			} else if subimgs[i].Location != location {
				uploaded = true
				_, err := db.Update(&subimgs[i], func() error {
					subimgs[i].Location = location
					return nil
				})
				if err != nil {
					log.Errorf("failed update subimg %s", err)
				}
				if err = procutils.NewCommand("rm", "-f", imagePath).Run(); err != nil {
					log.Errorf("failed remove file %s: %s", imagePath, err)
				}
			}
			db.Update(&subimgs[i], func() error {
				subimgs[i].Status = api.IMAGE_STATUS_ACTIVE
				return nil
			})
		}
	}
	return uploaded, nil
}

func (img *SImage) doConvert(ctx context.Context, userCred mcclient.TokenCredential) (bool, error) {
	if img.IsGuestImage.IsTrue() {
		// for image the part of a guest image, convert is not necessary.
		return false, nil
	}
	needConvert := false
	subimgs := ImageSubformatManager.GetAllSubImages(img.Id)
	if len(subimgs) == 0 {
		needConvert = true
	} else {
		supportedFormats := make([]string, 0)
		for i := 0; i < len(subimgs); i += 1 {
			if !utils.IsInStringArray(subimgs[i].Format, options.Options.TargetImageFormats) && subimgs[i].Format != img.DiskFormat {
				// no need to have this subformat
				err := subimgs[i].cleanup(ctx, userCred)
				if err != nil {
					return false, errors.Wrap(err, "cleanup sub image")
				}
				continue
			}
			subimgs[i].checkStatus(true, false)
			if subimgs[i].Status != api.IMAGE_STATUS_ACTIVE {
				needConvert = true
			}
			supportedFormats = append(supportedFormats, subimgs[i].Format)
		}
		if len(supportedFormats) < len(options.Options.TargetImageFormats) {
			needConvert = true
		}
	}
	log.Debugf("doConvert imageStatus %s %v", img.Status, needConvert)
	if (img.Status == api.IMAGE_STATUS_SAVED || img.Status == api.IMAGE_STATUS_ACTIVE || img.Status == api.IMAGE_STATUS_PROBING) && needConvert {
		err := img.migrateSubImage(ctx)
		if err != nil {
			return false, errors.Wrap(err, "migrateSubImage")
		}
		err = img.makeSubImages(ctx)
		if err != nil {
			return false, errors.Wrap(err, "makeSubImages")
		}
		err = img.doConvertAllSubformats()
		if err != nil {
			return false, errors.Wrap(err, "doConvertAllSubformats")
		}
	}
	return needConvert, nil
}

func (img *SImage) doEncrypt(ctx context.Context, userCred mcclient.TokenCredential) (bool, error) {
	if len(img.EncryptKeyId) == 0 {
		return false, nil
	}
	if img.DiskFormat != string(qemuimgfmt.QCOW2) {
		// only qcow2 support encryption
		return false, nil
	}
	if img.EncryptStatus == api.IMAGE_ENCRYPT_STATUS_ENCRYPTED {
		return false, nil
	}
	session := auth.GetSession(ctx, userCred, options.Options.Region)
	keyObj, err := identity_modules.Credentials.GetById(session, img.EncryptKeyId, nil)
	if err != nil {
		return false, errors.Wrap(err, "GetByEncryptKeyId")
	}
	key, err := identity_modules.DecodeEncryptKey(keyObj)
	if err != nil {
		return false, errors.Wrap(err, "DecodeEncryptKey")
	}
	qemuImg, err := img.getQemuImage()
	if err != nil {
		return false, errors.Wrap(err, "getQemuImage")
	}
	var failMsg string
	if qemuImg.Encrypted {
		qemuImg.SetPassword(key.Key)
		err = qemuImg.Check()
		if err != nil {
			failMsg = "check encrypted image fail"
		}
	} else {
		img.setEncryptStatus(userCred, api.IMAGE_ENCRYPT_STATUS_ENCRYPTING, db.ACT_ENCRYPT_START, "start encrypt")
		err = qemuImg.Convert2Qcow2(true, key.Key, qemuimg.EncryptFormatLuks, key.Alg)
		if err != nil {
			failMsg = "Convert1Qcow2 fail"
		}
	}
	if err != nil {
		img.setEncryptStatus(userCred, api.IMAGE_ENCRYPT_STATUS_UNENCRYPTED, db.ACT_ENCRYPT_FAIL, fmt.Sprintf("%s %s", failMsg, err))
		return false, errors.Wrap(err, failMsg)
	}
	img.setEncryptStatus(userCred, api.IMAGE_ENCRYPT_STATUS_ENCRYPTED, db.ACT_ENCRYPT_DONE, "success")
	return true, nil
}

func (img *SImage) setEncryptStatus(userCred mcclient.TokenCredential, status string, event string, reason string) error {
	_, err := db.Update(img, func() error {
		img.EncryptStatus = status
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "update encrypted")
	}
	db.OpsLog.LogEvent(img, event, reason, userCred)
	switch event {
	case db.ACT_ENCRYPT_FAIL:
		logclient.AddSimpleActionLog(img, logclient.ACT_ENCRYPTION, reason, userCred, false)
	case db.ACT_ENCRYPT_DONE:
		logclient.AddSimpleActionLog(img, logclient.ACT_ENCRYPTION, reason, userCred, true)
	}
	return nil
}

func (img *SImage) Pipeline(ctx context.Context, userCred mcclient.TokenCredential, skipProbe bool) error {
	updated := false
	needChecksum := false
	// do probe
	if !skipProbe {
		alterd, err := img.doProbeImageInfo(ctx, userCred)
		if err != nil {
			log.Errorf("fail to doProbeImageInfo %s", err)
		}
		if alterd {
			needChecksum = true
		}
	} else {
		log.Debugf("skipProbe image...")
	}
	// do encrypt
	{
		altered, err := img.doEncrypt(ctx, userCred)
		if err != nil {
			return errors.Wrap(err, "doEncrypt")
		}
		if altered {
			needChecksum = true
		}
	}
	if needChecksum || len(img.Checksum) == 0 || len(img.FastHash) == 0 {
		err := img.updateChecksum()
		if err != nil {
			return errors.Wrap(err, "updateChecksum")
		}
	}
	{
		// do conert
		converted, err := img.doConvert(ctx, userCred)
		if err != nil {
			return errors.Wrap(err, "doConvert")
		}
		if converted {
			updated = true
		}
	}
	{
		// do doUploadPermanent
		uploaded, err := img.doUploadPermanentStorage(ctx, userCred)
		if err != nil {
			return errors.Wrap(err, "doUploadPermanentStorage")
		}
		if uploaded {
			updated = true
		}
	}
	if img.Status != api.IMAGE_STATUS_ACTIVE {
		img.SetStatus(ctx, userCred, api.IMAGE_STATUS_ACTIVE, "image pipeline complete")
	}
	if updated && img.IsGuestImage.IsFalse() {
		kwargs := jsonutils.NewDict()
		kwargs.Set("name", jsonutils.NewString(img.GetName()))
		osType, err := ImagePropertyManager.GetProperty(img.Id, api.IMAGE_OS_TYPE)
		if err == nil {
			kwargs.Set("os_type", jsonutils.NewString(osType.Value))
		}
		notifyclient.SystemNotifyWithCtx(ctx, notify.NotifyPriorityNormal, notifyclient.IMAGE_ACTIVED, kwargs)
		notifyclient.NotifyImportantWithCtx(ctx, []string{userCred.GetUserId()}, false, notifyclient.IMAGE_ACTIVED, kwargs)
	}
	return nil
}

func (img *SImage) getGuestImageCount() (int, error) {
	gis, err := GuestImageJointManager.GetByImageId(img.Id)
	if err != nil {
		return -1, errors.Wrap(err, "GuestImageJointManager.GetByImageId")
	}
	return len(gis), nil
}

func (img *SImage) markDataImage(userCred mcclient.TokenCredential) error {
	if img.IsData.IsTrue() {
		return nil
	}
	diff, err := db.Update(img, func() error {
		img.IsData = tristate.True
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "update")
	}
	db.OpsLog.LogEvent(img, db.ACT_UPDATE, diff, userCred)
	return nil
}
