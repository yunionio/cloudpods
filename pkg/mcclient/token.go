package mcclient

import (
	"time"

	"github.com/yunionio/pkg/gotypes"
)

type ExternalService struct {
	Name string
	Url  string
}

type TokenCredential interface {
	gotypes.ISerializable

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
	GetServiceURL(service, region, zone, endpointType string) (string, error)
	GetInternalServices(region string) []string
	GetExternalServices(region string) []ExternalService
}
