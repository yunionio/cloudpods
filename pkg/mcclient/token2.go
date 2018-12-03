package mcclient

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
)

type KeystoneEndpointV2 struct {
	Id          string
	InternalURL string
	PublicURL   string
	AdminURL    string
	Region      string
}

type KeystoneServiceV2 struct {
	Name      string
	Type      string
	Endpoints []KeystoneEndpointV2
}

type KeystoneRoleV2 struct {
	Name string
}

type KeystoneUserV2 struct {
	Id       string
	Name     string
	Username string
	Roles    []KeystoneRoleV2
}

type KeystoneTenantV2 struct {
	Id          string
	Name        string
	Enabled     bool
	Description string
}

type KeystoneTokenV2 struct {
	Id      string
	Expires time.Time
	Tenant  KeystoneTenantV2
}

type KeystoneMetadataV2 struct {
	IsAdmin int
	Roles   []string
}

type KeystoneServiceCatalogV2 []KeystoneServiceV2

type TokenCredentialV2 struct {
	Token          KeystoneTokenV2
	ServiceCatalog KeystoneServiceCatalogV2
	User           KeystoneUserV2
	Metadata       KeystoneMetadataV2
}

func (token *TokenCredentialV2) GetTokenString() string {
	return token.Token.Id
}

func (token *TokenCredentialV2) GetDomainId() string {
	return "default"
}

func (token *TokenCredentialV2) GetDomainName() string {
	return "Default"
}

func (token *TokenCredentialV2) GetTenantId() string {
	return token.Token.Tenant.Id
}

func (token *TokenCredentialV2) GetTenantName() string {
	return token.Token.Tenant.Name
}

func (token *TokenCredentialV2) GetProjectId() string {
	return token.GetTenantId()
}

func (token *TokenCredentialV2) GetProjectName() string {
	return token.GetTenantName()
}

func (token *TokenCredentialV2) GetUserName() string {
	return token.User.Username
}

func (token *TokenCredentialV2) GetUserId() string {
	return token.User.Id
}

func (token *TokenCredentialV2) GetRoles() []string {
	roles := make([]string, 0)
	for i := 0; i < len(token.User.Roles); i++ {
		roles = append(roles, token.User.Roles[i].Name)
	}
	return roles
}

func (this *TokenCredentialV2) GetExpires() time.Time {
	return this.Token.Expires
}

func (this *TokenCredentialV2) IsValid() bool {
	return this.ValidDuration() > 0
}

func (this *TokenCredentialV2) ValidDuration() time.Duration {
	return time.Until(this.Token.Expires)
}

func (this *TokenCredentialV2) IsAdmin() bool {
	for i := 0; i < len(this.User.Roles); i++ {
		if this.User.Roles[i].Name == "admin" {
			return true
		}
	}
	return false
}

func (this *TokenCredentialV2) GetRegions() []string {
	return this.ServiceCatalog.getRegions()
}

func (this *TokenCredentialV2) HasSystemAdminPrivelege() bool {
	return this.IsAdmin() && this.GetTenantName() == "system"
}

func (this *TokenCredentialV2) IsAdminAllow(service string, resource string, action string, extra ...string) bool {
	return this.HasSystemAdminPrivelege()
}

func (this *TokenCredentialV2) GetServiceURL(service, region, zone, endpointType string) (string, error) {
	return this.ServiceCatalog.GetServiceURL(service, region, zone, endpointType)
}

func (this *TokenCredentialV2) GetServiceURLs(service, region, zone, endpointType string) ([]string, error) {
	return this.ServiceCatalog.GetServiceURLs(service, region, zone, endpointType)
}

func (this *TokenCredentialV2) GetInternalServices(region string) []string {
	return nil
}

func (this *TokenCredentialV2) GetExternalServices(region string) []ExternalService {
	return nil
}

func (this *TokenCredentialV2) GetEndpoints(region string, endpointType string) []Endpoint {
	return nil
}

func (this *TokenCredentialV2) GetServiceCatalog() IServiceCatalog {
	return this.ServiceCatalog
}

func stringArrayContains(arr []string, needle string) bool {
	for i := 0; i < len(arr); i++ {
		if arr[i] == needle {
			return true
		}
	}
	return false
}

func (catalog KeystoneServiceCatalogV2) getRegions() []string {
	regions := make([]string, 0)
	for i := 0; i < len(catalog); i++ {
		for j := 0; j < len(catalog[i].Endpoints); j++ {
			r := catalog[i].Endpoints[j].Region
			slash := strings.IndexByte(r, '/')
			if slash > 0 {
				r = r[:slash]
			}
			if !stringArrayContains(regions, r) {
				regions = append(regions, r)
			}
		}
	}
	return regions
}

func (catalog KeystoneServiceCatalogV2) getServiceEndpoint(service, region, zone string) (KeystoneEndpointV2, error) {
	var selected KeystoneEndpointV2
	for i := 0; i < len(catalog); i++ {
		if service == catalog[i].Type {
			if len(region) == 0 {
				if len(catalog[i].Endpoints) >= 1 {
					selected = catalog[i].Endpoints[0]
				} else {
					return selected, fmt.Errorf("No default region")
				}
			} else {
				regionEps := make([]KeystoneEndpointV2, 0)
				zoneEps := make([]KeystoneEndpointV2, 0)
				if len(zone) > 0 {
					zone = fmt.Sprintf("%s/%s", region, zone)
				}
				for j := 0; j < len(catalog[i].Endpoints); j++ {
					if catalog[i].Endpoints[j].Region == region {
						regionEps = append(regionEps, catalog[i].Endpoints[j])
					} else if len(zone) > 0 && catalog[i].Endpoints[j].Region == zone {
						zoneEps = append(zoneEps, catalog[i].Endpoints[j])
					}
				}
				if len(zone) > 0 && len(zoneEps) > 0 {
					selected = zoneEps[rand.Intn(len(zoneEps))]
				} else if len(regionEps) > 0 {
					selected = regionEps[rand.Intn(len(regionEps))]
				} else {
					return selected, fmt.Errorf("No such region %s", region)
				}
			}
			return selected, nil
		}
	}
	return selected, fmt.Errorf("No such service %s", service)
}

func (catalog KeystoneServiceCatalogV2) GetServiceURL(service, region, zone, endpointType string) (string, error) {
	ep, err := catalog.getServiceEndpoint(service, region, zone)
	if err != nil {
		return "", err
	}
	return ep.getURL(endpointType), nil
}

func (catalog KeystoneServiceCatalogV2) GetServiceURLs(service, region, zone, endpointType string) ([]string, error) {
	url, err := catalog.GetServiceURL(service, region, zone, endpointType)
	if err != nil {
		return nil, err
	}
	return []string{url}, nil
}

func (ep KeystoneEndpointV2) getURL(epType string) string {
	switch epType {
	case "publicURL":
		return ep.PublicURL
	case "adminURL":
		return ep.AdminURL
	default:
		return ep.InternalURL
	}
}

func (self *TokenCredentialV2) GetCatalogData(serviceTypes []string, region string) jsonutils.JSONObject {
	return jsonutils.Marshal(self.GetServiceCatalog())
}

func (self *TokenCredentialV2) String() string {
	token := SimplifyToken(self)
	return token.String()
}

func (self *TokenCredentialV2) IsZero() bool {
	return len(self.GetUserId()) == 0 && len(self.GetProjectId()) == 0
}

func (self *TokenCredentialV2) ToJson() jsonutils.JSONObject {
	return SimplifyToken(self).ToJson()
}
