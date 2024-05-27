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

package glance

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type ImageListOptions struct {
	options.BaseListOptions

	IsPublic                 string   `help:"filter images public or not(True, False or None)" choices:"true|false"`
	IsStandard               string   `help:"filter images standard or non-standard" choices:"true|false"`
	Protected                string   `help:"filter images by protected" choices:"true|false"`
	IsUefi                   bool     `help:"list uefi image"`
	Format                   []string `help:"Disk formats"`
	SubFormats               []string `help:"Sub formats"`
	Name                     string   `help:"Name filter"`
	OsType                   []string `help:"Type of OS filter e.g. 'Windows, Linux, Freebsd, Android, macOS, VMWare'"`
	OsTypePreciseMatch       bool     `help:"OS precise match"`
	OsArch                   []string `help:"Type of OS arch filter e.g. 'x86, arm, arm64, x86_64'"`
	OsArchPreciseMatch       bool     `help:"OS arch precise match"`
	Distribution             []string `help:"Distribution filter, e.g. 'CentOS, Ubuntu, Debian, Windows'"`
	DistributionPreciseMatch bool     `help:"Distribution precise match"`
}

func (o *ImageListOptions) Params() (jsonutils.JSONObject, error) {
	param, err := o.BaseListOptions.Params()
	if err != nil {
		return nil, err
	}
	params := param.(*jsonutils.JSONDict)
	if len(o.IsPublic) > 0 {
		params.Add(jsonutils.NewString(o.IsPublic), "is_public")
	}
	if len(o.IsStandard) > 0 {
		params.Add(jsonutils.NewString(o.IsStandard), "is_standard")
	}
	if len(o.Protected) > 0 {
		params.Add(jsonutils.NewString(o.Protected), "protected")
	}
	if o.IsUefi {
		params.Add(jsonutils.JSONTrue, "uefi")
	}
	if len(o.Tenant) > 0 {
		params.Add(jsonutils.NewString(o.Tenant), "owner")
	}
	if len(o.Name) > 0 {
		params.Add(jsonutils.NewString(o.Name), "name")
	}
	if len(o.Format) > 0 {
		fs := jsonutils.NewArray()
		for _, f := range o.Format {
			fs.Add(jsonutils.NewString(f))
		}
		params.Add(fs, "disk_formats")
	}
	if len(o.SubFormats) > 0 {
		params.Add(jsonutils.Marshal(o.SubFormats), "sub_formats")
	}
	if len(o.OsType) > 0 {
		params.Add(jsonutils.NewStringArray(o.OsType), "os_types")
	}
	if o.OsTypePreciseMatch {
		params.Add(jsonutils.JSONTrue, "os_type_precise_match")
	}
	if len(o.OsArch) > 0 {
		params.Add(jsonutils.NewStringArray(o.OsArch), "os_archs")
	}
	if o.OsArchPreciseMatch {
		params.Add(jsonutils.JSONTrue, "os_arch_precise_match")
	}
	if len(o.Distribution) > 0 {
		params.Add(jsonutils.NewStringArray(o.Distribution), "distributions")
	}
	if o.DistributionPreciseMatch {
		params.Add(jsonutils.JSONTrue, "distribution_precise_match")
	}
	return params, nil
}

type ImageStatusStatisticsOptions struct {
	ImageListOptions
	options.StatusStatisticsOptions
}
