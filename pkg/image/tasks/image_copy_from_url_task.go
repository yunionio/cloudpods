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

package tasks

import (
	"context"
	"net/http"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/httputils"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/image/models"
	"yunion.io/x/onecloud/pkg/image/options"
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
		client := httputils.GetTimeoutClient(0)
		transport := httputils.GetTransport(true)
		transport.Proxy = options.Options.HttpTransportProxyFunc()
		client.Transport = transport
		resp, err := httputils.Request(client, ctx, httputils.GET, copyFrom, header, nil, false)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		err = image.SaveImageFromStream(resp.Body, resp.ContentLength, false)
		if err != nil {
			return nil, err
		}
		md5 := resp.Header.Get("x-oss-meta-yunion-os-checksum")
		if len(md5) > 0 {
			db.Update(image, func() error {
				image.OssChecksum = md5
				return nil
			})
		}
		return nil, nil
	})
}

func (self *ImageCopyFromUrlTask) OnImageImportComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	image := obj.(*models.SImage)
	image.OnSaveTaskSuccess(self, self.UserCred, "create upload success")
	image.StartImagePipeline(ctx, self.UserCred, false)
	self.SetStageComplete(ctx, nil)
}

func (self *ImageCopyFromUrlTask) OnImageImportCompleteFailed(ctx context.Context, obj db.IStandaloneModel, err jsonutils.JSONObject) {
	image := obj.(*models.SImage)
	copyFrom, _ := self.Params.GetString("copy_from")
	msg := jsonutils.NewDict()
	msg.Add(err, "reason")
	msg.Add(jsonutils.NewString(copyFrom), "copy_from")
	image.OnSaveTaskFailed(self, self.UserCred, msg)
	self.SetStageFailed(ctx, msg)
}
