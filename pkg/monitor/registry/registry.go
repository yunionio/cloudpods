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

package registry

import (
	"context"
	"reflect"
	"sort"
)

type Descriptor struct {
	Name         string
	Instance     Service
	InitPriority Priority
}

var services []*Descriptor

func RegisterService(instance Service) {
	services = append(services, &Descriptor{
		Name:         reflect.TypeOf(instance).Elem().Name(),
		Instance:     instance,
		InitPriority: Low,
	})
}

func Register(descriptor *Descriptor) {
	services = append(services, descriptor)
}

func GetServices() []*Descriptor {
	slice := getServicesWithOverrides()

	sort.Slice(slice, func(i, j int) bool {
		return slice[i].InitPriority > slice[j].InitPriority
	})

	return slice
}

type OverrideServiceFunc func(descriptor Descriptor) (*Descriptor, bool)

var overrides []OverrideServiceFunc

func getServicesWithOverrides() []*Descriptor {
	slice := []*Descriptor{}
	for _, s := range services {
		var descriptor *Descriptor
		for _, fn := range overrides {
			if newDescriptor, override := fn(*s); override {
				descriptor = newDescriptor
				break
			}
		}

		if descriptor != nil {
			slice = append(slice, descriptor)
		} else {
			slice = append(slice, s)
		}
	}

	return slice
}

// Service interface is the lowest common shape that services
// are expected to forfill to be started within monitor.
type Service interface {

	// Init is called by monitor main process which gives the service
	// the possibility do some initial work before its started. Things
	// like adding routes, bus handlers should be done in the Init function
	Init() error
}

// CanBeDisabled allows the services to decide if it should
// be started or not by itself. This is useful for services
// that might not always be started, ex alerting.
// This will be called after `Init()`.
type CanBeDisabled interface {

	// IsDisabled should return a bool saying if it can be started or not.
	IsDisabled() bool
}

// BackgroundService should be implemented for services that have
// long running tasks in the background.
type BackgroundService interface {
	// Run starts the background process of the service after `Init` have been called
	// on all services. The `context.Context` passed into the function should be used
	// to subscribe to ctx.Done() so the service can be notified when monitor shuts down.
	Run(ctx context.Context) error
}

// IsDisabled takes an service and return true if its disabled
func IsDisabled(srv Service) bool {
	canBeDisabled, ok := srv.(CanBeDisabled)
	return ok && canBeDisabled.IsDisabled()
}

type Priority int

const (
	High Priority = 100
	Low  Priority = 0
)
