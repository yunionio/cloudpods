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

	"github.com/pkg/errors"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudevent/models"
	"yunion.io/x/onecloud/pkg/cloudevent/options"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type CloudeventSyncTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(CloudeventSyncTask{})
}

func (self *CloudeventSyncTask) taskFailed(ctx context.Context, cloudprovider *models.SCloudprovider, err error) {
	cloudprovider.MarkEndSync(self.UserCred)
	self.SetStageFailed(ctx, err.Error())
}

func (self *CloudeventSyncTask) taskComplete(ctx context.Context, cloudprovider *models.SCloudprovider) {
	cloudprovider.MarkEndSync(self.UserCred)
	self.SetStageComplete(ctx, nil)
}

func (self *CloudeventSyncTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	provider := obj.(*models.SCloudprovider)

	factory, err := provider.GetProviderFactory()
	if err != nil {
		self.taskFailed(ctx, provider, errors.Wrap(err, "cloudprovider.GetProviderFactory"))
		return
	}

	iProvider, err := provider.GetProvider()
	if err != nil {
		self.taskFailed(ctx, provider, errors.Wrap(err, "cloudprovider.GetProvider"))
		return
	}

	start, end, err := provider.GetNextTimeRange()
	if err != nil {
		self.taskFailed(ctx, provider, errors.Wrap(err, "provider.GetNextTimeRange"))
		return
	}

	//小于1小时的暂时不同步
	duration := end.Sub(start) + time.Second
	if duration < time.Hour {
		self.taskComplete(ctx, provider)
		return
	}

	count := 0
	for {
		events := []cloudprovider.ICloudEvent{}
		regions := iProvider.GetIRegions()
		for i := range regions {
			if factory.IsCloudeventRegional() || i == 0 {
				_events, err := regions[i].GetICloudEvents(start, end, options.Options.SyncWithReadEvent)
				if err != nil {
					if err == cloudprovider.ErrNotSupported {
						continue
					}
					self.taskFailed(ctx, provider, errors.Wrapf(err, "regions[%d].GetICloudEvents", i))
					return
				}
				events = append(events, _events...)
			}
		}
		_count := models.CloudeventManager.SyncCloudevent(ctx, self.UserCred, provider, events)
		log.Infof("Sync %d events for %s(%s) from %s(%d) hours", _count, provider.Name, provider.Id, start.Format("2006-01-02T15:04:05Z"), duration/time.Hour)
		count += _count

		if time.Now().Sub(end) < duration {
			break
		}
		start = start.Add(duration)
		end = end.Add(duration)
	}
	self.taskComplete(ctx, provider)
}
