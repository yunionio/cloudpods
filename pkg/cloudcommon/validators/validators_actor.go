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

package validators

import (
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/netutils"
)

type IValidateActor interface {
	Validate(val interface{}) (bool, error)
	Set(data *jsonutils.JSONDict, key string, val interface{})
	getValue() interface{}
}

type actorSetString struct{}

func (actor *actorSetString) Set(data *jsonutils.JSONDict, key string, val interface{}) {
	s := val.(string)
	data.Set(key, jsonutils.NewString(s))
}

type ActorIPv4Prefix struct {
	actorSetString

	val *netutils.IPV4Prefix
}

func NewActorIPv4Prefix() *ActorIPv4Prefix {
	return &ActorIPv4Prefix{}
}

func (actor *ActorIPv4Prefix) Validate(val interface{}) (bool, error) {
	s, ok := val.(string)
	if !ok {
		return false, fmt.Errorf("invalid type, want string, got %t", val)
	}
	p, err := netutils.NewIPV4Prefix(s)
	if err != nil {
		return false, err
	}
	actor.val = &p
	return true, nil
}

func (actor *ActorIPv4Prefix) getValue() interface{} {
	return actor.val
}

type ActorJoinedBy struct {
	sep         string
	trimSpace   bool
	ignoreEmpty bool
	actor       IValidateActor

	sanitized string
	vals      []interface{}
}

func NewActorJoinedBy(sep string, subActor IValidateActor) *ActorJoinedBy {
	return &ActorJoinedBy{
		sep:         sep,
		trimSpace:   true,
		ignoreEmpty: true,
		actor:       subActor,
	}
}

func (actor *ActorJoinedBy) TrimSpace(b bool) *ActorJoinedBy {
	actor.trimSpace = b
	return actor
}

func (actor *ActorJoinedBy) IgnoreEmpty(b bool) *ActorJoinedBy {
	actor.ignoreEmpty = b
	return actor
}

func (actor *ActorJoinedBy) Validate(val interface{}) (bool, error) {
	s, ok := val.(string)
	if !ok {
		return false, fmt.Errorf("invalid type, want string, got %t", val)
	}
	parts := strings.Split(s, actor.sep)
	sanitizedParts := make([]string, 0, len(parts))
	vals := make([]interface{}, 0, len(parts))
	for i, part := range parts {
		if actor.trimSpace {
			part = strings.TrimSpace(part)
		}
		if actor.ignoreEmpty && part == "" {
			continue
		}
		if _, err := actor.actor.Validate(part); err != nil {
			return false, fmt.Errorf("bad value at index %d: %v", i, err)
		}
		sanitizedParts = append(sanitizedParts, part)
		vals = append(vals, actor.actor.getValue())
	}
	actor.sanitized = strings.Join(sanitizedParts, actor.sep)
	actor.vals = vals
	return true, nil
}

func (actor *ActorJoinedBy) Set(data *jsonutils.JSONDict, key string, val interface{}) {
	data.Set(key, jsonutils.NewString(actor.sanitized))
}

func (actor *ActorJoinedBy) getValue() interface{} {
	return actor.vals
}

type ValidatorByActor struct {
	Validator
	Value interface{}

	actor IValidateActor
}

func NewValidatorByActor(key string, actor IValidateActor) *ValidatorByActor {
	v := &ValidatorByActor{
		Validator: Validator{Key: key},
		actor:     actor,
	}
	v.SetParent(v)
	return v
}

func (v *ValidatorByActor) Default(val interface{}) IValidator {
	if ok, err := v.actor.Validate(val); !ok {
		panic(fmt.Sprintf("invalid default value: %v", err))
	}
	return v.Validator.Default(val)
}

func (v *ValidatorByActor) getValue() interface{} {
	return v.Value
}

func (v *ValidatorByActor) Validate(data *jsonutils.JSONDict) error {
	if err, isSet := v.Validator.validateEx(data); err != nil || !isSet {
		return err
	}
	val := v.value.Interface()
	if ok, err := v.actor.Validate(val); !ok {
		return newInvalidValueErrorEx(v.Key, err)
	}
	v.actor.Set(data, v.Key, val)
	v.Value = v.actor.getValue()
	return nil
}
