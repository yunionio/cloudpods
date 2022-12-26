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

package ceph

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"
	"yunion.io/x/s3cli"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud/objectstore"
)

type SCephRadosClient struct {
	*objectstore.SObjectStoreClient

	adminApi *SCephAdminApi

	userQuota   *SQuota
	bucketQuota *SQuota
	userInfo    *SUserInfo
}

func NewCephRados(cfg *objectstore.ObjectStoreClientConfig) (*SCephRadosClient, error) {
	s3store, err := objectstore.NewObjectStoreClientAndFetch(cfg, false)
	if err != nil {
		return nil, errors.Wrap(err, "NewObjectStoreClient")
	}
	adminApi := newCephAdminApi(
		cfg.GetAccessKey(),
		cfg.GetAccessSecret(),
		cfg.GetEndpoint(),
		cfg.GetDebug(),
		"",
	)
	httputils.SetClientProxyFunc(adminApi.httpClient(), cfg.GetCloudproviderConfig().ProxyFunc)

	client := SCephRadosClient{
		SObjectStoreClient: s3store,
		adminApi:           adminApi,
	}

	client.SetVirtualObject(&client)

	err = client.FetchBuckets()
	if err != nil {
		return nil, errors.Wrap(err, "fetchBuckets")
	}

	userQuota, bucketQuota, err := adminApi.GetUserQuota(context.Background(), s3store.GetAccountId())
	if err != nil {
		if errors.Cause(err) != cloudprovider.ErrForbidden {
			return nil, errors.Wrap(err, "adminApi.GetUserQuota")
		} else {
			// skip the error
			log.Errorf("adminApi.GetUserQuota fail: %s", err)
		}
	}
	userInfo, err := adminApi.GetUserInfo(context.Background(), s3store.GetAccountId())
	if err != nil {
		if errors.Cause(err) != cloudprovider.ErrForbidden {
			return nil, errors.Wrap(err, "adminApi.GetUserInfo")
		} else {
			// skip the error
			log.Errorf("adminApi.GetUserInfo fail: %s", err)
		}
	}
	if cfg.GetDebug() {
		log.Debugf("%#v %#v %#v", userQuota, bucketQuota, userInfo)
	}
	client.userQuota = userQuota
	client.bucketQuota = bucketQuota
	client.userInfo = userInfo

	return &client, nil
}

func (cli *SCephRadosClient) GetVersion() string {
	return ""
}

func (cli *SCephRadosClient) About() jsonutils.JSONObject {
	about := jsonutils.NewDict()
	if cli.userQuota != nil {
		about.Add(jsonutils.Marshal(cli.userQuota), "user_quota")
	}
	if cli.bucketQuota != nil {
		about.Add(jsonutils.Marshal(cli.bucketQuota), "bucket_quota")
	}
	if cli.userInfo != nil {
		about.Add(jsonutils.Marshal(cli.userInfo), "user_info")
	}
	return about
}

func (cli *SCephRadosClient) GetProvider() string {
	return api.CLOUD_PROVIDER_CEPH
}

func (cli *SCephRadosClient) NewBucket(bucket s3cli.BucketInfo) cloudprovider.ICloudBucket {
	generalBucket := cli.SObjectStoreClient.NewBucket(bucket)
	return &SCephRadosBucket{
		SBucket: generalBucket.(*objectstore.SBucket),
	}
}
