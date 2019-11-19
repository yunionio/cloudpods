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
	"fmt"
	"strconv"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type SDnsRecordManager struct {
	db.SAdminSharableVirtualResourceBaseManager
}

var DnsRecordManager *SDnsRecordManager

func init() {
	DnsRecordManager = &SDnsRecordManager{
		SAdminSharableVirtualResourceBaseManager: db.NewAdminSharableVirtualResourceBaseManager(
			SDnsRecord{},
			"dnsrecord_tbl",
			"dnsrecord",
			"dnsrecords",
		),
	}
	DnsRecordManager.SetVirtualObject(DnsRecordManager)
}

const DNS_RECORDS_SEPARATOR = ","

type SDnsRecord struct {
	db.SAdminSharableVirtualResourceBase
	Ttl     int               `nullable:"true" default:"1" create:"optional" list:"user" update:"user"`
	Enabled tristate.TriState `nullable:"false" default:"true" create:"optional" list:"user"`
}

// GetRecordsSeparator implements IAdminSharableVirtualModelManager
func (man *SDnsRecordManager) GetRecordsSeparator() string {
	return DNS_RECORDS_SEPARATOR
}

// GetRecordsLimit implements IAdminSharableVirtualModelManager
func (man *SDnsRecordManager) GetRecordsLimit() int {
	return 0
}

// ParseInputInfo implements IAdminSharableVirtualModelManager
func (man *SDnsRecordManager) ParseInputInfo(data *jsonutils.JSONDict) ([]string, error) {
	records := []string{}
	for _, typ := range []string{"A", "AAAA"} {
		for i := 0; ; i++ {
			key := fmt.Sprintf("%s.%d", typ, i)
			if !data.Contains(key) {
				break
			}
			addr, err := data.GetString(key)
			if err != nil {
				return nil, err
			}
			if err := man.checkRecordValue(typ, addr); err != nil {
				return nil, err
			}
			records = append(records, fmt.Sprintf("%s:%s", typ, addr))
		}
	}
	{
		// - SRV.i
		// - (deprecated) SRV_host and SRV_port
		//
		// - rfc2782, A DNS RR for specifying the location of services (DNS SRV),
		//   https://tools.ietf.org/html/rfc2782
		parseSrvParam := func(s string) (string, error) {
			parts := strings.SplitN(s, ":", 4)
			if len(parts) < 2 {
				return "", httperrors.NewNotAcceptableError("SRV: insufficient param: %s", s)
			}
			host := parts[0]
			if err := man.checkRecordValue("SRV", host); err != nil {
				return "", err
			}
			port, err := strconv.Atoi(parts[1])
			if err != nil || port <= 0 || port >= 65536 {
				return "", httperrors.NewNotAcceptableError("SRV: invalid port number: %s", parts[1])
			}
			weight := 100
			priority := 0
			if len(parts) >= 3 {
				var err error
				weight, err = strconv.Atoi(parts[2])
				if err != nil {
					return "", httperrors.NewNotAcceptableError("SRV: invalid weight number: %s", parts[2])
				}
				if weight < 0 || weight > 65535 {
					return "", httperrors.NewNotAcceptableError("SRV: weight number %d not in range [0,65535]", weight)
				}
				if len(parts) >= 4 {
					priority, err = strconv.Atoi(parts[3])
					if err != nil {
						return "", httperrors.NewNotAcceptableError("SRV: invalid priority number: %s", parts[3])
					}
					if priority < 0 || priority > 65535 {
						return "", httperrors.NewNotAcceptableError("SRV: priority number %d not in range [0,65535]", priority)
					}
				}
			}
			rec := fmt.Sprintf("SRV:%s:%d:%d:%d", host, port, weight, priority)
			return rec, nil
		}
		recSrv := []string{}
		for i := 0; ; i++ {
			k := fmt.Sprintf("SRV.%d", i)
			if !data.Contains(k) {
				break
			}
			s, err := data.GetString(k)
			if err != nil {
				return nil, err
			}
			rec, err := parseSrvParam(s)
			if err != nil {
				return nil, err
			}
			recSrv = append(recSrv, rec)
		}
		if data.Contains("SRV_host") && data.Contains("SRV_port") {
			host, err := data.GetString("SRV_host")
			if err != nil {
				return nil, err
			}
			port, err := data.GetString("SRV_port")
			if err != nil {
				return nil, err
			}
			s := fmt.Sprintf("%s:%s", host, port)
			rec, err := parseSrvParam(s)
			if err != nil {
				return nil, err
			}
			recSrv = append(recSrv, rec)
		}
		if len(recSrv) > 0 {
			if len(records) > 0 {
				return nil, httperrors.NewNotAcceptableError("SRV cannot mix with other types")
			}
			records = recSrv
		}
	}
	if data.Contains("CNAME") {
		if len(records) > 0 {
			return nil, httperrors.NewNotAcceptableError("CNAME cannot mix with other types")
		}
		if cname, err := data.GetString("CNAME"); err != nil {
			return nil, err
		} else if err := man.checkRecordValue("CNAME", cname); err != nil {
			return nil, err
		} else {
			records = []string{fmt.Sprintf("%s:%s", "CNAME", cname)}
		}
	}
	if data.Contains("PTR") {
		if len(records) > 0 {
			return nil, httperrors.NewNotAcceptableError("PTR cannot mix with other types")
		}
		name, err := data.GetString("name")
		{
			if err != nil {
				return nil, err
			}
			if err := man.checkRecordName("PTR", name); err != nil {
				return nil, err
			}
		}
		domainName, err := data.GetString("PTR")
		{
			if err != nil {
				return nil, err
			}
			if err := man.checkRecordValue("PTR", domainName); err != nil {
				return nil, err
			}
		}
		records = []string{fmt.Sprintf("%s:%s", "PTR", domainName)}
	}
	return records, nil
}

func (man *SDnsRecordManager) getRecordsType(recs []string) string {
	for _, rec := range recs {
		switch typ := rec[:strings.Index(rec, ":")]; typ {
		case "A", "AAAA":
			return "A"
		case "CNAME":
			return "CNAME"
		case "SRV":
			return "SRV"
		case "PTR":
			return "PTR"
		}
	}
	return ""
}

func (man *SDnsRecordManager) checkRecordName(typ, name string) error {
	switch typ {
	case "A", "CNAME":
		if !regutils.MatchDomainName(name) {
			return httperrors.NewNotAcceptableError("%s: invalid domain name: %s", typ, name)
		}
	case "SRV":
		if !regutils.MatchDomainSRV(name) {
			return httperrors.NewNotAcceptableError("SRV: invalid srv record name: %s", typ, name)
		}
	case "PTR":
		if !regutils.MatchPtr(name) {
			return httperrors.NewNotAcceptableError("PTR: invalid ptr record name: %s", typ, name)
		}
	}
	if regutils.MatchIPAddr(name) {
		return httperrors.NewNotAcceptableError("%s: name cannot be ip address: %s", typ, name)
	}
	return nil
}

func (man *SDnsRecordManager) checkRecordValue(typ, val string) error {
	switch typ {
	case "A":
		if !regutils.MatchIP4Addr(val) {
			return httperrors.NewNotAcceptableError("A: record value must be ipv4 address: %s", val)
		}
	case "AAAA":
		if !regutils.MatchIP6Addr(val) {
			return httperrors.NewNotAcceptableError("AAAA: record value must be ipv6 address: %s", val)
		}
	case "CNAME", "PTR", "SRV":
		fieldMsg := "record value"
		if typ == "SRV" {
			fieldMsg = "target"
		}
		if !regutils.MatchDomainName(val) {
			return httperrors.NewNotAcceptableError("%s: %s must be domain name: %s", typ, fieldMsg, val)
		}
		if regutils.MatchIPAddr(val) {
			return httperrors.NewNotAcceptableError("%s: %s cannot be ip address: %s", typ, fieldMsg, val)
		}
	default:
		// internal error
		return httperrors.NewNotAcceptableError("%s: unknown record type", typ)
	}
	return nil
}

func (man *SDnsRecordManager) validateModelData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	data *jsonutils.JSONDict,
) (*jsonutils.JSONDict, error) {
	records, err := man.ParseInputInfo(data)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, httperrors.NewInputParameterError("Empty record")
	}
	recType := man.getRecordsType(records)
	name, err := data.GetString("name")
	if err != nil {
		return nil, err
	}
	err = man.checkRecordName(recType, name)
	if err != nil {
		return nil, err
	}
	if data.Contains("ttl") {
		jo, err := data.Get("ttl")
		if err != nil {
			return nil, err
		}
		ttl, err := jo.Int()
		if err != nil {
			return nil, httperrors.NewInputParameterError("invalid ttl: %s", err)
		}
		if ttl == 0 {
			// - Create: use the database default
			// - Update: unchanged
			data.Remove("ttl")
		} else if ttl < 0 || ttl > 0x7fffffff {
			// positive values of a signed 32 bit number.
			return nil, httperrors.NewInputParameterError("invalid ttl: %d", ttl)
		}
	}
	return data, err
}

func (man *SDnsRecordManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	data *jsonutils.JSONDict,
) (*jsonutils.JSONDict, error) {
	data, err := man.validateModelData(ctx, userCred, ownerId, query, data)
	if err != nil {
		return nil, err
	}
	return man.SAdminSharableVirtualResourceBaseManager.ValidateCreateData(man, data)
}

func (man *SDnsRecordManager) QueryDns(projectId, name string) *SDnsRecord {
	q := man.Query().
		Equals("name", name).
		IsTrue("enabled")
	if len(projectId) == 0 {
		q = q.IsTrue("is_public")
	} else {
		q = q.Filter(sqlchemy.OR(
			sqlchemy.IsTrue(q.Field("is_public")),
			sqlchemy.Equals(q.Field("tenant_id"), projectId),
		))
	}
	rec := &SDnsRecord{}
	rec.SetModelManager(DnsRecordManager, rec)
	if err := q.First(rec); err != nil {
		return nil
	}
	return rec
}

type DnsIp struct {
	Addr string
	Ttl  int
}

func (man *SDnsRecordManager) QueryDnsIps(projectId, name, kind string) []*DnsIp {
	rec := man.QueryDns(projectId, name)
	if rec == nil {
		return nil
	}
	pref := kind + ":"
	prefLen := len(pref)
	dnsIps := []*DnsIp{}
	for _, r := range rec.GetInfo() {
		if strings.HasPrefix(r, pref) {
			dnsIps = append(dnsIps, &DnsIp{
				Addr: r[prefLen:],
				Ttl:  rec.Ttl,
			})
		}
	}
	return dnsIps
}

func (rec *SDnsRecord) GetInfo() []string {
	return strings.Split(rec.Records, DNS_RECORDS_SEPARATOR)
}

func (rec *SDnsRecord) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	data.UpdateDefault(jsonutils.Marshal(rec))
	data, err := DnsRecordManager.validateModelData(ctx, userCred, rec.GetOwnerId(), query, data)
	if err != nil {
		return nil, err
	}
	{
		records, err := DnsRecordManager.ParseInputInfo(data)
		if err != nil {
			return nil, err
		}
		data.Set("records", jsonutils.NewString(strings.Join(records, DNS_RECORDS_SEPARATOR)))
	}
	return rec.SAdminSharableVirtualResourceBase.ValidateUpdateData(ctx, userCred, query, data)
}

func (rec *SDnsRecord) AddInfo(ctx context.Context, userCred mcclient.TokenCredential, data jsonutils.JSONObject) error {
	return rec.SAdminSharableVirtualResourceBase.AddInfo(ctx, userCred, DnsRecordManager, rec, data)
}

func (rec *SDnsRecord) AllowPerformAddRecords(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return rec.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, rec, "add-records")
}

func (rec *SDnsRecord) PerformAddRecords(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	records, err := DnsRecordManager.ParseInputInfo(data.(*jsonutils.JSONDict))
	if err != nil {
		return nil, err
	}
	oldRecs := rec.GetInfo()
	oldType := DnsRecordManager.getRecordsType(oldRecs)
	newType := DnsRecordManager.getRecordsType(records)
	if oldType != "" && oldType != newType {
		return nil, httperrors.NewNotAcceptableError("Cannot mix different types of records, %s != %s", oldType, newType)
	}
	err = rec.AddInfo(ctx, userCred, data)
	return nil, err
}

func (rec *SDnsRecord) AllowPerformRemoveRecords(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return rec.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, rec, "remove-records")
}

func (rec *SDnsRecord) PerformRemoveRecords(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	err := rec.SAdminSharableVirtualResourceBase.RemoveInfo(ctx, userCred, DnsRecordManager, rec, data, false)
	return nil, err
}

func (rec *SDnsRecord) AllowPerformEnable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return rec.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, rec, "enable")
}

func (rec *SDnsRecord) PerformEnable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if rec.Enabled.IsFalse() {
		diff, err := db.Update(rec, func() error {
			rec.Enabled = tristate.True
			return nil
		})
		if err != nil {
			log.Errorf("enabling dnsrecords for %s failed: %s", rec.Name, err)
			return nil, err
		}
		db.OpsLog.LogEvent(rec, db.ACT_ENABLE, diff, userCred)
		logclient.AddActionLogWithContext(ctx, rec, logclient.ACT_ENABLE, diff, userCred, true)
	}
	return nil, nil
}

func (rec *SDnsRecord) AllowPerformDisable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return rec.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, rec, "disable")
}

func (rec *SDnsRecord) PerformDisable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if rec.Enabled.IsTrue() {
		diff, err := db.Update(rec, func() error {
			rec.Enabled = tristate.False
			return nil
		})
		if err != nil {
			log.Errorf("disabling dnsrecords for %s failed: %s", rec.Name, err)
			return nil, err
		}
		db.OpsLog.LogEvent(rec, db.ACT_DISABLE, diff, userCred)
		logclient.AddActionLogWithContext(ctx, rec, logclient.ACT_DISABLE, diff, userCred, true)
	}
	return nil, nil
}
