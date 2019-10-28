package apis

import (
	"time"
)

type SResourceBase struct {
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	UpdateVersion int       `json:"updated_version"`
	DeletedAt     time.Time `json:"deleted_at"`
	Deleted       bool      `json:"deleted"`
}

type SStandaloneResourceBase struct {
	SResourceBase
	Id          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	IsEmulated  bool   `json:"is_emulated"`
}

type SStatusStandaloneResourceBase struct {
	SStandaloneResourceBase
	Status string `json:"status"`
}

type SDomainizedResourceBase struct {
	DomainId string `json:"domain_id"`
}

type SProjectizedResourceBase struct {
	SDomainizedResourceBase
	ProjectId string `json:"tenant_id"`
}

type SVirtualResourceBase struct {
	SStatusStandaloneResourceBase
	SProjectizedResourceBase

	ProjectSrc string `json:"project_src"`
	IsSystem   bool   `json:"is_system"`

	PendingDeletedAt time.Time `json:"pending_deleted_at"`
	PendingDeleted   bool      `json:"pending_deleted"`
}

type SSharableVirtualResourceBase struct {
	SVirtualResourceBase

	IsPublic    bool   `json:"is_public"`
	PublicScope string `json:"public_scope"`
}

type SExternalizedResourceBase struct {
	ExternalId string `json:"external_id"`
}

type SBillingResourceBase struct {
	BillingType  string    `json:"billing_type"`
	ExpiredAt    time.Time `json:"expired_at"`
	BillingCycle string    `json:"billing_cycle"`
}
