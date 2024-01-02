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

package options

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/util/reflectutils"

	dbapi "yunion.io/x/onecloud/pkg/apis/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/util/tagutils"
)

// Int returns a pointer to int type with the same value as the argument.  This
// is intended to be used for literal initialization of options
func Int(v int) *int {
	return &v
}

// Bool returns a pointer to bool type with the same value as the argument.
// This is intended to be used for literal initialization of options
func Bool(v bool) *bool {
	return &v
}

func String(v string) *string {
	return &v
}

// IntV returns the integer value as pointed to by the argument if it's
// non-nil, return 0 otherwise
func IntV(p *int) int {
	if p != nil {
		return *p
	}
	return 0
}

// BoolV returns the bool value as pointed to by the argument if it's non-nil,
// return false otherwise
func BoolV(p *bool) bool {
	if p != nil {
		return *p
	}
	return false
}

func StringV(p *string) string {
	if p != nil {
		return *p
	}
	return ""
}

type IParamsOptions interface {
	Params() (*jsonutils.JSONDict, error)
}

var BaseListOptionsType = reflect.TypeOf((*BaseListOptions)(nil)).Elem()

func optionsStructRvToParams(rv reflect.Value) (*jsonutils.JSONDict, error) {
	p := jsonutils.NewDict()
	rvType := rv.Type()
	for i := 0; i < rvType.NumField(); i++ {
		ft := rvType.Field(i)
		jsonInfo := reflectutils.ParseStructFieldJsonInfo(ft)
		name := jsonInfo.MarshalName()
		if name == "" {
			continue
		}
		if jsonInfo.Ignore {
			continue
		}
		f := rv.Field(i)
	begin:
		switch f.Kind() {
		case reflect.Ptr:
			if f.IsNil() {
				continue
			}
			f = f.Elem()
			goto begin
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			rv64 := f.Convert(gotypes.Int64Type)
			i64 := rv64.Interface().(int64)
			if i64 != 0 || !jsonInfo.OmitZero {
				p.Set(name, jsonutils.NewInt(i64))
			}
		case reflect.Bool:
			b := f.Interface().(bool)
			if b || !jsonInfo.OmitFalse {
				p.Set(name, jsonutils.NewBool(b))
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			// NOTE uint64 converted to int64
			rv64 := f.Convert(gotypes.Uint64Type)
			i64 := rv64.Interface().(uint64)
			if i64 != 0 || !jsonInfo.OmitZero {
				p.Set(name, jsonutils.NewInt(int64(i64)))
			}
		case reflect.Float32, reflect.Float64:
			rv64 := f.Convert(gotypes.Float64Type)
			f64 := rv64.Interface().(float64)
			if f64 != 0 || !jsonInfo.OmitZero {
				p.Set(name, jsonutils.NewFloat64(f64))
			}
		case reflect.String:
			s := f.Interface().(string)
			if len(s) > 0 || !jsonInfo.OmitEmpty {
				p.Set(name, jsonutils.NewString(s))
			}
		case reflect.Struct:
			if ft.Anonymous {
				continue
			}
			if f.Type() == gotypes.TimeType {
				t := f.Interface().(time.Time)
				p.Set(name, jsonutils.NewTimeString(t))
				continue
			}
			// TODO
			msg := fmt.Sprintf("do not know what to do with non-anonymous struct field: %s", ft.Name)
			panic(msg)
		case reflect.Map:
			p.Set(name, jsonutils.Marshal(f.Interface()))
		case reflect.Slice, reflect.Array:
			l := f.Len()
			for i := 0; i < l; i++ {
				namei := fmt.Sprintf("%s.%d", name, i)
				vali := jsonutils.Marshal(f.Index(i).Interface())
				p.Set(namei, vali)
			}
		default:
			msg := fmt.Sprintf("unsupported field type %s: %s", ft.Name, ft.Type)
			panic(msg)
		}
	}
	return p, nil
}

func optionsStructToParams(v interface{}) (*jsonutils.JSONDict, error) {
	rv := reflect.Indirect(reflect.ValueOf(v))
	return optionsStructRvToParams(rv)
}

// StructToParams converts the struct as pointed to by the argument to JSON
// dict params, ignoring any embedded in struct
func StructToParams(v interface{}) (*jsonutils.JSONDict, error) {
	return optionsStructToParams(v)
}

// ListStructToParams converts the struct as pointed to by the argument to JSON
// dict params, taking into account .BaseListOptions.Params() if it exists
func ListStructToParams(v interface{}) (*jsonutils.JSONDict, error) {
	rv := reflect.Indirect(reflect.ValueOf(v))
	params, err := optionsStructRvToParams(rv)
	if err != nil {
		return nil, err
	}
	{
		for _, oName := range []string{"BaseListOptions", "MultiArchListOptions"} {
			f := rv.FieldByName(oName)
			if f.IsValid() {
				listOpts, ok := f.Addr().Interface().(IParamsOptions)
				if ok {
					listParams, err := listOpts.Params()
					if err != nil {
						return nil, err
					}
					params.Update(listParams)
				}
			}
		}
	}
	return params, nil
}

const (
	ListOrderAsc  = "asc"
	ListOrderDesc = "desc"
)

type BaseListOptions struct {
	Limit          *int     `default:"20" help:"Page limit"`
	Offset         *int     `default:"0" help:"Page offset"`
	OrderBy        []string `help:"Name of the field to be ordered by"`
	Order          string   `help:"List order" choices:"desc|asc"`
	Details        *bool    `help:"Show more details" default:"false"`
	ShowFailReason *bool    `help:"show fail reason fields"`
	Search         string   `help:"Filter results by a simple keyword search"`
	Meta           *bool    `help:"Piggyback metadata information" json:"with_meta" token:"meta"`
	Filter         []string `help:"Filters"`
	JointFilter    []string `help:"Filters with joint table col; joint_tbl.related_key(origin_key).filter_col.filter_cond(filters)"`
	FilterAny      *bool    `help:"If true, match if any of the filters matches; otherwise, match if all of the filters match"`

	Admin         *bool    `help:"Is an admin call?"`
	Tenant        string   `help:"Tenant ID or Name" alias:"project"`
	ProjectDomain string   `help:"Project domain filter"`
	User          string   `help:"User ID or Name"`
	Field         []string `help:"Show only specified fields"`
	Scope         string   `help:"resource scope" choices:"system|domain|project|user"`

	System           *bool `help:"Show system resource"`
	PendingDelete    *bool `help:"Show only pending deleted resources"`
	PendingDeleteAll *bool `help:"Show also pending-deleted resources" json:"-"`
	DeleteAll        *bool `help:"Show also deleted resources" json:"-"`
	ShowEmulated     *bool `help:"Show all resources including the emulated resources"`

	ExportKeys string `help:"Export field keys"`
	ExtraListOptions

	Tags      []string `help:"filter by Tags, key and value separated by \"=\", keyvalue pairs separated by \";\", eg: \"hypervisor=aliyun;os_type=Linux;os_version\"" json:"-"`
	NoTags    []string `help:"List resources without this tags, key and value separated by \"=\", keyvalue pairs separated by \";\", eg: os_type=Linux, os_version" json:"-"`
	UserTags  []string `help:"UserTags info, key and value separated by \"=\", keyvalue pairs separated by \";\", eg: group=rd" json:"-"`
	CloudTags []string `help:"CloudTags info, key and value separated by \"=\", keyvalue pairs separated by \";\", eg: price_key=cn-beijing" json:"-"`

	NoUserTags  []string `help:"filter by without these UserTags, key and value separated by \"=\", keyvalue pairs separated by \";\", eg: group=rd" json:"-"`
	NoCloudTags []string `help:"filter by no these CloudTags, key and value separated by \"=\", keyvalue pairs separated by \";\", eg: price_key=cn-beijing" json:"-"`

	ProjectTags   []string `help:"filter by project tags, key and value separated by \"=\", keyvalue pairs separated by \";\"" json:"-"`
	NoProjectTags []string `help:"filter by no these project tags, key and value separated by \"=\", keyvalue pairs separated by \";\"" json:"-"`
	DomainTags    []string `help:"filter by domain project tags, key and value separated by \"=\", keyvalue pairs separated by \";\"" json:"-"`
	NoDomainTags  []string `help:"filter by no these domain tags, key and value separated by \"=\", keyvalue pairs separated by \";\"" json:"-"`

	ProjectOrganizations []string `help:"filter by projects of specified organizations"`
	DomainOrganizations  []string `help:"filter by domains of specified organizations"`

	Manager      []string `help:"List objects belonging to the cloud provider" json:"manager,omitempty"`
	Account      string   `help:"List objects belonging to the cloud account" json:"account,omitempty"`
	Provider     []string `help:"List objects from the provider" choices:"OneCloud|VMware|Aliyun|Apsara|Qcloud|Azure|Aws|Huawei|OpenStack|Ucloud|VolcEngine|ZStack|Google|Ctyun|Cloudpods|Nutanix|BingoCloud|IncloudSphere|JDcloud|Proxmox|Ceph|Ecloud|HCSO|HCS|HCSOP|H3C|S3|RemoteFile|Ksyun|Baidu|QingCloud" json:"provider,omitempty"`
	Brand        []string `help:"List objects belonging to a special brand"`
	CloudEnv     string   `help:"Cloud environment" choices:"public|private|onpremise|private_or_onpremise" json:"cloud_env,omitempty"`
	PublicCloud  *bool    `help:"List objects belonging to public cloud" json:"public_cloud"`
	PrivateCloud *bool    `help:"List objects belonging to private cloud" json:"private_cloud"`
	IsOnPremise  *bool    `help:"List objects belonging to on premise infrastructures" token:"on-premise" json:"is_on_premise"`
	IsManaged    *bool    `help:"List objects managed by external providers" token:"managed" json:"is_managed"`

	PagingMarker string `help:"Marker for pagination" json:"paging_marker"`
	PagingOrder  string `help:"paging order" choices:"DESC|ASC"`

	OrderByTag string `help:"Order results by tag values, composed by a tag key and order, e.g user:部门:ASC"`

	Delete string `help:"show deleted records"`

	Id []string `help:"filter by id"`
	// Name []string `help:"fitler by name"`
}

func (opts *BaseListOptions) addTag(keyPrefix, tagstr string, idx int, params *jsonutils.JSONDict) error {
	return opts.addTagInternal(keyPrefix, "obj_tags", tagstr, idx, params)
}

func (opts *BaseListOptions) addNoTag(keyPrefix, tagstr string, idx int, params *jsonutils.JSONDict) error {
	return opts.addTagInternal(keyPrefix, "no_obj_tags", tagstr, idx, params)
}

func (opts *BaseListOptions) addTagInternal(keyPrefix, tagName, tagstr string, idx int, params *jsonutils.JSONDict) error {
	tags := SplitTag(tagstr)
	if len(tags) == 0 {
		return fmt.Errorf("empty tags")
	}
	for i, tag := range tags {
		params.Add(jsonutils.NewString(keyPrefix+tag.Key), fmt.Sprintf("%s.%d.%d.key", tagName, idx, i))
		params.Add(jsonutils.NewString(tag.Value), fmt.Sprintf("%s.%d.%d.value", tagName, idx, i))
	}
	return nil
}

func SplitTag(tag string) tagutils.TTagSet {
	tags := make(tagutils.TTagSet, 0)
	tagInfoList := strings.Split(tag, ";")
	for _, tagInfo := range tagInfoList {
		if len(tagInfo) == 0 {
			continue
		}
		tagInfo := strings.Split(tagInfo, "=")
		tag := tagutils.STag{Key: tagInfo[0]}
		if len(tagInfo) > 1 {
			tag.Value = tagInfo[1]
		}
		tags = append(tags, tag)
	}
	return tags
}

func (opts *BaseListOptions) Params() (*jsonutils.JSONDict, error) {
	params, err := optionsStructToParams(opts)
	if err != nil {
		return nil, err
	}
	if len(opts.Filter) == 0 {
		params.Remove("filter_any")
	}
	if BoolV(opts.DeleteAll) {
		params.Set("delete", jsonutils.NewString("all"))
	}
	if BoolV(opts.PendingDeleteAll) {
		params.Set("pending_delete", jsonutils.NewString("all"))
	}
	/*if opts.Admin == nil {
		requiresSystem := len(opts.Tenant) > 0 ||
			len(opts.ProjectDomain) > 0 ||
			BoolV(opts.System) ||
			BoolV(opts.PendingDelete) ||
			BoolV(opts.PendingDeleteAll)
		if requiresSystem {
			params.Set("admin", jsonutils.JSONTrue)
		}
	}*/
	tagIdx, noTagIdx := 0, 0
	for _, tag := range opts.Tags {
		err = opts.addTag("", tag, tagIdx, params)
		if err != nil {
			return nil, errors.Wrap(err, "Tags")
		}
		tagIdx++
	}
	for _, tag := range opts.NoTags {
		err = opts.addNoTag("", tag, noTagIdx, params)
		if err != nil {
			return nil, errors.Wrap(err, "NoTags")
		}
		noTagIdx++
	}
	for _, tag := range opts.UserTags {
		err = opts.addTag(dbapi.USER_TAG_PREFIX, tag, tagIdx, params)
		if err != nil {
			return nil, errors.Wrap(err, "UserTags")
		}
		tagIdx++
	}
	for _, tag := range opts.NoUserTags {
		err = opts.addNoTag(dbapi.USER_TAG_PREFIX, tag, noTagIdx, params)
		if err != nil {
			return nil, errors.Wrap(err, "NoUserTags")
		}
		noTagIdx++
	}
	for _, tag := range opts.CloudTags {
		err = opts.addTag(dbapi.CLOUD_TAG_PREFIX, tag, tagIdx, params)
		if err != nil {
			return nil, errors.Wrap(err, "CloudTags")
		}
		tagIdx++
	}
	for _, tag := range opts.NoCloudTags {
		err = opts.addNoTag(dbapi.CLOUD_TAG_PREFIX, tag, noTagIdx, params)
		if err != nil {
			return nil, errors.Wrap(err, "NoCloudTags")
		}
		noTagIdx++
	}
	projTagIdx := 0
	for _, tag := range opts.ProjectTags {
		err := opts.addTagInternal("", "project_tags", tag, projTagIdx, params)
		if err != nil {
			return nil, errors.Wrap(err, "ProjectTags")
		}
		projTagIdx++
	}
	noProjTagIdx := 0
	for _, tag := range opts.NoProjectTags {
		err := opts.addTagInternal("", "no_project_tags", tag, noProjTagIdx, params)
		if err != nil {
			return nil, errors.Wrap(err, "NoProjectTags")
		}
		noProjTagIdx++
	}
	for i, orgId := range opts.ProjectOrganizations {
		params.Add(jsonutils.NewString(orgId), fmt.Sprintf("project_organizations.%d", i))
	}
	domainTagIdx := 0
	for _, tag := range opts.DomainTags {
		err := opts.addTagInternal("", "domain_tags", tag, domainTagIdx, params)
		if err != nil {
			return nil, errors.Wrap(err, "DomainTags")
		}
		domainTagIdx++
	}
	noDomainTagIdx := 0
	for _, tag := range opts.NoDomainTags {
		err := opts.addTagInternal("", "no_domain_tags", tag, noDomainTagIdx, params)
		if err != nil {
			return nil, errors.Wrap(err, "NoDomainTags")
		}
		noDomainTagIdx++
	}
	for i, orgId := range opts.DomainOrganizations {
		params.Add(jsonutils.NewString(orgId), fmt.Sprintf("domain_organizations.%d", i))
	}
	return params, nil
}

func (o *BaseListOptions) GetExportKeys() string {
	return o.ExportKeys
}

type ExtraListOptions struct {
	ExportFile  string `help:"Export to file" metavar:"<EXPORT_FILE_PATH>" json:"-"`
	ExportTexts string `help:"Export field displayname texts" json:"-"`
}

func (o ExtraListOptions) GetExportFile() string {
	return o.ExportFile
}

func (o ExtraListOptions) GetExportTexts() string {
	return o.ExportTexts
}

func (o ExtraListOptions) GetContextId() string {
	return ""
}

type ScopedResourceListOptions struct {
	BelongScope string `help:"Filter by resource belong scope" choices:"system|domain|project"`
}

func (o *ScopedResourceListOptions) Params() (*jsonutils.JSONDict, error) {
	return optionsStructToParams(o)
}

type MultiArchListOptions struct {
	OsArch string `help:"Filter resource by arch" choices:"i386|x86|x86_32|x86_64|arm|aarch32|aarch64"`
}

func (o *MultiArchListOptions) Params() (*jsonutils.JSONDict, error) {
	return optionsStructToParams(o)
}

type BaseUpdateOptions struct {
	ID   string `help:"ID or Name of resource to update" json:"-"`
	Name string `help:"Name of resource to update" json:"name"`
	Desc string `metavar:"<DESCRIPTION>" help:"Description" json:"description"`
}

func (opts *BaseUpdateOptions) GetId() string {
	return opts.ID
}

func (opts *BaseUpdateOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.NewDict()
	if len(opts.Name) > 0 {
		params.Add(jsonutils.NewString(opts.Name), "name")
	}
	if len(opts.Desc) > 0 {
		params.Add(jsonutils.NewString(opts.Desc), "description")
	}
	return params, nil
}

type BasePublicOptions struct {
	ID             string   `help:"ID or name of resource" json:"-"`
	Scope          string   `help:"sharing scope" choices:"system|domain|project"`
	SharedDomains  []string `help:"share to domains"`
	SharedProjects []string `help:"share to projects"`
}

func (opts *BasePublicOptions) GetId() string {
	return opts.ID
}

func (opts *BasePublicOptions) Params() (jsonutils.JSONObject, error) {
	params := jsonutils.Marshal(opts).(*jsonutils.JSONDict)
	params.Remove("id")
	return params, nil
}

type BaseCreateOptions struct {
	NAME string `json:"name" help:"Resource Name"`
	Desc string `metavar:"<DESCRIPTION>" help:"Description" json:"description"`
}

func (opts *BaseCreateOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(opts), nil
}

type EnabledStatusCreateOptions struct {
	BaseCreateOptions
	Status  string
	Enabled *bool `help:"turn on enabled flag"`
}

type BaseIdOptions struct {
	ID string `json:"-"`
}

func (o *BaseIdOptions) GetId() string {
	return o.ID
}

func (o *BaseIdOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type BaseIdsOptions struct {
	ID []string `json:"-"`
}

func (o *BaseIdsOptions) GetIds() []string {
	return o.ID
}

func (o *BaseIdsOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type BaseShowOptions struct {
	BaseIdOptions
	WithMeta       *bool `help:"With meta data"`
	ShowFailReason *bool `help:"show fail reason fields"`
}

func (o BaseShowOptions) Params() (jsonutils.JSONObject, error) {
	return StructToParams(o)
}

type ChangeOwnerOptions struct {
	BaseIdOptions
	ProjectDomain string `json:"project_domain" help:"target domain"`
}

func (o ChangeOwnerOptions) Params() (jsonutils.JSONObject, error) {
	if len(o.ProjectDomain) == 0 {
		return nil, fmt.Errorf("empty project_domain")
	}
	return jsonutils.Marshal(map[string]string{"project_domain": o.ProjectDomain}), nil
}

type StatusStatisticsOptions struct {
}

func (o StatusStatisticsOptions) Property() string {
	return "statistics"
}

type ProjectStatisticsOptions struct {
}

func (o ProjectStatisticsOptions) Property() string {
	return "project-statistics"
}

type DomainStatisticsOptions struct {
}

func (o DomainStatisticsOptions) Property() string {
	return "domain-statistics"
}
