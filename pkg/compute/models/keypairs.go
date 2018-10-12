package models

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/sqlchemy"

	"golang.org/x/crypto/ssh"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/seclib2"
	"yunion.io/x/pkg/utils"
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

	Scheme      string `width:"12" charset:"ascii" nullable:"true" default:"RSA" list:"user" create:"required"` // Column(VARCHAR(length=12, charset='ascii'), nullable=True, default='RSA')
	Fingerprint string `width:"48" charset:"ascii" nullable:"false" list:"user" create:"required"`              // Column(VARCHAR(length=48, charset='ascii'), nullable=False)
	PrivateKey  string `width:"2048" charset:"ascii" nullable:"false" create:"optional"`                        // Column(VARCHAR(length=2048, charset='ascii'), nullable=False)
	PublicKey   string `width:"1024" charset:"ascii" nullable:"false" list:"user" create:"required"`            // Column(VARCHAR(length=1024, charset='ascii'), nullable=False)
	OwnerId     string `width:"128" charset:"ascii" index:"true" nullable:"false" create:"required"`            // Column(VARCHAR(length=36, charset='ascii'), index=True, nullable=False)
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
	publicKey, _ := data.GetString("public_key")
	if len(publicKey) == 0 {
		scheme, _ := data.GetString("scheme")
		if len(scheme) > 0 {
			if !utils.IsInStringArray(scheme, []string{"RSA", "DSA"}) {
				return nil, httperrors.NewInputParameterError("Unsupported scheme %s", scheme)
			}
		} else {
			scheme = "RSA"
		}
		var privKey, pubKey string
		var err error
		if scheme == "RSA" {
			privKey, pubKey, err = seclib2.GenerateRSASSHKeypair()
		} else {
			privKey, pubKey, err = seclib2.GenerateDSASSHKeypair()
		}
		if err != nil {
			log.Errorf("fail to generate ssh keypair %s", err)
			return nil, httperrors.NewGeneralError(err)
		}
		publicKey = pubKey
		data.Set("public_key", jsonutils.NewString(pubKey))
		data.Set("private_key", jsonutils.NewString(privKey))
	}
	pubKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(publicKey))
	if err != nil {
		log.Errorf("invalid public key %s", err)
		return nil, httperrors.NewInputParameterError("invalid public")
	}
	data.Set("fingerprint", jsonutils.NewString(ssh.FingerprintLegacyMD5(pubKey)))
	data.Set("scheme", jsonutils.NewString(seclib2.GetPublicKeyScheme(pubKey)))
	data.Set("owner_id", jsonutils.NewString(userCred.GetUserId()))

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

func (manager *SKeypairManager) FilterByOwner(q *sqlchemy.SQuery, owner string) *sqlchemy.SQuery {
	return q.Equals("owner_id", owner)
}

func (self *SKeypair) GetOwnerProjectId() string {
	return self.OwnerId
}

func (manager *SKeypairManager) GetOwnerId(userCred mcclient.IIdentityProvider) string {
	return userCred.GetUserId()
}

func (manager *SKeypairManager) FetchByName(userCred mcclient.IIdentityProvider, idStr string) (db.IModel, error) {
	return db.FetchByName(manager, userCred, idStr)
}

func (manager *SKeypairManager) FetchByIdOrName(userCred mcclient.IIdentityProvider, idStr string) (db.IModel, error) {
	return db.FetchByIdOrName(manager, userCred, idStr)
}

func (keypair *SKeypair) AllowGetDetailsPrivatekey(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return keypair.OwnerId == userCred.GetUserId()
}

func (keypair *SKeypair) GetDetailsPrivatekey(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	retval := jsonutils.NewDict()
	if len(keypair.PrivateKey) > 0 {
		retval.Add(jsonutils.NewString(keypair.PrivateKey), "private_key")
		retval.Add(jsonutils.NewString(keypair.Name), "name")
		retval.Add(jsonutils.NewString(keypair.Scheme), "scheme")
		_, err := keypair.GetModelManager().TableSpec().Update(keypair, func() error {
			keypair.PrivateKey = ""
			return nil
		})
		if err != nil {
			return nil, err
		}

		db.OpsLog.LogEvent(keypair, db.ACT_FETCH, nil, userCred)
	}
	return retval, nil
}
