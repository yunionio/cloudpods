package policy

import (
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/gotypes"

	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

type SPolicyTokenCredential struct {
	Token mcclient.TokenCredential
}

func (self *SPolicyTokenCredential) String() string {
	return self.Token.String()
}

func (self *SPolicyTokenCredential) IsZero() bool {
	return self.Token.IsZero()
}

func (self *SPolicyTokenCredential) GetProjectId() string {
	return self.Token.GetProjectId()
}

func (self *SPolicyTokenCredential) GetTenantId() string {
	return self.Token.GetTenantId()
}

func (self *SPolicyTokenCredential) GetUserId() string {
	return self.Token.GetUserId()
}

func (self *SPolicyTokenCredential) GetServiceURL(service, region, zone, endpointType string) (string, error) {
	return self.Token.GetServiceURL(service, region, zone, endpointType)
}

func (self *SPolicyTokenCredential) GetServiceURLs(service, region, zone, endpointType string) ([]string, error) {
	return self.Token.GetServiceURLs(service, region, zone, endpointType)
}

func (self *SPolicyTokenCredential) GetTokenString() string {
	return self.Token.GetTokenString()
}

func (self *SPolicyTokenCredential) GetDomainId() string {
	return self.Token.GetDomainId()
}

func (self *SPolicyTokenCredential) GetDomainName() string {
	return self.Token.GetDomainName()
}

func (self *SPolicyTokenCredential) GetTenantName() string {
	return self.Token.GetTenantName()
}

func (self *SPolicyTokenCredential) GetProjectName() string {
	return self.Token.GetProjectName()
}

func (self *SPolicyTokenCredential) GetUserName() string {
	return self.Token.GetUserName()
}

func (self *SPolicyTokenCredential) GetRoles() []string {
	return self.Token.GetRoles()
}

func (self *SPolicyTokenCredential) GetExpires() time.Time {
	return self.Token.GetExpires()
}

func (self *SPolicyTokenCredential) IsValid() bool {
	return self.Token.IsValid()
}

func (self *SPolicyTokenCredential) ValidDuration() time.Duration {
	return self.Token.ValidDuration()
}

func (self *SPolicyTokenCredential) GetRegions() []string {
	return self.Token.GetRegions()
}

func (self *SPolicyTokenCredential) GetServiceCatalog() mcclient.IServiceCatalog {
	return self.Token.GetServiceCatalog()
}

func (self *SPolicyTokenCredential) GetCatalogData(serviceTypes []string, region string) jsonutils.JSONObject {
	return self.Token.GetCatalogData(serviceTypes, region)
}

func (self *SPolicyTokenCredential) GetInternalServices(region string) []string {
	return self.Token.GetInternalServices(region)
}

func (self *SPolicyTokenCredential) GetExternalServices(region string) []mcclient.ExternalService {
	return self.Token.GetExternalServices(region)
}

func (self *SPolicyTokenCredential) GetEndpoints(region string, endpointType string) []mcclient.Endpoint {
	return self.Token.GetEndpoints(region, endpointType)
}

func (self *SPolicyTokenCredential) ToJson() jsonutils.JSONObject {
	return self.Token.ToJson()
}

func (self *SPolicyTokenCredential) HasSystemAdminPrivelege() bool {
	if consts.IsRbacEnabled() {
		return PolicyManager.IsAdminCapable(self.Token)
	}
	return self.Token.HasSystemAdminPrivelege()
}

func (self *SPolicyTokenCredential) IsAdminAllow(service string, resource string, action string, extra ...string) bool {
	if consts.IsRbacEnabled() {
		result := PolicyManager.Allow(true, self.Token, service, resource, action, extra...)
		return result == rbacutils.AdminAllow
	}
	return self.Token.IsAdminAllow(service, resource, action, extra...)
}

func init() {
	gotypes.RegisterSerializable(mcclient.TokenCredentialType, func() gotypes.ISerializable {
		return &SPolicyTokenCredential{}
	})
}

func FilterPolicyCredential(token mcclient.TokenCredential) mcclient.TokenCredential {
	if !consts.IsRbacEnabled() {
		return token
	}
	switch token.(type) {
	case *SPolicyTokenCredential:
		return token
	default:
		return &SPolicyTokenCredential{Token: token}
	}
}
