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

package cloudprovider

import (
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/httperrors"
)

const (
	CloudVMStatusRunning      = "running"
	CloudVMStatusStopping     = "stopping"
	CloudVMStatusSuspend      = "suspend"
	CloudVMStatusStopped      = "stopped"
	CloudVMStatusChangeFlavor = "change_flavor"
	CloudVMStatusDeploying    = "deploying"
	CloudVMStatusOther        = "other"

	ErrNotFound       = errors.ErrNotFound
	ErrDuplicateId    = errors.ErrDuplicateId
	ErrInvalidStatus  = errors.ErrInvalidStatus
	ErrTimeout        = errors.ErrTimeout
	ErrNotImplemented = errors.ErrNotImplemented
	ErrNotSupported   = errors.ErrNotSupported

	ErrInvalidProvider     = httperrors.ErrInvalidProvider
	ErrNoBalancePermission = httperrors.ErrNoBalancePermission
)

const (
	CLOUD_CAPABILITY_PROJECT         = "project"
	CLOUD_CAPABILITY_COMPUTE         = "compute"
	CLOUD_CAPABILITY_NETWORK         = "network"
	CLOUD_CAPABILITY_LOADBALANCER    = "loadbalancer"
	CLOUD_CAPABILITY_OBJECTSTORE     = "objectstore"
	CLOUD_CAPABILITY_RDS             = "rds"
	CLOUD_CAPABILITY_CACHE           = "cache" // 弹性缓存包含redis、memcached
	CLOUD_CAPABILITY_EVENT           = "event"
	CLOUD_CAPABILITY_CLOUDID         = "cloudid"
	CLOUD_CAPABILITY_DNSZONE         = "dnszone"
	CLOUD_CAPABILITY_PUBLIC_IP       = "public_ip"
	CLOUD_CAPABILITY_INTERVPCNETWORK = "intervpcnetwork"
	CLOUD_CAPABILITY_SAML_AUTH       = "saml_auth" // 是否支持SAML 2.0
	CLOUD_CAPABILITY_NAT             = "nat"       // NAT网关
	CLOUD_CAPABILITY_NAS             = "nas"       // NAS
)

const (
	CLOUD_ENV_PUBLIC_CLOUD  = "public"
	CLOUD_ENV_PRIVATE_CLOUD = "private"
	CLOUD_ENV_ON_PREMISE    = "onpremise"

	CLOUD_ENV_PRIVATE_ON_PREMISE = "private_or_onpremise"
)
