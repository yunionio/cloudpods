package mcclient

import (
	"time"

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

type TokenCredential interface {
	gotypes.ISerializable

	IServiceCatalog

	GetTokenString() string
	GetDomainId() string
	GetDomainName() string
	GetTenantId() string
	GetTenantName() string
	GetProjectId() string
	GetProjectName() string
	GetUserId() string
	GetUserName() string
	GetRoles() []string
	GetExpires() time.Time
	IsValid() bool
	ValidDuration() time.Duration
	IsAdmin() bool
	IsSystemAdmin() bool
	GetRegions() []string

	GetServiceCatalog() IServiceCatalog

	GetInternalServices(region string) []string
	GetExternalServices(region string) []ExternalService
	GetEndpoints(region string, endpointType string) []Endpoint
}
