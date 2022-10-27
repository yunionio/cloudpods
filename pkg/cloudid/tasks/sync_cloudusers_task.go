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

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudid/models"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SyncCloudusersTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(SyncCloudusersTask{})
}

func (self *SyncCloudusersTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	account := obj.(*models.SCloudaccount)
	self.SetStage("OnClouduserSyncComplete", nil)
	taskman.LocalTaskRun(self, func() (jsonutils.JSONObject, error) {
		factory, err := account.GetProviderFactory()
		if err != nil {
			return nil, errors.Wrap(err, "account.GetProviderFactory")
		}
		if !factory.IsSupportCloudIdService() {
			return nil, nil
		}
		provider, err := account.GetProvider()
		if err != nil {
			return nil, errors.Wrap(err, "GetProvider")
		}
		iUsers, err := provider.GetICloudusers()
		if err != nil {
			return nil, errors.Wrap(err, "provider.GetICloudusers")
		}
		localUsers, remoteUsers, result := account.SyncCloudusers(ctx, self.UserCred, iUsers)
		msg := fmt.Sprintf("SyncCloudusers for account %s(%s) result: %s", account.Name, account.Provider, result.Result())
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
		return nil, nil
	})
}

func syncClouduserPolicies(ctx context.Context, userCred mcclient.TokenCredential, localUser *models.SClouduser, remoteUser cloudprovider.IClouduser) {
	account, err := localUser.GetCloudaccount()
	if err != nil {
		log.Errorf("failed to get user %s cloud account error: %v", remoteUser.GetName(), err)
		return
	}
	factory, err := account.GetProviderFactory()
	if err != nil {
		log.Errorf("failed to get user %s cloud account error: %v", remoteUser.GetName(), err)
		return
	}
	if factory.IsClouduserpolicyWithSubscription() {
		providers, err := account.GetCloudproviders()
		if err != nil {
			log.Errorf("failed get cloudproviders for account %s error: %v", account.Name, err)
			return
		}
		for i := range providers {
			iProvider, err := providers[i].GetProvider()
			if err != nil {
				log.Errorf("failed to get provider for cloudprovider %s error: %v", providers[i].Name, err)
				continue
			}
			iUser, err := iProvider.GetIClouduserByName(remoteUser.GetName())
			if err != nil {
				log.Errorf("failed to get clouduser %s for cloudprovider %s error: %v", remoteUser.GetName(), providers[i].Name, err)
				continue
			}
			iPolicies, err := iUser.GetISystemCloudpolicies()
			if err != nil {
				log.Errorf("failed to get user %s policies error: %v", remoteUser.GetName(), err)
				continue
			}
			result := localUser.SyncSystemCloudpolicies(ctx, userCred, iPolicies, providers[i].Id)
			msg := fmt.Sprintf("SyncSystemCloudpolicies for user %s(%s) in subscription %s(%s) result: %s", localUser.Name, localUser.Id, providers[i].Name, providers[i].Id, result.Result())
			log.Infof(msg)

			iPolicies, err = remoteUser.GetICustomCloudpolicies()
			if err != nil {
				if errors.Cause(err) != cloudprovider.ErrNotImplemented {
					log.Errorf("failed to get user %s custom policies error: %v", remoteUser.GetName(), err)
					return
				}
				return
			}
			result = localUser.SyncCustomCloudpolicies(ctx, userCred, iPolicies, "")
			msg = fmt.Sprintf("SyncCustomCloudpolicies for user %s(%s) result: %s", localUser.Name, localUser.Id, result.Result())
			log.Infof(msg)
		}
	} else {
		iPolicies, err := remoteUser.GetISystemCloudpolicies()
		if err != nil {
			log.Errorf("failed to get user %s policies error: %v", remoteUser.GetName(), err)
			return
		}
		result := localUser.SyncSystemCloudpolicies(ctx, userCred, iPolicies, "")
		msg := fmt.Sprintf("SyncSystemCloudpolicies for user %s(%s) result: %s", localUser.Name, localUser.Id, result.Result())
		log.Infof(msg)
		iPolicies, err = remoteUser.GetICustomCloudpolicies()
		if err != nil {
			if errors.Cause(err) != cloudprovider.ErrNotImplemented {
				log.Errorf("failed to get user %s custom policies error: %v", remoteUser.GetName(), err)
				return
			}
			return
		}
		result = localUser.SyncCustomCloudpolicies(ctx, userCred, iPolicies, "")
		msg = fmt.Sprintf("SyncCustomCloudpolicies for user %s(%s) result: %s", localUser.Name, localUser.Id, result.Result())
		log.Infof(msg)
	}
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
