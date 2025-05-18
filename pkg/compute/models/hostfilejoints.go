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
	"database/sql"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SHostFileJontsManager struct {
	db.SModelBaseManager
}

var HostFileJointsManager *SHostFileJontsManager

func init() {
	HostFileJointsManager = &SHostFileJontsManager{
		SModelBaseManager: db.NewModelBaseManager(
			SHostFileJoint{},
			"hostfilejoints_tbl",
			"hostfilejoint",
			"hostfilejoints",
		),
	}
	HostFileJointsManager.SetVirtualObject(HostFileJointsManager)
}

// +onecloud:model-api-gen
type SHostFileJoint struct {
	db.SModelBase

	HostId     string `width:"128" charset:"ascii" nullable:"false" primary:"true"`
	HostFileId string `width:"128" charset:"ascii" nullable:"false" primary:"true"`
	Deleted    bool   `nullable:"false"`
}

func (host *SHost) PerformSetHostFiles(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input computeapi.HostSetHostFilesInput,
) (jsonutils.JSONObject, error) {
	hostFileIds := make([]string, 0)
	for _, hostFileName := range input.HostFiles {
		hostFileObj, err := HostFileManager.FetchByIdOrName(ctx, userCred, hostFileName)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(HostFileManager.Keyword(), hostFileName)
			}
			return nil, errors.Wrap(err, "HostFileManager.FetchByIdOrName")
		}
		hostFile := hostFileObj.(*SHostFile)
		if hostFile.DomainId != host.DomainId {
			domains := hostFile.GetSharedDomains()
			if !utils.IsInArray(host.DomainId, domains) {
				return nil, errors.Wrapf(httperrors.ErrNoPermission, "host file %s not accessible to host %s", hostFile.Name, host.Name)
			}
		}
		hostFileIds = append(hostFileIds, hostFileObj.GetId())
	}

	err := HostFileJointsManager.setHostFiles(ctx, userCred, host.Id, hostFileIds)
	if err != nil {
		return nil, errors.Wrap(err, "HostFileJontsManager.setHostFiles")
	}

	return nil, nil
}

func (manager *SHostFileJontsManager) getHostFiles(hostId string) ([]SHostFileJoint, error) {
	q := manager.Query().Equals("host_id", hostId)
	hostFiles := make([]SHostFileJoint, 0)
	err := db.FetchModelObjects(manager, q, &hostFiles)
	if err != nil {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	return hostFiles, nil
}

func (manager *SHostFileJontsManager) setHostFiles(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	hostId string,
	hostFileIds []string,
) error {
	hostFiles, err := manager.getHostFiles(hostId)
	if err != nil {
		return errors.Wrap(err, "manager.getHostFiles")
	}

	existingIds := make(map[string]bool)

	for i := range hostFiles {
		existingIds[hostFiles[i].HostFileId] = true
		if utils.IsInStringArray(hostFiles[i].HostFileId, hostFileIds) {
			hostFiles[i].Deleted = false
		} else {
			hostFiles[i].Deleted = true
		}
	}

	for i := range hostFileIds {
		if _, ok := existingIds[hostFileIds[i]]; !ok {
			hostFileJoint := SHostFileJoint{
				HostId:     hostId,
				HostFileId: hostFileIds[i],
				Deleted:    false,
			}
			hostFileJoint.SetModelManager(manager, &hostFileJoint)
			hostFiles = append(hostFiles, hostFileJoint)
		}
	}

	errs := make([]error, 0)
	for i := range hostFiles {
		err := manager.TableSpec().InsertOrUpdate(ctx, &hostFiles[i])
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errors.NewAggregate(errs)
	}

	return nil
}

func (h *SHost) getHostFiles() ([]computeapi.SHostFile, error) {
	q := HostFileManager.Query()
	jointQ := HostFileJointsManager.Query().Equals("host_id", h.Id).SubQuery()
	q = q.Join(jointQ, sqlchemy.Equals(q.Field("id"), jointQ.Field("host_file_id")))

	hostFiles := make([]computeapi.SHostFile, 0)

	err := q.All(&hostFiles)
	if err != nil {
		return nil, errors.Wrap(err, "q.All")
	}

	return hostFiles, nil
}

func fetchHostHostFiles(hostIds []string) (map[string][]string, error) {
	q := HostFileJointsManager.Query().In("host_id", hostIds)
	hostFilesQ := HostFileManager.Query().SubQuery()
	q = q.Join(hostFilesQ, sqlchemy.Equals(q.Field("host_file_id"), hostFilesQ.Field("id")))
	q = q.AppendField(q.Field("host_id"))
	q = q.AppendField(hostFilesQ.Field("name"))

	results := make([]struct {
		HostId string
		Name   string
	}, 0)
	err := q.All(&results)
	if err != nil {
		return nil, errors.Wrap(err, "q.All")
	}

	hostFiles := make(map[string][]string)
	for _, result := range results {
		hostFiles[result.HostId] = append(hostFiles[result.HostId], result.Name)
	}

	return hostFiles, nil
}

func fetchHostFilesHosts(hostFileIds []string) (map[string][]string, error) {
	q := HostFileJointsManager.Query().In("host_file_id", hostFileIds)
	hostQ := HostManager.Query().SubQuery()
	q = q.Join(hostQ, sqlchemy.Equals(q.Field("host_id"), hostQ.Field("id")))
	q = q.AppendField(q.Field("host_file_id"))
	q = q.AppendField(hostQ.Field("name"))

	results := make([]struct {
		HostFileId string
		Name       string
	}, 0)
	err := q.All(&results)
	if err != nil {
		return nil, errors.Wrap(err, "q.All")
	}

	hostFiles := make(map[string][]string)
	for _, result := range results {
		hostFiles[result.HostFileId] = append(hostFiles[result.HostFileId], result.Name)
	}

	return hostFiles, nil
}

func (host *SHost) GetDetailsHostFiles(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
) (jsonutils.JSONObject, error) {
	hostFiles, err := host.getHostFiles()
	if err != nil {
		return nil, errors.Wrap(err, "GetHostFiles")
	}

	hostFilesObj := jsonutils.NewDict()
	hostFilesObj.Add(jsonutils.Marshal(hostFiles), "host_files")
	return hostFilesObj, nil
}
