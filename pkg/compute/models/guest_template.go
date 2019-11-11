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
	"context"
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/sets"

	"yunion.io/x/onecloud/pkg/cloudcommon/cmdline"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	computeapis "yunion.io/x/onecloud/pkg/apis/compute"
)

const (
	IMAGE_TYPE_NORMAL = "normal"
	IMAGE_TYPE_GUEST  = "guest"
)

type SGuestTemplateManager struct {
	db.SSharableVirtualResourceBaseManager
}

type SGuestTemplate struct {
	db.SSharableVirtualResourceBase

	VcpuCount  int    `nullable:"false" default:"1" create:"optional"`
	VmemSize   int    `nullable:"false" create:"optional"`
	OsType     string `width:"36" charset:"ascii" nullable:"true" create:"optional"`
	ImageType  string `width:"10" charset:"ascii" nullabel:"true" default:"normal" create:"optional"`
	ImageId    string `width:"128" charset:"ascii" create:"optional"`
	Hypervisor string `width:"16" charset:"ascii" default:"kvm" create:"optional"`

	Content jsonutils.JSONObject `nullable:"false" list:"user" update:"user" create:"optional"`
}

var GuestTemplateManager *SGuestTemplateManager

func init() {
	GuestTemplateManager = &SGuestTemplateManager{
		SSharableVirtualResourceBaseManager: db.NewSharableVirtualResourceBaseManager(
			SGuestTemplate{},
			"guesttemplates_tbl",
			"servertemplate",
			"servertemplates",
		),
	}

	GuestTemplateManager.SetVirtualObject(GuestTemplateManager)
}

func (gtm *SGuestTemplateManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.
	IIdentityProvider, query jsonutils.JSONObject, input *computeapis.GuesttemplateCreateInput) (*jsonutils.JSONDict, error) {

	if input.Content == nil {
		return nil, httperrors.NewMissingParameterError("content")
	}

	return gtm.validateData(ctx, userCred, ownerId, query, input)
}

func (gtm *SGuestTemplateManager) validateData(ctx context.Context, userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, cinput *computeapis.GuesttemplateCreateInput) (
	*jsonutils.JSONDict, error) {

	content := cinput.Content
	data := jsonutils.NewDict()
	data.Add(jsonutils.NewString(cinput.Name), "name")
	// not support guest image and guest snapshot for now
	if content.Contains("instance_snapshot_id") {
		return nil, httperrors.NewInputParameterError(
			"no support for instance snapshot in guest template for now")
	}
	copy := jsonutils.DeepCopy(content).(*jsonutils.JSONDict)
	input, err := GuestManager.validateCreateData(ctx, userCred, ownerId, query, copy)
	if err != nil {
		return nil, httperrors.NewInputParameterError(err.Error())
	}
	// fill field
	data.Add(jsonutils.NewInt(int64(input.VmemSize)), "vmem_size")
	data.Add(jsonutils.NewInt(int64(input.VcpuCount)), "vcpu_count")
	data.Add(jsonutils.NewString(input.OsType), "os_type")
	data.Add(jsonutils.NewString(input.Hypervisor), "hypervisor")

	if len(input.GuestImageID) > 0 {
		data.Add(jsonutils.NewString(IMAGE_TYPE_GUEST), "image_type")
		data.Add(jsonutils.NewString(input.GuestImageID), "image_id")
	} else {
		data.Add(jsonutils.NewString(input.Disks[0].ImageId), "image_id")
		data.Add(jsonutils.NewString(IMAGE_TYPE_NORMAL), "image_type")
	}

	// hide some properties
	contentDict := content.(*jsonutils.JSONDict)
	contentDict.Remove("name")
	contentDict.Remove("generate_name")
	// "__count__" was converted to "count" by apigateway
	contentDict.Remove("count")
	contentDict.Remove("project_id")

	data.Add(contentDict, "content")
	return data, nil
}

func (gt *SGuestTemplate) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, data *computeapis.GuesttemplateCreateInput) (*jsonutils.JSONDict, error) {

	return GuestTemplateManager.validateData(ctx, userCred, gt.GetOwnerId(), query, data)
}

func (gt *SGuestTemplate) ValidateDeleteCondition(ctx context.Context) error {
	return nil
}

func (gt *SGuestTemplate) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {

	extra, err := gt.SSharableVirtualResourceBase.GetExtraDetails(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	return gt.getMoreDetails(ctx, userCred, extra), nil
}

func (gt *SGuestTemplate) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject) *jsonutils.JSONDict {

	return gt.getMoreDetails(ctx, userCred, gt.SSharableVirtualResourceBase.GetCustomizeColumns(ctx, userCred, query))
}

func (gt *SGuestTemplate) getMoreDetails(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject) *jsonutils.JSONDict {

	dict := query.(*jsonutils.JSONDict)
	input, err := cmdline.FetchServerCreateInputByJSON(gt.Content)
	if err != nil {
		return dict
	}
	configInfo := make(map[string]interface{}, 12)
	if len(input.PreferRegion) != 0 {
		region := CloudregionManager.FetchRegionById(input.PreferRegion)
		if region != nil {
			input.PreferRegion = region.GetName()
		}
		configInfo["region"] = input.PreferRegion

	}
	if len(input.PreferZone) != 0 {
		zone := ZoneManager.FetchZoneById(input.PreferZone)
		if zone != nil {
			input.PreferZone = zone.GetName()
		}
		configInfo["zone"] = input.PreferZone
	}
	configInfo["hypervisor"] = gt.Hypervisor
	configInfo["os_type"] = gt.OsType
	if len(input.InstanceType) > 0 {
		configInfo["sku"] = input.InstanceType
	}
	type disk struct {
		Backend  string
		DiskType string
		Index    int
		SizeMb   int
	}
	disks := make([]disk, len(input.Disks))
	for i := range input.Disks {
		disks[i] = disk{
			Backend:  input.Disks[i].Backend,
			DiskType: input.Disks[i].DiskType,
			Index:    input.Disks[i].Index,
			SizeMb:   input.Disks[i].SizeMb,
		}
	}
	configInfo["disks"] = disks
	if len(input.KeypairId) > 0 {
		model, err := KeypairManager.FetchById(input.KeypairId)
		if err == nil {
			keypair := model.(*SKeypair)
			configInfo["keypair"] = keypair.GetName()
		}
	}
	if len(input.Networks) > 0 {
		networkNameList := make([]string, 0, len(input.Networks))
		networkList := make([]string, len(input.Networks))
		for i := range input.Networks {
			networkList[i] = input.Networks[i].Network
		}
		networkSet := sets.NewString(networkList...)
		pairs := make([]struct {
			ID   string
			Name string
		}, 0, len(networkList))

		q := NetworkManager.Query("id", "name").In("id", networkSet)
		q.All(&pairs)

		for _, p := range pairs {
			if networkSet.Has(p.ID) {
				networkSet.Delete(p.ID)
			}
			networkNameList = append(networkNameList, p.Name)
		}

		for _, name := range networkSet.UnsortedList() {
			networkNameList = append(networkNameList, name)
		}

		configInfo["nets"] = networkNameList
	}
	if len(input.SecgroupId) > 0 {
		secgroup := SecurityGroupManager.FetchSecgroupById(input.SecgroupId)
		if secgroup != nil {
			input.SecgroupId = secgroup.GetName()
		}
		configInfo["secgroup"] = input.SecgroupId
	}
	if input.IsolatedDevices != nil && len(input.IsolatedDevices) != 0 {
		configInfo["isolated_device_config"] = input.IsolatedDevices
	}

	// fill image info
	switch gt.ImageType {
	case IMAGE_TYPE_NORMAL:
		image, err := CachedimageManager.getImageInfo(ctx, userCred, gt.ImageId, false)
		if err == nil {
			configInfo["image"] = image.Name
		} else {
			configInfo["image"] = gt.ImageId
		}
	case IMAGE_TYPE_GUEST:
		s := auth.GetSession(ctx, userCred, options.Options.Region, "")
		ret, err := modules.GuestImages.Get(s, gt.ImageId, jsonutils.JSONNull)
		if err != nil || !ret.Contains("id") {
			configInfo["image"] = gt.ImageId
		} else {
			name, _ := ret.GetString("id")
			configInfo["image"] = name
		}
	default:
		// no arrivals
	}

	dict.Add(jsonutils.Marshal(configInfo), "config_info")
	return dict
}

func (gt *SGuestTemplate) PerformPublic(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {

	// image, network, secgroup, instancegroup
	input, err := cmdline.FetchServerCreateInputByJSON(gt.Content)
	if err != nil {
		return nil, errors.Wrap(err, "fail to convert content of guest template to ServerCreateInput")
	}

	// check for below private resource in the guest template
	privateResource := map[string]int{
		"keypair":           len(input.KeypairId),
		"instance group":    len(input.InstanceGroupIds),
		"instance snapshot": len(input.InstanceSnapshotId),
	}
	for k, v := range privateResource {
		if v > 0 {
			return nil, gt.genForbiddenError(k, "", "")
		}
	}

	targetScopeStr, _ := data.GetString("scope")
	targetScope := rbacutils.String2ScopeDefault(targetScopeStr, rbacutils.ScopeSystem)

	// check if secgroup is public
	if len(input.SecgroupId) > 0 {
		model, err := SecurityGroupManager.FetchByIdOrName(userCred, input.SecgroupId)
		if err != nil {
			return nil, httperrors.NewResourceNotFoundError("there is no such secgroup %s descripted by guest template",
				input.SecgroupId)
		}
		secgroup := model.(*SSecurityGroup)
		sgScope := rbacutils.String2Scope(secgroup.PublicScope)
		if !secgroup.IsPublic || !sgScope.HigherEqual(targetScope) {
			return nil, gt.genForbiddenError("security group", input.SecgroupId, string(targetScope))
		}
	}

	// check if networks is public
	if len(input.Networks) > 0 {
		for i := range input.Networks {
			str := input.Networks[i].Network
			model, err := NetworkManager.FetchByIdOrName(userCred, str)
			if err != nil {
				return nil, httperrors.NewResourceNotFoundError(
					"there is no such secgroup %s descripted by guest template")
			}
			network := model.(*SNetwork)
			netScope := rbacutils.String2Scope(network.PublicScope)
			if !network.IsPublic || !netScope.HigherEqual(targetScope) {
				return nil, gt.genForbiddenError("network", str, string(targetScope))
			}
		}
	}

	// check if image is public
	var (
		isPublic    bool
		publicScope string
	)
	switch gt.ImageType {
	case IMAGE_TYPE_NORMAL:
		image, err := CachedimageManager.GetImageById(ctx, userCred, gt.ImageId, false)
		if err != nil {
			return nil, errors.Wrapf(err, "fail to fetch image %s descripted by guest template", gt.ImageId)
		}
		isPublic, publicScope = image.IsPublic, image.PublicScope
	case IMAGE_TYPE_GUEST:
		s := auth.GetSession(ctx, userCred, options.Options.Region, "")
		ret, err := modules.GuestImages.Get(s, gt.ImageId, jsonutils.JSONNull)
		if err != nil {
			return nil, errors.Wrapf(err, "fail to fetch guest image %s descripted by guest template", gt.ImageId)
		}
		isPublic = jsonutils.QueryBoolean(ret, "is_public", false)
		publicScope, _ = ret.GetString("public_scope")
	default:
		//no arrivals
	}
	igScope := rbacutils.String2Scope(publicScope)
	if !isPublic || !igScope.HigherEqual(targetScope) {
		return nil, gt.genForbiddenError("image", "", string(targetScope))
	}

	return gt.SSharableVirtualResourceBase.PerformPublic(ctx, userCred, query, data)
}

func (gt *SGuestTemplate) genForbiddenError(resourceName, resourceStr, scope string) error {
	var msg string
	if len(resourceStr) == 0 {
		msg = fmt.Sprintf("the %s in guest template is not a public resource", resourceName)
	} else {
		msg = fmt.Sprintf("the %s '%s' in guest template is not a public resource", resourceName, resourceStr)
	}
	if len(scope) > 0 {
		msg += fmt.Sprintf(" in %s scope", scope)
	}
	return httperrors.NewForbiddenError(msg)
}
