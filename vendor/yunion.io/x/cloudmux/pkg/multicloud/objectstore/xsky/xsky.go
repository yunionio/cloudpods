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

package xsky

import (
	"context"
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"
	"yunion.io/x/s3cli"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud/objectstore"
)

type SXskyClient struct {
	*objectstore.SObjectStoreClient

	adminApi *SXskyAdminApi

	adminUser   *sUser
	initAccount string
}

func parseAccount(account string) (user string, accessKey string) {
	accountInfo := strings.Split(account, "/")
	user = accountInfo[0]
	if len(accountInfo) > 1 {
		accessKey = strings.Join(accountInfo[1:], "/")
	}
	return
}

func NewXskyClient(cfg *objectstore.ObjectStoreClientConfig) (*SXskyClient, error) {
	usrname, accessKey := parseAccount(cfg.GetAccessKey())
	adminApi := newXskyAdminApi(
		usrname,
		cfg.GetAccessSecret(),
		cfg.GetEndpoint(),
		cfg.GetDebug(),
	)
	httputils.SetClientProxyFunc(adminApi.httpClient(), cfg.GetCloudproviderConfig().ProxyFunc)

	gwEp, err := adminApi.getS3GatewayEndpoint(context.Background())
	if err != nil {
		return nil, errors.Wrap(err, "adminApi.getS3GatewayIP")
	}

	var usr *sUser
	var key *sKey
	if len(accessKey) > 0 {
		usr, key, err = adminApi.findUserByAccessKey(context.Background(), accessKey)
		if err != nil {
			return nil, errors.Wrap(err, "adminApi.findUserByAccessKey")
		}
	} else {
		usr, key, err = adminApi.findFirstUserWithAccessKey(context.Background())
		if err != nil {
			return nil, errors.Wrap(err, "adminApi.findFirstUserWithAccessKey")
		}
	}

	s3store, err := objectstore.NewObjectStoreClientAndFetch(
		objectstore.NewObjectStoreClientConfig(
			gwEp, accessKey, key.SecretKey,
		).Debug(cfg.GetDebug()).CloudproviderConfig(cfg.GetCloudproviderConfig()),
		false,
	)
	if err != nil {
		return nil, errors.Wrap(err, "NewObjectStoreClient")
	}

	client := SXskyClient{
		SObjectStoreClient: s3store,
		adminApi:           adminApi,
		adminUser:          usr,
	}

	if len(accessKey) > 0 {
		client.initAccount = cfg.GetAccessKey()
	}

	client.SetVirtualObject(&client)

	err = client.FetchBuckets()
	if err != nil {
		return nil, errors.Wrap(err, "fetchBuckets")
	}

	return &client, nil
}

func (cli *SXskyClient) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	if len(cli.initAccount) > 0 {
		return []cloudprovider.SSubAccount{
			{
				Id:           fmt.Sprintf("%d", cli.adminUser.Id),
				Account:      cli.initAccount,
				Name:         cli.adminUser.Name,
				HealthStatus: api.CLOUD_PROVIDER_HEALTH_NORMAL,
			},
		}, nil
	}
	usrs, err := cli.adminApi.getUsers(context.Background())
	if err != nil {
		return nil, errors.Wrap(err, "api.getUsers")
	}
	subAccounts := make([]cloudprovider.SSubAccount, 0)
	for i := range usrs {
		ak := usrs[i].getMinKey()
		if len(ak) > 0 {
			subAccount := cloudprovider.SSubAccount{
				Id:           fmt.Sprintf("%d", usrs[i].Id),
				Account:      fmt.Sprintf("%s/%s", cli.adminApi.username, ak),
				Name:         usrs[i].Name,
				HealthStatus: api.CLOUD_PROVIDER_HEALTH_NORMAL,
			}
			subAccounts = append(subAccounts, subAccount)
		}
	}
	return subAccounts, nil
}

func (cli *SXskyClient) GetAccountId() string {
	return cli.adminApi.username
}

func (cli *SXskyClient) GetVersion() string {
	return ""
}

func (cli *SXskyClient) About() jsonutils.JSONObject {
	about := jsonutils.NewDict()
	if cli.adminUser != nil {
		about.Add(jsonutils.Marshal(cli.adminUser), "admin_user")
	}
	return about
}

func (cli *SXskyClient) GetProvider() string {
	return api.CLOUD_PROVIDER_XSKY
}

func (cli *SXskyClient) NewBucket(bucket s3cli.BucketInfo) cloudprovider.ICloudBucket {
	if cli.SObjectStoreClient == nil {
		return nil
	}
	generalBucket := cli.SObjectStoreClient.NewBucket(bucket)
	return &SXskyBucket{
		SBucket: generalBucket.(*objectstore.SBucket),
		client:  cli,
	}
}
