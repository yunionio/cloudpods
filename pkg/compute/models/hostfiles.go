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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SHostFileManager struct {
	db.SInfrasResourceBaseManager
}

var HostFileManager *SHostFileManager

func init() {
	HostFileManager = &SHostFileManager{
		SInfrasResourceBaseManager: db.NewInfrasResourceBaseManager(
			SHostFile{},
			"hostfiles_tbl",
			"hostfile",
			"hostfiles",
		),
	}
	HostFileManager.SetVirtualObject(HostFileManager)
}

// +onecloud:model-api-gen
type SHostFile struct {
	db.SInfrasResourceBase

	Type    computeapi.HostFileType `width:"64" charset:"ascii" nullable:"false" list:"domain" create:"domain_required"`
	Path    string                  `width:"256" charset:"utf8" nullable:"true" list:"domain" update:"domain" create:"domain_optional"`
	Content string                  `charset:"utf8" nullable:"true" get:"domain" update:"domain" create:"domain_optional"`
}

func (manager *SHostFileManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input computeapi.HostFileCreateInput,
) (computeapi.HostFileCreateInput, error) {
	var err error

	input.InfrasResourceBaseCreateInput, err = manager.SInfrasResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.InfrasResourceBaseCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "SInfrasResourceBaseManager.ValidateCreateData")
	}

	if len(input.Type) == 0 {
		return input, httperrors.NewInputParameterError("Type is required")
	}

	return input, nil
}

func (manager *SHostFileManager) ValidateUpdateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input computeapi.HostFileUpdateInput,
) (computeapi.HostFileUpdateInput, error) {
	return input, nil
}

func (manager *SHostFileManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query computeapi.HostFileListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SInfrasResourceBaseManager.ListItemFilter(ctx, q, userCred, query.InfrasResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SInfrasResourceBaseManager.ListItemFilter")
	}

	if len(query.Type) > 0 {
		q = q.In("type", query.Type)
	}

	if len(query.Path) > 0 {
		q = q.Equals("path", query.Path)
	}

	return q, nil
}

func (manager *SHostFileManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.HostFileDetails {
	rows := make([]api.HostFileDetails, len(objs))
	infrasRows := manager.SInfrasResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	hostFileIds := make([]string, 0, len(objs))
	for i := range rows {
		rows[i] = api.HostFileDetails{
			InfrasResourceBaseDetails: infrasRows[i],
		}
		hostFileIds = append(hostFileIds, objs[i].(*SHostFile).Id)
	}
	hostFiles, err := fetchHostFilesHosts(hostFileIds)
	if err != nil {
		log.Errorf("fetchHostFilesHosts error: %v", err)
	}
	for i := range rows {
		rows[i].Hosts = hostFiles[hostFileIds[i]]
	}
	return rows
}

func (hostfile *SHostFile) ValidateDeleteCondition(ctx context.Context, info api.HostFileDetails) error {
	err := hostfile.SInfrasResourceBase.ValidateDeleteCondition(ctx, jsonutils.Marshal(info))
	if err != nil {
		return errors.Wrap(err, "SInfrasResourceBase.ValidateDeleteCondition")
	}
	if len(info.Hosts) > 0 {
		return httperrors.NewNotEmptyError("hostfile is used by hosts")
	}
	return nil
}
