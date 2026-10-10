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

// Package reflectutils walks struct values by reflection, mainly to give the
// json encoder and decoder in yunion.io/x/jsonutils the field names and the
// tags to honour.
//
// # Field naming
//
// Fields are collected in declaration order, with the fields of an embedded
// struct expanded in place.  A name is taken from the "name" tag, then from
// the "json" tag, and finally from the kebab form of the field name, so it can
// be looked up by any of those.
//
// Two collected fields can carry the same name.  An embedded struct may bring
// in a field that another embedded struct also contributes, and unlike
// encoding/json an outer field does not hide a field of an embedded struct.
// All the fields sharing a name are reported, and a value decoded into the
// struct is written to every one of them, so that code reaching the field
// through any of the embedded paths sees it.  Shadowing is deliberately not
// implemented: the list inputs of the resource APIs rely on this, e.g.
// ServerListInput carries several fields named "domain" coming from different
// embedded structs.
//
// To set such fields apart, tag an embedded struct with
// TAG_AMBIGUOUS_PREFIX, whose value is prepended to the names of the fields
// that embedded struct contributes.
package reflectutils // import "yunion.io/x/pkg/util/reflectutils"
