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
	"time"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type DBInstancePublicConnectionTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(DBInstancePublicConnectionTask{})
}

func (self *DBInstancePublicConnectionTask) getAction() string {
	act := logclient.ACT_OPEN_PUBLIC_CONNECTION
	if isOpen, _ := self.GetParams().Bool("open"); !isOpen {
		act = logclient.ACT_CLOSE_PUBLIC_CONNECTION
	}
	return act
}

func (self *DBInstancePublicConnectionTask) taskFailed(ctx context.Context, dbinstance *models.SDBInstance, err error) {
	dbinstance.SetStatus(ctx, self.UserCred, api.DBINSTANCE_FAILE, err.Error())
	logclient.AddActionLogWithStartable(self, dbinstance, self.getAction(), err, self.UserCred, false)
	self.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
}

func (self *DBInstancePublicConnectionTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	dbinstance := obj.(*models.SDBInstance)
	self.DBInstancePublicConnectionOperation(ctx, dbinstance)
}

func (self *DBInstancePublicConnectionTask) DBInstancePublicConnectionOperation(ctx context.Context, instance *models.SDBInstance) {
	idbinstance, err := instance.GetIDBInstance(ctx)
	if err != nil {
		self.taskFailed(ctx, instance, errors.Wrap(err, "instance.GetIDBInstance"))
		return
	}

	isOpen := jsonutils.QueryBoolean(self.GetParams(), "open", false)

	if isOpen {
		err = idbinstance.OpenPublicConnection()
	} else {
		err = idbinstance.ClosePublicConnection()
	}
	if err != nil {
		self.taskFailed(ctx, instance, errors.Wrap(err, "idbinstance.OpenPublicConnection || idbinstance.ClosePublicConnection"))
		return
	}

	connectionStr := ""

	err = cloudprovider.Wait(time.Second*10, time.Minute*5, func() (bool, error) {
		iRds, err := instance.GetIDBInstance(ctx)
		if err != nil {
			return false, errors.Wrapf(err, "GetIDBInstance")
		}
		connectionStr = iRds.GetConnectionStr()
		if (isOpen && len(connectionStr) > 0) || (!isOpen && len(connectionStr) == 0) {
			return true, nil
		}
		return false, nil
	})

	if err != nil {
		self.taskFailed(ctx, instance, errors.Wrapf(err, "cloudprovider.Wait"))
		return
	}

	_, err = db.Update(instance, func() error {
		instance.ConnectionStr = connectionStr
		return nil
	})

	if err != nil {
		self.taskFailed(ctx, instance, errors.Wrap(err, "db.Update"))
		return
	}

	self.taskComplete(ctx, instance)
}

func (self *DBInstancePublicConnectionTask) taskComplete(ctx context.Context, dbinstance *models.SDBInstance) {
	dbinstance.SetStatus(ctx, self.UserCred, api.DBINSTANCE_RUNNING, "")
	logclient.AddActionLogWithStartable(self, dbinstance, self.getAction(), nil, self.UserCred, true)

	self.SetStageComplete(ctx, nil)
}
