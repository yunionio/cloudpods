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

package clientman

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/golang-plus/uuid"
	"github.com/pkg/errors"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/apigateway/options"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
)

func authVersion() string {
	pos := strings.LastIndexByte(options.Options.AuthURL, '/')
	if pos > 0 {
		return options.Options.AuthURL[pos+1:]
	} else {
		return ""
	}
}

/*
  mapTokenManagerV2 在mapTokenManager的基础上增加了 token持久化.后端为sqlite
  token存储了两份：分别位于缓存和后端sqlite.
  save同时会从sqlite中清理过期的缓存
*/
type ITokenManagerV2 interface {
	mcclient.TokenManager
	ITotpManager
}

type credential struct {
	Token mcclient.TokenCredential
	Totp  ITotp // 双因子认证。只有options totp enable的情况下该字段才有意义
}

type SMapTokenManagerV2 struct {
	table map[string]*credential
}

func UnmarshalV2Token(rbody jsonutils.JSONObject) (cred mcclient.TokenCredential, err error) {
	access, err := rbody.Get("access")
	if err == nil {
		cred = &mcclient.TokenCredentialV2{}
		err = access.Unmarshal(cred)
		if err != nil {
			err = fmt.Errorf("Invalid response when unmarshal V2 Token: %s", err)
		}

		return
	}
	err = fmt.Errorf("Invalid response: no access object")
	return
}

func UnmarshalV3Token(rbody jsonutils.JSONObject, tokenId string) (cred mcclient.TokenCredential, err error) {
	cred = &mcclient.TokenCredentialV3{Id: tokenId}
	err = rbody.Unmarshal(cred)
	if err != nil {
		err = fmt.Errorf("Invalid response when unmarshal V3 Token: %v", err)
	}

	return
}

func (this *SMapTokenManagerV2) Save(token mcclient.TokenCredential) string {
	key, e := uuid.NewV4()
	if e != nil {
		log.Fatalf("uuid.NewV4 returns error!")
	}

	var totp ITotp
	kkey := key.String()
	totp = &STotp{}
	this.table[kkey] = &credential{Token: token, Totp: totp}
	// 存储到后端服务
	go func() {
		t := jsonutils.Marshal(token).String()
		t2 := jsonutils.Marshal(totp).String()
		tr := TokenRecord{
			TokenID:  kkey,
			Token:    t,
			TokenStr: token.GetTokenString(),
			Totp:     t2,
			Version:  authVersion(),
			ExpireAt: token.GetExpires(),
		}
		if DB.Create(&tr); DB.Error != nil {
			log.Errorf("%v", DB.Error)
		} else {
			// 删除已经过期的token避免无效token堆积
			now := time.Now()
			if DB.Delete(TokenRecord{}, "expire_at < ?", now); DB.Error != nil {
				log.Errorf("%v", DB.Error)
			}
		}
	}()
	return kkey
}

func (this *SMapTokenManagerV2) Get(tid string) mcclient.TokenCredential {
	if cred := this.table[tid]; cred != nil && cred.Token != nil {
		return cred.Token
	}

	// load from db
	err := this.loadFromDB(tid)
	if err != nil {
		log.Errorf("loadFromDB by tid %s: %v", tid, err)
		return nil
	} else {
		return this.table[tid].Token
	}
}

func (this *SMapTokenManagerV2) Remove(tid string) {
	delete(this.table, tid)
	if DB.Delete(TokenRecord{}, "token_id = ?", tid); DB.Error != nil {
		log.Errorf("%v", DB.Error)
	}
}

func (this *SMapTokenManagerV2) GetTotp(tid string) ITotp {
	if cred := this.table[tid]; cred != nil && cred.Totp != nil {
		return cred.Totp
	}

	// load from db
	err := this.loadFromDB(tid)
	if err != nil {
		log.Errorf(err.Error())
		return nil
	} else {
		return this.table[tid].Totp
	}
}

func (this *SMapTokenManagerV2) SaveTotp(tid string) {
	totp := this.table[tid].Totp
	if totp == nil {
		log.Errorf("%s totp is nil", tid)
		return
	}

	// 存储到后端服务
	go func() {
		tr := TokenRecord{}
		if DB.Where("token_id = ?", tid).First(&tr); DB.Error != nil {
			log.Errorf("%v", DB.Error)
		} else {
			DB.Model(&tr).Update("Totp", jsonutils.Marshal(totp).String())
		}
	}()
}

func (this *SMapTokenManagerV2) loadFromDB(tid string) error {
	// load from db
	tr := TokenRecord{}
	if DB.Where("token_id = ?", tid).First(&tr); DB.Error != nil {
		return fmt.Errorf("%v", DB.Error)
	}

	tjson, err := jsonutils.ParseString(tr.Token)
	if err != nil {
		return errors.Wrapf(err, "parse token to json: %s", tr.Token)
	}

	var token mcclient.TokenCredential
	if authVersion() == "v3" {
		if token, err = UnmarshalV3Token(tjson, tr.TokenStr); err != nil {
			return err
		}
	} else {
		if token, err = UnmarshalV2Token(tjson); err != nil {
			return err
		}
	}

	var totp STotp
	totpJson, err := jsonutils.ParseString(tr.Totp)
	if err != nil {
		return err
	}

	if err := totpJson.Unmarshal(&totp); err != nil {
		return err
	}

	this.table[tid] = &credential{Token: token, Totp: &totp}
	return nil
}

func NewMapTokenManagerV2() ITokenManagerV2 {
	return &SMapTokenManagerV2{table: make(map[string]*credential)}
}

var (
	TokenMan ITokenManagerV2
)

func InitClient(dbPath string) error {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return errors.Wrapf(err, "ensure sqlite dir %s", filepath.Dir(dbPath))
	}
	if err := initDB("sqlite3", dbPath); err != nil {
		return errors.Wrap(err, "init sqlite db")
	}

	info := auth.NewAuthInfo(options.Options.AuthURL,
		options.Options.AdminDomain,
		options.Options.AdminUser,
		options.Options.AdminPassword,
		options.Options.AdminProject,
		options.Options.AdminProjectDomain,
	)

	auth.Init(info, options.Options.DebugClient, true,
		options.Options.SslCertfile, options.Options.SslKeyfile)

	TokenMan = NewMapTokenManagerV2()
	return nil
}
