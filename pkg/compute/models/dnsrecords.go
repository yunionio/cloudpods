package models

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SDnsRecordManager struct {
	db.SAdminSharableVirtualResourceBaseManager
}

var DnsRecordManager *SDnsRecordManager

func init() {
	DnsRecordManager = &SDnsRecordManager{SAdminSharableVirtualResourceBaseManager: db.NewAdminSharableVirtualResourceBaseManager(SDnsRecord{}, "dnsrecord_tbl", "dnsrecord", "dnsrecords")}
}

const DNS_RECORDS_SEPARATOR = ","

type SDnsRecord struct {
	db.SAdminSharableVirtualResourceBase
	Ttl     int  `nullable:"true" default:"1" create:"optional" list:"user" update:"user"`
	Enabled bool `nullable:"false" default:"true" create:"optional" list:"user"`
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
			if (typ == "A" && !regutils.MatchIP4Addr(addr)) || (typ == "AAAA" && !regutils.MatchIP6Addr(addr)) {
				return nil, httperrors.NewNotAcceptableError("Invalid type %s address: %s", typ, addr)
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
			if !regutils.MatchDomainName(host) &&
				!regutils.MatchIPAddr(host) {
				return "", httperrors.NewNotAcceptableError("SRV: invalid host part: %s", host)
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
		} else if !regutils.MatchDomainName(cname) {
			return nil, httperrors.NewNotAcceptableError(fmt.Sprintf("Invalid type CNAME domain %s", cname))
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
			if !regutils.MatchPtr(name) {
				return nil, httperrors.NewNotAcceptableError(fmt.Sprintf("Invalid ptr %s", name))
			}
		}
		domainName, err := data.GetString("PTR")
		{
			if err != nil {
				return nil, err
			}
			if !regutils.MatchDomainName(domainName) {
				return nil, httperrors.NewNotAcceptableError(fmt.Sprintf("Invalid domain %s", domainName))
			}
		}
		records = []string{fmt.Sprintf("%s:%s", "PTR", domainName)}
	}
	return records, nil
}

func (man *SDnsRecordManager) GetRecordsType(recs []string) string {
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

func (man *SDnsRecordManager) CheckNameForDnsType(name, recType string) (err error) {
	switch recType {
	case "A", "CNAME":
		if !regutils.MatchDomainName(name) {
			err = httperrors.NewNotAcceptableError(fmt.Sprintf("Invalid domain name %s for type %s", name, recType))
		}
	case "SRV":
		if !regutils.MatchDomainSRV(name) {
			err = httperrors.NewNotAcceptableError(fmt.Sprintf("Invalid SRV name %s for type %s", name, recType))
		}
	case "PTR":
		if !regutils.MatchPtr(name) {
			err = httperrors.NewNotAcceptableError(fmt.Sprintf("Invalid ptr name %s", name))
		}
	}
	return
}

func (man *SDnsRecordManager) validateModelData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerProjId string,
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
	recType := man.GetRecordsType(records)
	name, err := data.GetString("name")
	if err != nil {
		return nil, err
	}
	err = man.CheckNameForDnsType(name, recType)
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
	ownerProjId string,
	query jsonutils.JSONObject,
	data *jsonutils.JSONDict,
) (*jsonutils.JSONDict, error) {
	data, err := man.validateModelData(ctx, userCred, ownerProjId, query, data)
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
	rec.SetModelManager(DnsRecordManager)
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
	data, err := DnsRecordManager.validateModelData(ctx, userCred, rec.GetOwnerProjectId(), query, data)
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

func (rec *SDnsRecord) AddInfo(userCred mcclient.TokenCredential, data jsonutils.JSONObject) error {
	return rec.SAdminSharableVirtualResourceBase.AddInfo(userCred, DnsRecordManager, rec, data)
}

func (rec *SDnsRecord) AllowPerformAddRecords(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return rec.IsOwner(userCred)
}

func (rec *SDnsRecord) PerformAddRecords(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	records, err := DnsRecordManager.ParseInputInfo(data.(*jsonutils.JSONDict))
	if err != nil {
		return nil, err
	}
	oldRecs := rec.GetInfo()
	oldType := DnsRecordManager.GetRecordsType(oldRecs)
	newType := DnsRecordManager.GetRecordsType(records)
	if oldType != "" && oldType != newType {
		return nil, httperrors.NewNotAcceptableError("Cannot mix different types of records, %s != %s", oldType, newType)
	}
	err = rec.AddInfo(userCred, data)
	return nil, err
}

func (rec *SDnsRecord) AllowPerformRemoveRecords(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return rec.IsOwner(userCred)
}

func (rec *SDnsRecord) PerformRemoveRecords(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	err := rec.SAdminSharableVirtualResourceBase.RemoveInfo(userCred, DnsRecordManager, rec, data, false)
	return nil, err
}

func (rec *SDnsRecord) AllowPerformEnable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return rec.IsOwner(userCred) || userCred.IsSystemAdmin()
}

func (rec *SDnsRecord) PerformEnable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if !rec.Enabled {
		_, err := rec.GetModelManager().TableSpec().Update(rec, func() error {
			rec.Enabled = true
			return nil
		})
		if err != nil {
			log.Errorf("enabling dnsrecords for %s failed: %s", rec.Name, err)
			return nil, err
		}
	}
	return nil, nil
}

func (rec *SDnsRecord) AllowPerformDisable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return rec.IsOwner(userCred) || userCred.IsSystemAdmin()
}

func (rec *SDnsRecord) PerformDisable(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if rec.Enabled {
		_, err := rec.GetModelManager().TableSpec().Update(rec, func() error {
			rec.Enabled = false
			return nil
		})
		if err != nil {
			log.Errorf("disabling dnsrecords for %s failed: %s", rec.Name, err)
			return nil, err
		}
	}
	return nil, nil
}
