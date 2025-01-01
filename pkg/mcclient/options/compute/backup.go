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

package compute

import (
	"yunion.io/x/jsonutils"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type DiskBackupListOptions struct {
	options.BaseListOptions
	DiskId           string `help:"disk id" json:"disk_id"`
	BackupStorageId  string `help:"backup storage id" json:"backup_storage_id"`
	IsInstanceBackup *bool  `help:"if part of instance backup" json:"is_instance_backup"`
	OrderByDiskName  string
}

func (opts *DiskBackupListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(opts)
}

type DiskBackupIdOptions struct {
	ID string `help:"disk backup id" json:"-"`
}

func (opts *DiskBackupIdOptions) GetId() string {
	return opts.ID
}

func (opts *DiskBackupIdOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type DiskBackupDeleteOptions struct {
	DiskBackupIdOptions
	Force bool `help:"force delete"`
}

func (opts *DiskBackupDeleteOptions) QueryParams() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}

type DiskBackupCreateOptions struct {
	options.BaseCreateOptions
	AsTarContainerId        string   `help:"container id of tar process"`
	AsTarIncludeFile        []string `help:"include file path of tar process"`
	AsTarExcludeFile        []string `help:"exclude file path of tar process"`
	AsTarIgnoreNotExistFile bool     `help:"ignore not exist file when using tar"`

	DISKID          string `help:"disk id" json:"disk_id"`
	BACKUPSTORAGEID string `help:"back storage id" json:"backup_storage_id"`
}

func (opts *DiskBackupCreateOptions) Params() (jsonutils.JSONObject, error) {
	input := &computeapi.DiskBackupCreateInput{
		DiskId:          opts.DISKID,
		BackupStorageId: opts.BACKUPSTORAGEID,
		BackupAsTar:     new(computeapi.DiskBackupAsTarInput),
	}
	input.Name = opts.NAME
	input.Description = opts.Desc
	if opts.AsTarContainerId != "" {
		input.BackupAsTar.ContainerId = opts.AsTarContainerId
	}
	if len(opts.AsTarIncludeFile) > 0 {
		input.BackupAsTar.IncludeFiles = opts.AsTarIncludeFile
	}
	if len(opts.AsTarExcludeFile) > 0 {
		input.BackupAsTar.ExcludeFiles = opts.AsTarExcludeFile
	}
	if opts.AsTarIgnoreNotExistFile {
		input.BackupAsTar.IgnoreNotExistFile = opts.AsTarIgnoreNotExistFile
	}
	return jsonutils.Marshal(input), nil
}

type DiskBackupRecoveryOptions struct {
	DiskBackupIdOptions
	Name string `help:"disk name" json:"name"`
}

func (opt *DiskBackupRecoveryOptions) GetId() string {
	return opt.ID
}

func (opt *DiskBackupRecoveryOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.NewDict()
	params.Set("name", jsonutils.NewString(opt.Name))
	return params, nil
}

type DiskBackupSyncstatusOptions struct {
	DiskBackupIdOptions
}

func (opt *DiskBackupSyncstatusOptions) GetId() string {
	return opt.ID
}

func (opt *DiskBackupSyncstatusOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type BackupStorageListOptions struct {
	options.BaseListOptions
}

func (opts *BackupStorageListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(opts)
}

type BackupStorageIdOptions struct {
	ID string `help:"backup storage id"`
}

func (opts *BackupStorageIdOptions) GetId() string {
	return opts.ID
}

func (opts *BackupStorageIdOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type BackupStorageCreateOptions struct {
	options.BaseCreateOptions
	StorageType string `help:"storage type" choices:"nfs|object"`

	NfsHost      string `help:"nfs host, required when storage_type is nfs"`
	NfsSharedDir string `help:"nfs shared dir, required when storage_type is nfs" `

	ObjectBucketUrl string `help:"object bucket url, required when storage_type is object"`
	ObjectAccessKey string `help:"object storage access key, required when storage_type is object"`
	ObjectSecret    string `help:"object storage secret, required when storage_type is object"`
	ObjectSignVer   string `help:"object storage signing alogirithm version, optional" choices:"v2|v4"`

	CapacityMb int `help:"capacity, unit mb"`
}

func (opts *BackupStorageCreateOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}

type BackupStorageUpdateOptions struct {
	options.BaseUpdateOptions

	NfsHost      string `help:"nfs host, required when storage_type is nfs"`
	NfsSharedDir string `help:"nfs shared dir, required when storage_type is nfs" `

	ObjectBucketUrl string `help:"object bucket url, required when storage_type is object"`
	ObjectAccessKey string `help:"object storage access key, required when storage_type is object"`
	ObjectSecret    string `help:"object storage secret, required when storage_type is object"`
	ObjectSignVer   string `help:"object storage signing alogirithm version, optional" choices:"v2|v4"`
}

func (opts *BackupStorageUpdateOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}

type InstanceBackupListOptions struct {
	options.BaseListOptions

	OrderByGuest string
}

func (opts *InstanceBackupListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(opts)
}

type InstanceBackupIdOptions struct {
	ID string `help:"instance backup id" json:"-"`
}

func (opts *InstanceBackupIdOptions) GetId() string {
	return opts.ID
}

func (opts *InstanceBackupIdOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type InstanceBackupDeleteOptions struct {
	InstanceBackupIdOptions
	Force bool `help:"force delete"`
}

func (opts *InstanceBackupDeleteOptions) QueryParams() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}

type InstanceBackupRecoveryOptions struct {
	DiskBackupIdOptions
	Name string `help:"server name" json:"name"`
}

func (opts *InstanceBackupRecoveryOptions) GetId() string {
	return opts.ID
}

func (opts *InstanceBackupRecoveryOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}

type InstanceBackupPackOptions struct {
	DiskBackupIdOptions
	PackageName string `help:"package name" json:"package_name"`
}

func (opts *InstanceBackupPackOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}

type InstanceBackupManagerCreateFromPackageOptions struct {
	PackageName     string `help:"package name" json:"package_name"`
	Name            string `help:"instance backup name" json:"name"`
	BackupStorageId string `help:"backup storage id" json:"backup_storage_id"`
	ProjectId       string `help:"target project id" json:"project_id"`
}

func (opts *InstanceBackupManagerCreateFromPackageOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}

type HostBackupStorageListOptions struct {
	options.BaseListOptions
	Host          string `json:"-" help:"filter by host"`
	Backupstorage string `json:"-" help:"filter by backupstorage"`
}

func (opts HostBackupStorageListOptions) GetMasterOpt() string {
	return opts.Host
}

func (opts HostBackupStorageListOptions) GetSlaveOpt() string {
	return opts.Backupstorage
}

func (opts *HostBackupStorageListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(opts)
}

type HostBackupStorageJoinOptions struct {
	HOST          string `json:"-" help:"host id"`
	BACKUPSTORAGE string `json:"-" help:"backup storage id"`
}

func (opts HostBackupStorageJoinOptions) GetMasterId() string {
	return opts.HOST
}

func (opts HostBackupStorageJoinOptions) GetSlaveId() string {
	return opts.BACKUPSTORAGE
}

func (opts HostBackupStorageJoinOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.NewDict(), nil
}
