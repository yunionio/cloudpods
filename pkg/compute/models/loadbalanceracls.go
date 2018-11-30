package models

import (
	"context"
	"fmt"
	"net"
	"reflect"
	"strings"
	"unicode"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
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

	AclEntries *SLoadbalancerAclEntries `list:"user" update:"user" create:"required"`
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
	return lbacl.IsOwner(userCred) || userCred.IsAdminAllow(consts.GetServiceType(), lbacl.KeywordPlural(), policy.PolicyActionPerform, "patch")
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
