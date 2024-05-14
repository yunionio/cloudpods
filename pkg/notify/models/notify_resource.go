// Copyright  Yunion
//
// Licensed under the Apache License, Version . (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http//www.apache.org/licenses/LICENSE-.
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package models

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"

	api "yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SNotifyResourceManager struct {
	db.SEnabledStatusStandaloneResourceBaseManager
}

var NotifyResourceManager *SNotifyResourceManager

func init() {
	NotifyResourceManager = &SNotifyResourceManager{
		SEnabledStatusStandaloneResourceBaseManager: db.NewEnabledStatusStandaloneResourceBaseManager(
			SNotifyResource{},
			"notify_resources_tbl",
			"notify_resource",
			"notify_resources",
		),
	}
	NotifyResourceManager.SetVirtualObject(NotifyResourceManager)
}

type SNotifyResource struct {
	db.SEnabledStatusStandaloneResourceBase
}

func (sm *SNotifyResourceManager) InitializeData() error {
	ctx := context.Background()
	resources := []string{
		api.TOPIC_RESOURCE_SERVER,
		api.TOPIC_RESOURCE_SCALINGGROUP,
		api.TOPIC_RESOURCE_SCALINGPOLICY,
		api.TOPIC_RESOURCE_IMAGE,
		api.TOPIC_RESOURCE_DISK,
		api.TOPIC_RESOURCE_SNAPSHOT,
		api.TOPIC_RESOURCE_INSTANCESNAPSHOT,
		api.TOPIC_RESOURCE_SNAPSHOTPOLICY,
		api.TOPIC_RESOURCE_NETWORK,
		api.TOPIC_RESOURCE_EIP,
		api.TOPIC_RESOURCE_SECGROUP,
		api.TOPIC_RESOURCE_LOADBALANCER,
		api.TOPIC_RESOURCE_LOADBALANCERACL,
		api.TOPIC_RESOURCE_LOADBALANCERCERTIFICATE,
		api.TOPIC_RESOURCE_BUCKET,
		api.TOPIC_RESOURCE_DBINSTANCE,
		api.TOPIC_RESOURCE_ELASTICCACHE,
		api.TOPIC_RESOURCE_SCHEDULEDTASK,
		api.TOPIC_RESOURCE_BAREMETAL,
		api.TOPIC_RESOURCE_VPC,
		api.TOPIC_RESOURCE_DNSZONE,
		api.TOPIC_RESOURCE_NATGATEWAY,
		api.TOPIC_RESOURCE_WEBAPP,
		api.TOPIC_RESOURCE_CDNDOMAIN,
		api.TOPIC_RESOURCE_FILESYSTEM,
		api.TOPIC_RESOURCE_WAF,
		api.TOPIC_RESOURCE_KAFKA,
		api.TOPIC_RESOURCE_ELASTICSEARCH,
		api.TOPIC_RESOURCE_MONGODB,
		api.TOPIC_RESOURCE_DNSRECORDSET,
		api.TOPIC_RESOURCE_LOADBALANCERLISTENER,
		api.TOPIC_RESOURCE_LOADBALANCERBACKEDNGROUP,
		api.TOPIC_RESOURCE_HOST,
		api.TOPIC_RESOURCE_TASK,
		api.TOPIC_RESOURCE_CLOUDPODS_COMPONENT,
		api.TOPIC_RESOURCE_DB_TABLE_RECORD,
		api.TOPIC_RESOURCE_USER,
		api.TOPIC_RESOURCE_ACTION_LOG,
		api.TOPIC_RESOURCE_ACCOUNT_STATUS,
		api.TOPIC_RESOURCE_NET,
		api.TOPIC_RESOURCE_SERVICE,
		api.TOPIC_RESOURCE_VM_INTEGRITY_CHECK,
		api.TOPIC_RESOURCE_PROJECT,
		api.TOPIC_RESOURCE_CLOUDPHONE,
	}
	dbResources := []SNotifyResource{}
	q := NotifyResourceManager.Query().In("id", resources)
	err := db.FetchModelObjects(NotifyResourceManager, q, &dbResources)
	if err != nil {
		return errors.Wrap(err, "fetch topic_resources")
	}
	dbResourceMap := map[string]struct{}{}
	for _, dbResource := range dbResources {
		dbResourceMap[dbResource.Id] = struct{}{}
	}
	for i, resource := range resources {
		if _, ok := dbResourceMap[resource]; !ok {
			NotifyResource := SNotifyResource{}
			NotifyResource.Id = resources[i]
			NotifyResource.Name = resources[i]
			NotifyResource.Enabled = tristate.True
			NotifyResourceManager.TableSpec().InsertOrUpdate(ctx, &NotifyResource)
		}
	}
	return nil
}

func (manager *SNotifyResourceManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.SNotifyElementCreateInput) (api.SNotifyElementCreateInput, error) {
	_, err := manager.SEnabledStatusStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.EnabledStatusStandaloneResourceCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "validateCreate")
	}
	count, err := manager.Query().Equals("name", input.Name).CountWithError()
	if err != nil {
		return input, errors.Wrap(err, "fetch count")
	}
	if count > 0 {
		return input, errors.Wrap(httperrors.ErrDuplicateName, "%s has been exist")
	}
	return input, nil
}
