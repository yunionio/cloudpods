package models

import (
	"context"
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
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

const (
	DNS_RECORDS_SEPARATOR = ","
)

type SDnsRecord struct {
	db.SAdminSharableVirtualResourceBase
	Ttl int `nullable:"true" default:"0" create:"optional" list:"user" update:"user"`
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
func (man *SDnsRecordManager) ParseInputInfo(data *jsonutils.JSONDict) (rec []string, err error) {
	for _, typ := range []string{"A", "AAAA"} {
		idx := 0
		var addr string
		for {
			key := fmt.Sprintf("%s.%d", typ, idx)
			if data.Contains(key) {
				addr, err = data.GetString(key)
				if err != nil {
					return
				}
				if (typ == "A" && !regutils.MatchIP4Addr(addr)) || (typ == "AAAA" && !regutils.MatchIP6Addr(addr)) {
					err = httperrors.NewNotAcceptableError("Invalid address %s", addr)
					return
				}
				rec = append(rec, fmt.Sprintf("%s:%s", typ, addr))
			} else {
				break
			}
			idx += 1
		}
	}
	if data.Contains("CNAME") {
		if len(rec) > 0 {
			err = httperrors.NewNotAcceptableError("CNAME cannot mix with other types")
			return
		}
		var cname string
		cname, err = data.GetString("CNAME")
		if err != nil {
			return
		}
		if !regutils.MatchDomainName(cname) {
			err = httperrors.NewNotAcceptableError(fmt.Sprintf("Invalid domain %s", cname))
			return
		}
		rec = append(rec, fmt.Sprintf("%s:%s", "CNAME", cname))
	} else if data.Contains("SRV_host") && data.Contains("SRV_port") {
		if len(rec) > 0 {
			err = httperrors.NewNotAcceptableError("SRV cannot mix with other types")
			return
		}
		var port string
		port, err = data.GetString("SRV_port")
		if err != nil {
			return
		}
		if !regutils.MatchInteger(port) {
			err = httperrors.NewNotAcceptableError("Invalid port %s", port)
			return
		}
		var host string
		host, err = data.GetString("SRV_host")
		if err != nil {
			return
		}
		rec = append(rec, fmt.Sprintf("%s:%s:%s", "SRV", host, port))
	} else if data.Contains("PTR") {
		if len(rec) > 0 {
			err = httperrors.NewNotAcceptableError("PTR cannot mix with other types")
			return
		}
		var name string
		name, err = data.GetString("name")
		if err != nil {
			return
		}
		if !regutils.MatchPtr(name) {
			err = httperrors.NewNotAcceptableError(fmt.Sprintf("Invalid ptr %s", name))
			return
		}
		var domainName string
		domainName, err = data.GetString("PTR")
		if err != nil {
			return
		}
		if !regutils.MatchDomainName(domainName) {
			err = httperrors.NewNotAcceptableError(fmt.Sprintf("Invalid domain %s", domainName))
			return
		}
		rec = append(rec, fmt.Sprintf("%s:%s", "PTR", domainName))
	}
	return
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

func (man *SDnsRecordManager) ValidateCreateData(
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
		return nil, fmt.Errorf("Empty record")
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

	return man.SAdminSharableVirtualResourceBaseManager.ValidateCreateData(man, data)
}

func (man *SDnsRecordManager) QueryDns(projectId, name string) *SDnsRecord {
	q := man.Query()
	q = man.FilterByName(q, name)
	if len(projectId) == 0 {
		q = q.Filter(sqlchemy.IsTrue(q.Field("is_public")))
	} else {
		q = q.Filter(sqlchemy.OR(sqlchemy.IsTrue(q.Field("is_public")), sqlchemy.Equals(q.Field("tenant_id"), projectId)))
	}

	rec := &SDnsRecord{}
	rec.SetModelManager(DnsRecordManager)

	err := q.First(rec)
	if err != nil {
		return nil
	}
	return rec
}

type DnsIp struct {
	Ttl  int
	Addr string
}

func (man *SDnsRecordManager) QueryDnsIps(projectId, name, kind string) []DnsIp {
	rec := man.QueryDns(projectId, name)
	result := make([]DnsIp, 0)
	if rec == nil {
		return result
	}
	records := rec.GetInfo()
	for _, r := range records {
		if strings.HasPrefix(r, fmt.Sprintf("%s:", kind)) {
			result = append(result, DnsIp{
				Ttl:  rec.GetTtl(),
				Addr: r[len(kind)+1:],
			})
		}
	}
	return result
}

func (rec *SDnsRecord) GetInfo() []string {
	return strings.Split(rec.Records, DNS_RECORDS_SEPARATOR)
}

func (rec *SDnsRecord) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	if data.Contains("name") {
		records := rec.GetInfo()
		recType := DnsRecordManager.GetRecordsType(records)
		name, err := data.GetString("name")
		if err != nil {
			return nil, err
		}
		err = DnsRecordManager.CheckNameForDnsType(name, recType)
		if err != nil {
			return nil, err
		}
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

func (rec *SDnsRecord) GetTtl() int {
	return rec.Ttl
}
