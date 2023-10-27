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

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/mcclient"
)

func RefreshCloudproviderHostStatus(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	if isStart {
		return
	}
	vmwareFilter := api.ManagedResourceListInput{}
	vmwareFilter.Providers = []string{api.CLOUD_PROVIDER_VMWARE}
	for _, filter := range []api.ManagedResourceListInput{
		vmwareFilter,
	} {
		err := refreshCloudproviderHostStatus(ctx, userCred, filter)
		if err != nil {
			log.Errorf("refreshCloudproviderHostStatus fail %s", err)
		}
	}
}

func refreshCloudproviderHostStatus(ctx context.Context, userCred mcclient.TokenCredential, filter api.ManagedResourceListInput) error {
	crs, err := CloudproviderRegionManager.FetchCloudproviderRegions(func(q *sqlchemy.SQuery) (*sqlchemy.SQuery, error) {
		var err error
		q, err = CloudproviderRegionManager.SManagedResourceBaseManager.ListItemFilter(ctx, q, userCred, filter)
		if err != nil {
			return nil, errors.Wrap(err, "SManagedResourceBaseManager.ListItemFilter")
		}
		return q, nil
	})
	if err != nil {
		return errors.Wrap(err, "FetchCloudproviderRegions")
	}
	log.Debugf("refreshCloudproviderHostStatus count %d", len(crs))
	var errs []error
	for i := range crs {
		err := crs[i].doSyncHostsStatus(ctx, userCred)
		if err != nil {
			log.Errorf("DoSyncHostsStatus fail %s", err)
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return errors.NewAggregate(errs)
	}
	return nil
}

func (cr *SCloudproviderregion) doSyncHostsStatus(ctx context.Context, userCred mcclient.TokenCredential) error {
	localRegion, err := cr.GetRegion()
	if err != nil {
		return errors.Wrapf(err, "GetRegion")
	}
	provider, err := cr.GetProvider()
	if err != nil {
		return errors.Wrapf(err, "GetProvider")
	}

	log.Infof("doSyncHostsStatus for provider %s(%s) region %s(%s)", provider.Name, provider.Id, localRegion.Name, localRegion.Id)

	driver, err := provider.GetProvider(ctx)
	if err != nil {
		log.Errorf("Failed to get driver, connection problem?")
		return errors.Wrap(err, "GetProvider")
	}

	var remoteRegion cloudprovider.ICloudRegion
	if localRegion.isManaged() {
		var err error
		remoteRegion, err = driver.GetIRegionById(localRegion.ExternalId)
		if err != nil {
			return errors.Wrap(err, "GetIRegionById")
		}

	} else {
		var err error
		remoteRegion, err = driver.GetOnPremiseIRegion()
		if err != nil {
			return errors.Wrap(err, "GetOnPremiseIRegion")
		}
	}

	extHosts, err := remoteRegion.GetIHosts()
	if err != nil {
		return errors.Wrap(err, "GetIHosts")
	}

	_, _, results := HostManager.SyncHosts(ctx, userCred, provider, nil, localRegion, extHosts, false)
	if results.IsError() {
		return errors.Wrap(results.AllError(), "SyncHosts")
	}

	log.Infof("End of doSyncHostsStatus for provider %s(%s) region %s(%s)", provider.Name, provider.Id, localRegion.Name, localRegion.Id)

	return nil
}
