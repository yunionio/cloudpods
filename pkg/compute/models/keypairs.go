package models

import (
	"context"

	"github.com/yunionio/jsonutils"
	"github.com/yunionio/onecloud/pkg/cloudcommon/db"
	"github.com/yunionio/onecloud/pkg/httperrors"
	"github.com/yunionio/onecloud/pkg/mcclient"
	"github.com/yunionio/sqlchemy"
)

type SKeypairManager struct {
	db.SStandaloneResourceBaseManager
}

var KeypairManager *SKeypairManager

func init() {
	KeypairManager = &SKeypairManager{SStandaloneResourceBaseManager: db.NewStandaloneResourceBaseManager(SKeypair{}, "keypairs_tbl", "keypair", "keypairs")}
}

type SKeypair struct {
	db.SStandaloneResourceBase

	Scheme      string `width:"12" charset:"ascii" nullable:"true" default:"RSA" list:"user" create:"optional"` // Column(VARCHAR(length=12, charset='ascii'), nullable=True, default='RSA')
	Fingerprint string `width:"48" charset:"ascii" nullable:"false" list:"user"`                                // Column(VARCHAR(length=48, charset='ascii'), nullable=False)
	PrivateKey  string `width:"2048" charset:"ascii" nullable:"false"`                                          // Column(VARCHAR(length=2048, charset='ascii'), nullable=False)
	PublicKey   string `width:"1024" charset:"ascii" nullable:"false" list:"user"`                              // Column(VARCHAR(length=1024, charset='ascii'), nullable=False)
	OwnerId     string `width:"128" charset:"ascii" index:"true" nullable:"false"`                              // Column(VARCHAR(length=36, charset='ascii'), index=True, nullable=False)
}

func (manager *SKeypairManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	q, err := manager.SStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}
	if userCred.IsSystemAdmin() && jsonutils.QueryBoolean(query, "admin", false) {
		user, _ := query.GetString("user")
		if len(user) > 0 {
			uc, _ := db.UserCacheManager.FetchUserByIdOrName(user)
			if uc == nil {
				return nil, httperrors.NewUserNotFoundError("user %s not found", user)
			}
			q = q.Equals("owner_id", uc.Id)
		}
	} else {
		q = q.Equals("owner_id", userCred.GetUserId())
	}
	return q, nil
}

func (manager *SKeypairManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return true
}

func (self *SKeypair) IsOwner(userCred mcclient.TokenCredential) bool {
	return self.OwnerId == userCred.GetUserId()
}

func (self *SKeypair) AllowGetDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || userCred.IsSystemAdmin()
}

func (self *SKeypair) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SStandaloneResourceBase.GetCustomizeColumns(ctx, userCred, query)
	extra.Add(jsonutils.NewInt(int64(len(self.PrivateKey))), "private_key_len")
	extra.Add(jsonutils.NewInt(int64(self.GetLinkedGuestsCount())), "linked_guest_count")
	return extra
}

func (self *SKeypair) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SStandaloneResourceBase.GetExtraDetails(ctx, userCred, query)
	extra.Add(jsonutils.NewInt(int64(len(self.PrivateKey))), "private_key_len")
	extra.Add(jsonutils.NewInt(int64(self.GetLinkedGuestsCount())), "linked_guest_count")
	if userCred.IsSystemAdmin() {
		extra.Add(jsonutils.NewString(self.OwnerId), "owner_id")
		uc, _ := db.UserCacheManager.FetchUserById(self.OwnerId)
		if uc != nil {
			extra.Add(jsonutils.NewString(uc.Name), "owner_name")
		}
	}
	return extra
}

func (manager *SKeypairManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return true
}

func (self *SKeypair) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return self.IsOwner(userCred)
}

func (self *SKeypair) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return self.IsOwner(userCred) || userCred.IsSystemAdmin()
}

func (self *SKeypair) GetLinkedGuestsCount() int {
	return GuestManager.Query().Equals("keypair_id", self.Id).Count()
}

func (manager *SKeypairManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	// XXX: TODO
	return manager.SStandaloneResourceBaseManager.ValidateCreateData(ctx, userCred, ownerProjId, query, data)
}

func (self *SKeypair) ValidateDeleteCondition(ctx context.Context) error {
	if self.GetLinkedGuestsCount() > 0 {
		return httperrors.NewNotEmptyError("Cannot delete keypair used by servers")
	}
	return self.SStandaloneResourceBase.ValidateDeleteCondition(ctx)
}

func totalKeypairCount(userId string) int {
	q := KeypairManager.Query().Equals("owner_id", userId)
	return q.Count()
}

func (manager *SKeypairManager) FilterByOwner(q *sqlchemy.SQuery, ownerId string) *sqlchemy.SQuery {
	return q.Equals("owner_id", ownerId)
}

func (self *SKeypair) GetOwnerProjectId() string {
	return self.OwnerId
}

func (manager *SKeypairManager) GetOwnerId(userCred mcclient.TokenCredential) string {
	return userCred.GetUserId()
}
