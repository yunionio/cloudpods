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

package ssl_certificate

import (
	"context"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type SSLCertificateDeleteTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(SSLCertificateDeleteTask{})
}

func (self *SSLCertificateDeleteTask) taskFailed(ctx context.Context, cert *models.SSSLCertificate, err error) {
	cert.SetStatus(ctx, self.UserCred, apis.STATUS_DELETE_FAILED, err.Error())
	logclient.AddActionLogWithStartable(self, cert, logclient.ACT_DELOCATE, err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *SSLCertificateDeleteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	cert := obj.(*models.SSSLCertificate)

	iCert, err := cert.GetICloudSSLCertificate(ctx)
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotFound {
			self.taskComplete(ctx, cert)
			return
		}
		self.taskFailed(ctx, cert, errors.Wrapf(err, "cert.GetICloudSSLCertificate"))
		return
	}

	err = iCert.Delete()
	if err != nil {
		self.taskFailed(ctx, cert, errors.Wrapf(err, "iCert.Delete"))
		return
	}

	self.taskComplete(ctx, cert)
}

func (self *SSLCertificateDeleteTask) taskComplete(ctx context.Context, cert *models.SSSLCertificate) {
	cert.RealDelete(ctx, self.GetUserCred())
	notifyclient.EventNotify(ctx, self.UserCred, notifyclient.SEventNotifyParam{
		Obj:    cert,
		Action: notifyclient.ActionDelete,
	})
	self.SetStageComplete(ctx, nil)
}
