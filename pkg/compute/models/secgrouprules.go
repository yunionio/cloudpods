package models

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/yunionio/jsonutils"
	"github.com/yunionio/log"
	"github.com/yunionio/mcclient"
	"github.com/yunionio/pkg/httperrors"
	"github.com/yunionio/pkg/util/secrules"
	"github.com/yunionio/pkg/util/stringutils"
	"github.com/yunionio/sqlchemy"

	"github.com/yunionio/onecloud/pkg/cloudcommon/db"
)

type SSecurityGroupRuleManager struct {
	db.SResourceBaseManager
}

var SecurityGroupRuleManager *SSecurityGroupRuleManager

func init() {
	SecurityGroupRuleManager = &SSecurityGroupRuleManager{SResourceBaseManager: db.NewResourceBaseManager(SSecurityGroupRule{}, "secgrouprules_tbl", "secgrouprule", "secgrouprules")}
}

type SSecurityGroupRule struct {
	db.SResourceBase
	Id          string `width:"128" charset:"ascii" primary:"true" list:"user"`
	Priority    int64  `default:"1" list:"user" update:"user" list:"user"`
	Protocol    string `width:"5" charset:"ascii" nullable:"false" list:"user" update:"user"`
	Ports       string `width:"256" charset:"ascii" list:"user" update:"user"`
	Direction   string `width:"3" charset:"ascii" list:"user" create:"required"`
	CIDR        string `width:"256" charset:"ascii" list:"user" update:"user"`
	Action      string `width:"5" charset:"ascii" nullable:"false" list:"user" update:"user" create:"required"`
	Description string `width:"256" charset:"utf8" list:"user" update:"user"`
	SecgroupID  string `width:"128" charset:"ascii" create:"required"`
}

func (manager *SSecurityGroupRuleManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return true
}

func (manager *SSecurityGroupRuleManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return true
}

func (self *SSecurityGroupRule) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	if secgroup := self.GetSecGroup(); secgroup != nil {
		return secgroup.IsOwner(userCred)
	}
	return false
}

func (self *SSecurityGroupRule) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	if secgroup := self.GetSecGroup(); secgroup != nil {
		return secgroup.IsOwner(userCred)
	}
	return false
}

func (self *SSecurityGroupRule) GetSecGroup() *SSecurityGroup {
	if secgroup, _ := SecurityGroupManager.FetchById(self.SecgroupID); secgroup != nil {
		return secgroup.(*SSecurityGroup)
	}
	return nil
}

func (manager *SSecurityGroupRuleManager) FilterById(q *sqlchemy.SQuery, idStr string) *sqlchemy.SQuery {
	return q.Equals("id", idStr)
}

func (manager *SSecurityGroupRuleManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (sql *sqlchemy.SQuery, err error) {
	if sql, err = manager.SResourceBaseManager.ListItemFilter(ctx, q, userCred, query); err != nil {
		return nil, err
	}
	if defsecgroup, _ := query.GetString("secgroup"); len(defsecgroup) > 0 {
		if secgroup, _ := SecurityGroupManager.FetchByIdOrName(userCred.GetProjectId(), defsecgroup); secgroup != nil {
			sql = sql.Equals("secgroup_id", secgroup.GetId())
		} else {
			return nil, httperrors.NewNotFoundError(fmt.Sprintf("Security Group %s not found", defsecgroup))
		}
	}
	for _, field := range []string{"direction", "action", "protocol"} {
		if key, _ := query.GetString(field); len(key) > 0 {
			sql = sql.Equals(field, key)
		}
	}
	return sql, err
}

func (self *SSecurityGroupRule) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DeleteModel(ctx, userCred, self)
}

func (self *SSecurityGroupRule) BeforeInsert() {
	if len(self.Id) == 0 {
		self.Id = stringutils.UUID4()
	}
}

func (manager *SSecurityGroupRuleManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerProjId string,
	query jsonutils.JSONObject,
	data *jsonutils.JSONDict,
) (*jsonutils.JSONDict, error) {
	if defsecgroup, _ := data.GetString("secgroup"); len(defsecgroup) > 0 {
		if secgroup, _ := SecurityGroupManager.FetchByIdOrName(userCred.GetProjectId(), defsecgroup); secgroup != nil {
			data.Set("secgroup_id", jsonutils.NewString(secgroup.GetId()))
		} else {
			return nil, httperrors.NewNotFoundError(fmt.Sprintf("Security Group %s not found", defsecgroup))
		}
	} else {
		return nil, httperrors.NewInputParameterError("missing Security Group info")
	}
	if _priority, _ := data.GetString("priority"); len(_priority) > 0 {
		if priority, err := strconv.Atoi(_priority); err != nil {
			return nil, httperrors.NewInputParameterError("UnSupport priority %s, only support 1-100", err.Error())
		} else if priority < 1 || priority > 100 {
			return nil, httperrors.NewInputParameterError("UnSupport priority range, only support 1-100")
		}
	}
	var fields []string
	for _, field := range []string{"direction", "action", "cidr", "protocol", "ports"} {
		if key, _ := data.GetString(field); len(key) > 0 {
			if field == "direction" {
				key += ":"
			}
			fields = append(fields, key)
		}
	}
	if _, err := secrules.ParseSecurityRule(strings.Join(fields, " ")); err != nil {
		return nil, err
	}
	return manager.SModelBaseManager.ValidateCreateData(ctx, userCred, ownerProjId, query, data)
}

func (self *SSecurityGroupRule) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	if _priority, _ := data.GetString("priority"); len(_priority) > 0 {
		if priority, err := strconv.Atoi(_priority); err != nil {
			return nil, httperrors.NewInputParameterError("UnSupport priority %s, only support 1-100", err.Error())
		} else {
			if priority < 1 || priority > 100 {
				return nil, httperrors.NewInputParameterError("UnSupport priority range, only support 1-100")
			}
		}
	}
	var fields []string
	for _, field := range []string{"direction", "action", "cidr", "protocol", "ports"} {
		if key, _ := data.GetString(field); len(key) > 0 {
			if field == "direction" {
				key += ":"
			}
			fields = append(fields, key)
		} else {
			switch field {
			case "direction":
				fields = append(fields, self.Direction+":")
			case "action":
				fields = append(fields, self.Action)
			case "cidr":
				if len(self.CIDR) > 0 {
					fields = append(fields, self.CIDR)
				}
			case "protocol":
				fields = append(fields, self.Protocol)
			case "ports":
				if len(self.Ports) > 0 {
					fields = append(fields, self.Ports)
				}
			}
		}
	}
	if _, err := secrules.ParseSecurityRule(strings.Join(fields, " ")); err != nil {
		return nil, err
	}
	return self.SResourceBase.ValidateUpdateData(ctx, userCred, query, data)
}

func (self *SSecurityGroupRule) GetRule() string {
	var fields []string
	for _, field := range []string{"direction", "action", "cidr", "protocol", "ports"} {
		switch field {
		case "direction":
			if len(self.Direction) == 0 {
				self.Direction = "in"
			}
			fields = append(fields, self.Direction+":")
		case "action":
			fields = append(fields, self.Action)
		case "cidr":
			if len(self.CIDR) > 0 {
				fields = append(fields, self.CIDR)
			}
		case "protocol":
			fields = append(fields, self.Protocol)
		case "ports":
			if len(self.Ports) > 0 {
				fields = append(fields, self.Ports)
			}
		}
	}
	return fields[0] + strings.Join(fields[1:], " ")
}

func (self *SSecurityGroupRule) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SResourceBase.PostCreate(ctx, userCred, ownerProjId, query, data)

	log.Debugf("POST Create %s", data)
	if secgroup := self.GetSecGroup(); secgroup != nil {
		secgroup.DoSync()
	}
}

func (self *SSecurityGroupRule) PreDelete(ctx context.Context, userCred mcclient.TokenCredential) {
	self.SResourceBase.PreDelete(ctx, userCred)

	if secgroup := self.GetSecGroup(); secgroup != nil {
		secgroup.DoSync()
	}
}

func (self *SSecurityGroupRule) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SResourceBase.PostUpdate(ctx, userCred, query, data)

	log.Debugf("POST Update %s", data)
	if secgroup := self.GetSecGroup(); secgroup != nil {
		secgroup.DoSync()
	}
}
