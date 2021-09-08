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
	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	compute_models "yunion.io/x/onecloud/pkg/compute/models"
	keystone_models "yunion.io/x/onecloud/pkg/keystone/models"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	mcclient_modules "yunion.io/x/onecloud/pkg/mcclient/modules"
)

type IMonitorResModelSet interface {
	apihelper.IModelSet
	GetResType() string
	NeedSync() bool
}

type (
	Servers  map[string]*Guest
	Hosts    map[string]*Host
	Rds      map[string]*SRds
	Redis    map[string]*SRedis
	Oss      map[string]*SOss
	Accounts map[string]*SAccount
	Storages map[string]*SStorage
	Domains  map[string]*SDomain
	Projects map[string]*SProject
)

type Details struct {
	//com_apis.CloudproviderDetails
	//Host          string
	HostId        string
	Zone          string
	zoneId        string
	zoneExtId     string
	Cloudregion   string
	CloudregionId string
	Tenant        string
	TenantId      string
	Brand         string
	DomainId      string
	ProjectDomain string
	Ips           string
}
type Guest struct {
	compute_models.SGuest
	Details
}

type Host struct {
	Id string
	compute_models.SHost
	Details
}

type SRds struct {
	compute_models.SDBInstance
	Details
}

type SRedis struct {
	compute_models.SElasticcache
	Details
}

type SOss struct {
	compute_models.SBucket
	Details
}

type SStorage struct {
	Id string
	compute_models.SStorage
	Details
}

type SAccount struct {
	Id string
	compute_models.SCloudaccount
	Details
}

type SDomain struct {
	Id string
	keystone_models.SDomain
}

type SProject struct {
	Id string
	keystone_models.SProject
}

func (s Servers) ModelManager() modulebase.IBaseManager {
	return &mcclient_modules.Servers
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
	return []string{"hypervisor.notin(baremetal,container)"}
}

func (h Hosts) AddModel(i db.IModel) {
	resource := i.(*Host)
	h[resource.Id] = resource
}

func (h Hosts) Copy() apihelper.IModelSet {
	return h
}

func (h Hosts) ModelManager() modulebase.IBaseManager {
	return &mcclient_modules.Hosts
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
	return &mcclient_modules.DBInstance
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
	return &mcclient_modules.ElasticCache
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
	return &mcclient_modules.Buckets
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
	return &mcclient_modules.Cloudaccounts
}

func (a Accounts) NewModel() db.IModel {
	return &SAccount{}
}

func (a Accounts) AddModel(i db.IModel) {
	resource := i.(*SAccount)
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
	return &mcclient_modules.Storages
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

func (d Domains) ModelManager() modulebase.IBaseManager {
	return &mcclient_modules.Domains
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
	return &mcclient_modules.Projects
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
