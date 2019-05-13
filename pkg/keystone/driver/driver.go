package driver

import (
	"context"

	"yunion.io/x/onecloud/pkg/keystone/models"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type IIdentityBackend interface {
	Authenticate(ctx context.Context, identity mcclient.SAuthenticationIdentity) (*models.SUserExtended, error)
}

type SUserInfo struct {
	DN      string
	Id      string
	Name    string
	Enabled bool
	Extra   map[string]string
}

type SGroupInfo struct {
	Id      string
	Name    string
	Members []string
}
