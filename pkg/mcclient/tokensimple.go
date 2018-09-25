package mcclient

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/gotypes"
)

type SSimpleToken struct {
	Token     string
	Domain    string
	DomainId  string
	User      string
	UserId    string
	Project   string `json:"tenant"`
	ProjectId string `json:"tenant_id"`
	Roles     string
	Expires   time.Time
}

func (self *SSimpleToken) GetTokenString() string {
	return self.Token
}

func (self *SSimpleToken) GetDomainId() string {
	return self.DomainId
}

func (self *SSimpleToken) GetDomainName() string {
	return self.Domain
}

func (self *SSimpleToken) GetTenantId() string {
	return self.ProjectId
}

func (self *SSimpleToken) GetTenantName() string {
	return self.Project
}

func (self *SSimpleToken) GetProjectId() string {
	return self.ProjectId
}

func (self *SSimpleToken) GetProjectName() string {
	return self.Project
}

func (self *SSimpleToken) GetUserId() string {
	return self.UserId
}

func (self *SSimpleToken) GetUserName() string {
	return self.User
}

func (self *SSimpleToken) GetRoles() []string {
	return strings.Split(self.Roles, ",")
}

func (self *SSimpleToken) GetExpires() time.Time {
	return self.Expires
}

func (self *SSimpleToken) IsValid() bool {
	return self.ValidDuration() > 0
}

func (self *SSimpleToken) ValidDuration() time.Duration {
	return time.Until(self.Expires)
}

func (self *SSimpleToken) IsAdmin() bool {
	roles := self.GetRoles()
	for i := 0; i < len(roles); i++ {
		if roles[i] == "admin" {
			return true
		}
	}
	return false
}

func (self *SSimpleToken) IsSystemAdmin() bool {
	return self.IsAdmin() && self.Project == "system"
}

func (self *SSimpleToken) GetRegions() []string {
	return nil
}

func (self *SSimpleToken) GetServiceURL(service, region, zone, endpointType string) (string, error) {
	return "", fmt.Errorf("Not available")
}

func (self *SSimpleToken) GetServiceURLs(service, region, zone, endpointType string) ([]string, error) {
	return nil, fmt.Errorf("Not available")
}

func (self *SSimpleToken) GetInternalServices(region string) []string {
	return nil
}

func (self *SSimpleToken) GetExternalServices(region string) []ExternalService {
	return nil
}

func (this *SSimpleToken) GetEndpoints(region string, endpointType string) []Endpoint {
	return nil
}

func (this *SSimpleToken) GetServiceCatalog() IServiceCatalog {
	return nil
}

func SimplifyToken(token TokenCredential) TokenCredential {
	simToken, ok := token.(*SSimpleToken)
	if ok {
		return simToken
	}
	return &SSimpleToken{Token: token.GetTokenString(),
		Domain:    token.GetDomainName(),
		DomainId:  token.GetDomainId(),
		User:      token.GetUserName(),
		UserId:    token.GetUserId(),
		Project:   token.GetProjectName(),
		ProjectId: token.GetProjectId(),
		Roles:     strings.Join(token.GetRoles(), ","),
		Expires:   token.GetExpires(),
	}
}

func (self *SSimpleToken) GetCatalogData(serviceTypes []string, region string) jsonutils.JSONObject {
	return nil
}

func (self *SSimpleToken) String() string {
	return jsonutils.Marshal(self).String()
}

func (self *SSimpleToken) IsZero() bool {
	return len(self.UserId) == 0 && len(self.ProjectId) == 0
}

var TokenCredentialType reflect.Type

func init() {
	TokenCredentialType = reflect.TypeOf((*TokenCredential)(nil)).Elem()

	gotypes.RegisterSerializable(TokenCredentialType, func() gotypes.ISerializable {
		return &SSimpleToken{}
	})
}
