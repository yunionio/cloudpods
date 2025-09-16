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
	"fmt"
	"regexp"
	"sort"
	"strings"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

// +onecloud:swagger-gen-model-singular=dnsrecord
// +onecloud:swagger-gen-model-plural=dnsrecords
type SDnsRecordManager struct {
	db.SEnabledStatusStandaloneResourceBaseManager
	db.SExternalizedResourceBaseManager

	SDnsZoneResourceBaseManager
}

var DnsRecordManager *SDnsRecordManager

func init() {
	DnsRecordManager = &SDnsRecordManager{
		SEnabledStatusStandaloneResourceBaseManager: db.NewEnabledStatusStandaloneResourceBaseManager(
			SDnsRecord{},
			"dns_records_tbl",
			"dnsrecord",
			"dnsrecords",
		),
	}
	DnsRecordManager.SetVirtualObject(DnsRecordManager)
}

type SDnsRecord struct {
	db.SEnabledStatusStandaloneResourceBase
	db.SExternalizedResourceBase
	SDnsZoneResourceBase

	DnsType    string `width:"36" charset:"ascii" nullable:"false" list:"user" update:"user" create:"required"`
	DnsValue   string `width:"256" charset:"ascii" nullable:"false" list:"user" update:"user" create:"required"`
	TTL        int64  `nullable:"false" list:"user" update:"user" create:"required" json:"ttl"`
	MxPriority int64  `nullable:"false" list:"user" update:"user" create:"optional"`

	// 解析线路类型
	PolicyType string `width:"36" charset:"ascii" nullable:"false" list:"user" update:"user" create:"optional"`
	// 解析线路
	PolicyValue string `width:"256" charset:"ascii" nullable:"false" list:"user" update:"user" create:"optional"`

	// 目前存储阿里云GTM设置地址及AWS TrafficPolicy端点地址, 仅支持同步
	ExtraAddresses []string `width:"512" charset:"utf8" nullable:"true" list:"user"`
}

func (manager *SDnsRecordManager) EnableGenerateName() bool {
	return false
}

// 创建
func (manager *SDnsRecordManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input *api.DnsRecordCreateInput,
) (*api.DnsRecordCreateInput, error) {
	var err error
	input.EnabledStatusStandaloneResourceCreateInput, err = manager.SEnabledStatusStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.EnabledStatusStandaloneResourceCreateInput)
	if err != nil {
		return nil, err
	}
	input.Name = strings.ToLower(input.Name)
	rrRegexp := regexp.MustCompile(`^(([a-zA-Z0-9_][a-zA-Z0-9_-]{0,61}[a-zA-Z0-9_])|[a-zA-Z0-9_]|\*{1}){1}(\.(([a-zA-Z0-9_][a-zA-Z0-9_-]{0,61}[a-zA-Z0-9_])|[a-zA-Z0-9_]))*$|^@{1}$`)
	if !rrRegexp.MatchString(input.Name) {
		return nil, httperrors.NewInputParameterError("invalid record name %s", input.Name)
	}

	_, err = validators.ValidateModel(ctx, userCred, DnsZoneManager, &input.DnsZoneId)
	if err != nil {
		return nil, err
	}

	record := api.SDnsRecord{}
	record.DnsZoneId = input.DnsZoneId
	record.DnsType = input.DnsType
	record.DnsValue = input.DnsValue
	record.TTL = input.TTL
	record.MxPriority = input.MxPriority

	err = record.ValidateDnsrecordValue()
	if err != nil {
		return input, err
	}

	// 处理重复的记录

	// CNAME  dnsName不能和其他类型record相同

	// 同dnsName 同dnsType重复检查
	// 检查dnsrecord 是否通过policy重复
	// simple类型不能重复，不能和其他policy重复
	// 不同类型policy不能重复
	// 同类型policy的dnsrecord重复时，需要通过policyvalue区别

	// validate name type
	q := DnsRecordManager.Query().Equals("dns_zone_id", input.DnsZoneId).Equals("name", input.Name)
	recordTypeQuery := q
	switch input.DnsType {
	case "CNAME":
		recordTypeQuery = recordTypeQuery.NotEquals("dns_type", "CNAME")
	default:
		recordTypeQuery = recordTypeQuery.Equals("dns_type", "CNAME")
	}

	cnt, err := recordTypeQuery.CountWithError()
	if err != nil {
		return input, httperrors.NewGeneralError(err)
	}
	if cnt > 0 {
		return input, httperrors.NewNotSupportedError("duplicated with CNAME dnsrecord name not support")
	}
	input.Status = api.DNS_RECORDSET_STATUS_CREATING
	return input, nil
}

func (self *SDnsRecord) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SEnabledStatusStandaloneResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	self.StartCreateTask(ctx, userCred, "")
}

func (self *SDnsRecord) StartCreateTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	params := jsonutils.NewDict()
	task, err := taskman.TaskManager.NewTask(ctx, "DnsRecordCreateTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	self.SetStatus(ctx, userCred, api.DNS_RECORDSET_STATUS_CREATING, "")
	return task.ScheduleRun(nil)
}

// DNS记录列表
func (manager *SDnsRecordManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.DnsRecordListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SEnabledStatusStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.EnabledStatusStandaloneResourceListInput)
	if err != nil {
		return nil, err
	}
	q, err = manager.SDnsZoneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.DnsZoneFilterListBase)
	if err != nil {
		return nil, err
	}
	return q, nil
}

// 解析详情
func (manager *SDnsRecordManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.DnsRecordDetails {
	rows := make([]api.DnsRecordDetails, len(objs))
	enRows := manager.SEnabledStatusStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	zoneIds := make([]string, len(objs))
	for i := range rows {
		rows[i] = api.DnsRecordDetails{
			EnabledStatusStandaloneResourceDetails: enRows[i],
		}
		record := objs[i].(*SDnsRecord)
		zoneIds[i] = record.DnsZoneId
	}

	zoneNames, err := db.FetchIdNameMap2(DnsZoneManager, zoneIds)
	if err != nil {
		return rows
	}
	for i := range rows {
		rows[i].DnsZone, _ = zoneNames[zoneIds[i]]
	}

	return rows
}

func (self *SDnsRecord) ToZoneLine() string {
	result := self.Name + "\t" + fmt.Sprint(self.TTL) + "\tIN\t" + self.DnsType + "\t"
	if self.MxPriority != 0 {
		result += fmt.Sprint(self.MxPriority) + "\t"
	}
	result += self.DnsValue
	if self.DnsType == "CNAME" || self.DnsType == "MX" || self.DnsType == "SRV" {
		result += "."
	}
	return result
}

func (self *SDnsRecord) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (self *SDnsRecord) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return self.SEnabledStatusStandaloneResourceBase.Delete(ctx, userCred)
}

type sRecordUniqValues struct {
	DnsZoneId string
	DnsType   string
	DnsName   string
	DnsValue  string
}

func (self *SDnsRecord) GetUniqValues() jsonutils.JSONObject {
	return jsonutils.Marshal(sRecordUniqValues{
		DnsZoneId: self.DnsZoneId,
		DnsName:   self.Name,
		DnsType:   self.DnsType,
		DnsValue:  self.DnsValue,
	})
}

func (manager *SDnsRecordManager) FetchUniqValues(ctx context.Context, data jsonutils.JSONObject) jsonutils.JSONObject {
	values := &sRecordUniqValues{}
	data.Unmarshal(values)
	return jsonutils.Marshal(values)
}

func (manager *SDnsRecordManager) FilterByUniqValues(q *sqlchemy.SQuery, values jsonutils.JSONObject) *sqlchemy.SQuery {
	uniq := &sRecordUniqValues{}
	values.Unmarshal(uniq)
	if len(uniq.DnsZoneId) > 0 {
		q = q.Equals("dns_zone_id", uniq.DnsZoneId)
	}
	if len(uniq.DnsName) > 0 {
		q = q.Equals("name", uniq.DnsName)
	}
	if len(uniq.DnsType) > 0 {
		q = q.Equals("dns_type", uniq.DnsType)
	}
	if len(uniq.DnsValue) > 0 {
		q = q.Equals("dns_value", uniq.DnsValue)
	}

	return q
}

func (manager *SDnsRecordManager) FetchOwnerId(ctx context.Context, data jsonutils.JSONObject) (mcclient.IIdentityProvider, error) {
	dnsZoneId, _ := data.GetString("dns_zone_id")
	if len(dnsZoneId) > 0 {
		dnsZone, err := db.FetchById(DnsZoneManager, dnsZoneId)
		if err != nil {
			return nil, errors.Wrapf(err, "db.FetchById(DnsZoneManager, %s)", dnsZoneId)
		}
		return dnsZone.(*SDnsZone).GetOwnerId(), nil
	}
	return db.FetchDomainInfo(ctx, data)
}

func (manager *SDnsRecordManager) FilterByOwner(ctx context.Context, q *sqlchemy.SQuery, man db.FilterByOwnerProvider, userCred mcclient.TokenCredential, owner mcclient.IIdentityProvider, scope rbacscope.TRbacScope) *sqlchemy.SQuery {
	sq := DnsZoneManager.Query("id")
	sq = db.SharableManagerFilterByOwner(ctx, DnsZoneManager, sq, userCred, owner, scope)
	return q.In("dns_zone_id", sq.SubQuery())
}

func (manager *SDnsRecordManager) ResourceScope() rbacscope.TRbacScope {
	return rbacscope.ScopeDomain
}

func (self *SDnsRecord) IsSharable(reqUsrId mcclient.IIdentityProvider) bool {
	dnsZone, err := self.GetDnsZone()
	if err != nil {
		return false
	}
	return dnsZone.IsSharable(reqUsrId)
}

func (self *SDnsRecord) GetOwnerId() mcclient.IIdentityProvider {
	dnsZone, err := self.GetDnsZone()
	if err != nil {
		return nil
	}
	return dnsZone.GetOwnerId()
}

func (self *SDnsRecord) syncRemove(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)

	return self.RealDelete(ctx, userCred)
}

func (self *SDnsRecord) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return self.StartDeleteTask(ctx, userCred, "")
}

func (self *SDnsRecord) StartDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	params := jsonutils.NewDict()
	task, err := taskman.TaskManager.NewTask(ctx, "DnsRecordDeleteTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	self.SetStatus(ctx, userCred, apis.STATUS_DELETING, "")
	return task.ScheduleRun(nil)
}

// 更新
func (self *SDnsRecord) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input *api.DnsRecordUpdateInput) (*api.DnsRecordUpdateInput, error) {
	var err error
	input.EnabledStatusStandaloneResourceBaseUpdateInput, err = self.SEnabledStatusStandaloneResourceBase.ValidateUpdateData(ctx, userCred, query, input.EnabledStatusStandaloneResourceBaseUpdateInput)
	if err != nil {
		return input, err
	}

	dnsZone, err := self.GetDnsZone()
	if err != nil {
		return input, httperrors.NewGeneralError(errors.Wrapf(err, "GetDnsZone"))
	}

	record := api.SDnsRecord{
		DnsType:  self.DnsType,
		DnsValue: self.DnsValue,
	}

	if len(input.DnsType) == 0 {
		record.DnsType = input.DnsType
	}
	if len(input.DnsValue) == 0 {
		record.DnsValue = input.DnsValue
	}

	err = record.ValidateDnsrecordValue()
	if err != nil {
		return input, err
	}

	// 处理重复的记录

	// CNAME  dnsName不能和其他类型record相同

	// 同dnsName 同dnsType重复检查
	// 检查dnsrecord 是否通过policy重复
	// simple类型不能重复，不能和其他policy重复
	// 不同类型policy不能重复
	// 同类型policy的dnsrecord重复时，需要通过policyvalue区别

	// validate name type
	q := DnsRecordManager.Query().Equals("dns_zone_id", dnsZone.Id).NotEquals("id", self.Id).Equals("name", input.Name)
	recordTypeQuery := q
	switch input.DnsType {
	case "CNAME":
		recordTypeQuery = recordTypeQuery.NotEquals("dns_type", "CNAME")
	default:
		recordTypeQuery = recordTypeQuery.Equals("dns_type", "CNAME")
	}

	cnt, err := recordTypeQuery.CountWithError()
	if err != nil {
		return input, httperrors.NewGeneralError(err)
	}
	if cnt > 0 {
		return input, httperrors.NewNotSupportedError("duplicated with CNAME dnsrecord name not support")
	}

	return input, nil
}

func (self *SDnsRecord) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SEnabledStatusStandaloneResourceBase.PostUpdate(ctx, userCred, query, data)
	self.StartUpdateTask(ctx, userCred, "")
}

func (self *SDnsRecord) StartUpdateTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	params := jsonutils.NewDict()
	task, err := taskman.TaskManager.NewTask(ctx, "DnsRecordUpdateTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	self.SetStatus(ctx, userCred, apis.STATUS_SYNC_STATUS, "")
	return task.ScheduleRun(nil)
}

func (self *SDnsRecord) GetDnsZone() (*SDnsZone, error) {
	dnsZone, err := DnsZoneManager.FetchById(self.DnsZoneId)
	if err != nil {
		return nil, errors.Wrapf(err, "DnsZoneManager.FetchById(%s)", self.DnsZoneId)
	}
	return dnsZone.(*SDnsZone), nil
}

func (self *SDnsZone) SyncRecords(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ICloudDnsZone, xor bool) compare.SyncResult {
	lockman.LockRawObject(ctx, self.Keyword(), self.Id)
	defer lockman.ReleaseRawObject(ctx, self.Keyword(), self.Id)

	result := compare.SyncResult{}

	records, err := ext.GetIDnsRecords()
	if err != nil {
		result.Error(errors.Wrapf(err, "GetIDnsRecords"))
		return result
	}

	dbRecords, err := self.GetDnsRecords()
	if err != nil {
		result.Error(err)
		return result
	}

	removed := make([]SDnsRecord, 0)
	commondb := make([]SDnsRecord, 0)
	commonext := make([]cloudprovider.ICloudDnsRecord, 0)
	added := make([]cloudprovider.ICloudDnsRecord, 0)

	err = compare.CompareSets(dbRecords, records, &removed, &commondb, &commonext, &added)
	if err != nil {
		result.Error(err)
		return result
	}

	for i := 0; i < len(removed); i += 1 {
		err = removed[i].syncRemove(ctx, userCred)
		if err != nil {
			result.DeleteError(err)
			continue
		}
		result.Delete()
	}

	if !xor {
		for i := 0; i < len(commondb); i += 1 {
			err = commondb[i].syncWithDnsRecord(ctx, userCred, commonext[i])
			if err != nil {
				result.UpdateError(err)
				continue
			}
			result.Update()
		}
	}

	for i := 0; i < len(added); i += 1 {
		err := self.newFromCloudDnsRecord(ctx, userCred, added[i])
		if err != nil {
			result.AddError(err)
			continue
		}
		result.Add()
	}

	return result
}

func (self *SDnsRecord) syncWithDnsRecord(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ICloudDnsRecord) error {
	diff, err := db.Update(self, func() error {
		self.Name = ext.GetDnsName()
		self.Enabled = tristate.NewFromBool(ext.GetEnabled())
		self.Status = ext.GetStatus()
		self.TTL = ext.GetTTL()
		self.MxPriority = ext.GetMxPriority()
		self.DnsType = string(ext.GetDnsType())
		self.DnsValue = ext.GetDnsValue()
		self.PolicyType = string(ext.GetPolicyType())
		self.PolicyValue = string(ext.GetPolicyValue())
		extraAddresses, err := ext.GetExtraAddresses()
		if err != nil {
			log.Errorf("GetExtraAddresses for record %s error: %v", self.Name, err)
			return nil
		}
		self.ExtraAddresses = extraAddresses
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "update")
	}
	if len(diff) > 0 {
		notifyclient.EventNotify(ctx, userCred, notifyclient.SEventNotifyParam{
			Obj:    self,
			Action: notifyclient.ActionSyncUpdate,
		})
	}
	return nil
}

func (self *SDnsZone) newFromCloudDnsRecord(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ICloudDnsRecord) error {
	record := &SDnsRecord{}
	record.SetModelManager(DnsRecordManager, record)
	record.DnsZoneId = self.Id
	record.Name = ext.GetDnsName()
	record.Status = ext.GetStatus()
	record.Enabled = tristate.NewFromBool(ext.GetEnabled())
	record.TTL = ext.GetTTL()
	record.MxPriority = ext.GetMxPriority()
	record.DnsType = string(ext.GetDnsType())
	record.DnsValue = ext.GetDnsValue()
	record.ExternalId = ext.GetGlobalId()
	record.PolicyType = string(ext.GetPolicyType())
	record.PolicyValue = string(ext.GetPolicyValue())
	record.ExtraAddresses, _ = ext.GetExtraAddresses()

	err := DnsRecordManager.TableSpec().Insert(ctx, record)
	if err != nil {
		return errors.Wrapf(err, "Insert")
	}

	notifyclient.EventNotify(ctx, userCred, notifyclient.SEventNotifyParam{
		Obj:    record,
		Action: notifyclient.ActionSyncCreate,
	})

	return nil
}

type sDnsResolveResults []api.SDnsResolveResult

func (a sDnsResolveResults) Len() int      { return len(a) }
func (a sDnsResolveResults) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a sDnsResolveResults) Less(i, j int) bool {
	partsI := strings.Split(a[i].DnsName, ".")
	partsJ := strings.Split(a[j].DnsName, ".")
	if len(partsI) > len(partsJ) {
		return true
	} else if len(partsI) < len(partsJ) {
		return false
	}
	if len(partsI) > 0 && partsI[0] != partsJ[0] {
		if partsI[0] == "*" {
			return false
		} else if partsJ[0] == "*" {
			return true
		}
	}
	if a[i].DnsName < a[j].DnsName {
		return true
	} else if a[i].DnsName > a[j].DnsName {
		return false
	}
	return false
}

func (man *SDnsRecordManager) QueryPtr(projectId, ip string) ([]api.SDnsResolveResult, error) {
	zonesQ := DnsZoneManager.Query().IsNullOrEmpty("manager_id").IsTrue("enabled")
	if len(projectId) == 0 {
		zonesQ = zonesQ.IsTrue("is_public")
	} else {
		zonesQ = zonesQ.Filter(sqlchemy.OR(
			sqlchemy.IsTrue(zonesQ.Field("is_public")),
			sqlchemy.Equals(zonesQ.Field("tenant_id"), projectId),
		))
	}
	zones := zonesQ.SubQuery()
	recQ := DnsRecordManager.Query().IsTrue("enabled")
	if regutils.MatchIP6Addr(ip) {
		recQ = recQ.Equals("dns_type", "AAAA")
	} else {
		recQ = recQ.Equals("dns_type", "A")
	}
	recSQ := recQ.SubQuery()

	rec := recSQ.Query(
		recSQ.Field("dns_value"),
		recSQ.Field("ttl"),
		sqlchemy.CONCAT("dns_name", recSQ.Field("name"), sqlchemy.NewStringField("."), zones.Field("name")),
	).Join(zones, sqlchemy.Equals(recSQ.Field("dns_zone_id"), zones.Field("id")))

	sq := rec.SubQuery()

	q := sq.Query().Equals("dns_value", ip)

	results := make([]api.SDnsResolveResult, 0)
	err := q.All(&results)
	if err != nil && errors.Cause(err) != sql.ErrNoRows {
		return nil, errors.Wrap(err, "FetchModelObjects")
	}

	return results, nil
}

func (man *SDnsRecordManager) QueryDns(projectId, name, kind string) ([]api.SDnsResolveResult, error) {
	name = strings.TrimSuffix(name, ".")

	zonesQ := DnsZoneManager.Query().IsNullOrEmpty("manager_id").IsTrue("enabled")
	if len(projectId) == 0 {
		zonesQ = zonesQ.IsTrue("is_public")
	} else {
		zonesQ = zonesQ.Filter(sqlchemy.OR(
			sqlchemy.IsTrue(zonesQ.Field("is_public")),
			sqlchemy.Equals(zonesQ.Field("tenant_id"), projectId),
		))
	}
	zones := zonesQ.SubQuery()
	recQ := DnsRecordManager.Query().IsTrue("enabled")
	if len(kind) > 0 {
		recQ = recQ.Equals("dns_type", kind)
	}
	recSQ := recQ.SubQuery()

	rec := recSQ.Query(
		recSQ.Field("dns_value"),
		recSQ.Field("ttl"),
		sqlchemy.CONCAT("dns_name", recSQ.Field("name"), sqlchemy.NewStringField("."), zones.Field("name")),
	).Join(zones, sqlchemy.Equals(recSQ.Field("dns_zone_id"), zones.Field("id")))

	sq := rec.SubQuery()

	filters := sqlchemy.OR(
		// example match
		sqlchemy.Equals(sq.Field("dns_name"), name),
		// support *.example.com resolve
		sqlchemy.Equals(sq.Field("dns_name"), "*."+name),
	)

	strs := strings.Split(name, ".")
	if len(strs) > 2 {
		filters = sqlchemy.OR(
			// example match
			sqlchemy.Equals(sq.Field("dns_name"), name),
			// root match, support resolve example.com to *.example.com
			sqlchemy.Equals(sq.Field("dns_name"), "*."+name),
			// support resolve office.example.com to *.example.com
			sqlchemy.Equals(sq.Field("dns_name"), "*."+strings.Join(strs[1:], ".")),
			// support resolve saml.office.example.com to office.example.com
			sqlchemy.Equals(sq.Field("dns_name"), strings.Join(strs[1:], ".")),
		)
	}

	q := sq.Query().Filter(filters)

	results := make([]api.SDnsResolveResult, 0)
	err := q.All(&results)
	if err != nil && errors.Cause(err) != sql.ErrNoRows {
		return nil, errors.Wrap(err, "FetchModelObjects")
	}

	if len(results) == 0 {
		return results, nil
	}

	sort.Sort(sDnsResolveResults(results))
	ret := make([]api.SDnsResolveResult, 0, len(results))
	for i := range results {
		if i == 0 || results[i].DnsName == ret[0].DnsName {
			ret = append(ret, results[i])
		} else {
			break
		}
	}

	return ret, nil
}

func (self *SDnsRecord) IsCNAME() bool {
	return strings.ToUpper(self.DnsType) == "CNAME"
}

func (self *SDnsRecord) GetCNAME() string {
	return self.DnsValue
}

// 启用
func (self *SDnsRecord) PerformEnable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.DnsRecordEnableInput) (jsonutils.JSONObject, error) {
	_, err := self.SEnabledStatusStandaloneResourceBase.PerformEnable(ctx, userCred, query, input.PerformEnableInput)
	if err != nil {
		return nil, err
	}
	if err != nil {
		return nil, httperrors.NewGeneralError(errors.Wrapf(err, "GetDnsZone"))
	}
	return nil, self.StartSetEnabledTask(ctx, userCred, "")
}

func (self *SDnsRecord) StartSetEnabledTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	params := jsonutils.NewDict()
	task, err := taskman.TaskManager.NewTask(ctx, "DnsRecordSetEnabledTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	self.SetStatus(ctx, userCred, apis.STATUS_SYNC_STATUS, "")
	return task.ScheduleRun(nil)
}

// 禁用
func (self *SDnsRecord) PerformDisable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.DnsRecordDisableInput) (jsonutils.JSONObject, error) {
	_, err := self.SEnabledStatusStandaloneResourceBase.PerformDisable(ctx, userCred, query, input.PerformDisableInput)
	if err != nil {
		return nil, err
	}
	if err != nil {
		return nil, httperrors.NewGeneralError(errors.Wrapf(err, "GetDnsZone"))
	}
	return nil, self.StartSetEnabledTask(ctx, userCred, "")
}
