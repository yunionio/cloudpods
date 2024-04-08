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
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apihelper"
	compute_api "yunion.io/x/onecloud/pkg/apis/compute"
	keystone_api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/mcclient/modules/identity"
)

const (
	INTERNAL_ID = "internal"
)

type IMonitorResModelSet interface {
	apihelper.IModelSet
	GetResType() string
	NeedSync() bool
}

type (
	// +onecloud:swagger-gen-ignore
	Servers map[string]*Guest
	// +onecloud:swagger-gen-ignore
	Hosts map[string]*Host
	// +onecloud:swagger-gen-ignore
	Rds map[string]*SRds
	// +onecloud:swagger-gen-ignore
	Redis map[string]*SRedis
	// +onecloud:swagger-gen-ignore
	Oss map[string]*SOss
	// +onecloud:swagger-gen-ignore
	Accounts map[string]*SAccount
	// +onecloud:swagger-gen-ignore
	Storages map[string]*SStorage
	// +onecloud:swagger-gen-ignore
	Domains map[string]*SDomain
	// +onecloud:swagger-gen-ignore
	Projects map[string]*SProject
)

// +onecloud:swagger-gen-ignore
type Guest struct {
	db.SModelBase
	compute_api.ServerDetails
}

// +onecloud:swagger-gen-ignore
type Host struct {
	db.SModelBase
	compute_api.HostDetails
}

// +onecloud:swagger-gen-ignore
type SRds struct {
	db.SModelBase
	compute_api.DBInstanceDetails
}

// +onecloud:swagger-gen-ignore
type SRedis struct {
	db.SModelBase
	compute_api.ElasticcacheDetails
}

// +onecloud:swagger-gen-ignore
type SOss struct {
	db.SModelBase
	compute_api.BucketDetails
}

// +onecloud:swagger-gen-ignore
type SStorage struct {
	db.SModelBase
	compute_api.StorageDetails
}

// +onecloud:swagger-gen-ignore
type SAccount struct {
	db.SModelBase
	compute_api.CloudaccountDetail
}

// +onecloud:swagger-gen-ignore
type SDomain struct {
	db.SModelBase
	keystone_api.DomainDetails
}

// +onecloud:swagger-gen-ignore
type SProject struct {
	db.SModelBase
	keystone_api.ProjectDetails
}

func (s Servers) ModelManager() modulebase.IBaseManager {
	return &compute.Servers
}

func (s Servers) NewModel() db.IModel {
	return &Guest{}
}

func (s Servers) AddModel(i db.IModel) {
	resource := i.(*Guest)
	s[resource.Id] = resource
}

func (s Servers) Copy() apihelper.IModelSet {
	return s
}

func (s Servers) GetResType() string {
	return monitor.METRIC_RES_TYPE_GUEST
}

func (s Servers) NeedSync() bool {
	return true
}

func (s Servers) ModelFilter() []string {
	return []string{"hypervisor.notin(container)"}
}

func (h Hosts) AddModel(i db.IModel) {
	resource := i.(*Host)
	h[resource.Id] = resource
}

func (h Hosts) Copy() apihelper.IModelSet {
	return h
}

func (h Hosts) ModelManager() modulebase.IBaseManager {
	return &compute.Hosts
}

func (h Hosts) NewModel() db.IModel {
	return &Host{}
}

func (h Hosts) GetResType() string {
	return monitor.METRIC_RES_TYPE_HOST
}

func (s Hosts) NeedSync() bool {
	return true
}

func (s Hosts) ModelParamFilter() jsonutils.JSONObject {
	param := jsonutils.NewDict()
	param.Set("baremetal", jsonutils.NewBool(false))
	return param
}

func (r Rds) ModelManager() modulebase.IBaseManager {
	return &compute.DBInstance
}

func (r Rds) NewModel() db.IModel {
	return &SRds{}
}

func (r Rds) AddModel(i db.IModel) {
	resource := i.(*SRds)
	r[resource.Id] = resource
}

func (r Rds) Copy() apihelper.IModelSet {
	return r
}

func (r Rds) GetResType() string {
	return monitor.METRIC_RES_TYPE_RDS
}

func (s Rds) NeedSync() bool {
	return true
}

func (r Redis) ModelManager() modulebase.IBaseManager {
	return &compute.ElasticCache
}

func (r Redis) NewModel() db.IModel {
	return &SRedis{}
}

func (r Redis) AddModel(i db.IModel) {
	resource := i.(*SRedis)
	r[resource.Id] = resource
}

func (r Redis) Copy() apihelper.IModelSet {
	return r
}

func (r Redis) GetResType() string {
	return monitor.METRIC_RES_TYPE_REDIS
}

func (s Redis) NeedSync() bool {
	return true
}

func (o Oss) ModelManager() modulebase.IBaseManager {
	return &compute.Buckets
}

func (o Oss) NewModel() db.IModel {
	return &SOss{}
}

func (o Oss) AddModel(i db.IModel) {
	resource := i.(*SOss)
	o[resource.Id] = resource
}

func (o Oss) Copy() apihelper.IModelSet {
	return o
}

func (o Oss) GetResType() string {
	return monitor.METRIC_RES_TYPE_OSS
}

func (s Oss) NeedSync() bool {
	return true
}

func (a Accounts) ModelManager() modulebase.IBaseManager {
	return &compute.Cloudaccounts
}

func (a Accounts) NewModel() db.IModel {
	return &SAccount{}
}

func (a Accounts) AddModel(i db.IModel) {
	resource := i.(*SAccount)
	resource.TenantId = INTERNAL_ID
	a[resource.Id] = resource
}

func (a Accounts) Copy() apihelper.IModelSet {
	return a
}

func (a Accounts) GetResType() string {
	return monitor.METRIC_RES_TYPE_CLOUDACCOUNT
}

func (s Accounts) NeedSync() bool {
	return true
}

func (s Storages) ModelManager() modulebase.IBaseManager {
	return &compute.Storages
}

func (s Storages) NewModel() db.IModel {
	return &SStorage{}
}

func (s Storages) AddModel(i db.IModel) {
	resource := i.(*SStorage)
	s[resource.Id] = resource
}

func (s Storages) Copy() apihelper.IModelSet {
	return s
}

func (s Storages) GetResType() string {
	return monitor.METRIC_RES_TYPE_STORAGE
}

func (s Storages) NeedSync() bool {
	return true
}

func (s Storages) SetModelListParams(params *jsonutils.JSONDict) *jsonutils.JSONDict {
	// list storages with details is too long
	params.Set("details", jsonutils.JSONFalse)
	return params
}

func (d Domains) ModelManager() modulebase.IBaseManager {
	return &identity.Domains
}

func (d Domains) NewModel() db.IModel {
	return &SDomain{}
}

func (d Domains) AddModel(i db.IModel) {
	resource := i.(*SDomain)
	d[resource.Id] = resource
}

func (d Domains) Copy() apihelper.IModelSet {
	return d
}

func (d Domains) GetResType() string {
	return monitor.METRIC_RES_TYPE_DOMAIN
}

func (d Domains) NeedSync() bool {
	return false
}

func (p Projects) ModelManager() modulebase.IBaseManager {
	return &identity.Projects
}

func (p Projects) NewModel() db.IModel {
	return &SProject{}
}

func (p Projects) AddModel(i db.IModel) {
	resource := i.(*SProject)
	p[resource.Id] = resource
}

func (p Projects) Copy() apihelper.IModelSet {
	return p
}

func (p Projects) GetResType() string {
	return monitor.METRIC_RES_TYPE_TENANT
}

func (p Projects) NeedSync() bool {
	return false
}
