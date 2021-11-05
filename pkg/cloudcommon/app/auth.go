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

package app

import (
	"context"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	identity_modules "yunion.io/x/onecloud/pkg/mcclient/modules/identity"
)

func InitAuth(options *common_options.CommonOptions, authComplete auth.AuthCompletedCallback) {

	if len(options.AuthURL) == 0 {
		log.Fatalln("Missing AuthURL")
	}

	if len(options.AdminUser) == 0 {
		log.Fatalln("Mising AdminUser")
	}

	if len(options.AdminPassword) == 0 {
		log.Fatalln("Missing AdminPasswd")
	}

	if len(options.AdminProject) == 0 {
		log.Fatalln("Missing AdminProject")
	}

	a := auth.NewAuthInfo(
		options.AuthURL,
		options.AdminDomain,
		options.AdminUser,
		options.AdminPassword,
		options.AdminProject,
		options.AdminProjectDomain,
	)

	// debug := options.LogLevel == "debug"

	if options.SessionEndpointType != "" {
		if !utils.IsInStringArray(options.SessionEndpointType,
			[]string{identity.EndpointInterfacePublic, identity.EndpointInterfaceInternal}) {
			log.Fatalf("Invalid session endpoint type %s", options.SessionEndpointType)
		}
		auth.SetEndpointType(options.SessionEndpointType)
	}

	auth.Init(a, options.DebugClient, true, options.SslCertfile, options.SslKeyfile) // , authComplete)

	users := options.NotifyAdminUsers
	groups := options.NotifyAdminGroups
	if len(users) == 0 && len(groups) == 0 {
		users = []string{"sysadmin"}
	}
	notifyclient.FetchNotifyAdminRecipients(context.Background(), options.Region, users, groups)

	if authComplete != nil {
		authComplete()
	}

	consts.SetTenantCacheExpireSeconds(options.TenantCacheExpireSeconds)

	InitBaseAuth(&options.BaseOptions)

	watcher := newEndpointChangeManager()
	watcher.StartWatching(&identity_modules.EndpointsV3)

	startEtcdEndpointPuller()
}

func InitBaseAuth(options *common_options.BaseOptions) {
	if options.EnableRbac {
		policy.EnableGlobalRbac(
			time.Second*time.Duration(options.RbacPolicyRefreshIntervalSeconds),
			options.RbacDebug,
		)
	}
	consts.SetNonDefaultDomainProjects(options.NonDefaultDomainProjects)
}

func FetchEtcdServiceInfo() (*identity.EndpointDetails, error) {
	s := auth.GetAdminSession(context.Background(), "", "")
	return s.GetCommonEtcdEndpoint()
}

func startEtcdEndpointPuller() {
	retryInterval := 60
	etecdUrl, err := auth.GetServiceURL(apis.SERVICE_TYPE_ETCD, consts.GetRegion(), "", "")
	if err != nil {
		log.Errorf("[etcd] GetServiceURL fail %s, retry after %d seconds", err, retryInterval)
	} else if len(etecdUrl) == 0 {
		log.Errorf("[etcd] no service url found, retry after %d seconds", retryInterval)
	} else {
		return
	}
	auth.ReAuth()
	time.AfterFunc(time.Second*time.Duration(retryInterval), startEtcdEndpointPuller)
}
