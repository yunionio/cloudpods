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

	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/util/seclib2"
)

const (
	sshAdminPrivateKey = "admin-ssh-private-key"
	sshAdminPublicKey  = "admin-ssh-public-key"

	sshPrivateKey = "project-ssh-private-key"
	sshPublicKey  = "project-ssh-public-key"
)

func _getKeys(ctx context.Context, tenantId string, privateKey, publicKey string) (string, string, error) {
	tenant, err := db.TenantCacheManager.FetchTenantById(ctx, tenantId)
	if err != nil {
		return "", "", err
	}
	private := tenant.GetMetadata(ctx, privateKey, nil)
	public := tenant.GetMetadata(ctx, publicKey, nil)
	userCred := auth.AdminCredential()
	if len(private) == 0 || len(public) == 0 {
		private, public, _ = seclib2.GenerateRSASSHKeypair()
		private, _ = utils.EncryptAESBase64(tenantId, private)
		tenant.SetMetadata(ctx, privateKey, private, userCred)
		tenant.SetMetadata(ctx, publicKey, public, userCred)
	}
	private, _ = utils.DescryptAESBase64(tenantId, private)
	return private, public, nil
}

func GetSshProjectKeypair(ctx context.Context, tenantId string) (string, string, error) {
	return _getKeys(ctx, tenantId, sshPrivateKey, sshPublicKey)
}

func GetSshAdminKeypair(ctx context.Context) (string, string, error) {
	userCred := auth.AdminCredential()
	return _getKeys(ctx, userCred.GetProjectId(), sshAdminPrivateKey, sshAdminPublicKey)
}
