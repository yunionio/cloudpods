package tasks

import (
	"context"
	"fmt"
	"net/http"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/image/models"
	"yunion.io/x/onecloud/pkg/util/httputils"
)

type ImageCopyFromUrlTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(ImageCopyFromUrlTask{})
}

func (self *ImageCopyFromUrlTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	image := obj.(*models.SImage)

	copyFrom, _ := self.Params.GetString("copy_from")

	log.Infof("Copy image from %s", copyFrom)

	self.SetStage("OnImageImportComplete", nil)
	taskman.LocalTaskRun(self, func() (jsonutils.JSONObject, error) {
		header := http.Header{}
		resp, err := httputils.Request(nil, ctx, httputils.GET, copyFrom, header, nil, false)
		if err != nil {
			return nil, err
		}
		err = image.SaveImageFromStream(resp.Body)
		if err != nil {
			return nil, err
		}
		return nil, nil
	})
}

func (self *ImageCopyFromUrlTask) OnImageImportComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	image := obj.(*models.SImage)
	image.OnSaveTaskSuccess(self, self.UserCred, "create upload success")
	image.StartImageConvertTask(ctx, self.UserCred, "")
	self.SetStageComplete(ctx, nil)
}

func (self *ImageCopyFromUrlTask) OnImageImportCompleteFailed(ctx context.Context, obj db.IStandaloneModel, err jsonutils.JSONObject) {
	image := obj.(*models.SImage)
	copyFrom, _ := self.Params.GetString("copy_from")
	msg := fmt.Sprintf("copy from url %s request fail %s", copyFrom, err)
	image.OnSaveTaskFailed(self, self.UserCred, msg)
	self.SetStageFailed(ctx, msg)
}
