package models

import (
	"context"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/util/seclib2"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
)

const (
	sshAdminPrivateKey = "admin-ssh-private-key"
	sshAdminPublicKey  = "admin-ssh-public-key"

	sshPrivateKey = "project-ssh-private-key"
	sshPublicKey  = "project-ssh-public-key"
)

func  _getKeys(ctx context.Context, tenantId string, privateKey, publicKey string) (string, string, error) {
	tenant, err := db.TenantCacheManager.FetchTenantById(ctx, tenantId)
	if err != nil {
		return "", "", err
	}
	private := tenant.GetMetadata(privateKey, nil)
	public := tenant.GetMetadata(publicKey, nil)
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

func getSshProjectKeypair(ctx context.Context, tenantId string) (string, string, error) {
	return _getKeys(ctx, tenantId, sshPrivateKey, sshPublicKey)
}

func getSshAdminKeypair(ctx context.Context) (string, string, error) {
	userCred := auth.AdminCredential()
	return _getKeys(ctx, userCred.GetProjectId(), sshAdminPrivateKey, sshAdminPublicKey)
}