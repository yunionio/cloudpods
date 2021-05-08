// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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

type EnabledStatusStandaloneResource struct {
	StatusStandaloneResource

	Enabled bool
}

type VirtualResource struct {
	StatusStandaloneResource

	DomainId         string
	ProjectId        string
	IsSystem         bool
	PendingDeletedAt time.Time
	PendingDeleted   bool
}

func (r *VirtualResource) GetPendingDeleted() bool {
	return r.PendingDeleted
}

type SharableVirtualResource struct {
	VirtualResource

	IsPublic bool
}

type ManagedResource struct {
	ManagerId string
}

type ExternalizedResource struct {
	ExternalId string
}
