package options

import (
	"yunion.io/x/onecloud/pkg/cloudcommon/options"
)

type SKeystoneOptions struct {
	options.BaseOptions

	options.DBOptions

	AdminPort int `default:"35357" help:"listening port for admin API(deprecated)"`

	TokenExpirationSeconds  int    `default:"86400" help:"token expiration seconds" token:"expiration"`
	TokenKeyRepository      string `help:"fernet key repo directory" token:"key_repository" default:"/etc/yunion/keystone/fernet-keys"`
	CredentialKeyRepository string `help:"fernet key repo directory for credential" token:"credential_key_repository"`

	AdminUserName        string `help:"Administrative user name" default:"sysadmin"`
	AdminUserDomainId    string `help:"Domain id of administrative user" default:"default"`
	AdminProjectName     string `help:"Administrative project name" default:"system"`
	AdminProjectDomainId string `help:"Domain id of administrative project" default:"default"`
}

var (
	Options SKeystoneOptions
)
