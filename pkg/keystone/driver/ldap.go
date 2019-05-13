package driver

import (
	"context"
	"database/sql"
	"strconv"
	"strings"

	"gopkg.in/ldap.v3"

	"github.com/pkg/errors"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/keystone/models"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/ldaputils"
)

type SLDAPDriver struct {
	SBaseDomainDriver
	ldapConfig *api.SDomainLDAPConfigOptions
}

func NewLDAPDriver(domainId string, conf models.TDomainConfigs) (IIdentityBackend, error) {
	drv := SLDAPDriver{
		SBaseDomainDriver: NewBaseDomainDriver(domainId, conf),
	}
	drv.virtual = &drv
	err := drv.prepareConfig()
	if err != nil {
		return nil, errors.WithMessage(err, "prepareConfig")
	}
	return &drv, nil
}

func (self *SLDAPDriver) prepareConfig() error {
	if self.ldapConfig == nil {
		conf := api.SDomainLDAPConfigOptions{}
		confJson := jsonutils.Marshal(self.config["ldap"])
		err := confJson.Unmarshal(&conf)
		if err != nil {
			return errors.WithMessage(err, "json.Unmarshal")
		}
		log.Debugf("%s %s %#v", self.config, confJson, self.ldapConfig)
		self.ldapConfig = &conf
	}
	return nil
}

func (self *SLDAPDriver) queryScope() int {
	if self.ldapConfig.QueryScope == api.QueryScopeOne {
		return ldap.ScopeSingleLevel
	} else {
		return ldap.ScopeWholeSubtree
	}
}

func (self *SLDAPDriver) userAttributeList() []string {
	attrs := []string{
		"dn",
		self.ldapConfig.UserIdAttribute,
		self.ldapConfig.UserNameAttribute,
		self.ldapConfig.UserEnabledAttribute,
	}
	for _, m := range self.ldapConfig.UserAdditionalAttribute {
		parts := strings.Split(m, ":")
		if len(parts) == 2 {
			attrs = append(attrs, parts[0])
		}
	}
	return attrs
}

func (self *SLDAPDriver) entry2Group(entry *ldap.Entry) SGroupInfo {
	info := SGroupInfo{}
	info.Id = entry.GetAttributeValue(self.ldapConfig.GroupIdAttribute)
	info.Name = entry.GetAttributeValue(self.ldapConfig.GroupNameAttribute)
	info.Members = entry.GetAttributeValues(self.ldapConfig.GroupMemberAttribute)
	return info
}

func (self *SLDAPDriver) entry2User(entry *ldap.Entry) SUserInfo {
	info := SUserInfo{}
	info.DN = entry.DN
	info.Id = entry.GetAttributeValue(self.ldapConfig.UserIdAttribute)
	info.Name = entry.GetAttributeValue(self.ldapConfig.UserNameAttribute)
	enabledStr := entry.GetAttributeValue(self.ldapConfig.UserEnabledAttribute)
	if len(enabledStr) == 0 {
		info.Enabled = utils.ToBool(self.ldapConfig.UserEnabledDefault)
	} else if self.ldapConfig.UserEnabledMask > 0 {
		enabled, _ := strconv.ParseInt(enabledStr, 0, 64)
		if (enabled & self.ldapConfig.UserEnabledMask) != 0 {
			info.Enabled = true
		}
	} else if self.ldapConfig.UserEnabledInvert {
		info.Enabled = !utils.ToBool(enabledStr)
	} else {
		info.Enabled = utils.ToBool(enabledStr)
	}
	info.Extra = make(map[string]string)
	for _, m := range self.ldapConfig.UserAdditionalAttribute {
		parts := strings.Split(m, ":")
		if len(parts) == 2 {
			info.Extra[parts[1]] = entry.GetAttributeValue(parts[0])
		}
	}
	return info
}

func (self *SLDAPDriver) groupAttributeList() []string {
	return []string{
		"dn",
		self.ldapConfig.GroupIdAttribute,
		self.ldapConfig.GroupNameAttribute,
		self.ldapConfig.GroupMemberAttribute,
	}
}

func (self *SLDAPDriver) getClient() (*ldaputils.SLDAPClient, error) {
	cli := ldaputils.NewLDAPClient(
		self.ldapConfig.Url,
		self.ldapConfig.User,
		self.ldapConfig.Password,
		self.ldapConfig.Suffix,
		false,
	)
	err := cli.Connect()
	if err != nil {
		return nil, errors.WithMessage(err, "Connect")
	}
	return cli, nil
}

func (self *SLDAPDriver) getUserTreeDN() string {
	if len(self.ldapConfig.UserTreeDN) > 0 {
		return self.ldapConfig.UserTreeDN
	}
	return self.ldapConfig.Suffix
}

func (self *SLDAPDriver) getGroupTreeDN() string {
	if len(self.ldapConfig.GroupTreeDN) > 0 {
		return self.ldapConfig.GroupTreeDN
	}
	return self.ldapConfig.Suffix
}

func (self *SLDAPDriver) Authenticate(ctx context.Context, ident mcclient.SAuthenticationIdentity) (*models.SUserExtended, error) {
	cli, err := self.getClient()
	if err != nil {
		return nil, errors.Wrap(err, "getClient")
	}
	defer cli.Close()

	username := ident.Password.User.Name
	password := ident.Password.User.Password

	entry, err := cli.Authenticate(
		self.getUserTreeDN(),
		self.ldapConfig.UserObjectclass,
		self.ldapConfig.UserIdAttribute,
		username,
		password,
		self.userAttributeList())
	if err != nil {
		return nil, errors.Wrap(err, "Authenticate error")
	}

	userinfo := self.entry2User(entry)
	groups := self.fetchUserGroups(cli, userinfo)

	return self.syncUserDB(ctx, models.GetDefaultAdminCred(), userinfo, groups)
}

func (self *SLDAPDriver) fetchUserGroups(cli *ldaputils.SLDAPClient, userinfo SUserInfo) []SGroupInfo {
	cond := make(map[string]string)
	if self.ldapConfig.GroupMembersAreIds {
		cond[self.ldapConfig.GroupMemberAttribute] = userinfo.Id
	} else {
		cond[self.ldapConfig.GroupMemberAttribute] = userinfo.DN
	}
	entries, _ := cli.Search(self.getGroupTreeDN(),
		self.ldapConfig.GroupObjectclass,
		cond,
		self.groupAttributeList(),
	)
	groups := make([]SGroupInfo, len(entries))
	for i := range entries {
		groups[i] = self.entry2Group(entries[i])
	}
	return groups
}

func copyUserInfo(ui SUserInfo, nonLocal *models.SNonlocalUser, user *models.SUser) {
	user.Name = ui.Id
	if ui.Enabled {
		user.Enabled = tristate.True
	} else {
		user.Enabled = tristate.False
	}
	user.Id = nonLocal.UserId
	user.DomainId = nonLocal.DomainId
	if val, ok := ui.Extra["email"]; ok && len(val) > 0 {
		user.Email = val
	}
	if val, ok := ui.Extra["displayname"]; ok && len(val) > 0 {
		user.Displayname = val
	}
	if val, ok := ui.Extra["mobile"]; ok && len(val) > 0 {
		user.Mobile = val
	}
}

func registerNonlocalUser(ctx context.Context, ui SUserInfo, nonLocal *models.SNonlocalUser) error {
	lockman.LockRawObject(ctx, models.UserManager.Keyword(), nonLocal.UserId)
	defer lockman.ReleaseRawObject(ctx, models.UserManager.Keyword(), nonLocal.UserId)

	userObj, err := db.NewModelObject(models.UserManager)
	if err != nil {
		return errors.WithMessage(err, "db.NewModelObject")
	}
	user := userObj.(*models.SUser)
	q := models.UserManager.Query().Equals("id", nonLocal.UserId)
	err = q.First(user)
	if err != nil && err != sql.ErrNoRows {
		return errors.WithMessage(err, "Query")
	}
	if err == nil {
		// update
		_, err := db.Update(user, func() error {
			copyUserInfo(ui, nonLocal, user)
			return nil
		})
		if err != nil {
			return errors.WithMessage(err, "Update")
		}
	} else {
		// insert
		copyUserInfo(ui, nonLocal, user)
		err = models.UserManager.TableSpec().Insert(user)
		if err != nil {
			return errors.WithMessage(err, "Insert")
		}
	}
	return nil
}

func (self *SLDAPDriver) syncUserDB(ctx context.Context, userCred mcclient.TokenCredential, ui SUserInfo, groups []SGroupInfo) (*models.SUserExtended, error) {
	nonLocalUser, err := models.NonlocalUserManager.Register(ctx, self.domainId, ui.Id)
	if err != nil {
		return nil, errors.WithMessage(err, "models.NonlocalUserManager.Register")
	}

	// insert nonlocal user
	err = registerNonlocalUser(ctx, ui, nonLocalUser)
	if err != nil {
		return nil, errors.WithMessage(err, "registerNonlocalUser")
	}

	// sync group
	groupIds := make([]string, 0)
	for i := range groups {
		grp, err := models.GroupManager.RegisterExternalGroup(ctx, self.domainId, groups[i].Id, groups[i].Name)
		if err != nil {
			log.Errorf("models.GroupManager.RegisterExternalGroup fail %s", err)
		} else {
			groupIds = append(groupIds, grp.Id)
		}
	}

	models.UsergroupManager.SyncUserGroups(ctx, userCred, nonLocalUser.UserId, groupIds)

	return models.UserManager.FetchUserExtended(nonLocalUser.UserId, "", "", "")
}
