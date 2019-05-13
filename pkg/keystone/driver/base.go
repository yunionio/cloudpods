package driver

import (
	"yunion.io/x/onecloud/pkg/keystone/models"
)

type SBaseDomainDriver struct {
	virtual interface{}

	config   models.TDomainConfigs
	domainId string
}

func (base *SBaseDomainDriver) IIdentityBackend() IIdentityBackend {
	return base.virtual.(IIdentityBackend)
}

func NewBaseDomainDriver(domainId string, conf models.TDomainConfigs) SBaseDomainDriver {
	return SBaseDomainDriver{
		domainId: domainId,
		config:   conf,
	}
}

func GetDriver(domainId string, conf models.TDomainConfigs) (IIdentityBackend, error) {
	if ident, ok := conf["identity"]; ok {
		if driverJson, ok := ident["driver"]; ok {
			driver, _ := driverJson.GetString()
			switch driver {
			case "ldap":
				return NewLDAPDriver(domainId, conf)
			}
		}
	}
	return NewSQLDriver(domainId, conf)
}
