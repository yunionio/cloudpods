package models

import (
	"context"
	"fmt"
	"net"
	"reflect"
	"strings"
	"unicode"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SLoadbalancerAclEntry struct {
	Cidr    string
	Comment string
}

func (aclEntry *SLoadbalancerAclEntry) Validate(data *jsonutils.JSONDict) error {
	if strings.Index(aclEntry.Cidr, "/") > 0 {
		_, ipNet, err := net.ParseCIDR(aclEntry.Cidr)
		if err != nil {
			return err
		}
		// normalize from 192.168.1.3/24 to 192.168.1.0/24
		aclEntry.Cidr = ipNet.String()
	} else {
		ip := net.ParseIP(aclEntry.Cidr).To4()
		if ip == nil {
			return httperrors.NewInputParameterError("invalid addr %s", aclEntry.Cidr)
		}
	}
	if commentLimit := 128; len(aclEntry.Comment) > commentLimit {
		return httperrors.NewInputParameterError("comment too long (%d>=%d)",
			len(aclEntry.Comment), commentLimit)
	}
	for _, r := range aclEntry.Comment {
		if !unicode.IsPrint(r) {
			return httperrors.NewInputParameterError("comment contains non-printable char: %v", r)
		}
	}
	return nil
}

type SLoadbalancerAclEntries []*SLoadbalancerAclEntry

func (aclEntries *SLoadbalancerAclEntries) String() string {
	return jsonutils.Marshal(aclEntries).String()
}
func (aclEntries *SLoadbalancerAclEntries) IsZero() bool {
	if len([]*SLoadbalancerAclEntry(*aclEntries)) == 0 {
		return true
	}
	return false
}

func (aclEntries *SLoadbalancerAclEntries) Validate(data *jsonutils.JSONDict) error {
	found := map[string]bool{}
	for _, aclEntry := range *aclEntries {
		if err := aclEntry.Validate(data); err != nil {
			return err
		}
		if _, ok := found[aclEntry.Cidr]; ok {
			// error so that the user has a chance to deal with comments
			return httperrors.NewInputParameterError("acl cidr duplicate %s", aclEntry.Cidr)
		}
		found[aclEntry.Cidr] = true
	}
	return nil
}

type SLoadbalancerAclManager struct {
	db.SSharableVirtualResourceBaseManager
}

var LoadbalancerAclManager *SLoadbalancerAclManager

func init() {
	gotypes.RegisterSerializable(reflect.TypeOf(&SLoadbalancerAclEntries{}), func() gotypes.ISerializable {
		return &SLoadbalancerAclEntries{}
	})
	LoadbalancerAclManager = &SLoadbalancerAclManager{
		SSharableVirtualResourceBaseManager: db.NewSharableVirtualResourceBaseManager(
			SLoadbalancerAcl{},
			"loadbalanceracls_tbl",
			"loadbalanceracl",
			"loadbalanceracls",
		),
	}
}

type SLoadbalancerAcl struct {
	db.SSharableVirtualResourceBase
	SManagedResourceBase

	CloudregionId string                   `width:"36" charset:"ascii" nullable:"false" list:"admin" default:"default"`
	AclEntries    *SLoadbalancerAclEntries `list:"user" update:"user" create:"required"`
}

func loadbalancerAclsValidateAclEntries(data *jsonutils.JSONDict, update bool) (*jsonutils.JSONDict, error) {
	aclEntries := SLoadbalancerAclEntries{}
	aclEntriesV := validators.NewStructValidator("acl_entries", &aclEntries)
	if update {
		aclEntriesV.Optional(true)
	}
	err := aclEntriesV.Validate(data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (man *SLoadbalancerAclManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	data, err := loadbalancerAclsValidateAclEntries(data, false)
	if err != nil {
		return nil, err
	}
	return man.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerProjId, query, data)
}

func (lbacl *SLoadbalancerAcl) AllowPerformStatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return false
}

func (lbacl *SLoadbalancerAcl) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	data, err := loadbalancerAclsValidateAclEntries(data, true)
	if err != nil {
		return nil, err
	}
	return lbacl.SSharableVirtualResourceBase.ValidateUpdateData(ctx, userCred, query, data)
}

func (lbacl *SLoadbalancerAcl) AllowPerformPatch(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) bool {
	return lbacl.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, lbacl, "patch")
}

// PerformPatch patches acl entries by adding then deleting the specified acls.
// This is intended mainly for command line operations.
func (lbacl *SLoadbalancerAcl) PerformPatch(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	aclEntries := gotypes.DeepCopy(*lbacl.AclEntries).(SLoadbalancerAclEntries)
	{
		adds := SLoadbalancerAclEntries{}
		addsV := validators.NewStructValidator("adds", &adds)
		addsV.Optional(true)
		err := addsV.Validate(data)
		if err != nil {
			return nil, err
		}
		for _, add := range adds {
			found := false
			for _, aclEntry := range aclEntries {
				if aclEntry.Cidr == add.Cidr {
					found = true
					aclEntry.Comment = add.Comment
					break
				}
			}
			if !found {
				aclEntries = append(aclEntries, add)
			}
		}
	}
	{
		dels := SLoadbalancerAclEntries{}
		delsV := validators.NewStructValidator("dels", &dels)
		delsV.Optional(true)
		err := delsV.Validate(data)
		if err != nil {
			return nil, err
		}
		for _, del := range dels {
			for i := len(aclEntries) - 1; i >= 0; i-- {
				aclEntry := aclEntries[i]
				if aclEntry.Cidr == del.Cidr {
					aclEntries = append(aclEntries[:i], aclEntries[i+1:]...)
					break
				}
			}
		}
	}
	_, err := lbacl.GetModelManager().TableSpec().Update(lbacl, func() error {
		lbacl.AclEntries = &aclEntries
		return nil
	})
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (lbacl *SLoadbalancerAcl) ValidateDeleteCondition(ctx context.Context) error {
	man := LoadbalancerListenerManager
	t := man.TableSpec().Instance()
	pdF := t.Field("pending_deleted")
	lbaclId := lbacl.Id
	n := t.Query().
		Filter(sqlchemy.OR(sqlchemy.IsNull(pdF), sqlchemy.IsFalse(pdF))).
		Equals("acl_id", lbaclId).
		Count()
	if n > 0 {
		return fmt.Errorf("acl %s is still referred to by %d %s",
			lbaclId, n, man.KeywordPlural())
	}
	return nil
}

func (lbacl *SLoadbalancerAcl) PreDelete(ctx context.Context, userCred mcclient.TokenCredential) {
	lbacl.DoPendingDelete(ctx, userCred)
}

func (lbacl *SLoadbalancerAcl) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (man *SLoadbalancerAclManager) getLoadbalancerAclsByRegion(region *SCloudregion, provider *SCloudprovider) ([]SLoadbalancerAcl, error) {
	acls := []SLoadbalancerAcl{}
	q := man.Query().Equals("cloudregion_id", region.Id).Equals("manager_id", provider.Id)
	if err := db.FetchModelObjects(man, q, &acls); err != nil {
		log.Errorf("failed to get acls for region: %v provider: %v error: %v", region, provider, err)
		return nil, err
	}
	return acls, nil
}

func (man *SLoadbalancerAclManager) SyncLoadbalancerAcls(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, region *SCloudregion, acls []cloudprovider.ICloudLoadbalancerAcl, syncRange *SSyncRange) compare.SyncResult {
	syncResult := compare.SyncResult{}

	dbAcls, err := man.getLoadbalancerAclsByRegion(region, provider)
	if err != nil {
		syncResult.Error(err)
		return syncResult
	}

	removed := []SLoadbalancerAcl{}
	commondb := []SLoadbalancerAcl{}
	commonext := []cloudprovider.ICloudLoadbalancerAcl{}
	added := []cloudprovider.ICloudLoadbalancerAcl{}

	err = compare.CompareSets(dbAcls, acls, &removed, &commondb, &commonext, &added)
	if err != nil {
		syncResult.Error(err)
		return syncResult
	}

	for i := 0; i < len(removed); i++ {
		err = removed[i].ValidateDeleteCondition(ctx)
		if err != nil { // cannot delete
			err = removed[i].SetStatus(userCred, LB_STATUS_UNKNOWN, "sync to delete")
			if err != nil {
				syncResult.DeleteError(err)
			} else {
				syncResult.Delete()
			}
		} else {
			err = removed[i].Delete(ctx, userCred)
			if err != nil {
				syncResult.DeleteError(err)
			} else {
				syncResult.Delete()
			}
		}
	}
	for i := 0; i < len(commondb); i++ {
		err = commondb[i].SyncWithCloudLoadbalancerAcl(ctx, userCred, commonext[i], provider.ProjectId, syncRange.ProjectSync)
		if err != nil {
			syncResult.UpdateError(err)
		} else {
			syncResult.Update()
		}
	}
	for i := 0; i < len(added); i++ {
		_, err := man.newFromCloudLoadbalancerAcl(ctx, userCred, provider, added[i], region)
		if err != nil {
			syncResult.AddError(err)
		} else {
			syncResult.Add()
		}
	}
	return syncResult
}

func (man *SLoadbalancerAclManager) newFromCloudLoadbalancerAcl(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, extAcl cloudprovider.ICloudLoadbalancerAcl, region *SCloudregion) (*SLoadbalancerAcl, error) {
	acl := SLoadbalancerAcl{}
	acl.SetModelManager(man)

	acl.ExternalId = extAcl.GetGlobalId()
	acl.Name = extAcl.GetName()
	acl.ManagerId = provider.Id
	acl.CloudregionId = region.Id

	acl.ProjectId = userCred.GetProjectId()
	if len(provider.ProjectId) > 0 {
		acl.ProjectId = provider.ProjectId
	}

	aclEntries := extAcl.GetAclEntries()
	acl.AclEntries = &SLoadbalancerAclEntries{}
	if err := aclEntries.Unmarshal(acl.AclEntries); err != nil {
		return nil, err
	}
	return &acl, man.TableSpec().Insert(&acl)
}

func (acl *SLoadbalancerAcl) SyncWithCloudLoadbalancerAcl(ctx context.Context, userCred mcclient.TokenCredential, extAcl cloudprovider.ICloudLoadbalancerAcl, projectId string, projectSync bool) error {
	_, err := acl.GetModelManager().TableSpec().Update(acl, func() error {
		acl.Name = extAcl.GetName()
		aclEntries := extAcl.GetAclEntries()

		if projectSync && len(projectId) > 0 {
			acl.ProjectId = projectId
		}

		return aclEntries.Unmarshal(acl.AclEntries)
	})
	return err
}
