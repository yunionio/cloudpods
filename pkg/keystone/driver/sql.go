package driver

import (
	"context"

	"yunion.io/x/onecloud/pkg/keystone/models"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SSQLDriver struct {
	SBaseDomainDriver
}

func NewSQLDriver(domainId string, conf models.TDomainConfigs) (IIdentityBackend, error) {
	drv := SSQLDriver{
		NewBaseDomainDriver(domainId, conf),
	}
	drv.virtual = &drv
	return &drv, nil
}

func (sql *SSQLDriver) Authenticate(ctx context.Context, ident mcclient.SAuthenticationIdentity) (*models.SUserExtended, error) {
	usrExt, err := models.UserManager.FetchUserExtended(
		ident.Password.User.Id,
		ident.Password.User.Name,
		ident.Password.User.Domain.Id,
		ident.Password.User.Domain.Name,
	)
	if err != nil {
		return nil, err
	}
	err = usrExt.VerifyPassword(ident.Password.User.Password)
	if err != nil {
		return nil, err
	}
	return usrExt, nil
}
