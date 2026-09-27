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

package sshkeys

import (
	"context"
	"strings"

	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/util/seclib2"
)

const (
	sshAdminRsaPrivateKey = "admin-ssh-private-key"
	sshAdminRsaPublicKey  = "admin-ssh-public-key"

	sshAdminEd25519PrivateKey = "admin-ssh-ed25519-private-key"
	sshAdminEd25519PublicKey  = "admin-ssh-ed25519-public-key"

	sshRsaPrivateKey = "project-ssh-private-key"
	sshRsaPublicKey  = "project-ssh-public-key"

	sshEd25519PrivateKey = "project-ssh-ed25519-private-key"
	sshEd25519PublicKey  = "project-ssh-ed25519-public-key"
)

func _getKeys(ctx context.Context, tenantId string, privateKey, publicKey string, autocreate bool) (string, string, error) {
	tenant, err := db.TenantCacheManager.FetchTenantById(ctx, tenantId)
	if err != nil {
		return "", "", err
	}
	private := tenant.GetMetadata(ctx, privateKey, nil)
	public := tenant.GetMetadata(ctx, publicKey, nil)
	userCred := auth.AdminCredential()
	if (len(private) == 0 || len(public) == 0) && autocreate {
		if strings.Contains(privateKey, "ed25519") {
			private, public, _ = seclib2.GenerateED25519SSHKeypair()
		} else {
			private, public, _ = seclib2.GenerateRSASSHKeypair()
		}
		private, _ = utils.EncryptAESBase64(tenantId, private)
		tenant.SetMetadata(ctx, privateKey, private, userCred)
		tenant.SetMetadata(ctx, publicKey, public, userCred)
	} else if len(private) > 0 {
		private, _ = utils.DescryptAESBase64(tenantId, private)
	}
	return private, public, nil
}

func GetSshProjectKeypair(ctx context.Context, tenantId string) ([]string, []string, error) {
	return getSshProjectKeypair(ctx, tenantId, sshEd25519PrivateKey, sshEd25519PublicKey, sshRsaPrivateKey, sshRsaPublicKey)
}

func getSshProjectKeypair(ctx context.Context, tenantId string, ed25519PrivKey, ed25519PubKey, rsaPrivKey, rsaPubKey string) ([]string, []string, error) {
	ed25519Priv, ed25519Pub, err := _getKeys(ctx, tenantId, ed25519PrivKey, ed25519PubKey, true)
	if err != nil {
		return nil, nil, err
	}
	rsaPriv, rsaPub, _ := _getKeys(ctx, tenantId, rsaPrivKey, rsaPubKey, false)
	private := []string{ed25519Priv}
	if len(rsaPriv) > 0 {
		private = append(private, rsaPriv)
	}
	public := []string{ed25519Pub}
	if len(rsaPub) > 0 {
		public = append(public, rsaPub)
	}
	return private, public, nil
}

func GetSshAdminKeypair(ctx context.Context) ([]string, []string, error) {
	userCred := auth.AdminCredential()
	return getSshProjectKeypair(ctx, userCred.GetProjectId(), sshAdminEd25519PrivateKey, sshAdminEd25519PublicKey, sshAdminRsaPrivateKey, sshAdminRsaPublicKey)
}
