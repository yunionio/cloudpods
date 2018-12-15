package models

import (
	"time"
)

type IResource interface {
	GetUpdateVersion() int
	GetUpdatedAt() time.Time
}

type IStandaloneResource interface {
	IResource
	GetId() string
}

type IVirtualResource interface {
	IStandaloneResource
	GetPendingDeleted() bool
}

type Resource struct {
	CreatedAt     time.Time
	UpdatedAt     time.Time
	UpdateVersion int
	DeletedAt     time.Time
	Deleted       bool
}

func (r *Resource) GetUpdateVersion() int {
	return r.UpdateVersion
}

func (r *Resource) GetUpdatedAt() time.Time {
	return r.UpdatedAt
}

type StandaloneResource struct {
	Resource

	Id          string
	Name        string
	ExternalId  string
	Description string
	IsEmulated  bool
}

func (r *StandaloneResource) GetId() string {
	return r.Id
}

type StatusStandaloneResource struct {
	StandaloneResource

	Status string
}

type VirtualResource struct {
	StatusStandaloneResource

	ProjectId        string
	IsSystem         bool
	PendingDeletedAt time.Time
	PendingDeleted   bool
}

type ManagedResource struct {
	ManagerId string
}

func (r *VirtualResource) GetPendingDeleted() bool {
	return r.PendingDeleted
}

type SharableVirtualResource struct {
	VirtualResource

	IsPublic bool
}
