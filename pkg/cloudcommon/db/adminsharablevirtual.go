package db

import (
	"strings"

	"github.com/yunionio/jsonutils"
	"github.com/yunionio/onecloud/pkg/mcclient"
	"github.com/yunionio/onecloud/pkg/httperrors"
	"github.com/yunionio/pkg/utils"
)

type SAdminSharableVirtualResourceBase struct {
	SSharableVirtualResourceBase
	Records string `charset:"ascii" list:"user" create:"optional" update:"user"`
}

type SAdminSharableVirtualResourceBaseManager struct {
	SSharableVirtualResourceBaseManager
	RecordsSeparator string
	RecordsLimit     int
}

func NewAdminSharableVirtualResourceBaseManager(dt interface{}, tableName string, keyword string, keywordPlural string) SAdminSharableVirtualResourceBaseManager {
	manager := SAdminSharableVirtualResourceBaseManager{SSharableVirtualResourceBaseManager: NewSharableVirtualResourceBaseManager(dt, tableName, keyword, keywordPlural)}
	manager.RecordsSeparator = ";"
	manager.RecordsLimit = 0
	return manager
}

func (manager *SAdminSharableVirtualResourceBaseManager) ValidateCreateData(man IAdminSharableVirtualModelManager, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	records, err := man.ParseInputInfo(data)
	if err != nil {
		return nil, err
	}
	data.Add(jsonutils.NewString(strings.Join(records, man.GetRecordsSeparator())), "records")
	return data, nil
}

func (model *SAdminSharableVirtualResourceBase) SetInfo(
	userCred mcclient.TokenCredential,
	man IAdminSharableVirtualModelManager,
	obj IAdminSharableVirtualModel,
	data *jsonutils.JSONDict,
) error {
	records, err := man.ParseInputInfo(data)
	if err != nil {
		return err
	}
	oldRecs := obj.GetInfo()
	isChanged := false
	if len(records) != len(oldRecs) {
		isChanged = true
	} else {
		for _, rec := range records {
			if !utils.IsInStringArray(rec, oldRecs) {
				isChanged = true
				break
			}
		}
	}
	if isChanged {
		return model.setInfo(userCred, man, records)
	}
	return nil
}

func (model *SAdminSharableVirtualResourceBase) AddInfo(
	userCred mcclient.TokenCredential,
	man IAdminSharableVirtualModelManager,
	obj IAdminSharableVirtualModel,
	data jsonutils.JSONObject,
) error {
	records, err := man.ParseInputInfo(data.(*jsonutils.JSONDict))
	if err != nil {
		return err
	}
	oldRecs := obj.GetInfo()
	adds := []string{}
	for _, r := range records {
		if !utils.IsInStringArray(r, oldRecs) {
			oldRecs = append(oldRecs, r)
			adds = append(adds, r)
		}
	}
	if len(adds) > 0 {
		return model.setInfo(userCred, man, oldRecs)
	}
	return nil
}

func (model *SAdminSharableVirtualResourceBase) RemoveInfo(
	userCred mcclient.TokenCredential,
	man IAdminSharableVirtualModelManager,
	obj IAdminSharableVirtualModel,
	data jsonutils.JSONObject,
	allowEmpty bool,
) error {
	records, err := man.ParseInputInfo(data.(*jsonutils.JSONDict))
	if err != nil {
		return err
	}
	oldRecs := obj.GetInfo()
	removes := []string{}
	for _, rec := range records {
		if ok, idx := utils.InStringArray(rec, oldRecs); ok {
			oldRecs = append(oldRecs[:idx], oldRecs[idx+1:]...) // remove record
			removes = append(removes, rec)
		}
	}
	if len(oldRecs) == 0 && !allowEmpty {
		return httperrors.NewNotAcceptableError("Not allow empty records")
	}
	if len(removes) > 0 {
		return model.setInfo(userCred, man, oldRecs)
	}
	return nil
}

func (model *SAdminSharableVirtualResourceBase) setInfo(
	userCred mcclient.TokenCredential,
	man IAdminSharableVirtualModelManager,
	records []string,
) error {
	if man.GetRecordsLimit() > 0 && len(records) > man.GetRecordsLimit() {
		return httperrors.NewNotAcceptableError("Records limit exceeded.")
	}
	diff, err := model.GetModelManager().TableSpec().Update(model, func() error {
		model.Records = strings.Join(records, man.GetRecordsSeparator())
		return nil
	})
	if err != nil {
		return err
	}
	if diff != nil {
		OpsLog.LogEvent(model, ACT_UPDATE, diff, userCred)
	}
	return err
}
