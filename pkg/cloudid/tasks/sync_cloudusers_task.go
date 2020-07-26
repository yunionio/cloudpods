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
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudid/models"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SyncCloudusersTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(SyncCloudusersTask{})
}

type IProvider interface {
	GetProvider() (cloudprovider.ICloudProvider, error)
	GetName() string
	GetCloudproviderId() string
}

func (self *SyncCloudusersTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	account := obj.(*models.SCloudaccount)
	self.SetStage("OnClouduserSyncComplete", nil)
	taskman.LocalTaskRun(self, func() (jsonutils.JSONObject, error) {
		factory, err := account.GetProviderFactory()
		if err != nil {
			return nil, errors.Wrap(err, "account.GetProviderFactory")
		}
		if !factory.IsSupportClouduser() {
			return nil, nil
		}
		iProviders := []IProvider{account}
		if factory.IsClouduserBelongCloudprovider() {
			iProviders = []IProvider{}
			providers, err := account.GetCloudproviders()
			if err != nil {
				return nil, errors.Wrap(err, "GetCloudproviders")
			}
			for i := range providers {
				iProviders = append(iProviders, &providers[i])
			}
		}
		for i := range iProviders {
			provider, err := iProviders[i].GetProvider()
			if err != nil {
				return nil, errors.Wrap(err, "GetProvider")
			}
			iUsers, err := provider.GetICloudusers()
			if err != nil {
				return nil, errors.Wrap(err, "provider.GetICloudusers")
			}
			localUsers, remoteUsers, result := account.SyncCloudusers(ctx, self.UserCred, iProviders[i].GetCloudproviderId(), iUsers)
			msg := fmt.Sprintf("SyncCloudusers for account %s(%s) result: %s", iProviders[i].GetName(), account.Provider, result.Result())
			log.Infof(msg)

			for i := 0; i < len(localUsers); i += 1 {
				func() {
					// lock clouduser
					lockman.LockObject(ctx, &localUsers[i])
					defer lockman.ReleaseObject(ctx, &localUsers[i])

					syncClouduserPolicies(ctx, self.GetUserCred(), &localUsers[i], remoteUsers[i])
					syncClouduserGroups(ctx, self.GetUserCred(), &localUsers[i], remoteUsers[i])
				}()
			}
		}
		return nil, nil
	})
}

func syncClouduserPolicies(ctx context.Context, userCred mcclient.TokenCredential, localUser *models.SClouduser, remoteUser cloudprovider.IClouduser) {
	iPolicies, err := remoteUser.GetISystemCloudpolicies()
	if err != nil {
		log.Errorf("failed to get user %s policies error: %v", remoteUser.GetName(), err)
		return
	}
	result := localUser.SyncCloudpolicies(ctx, userCred, iPolicies)
	msg := fmt.Sprintf("SyncCloudpolicies for user %s(%s) result: %s", localUser.Name, localUser.Id, result.Result())
	log.Infof(msg)
}

func syncClouduserGroups(ctx context.Context, userCred mcclient.TokenCredential, localUser *models.SClouduser, remoteUser cloudprovider.IClouduser) {
	iGroups, err := remoteUser.GetICloudgroups()
	if err != nil {
		log.Errorf("failed to get user %s groups error: %v", remoteUser.GetName(), err)
		return
	}
	result := localUser.SyncCloudgroups(ctx, userCred, iGroups)
	msg := fmt.Sprintf("SyncCloudgroups for user %s(%s) result: %s", localUser.Name, localUser.Id, result.Result())
	log.Infof(msg)
}

func (self *SyncCloudusersTask) OnClouduserSyncComplete(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStageComplete(ctx, nil)
}

func (self *SyncCloudusersTask) OnClouduserSyncCompleteFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	self.SetStageFailed(ctx, data)
}
