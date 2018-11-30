package mcclient

import (
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/gotypes"
)

type ExternalService struct {
	Name string
	Url  string
}

type Endpoint struct {
	Id          string
	RegionId    string
	ServiceId   string
	ServiceName string
	Url         string
	Interface   string
}

type IIdentityProvider interface {
	GetProjectId() string
	GetUserId() string
	GetTenantId() string
}

type TokenCredential interface {
	gotypes.ISerializable

	IServiceCatalog

	IIdentityProvider

	GetTokenString() string
	GetDomainId() string
	GetDomainName() string
	GetTenantName() string
	GetProjectName() string
	GetUserName() string
	GetRoles() []string
	GetExpires() time.Time
	IsValid() bool
	ValidDuration() time.Duration
	// IsAdmin() bool
	HasSystemAdminPrivelege() bool
	IsAdminAllow(service string, resource string, action string, extra ...string) bool
	GetRegions() []string

	GetServiceCatalog() IServiceCatalog
	GetCatalogData(serviceTypes []string, region string) jsonutils.JSONObject

	GetInternalServices(region string) []string
	GetExternalServices(region string) []ExternalService
	GetEndpoints(region string, endpointType string) []Endpoint

	ToJson() jsonutils.JSONObject
}
