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

	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type DiskBackupListOptions struct {
	options.BaseListOptions
	DiskId           string `help:"disk id" json:"disk_id"`
	BackupStorageId  string `help:"backup storage id" json:"backup_storage_id"`
	IsInstanceBackup *bool  `help:"if part of instance backup" json:"is_instance_backup"`
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
	DISKID          string `help:"disk id" json:"disk_id"`
	BACKUPSTORAGEID string `help:"back storage id" json:"backup_storage_id"`
}

func (opts *DiskBackupCreateOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
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
	StorageType  string `help:"storage type" choices:"nfs"`
	NfsHost      string `help:"nfs host, required when storage_type is nfs"`
	NfsSharedDir string `help:"nfs shared dir, required when storage_type is nfs" `
	CapacityMb   int    `help:"capacity, unit mb"`
}

func (opts *BackupStorageCreateOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}

type InstanceBackupListOptions struct {
	options.BaseListOptions
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
