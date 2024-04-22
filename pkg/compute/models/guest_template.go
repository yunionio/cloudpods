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
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/util/sets"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	computeapis "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/cmdline"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/image"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

const (
	IMAGE_TYPE_NORMAL = "normal"
	IMAGE_TYPE_GUEST  = "guest"
)

type SGuestTemplateManager struct {
	db.SSharableVirtualResourceBaseManager
	SCloudregionResourceBaseManager
	SVpcResourceBaseManager
}

type SGuestTemplate struct {
	db.SSharableVirtualResourceBase
	SCloudregionResourceBase
	SVpcResourceBase

	// 虚拟机CPU数量
	VcpuCount int `nullable:"false" default:"1" create:"optional" json:"vcpu_count"`

	// 虚拟机内存大小（MB）
	VmemSize int `nullable:"false" create:"optional" json:"vmem_size"`

	// 虚拟机操作系统类型
	// pattern:Linux|Windows|VMWare
	OsType string `width:"36" charset:"ascii" nullable:"true" create:"optional" json:"os_type" list:"user" get:"user"`

	// 镜像类型
	ImageType string `width:"10" charset:"ascii" nullabel:"true" default:"normal" create:"optional" json:"image_type"`

	// 镜像ID
	ImageId string `width:"128" charset:"ascii" create:"optional" json:"image_id"`

	// 虚拟机技术
	Hypervisor string `width:"16" charset:"ascii" default:"kvm" create:"optional" json:"hypervisor"`

	// 计费方式
	BillingType string `width:"16" charset:"ascii" default:"postpaid" create:"optional" list:"user" get:"user" json:"billing_type"`

	// 其他配置信息
	Content jsonutils.JSONObject `nullable:"false" list:"user" update:"user" create:"optional" json:"content"`

	LastCheckTime time.Time
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

func (gtm *SGuestTemplateManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input computeapis.GuestTemplateCreateInput,
) (computeapis.GuestTemplateCreateInput, error) {

	var err error
	if input.Content == nil {
		return input, httperrors.NewMissingParameterError("content")
	}

	if !input.Content.Contains("name") && !input.Content.Contains("generate_name") {
		input.Content.Set("generate_name", jsonutils.NewString(input.Name))
	}

	input.GuestTemplateInput, err = gtm.validateData(ctx, userCred, ownerId, query, input.GuestTemplateInput)
	if err != nil {
		return input, errors.Wrap(err, "gtm.validateData")
	}

	input.SharableVirtualResourceCreateInput, err = gtm.SSharableVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.SharableVirtualResourceCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "SSharableVirtualResourceBaseManager.ValidateCreateData")
	}

	return input, nil
}

func (gt *SGuestTemplate) PostCreate(ctx context.Context, userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	gt.SetStatus(ctx, userCred, computeapis.GT_READY, "")
	gt.updateCheckTime()
	logclient.AddActionLogWithContext(ctx, gt, logclient.ACT_CREATE, nil, userCred, true)
}

func (gt *SGuestTemplate) updateCheckTime() error {
	_, err := db.Update(gt, func() error {
		gt.LastCheckTime = time.Now()
		return nil
	})
	return err
}

func (gt *SGuestTemplate) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, data jsonutils.JSONObject) {
	logclient.AddActionLogWithContext(ctx, gt, logclient.ACT_UPDATE, nil, userCred, true)
}

var HypervisorBrandMap = map[string]string{
	computeapis.HYPERVISOR_KVM:       computeapis.CLOUD_PROVIDER_ONECLOUD,
	computeapis.HYPERVISOR_ESXI:      computeapis.CLOUD_PROVIDER_VMWARE,
	computeapis.HYPERVISOR_ALIYUN:    computeapis.CLOUD_PROVIDER_ALIYUN,
	computeapis.HYPERVISOR_QCLOUD:    computeapis.CLOUD_PROVIDER_QCLOUD,
	computeapis.HYPERVISOR_AZURE:     computeapis.CLOUD_PROVIDER_AZURE,
	computeapis.HYPERVISOR_AWS:       computeapis.CLOUD_PROVIDER_AWS,
	computeapis.HYPERVISOR_HUAWEI:    computeapis.CLOUD_PROVIDER_HUAWEI,
	computeapis.HYPERVISOR_OPENSTACK: computeapis.CLOUD_PROVIDER_OPENSTACK,
	computeapis.HYPERVISOR_UCLOUD:    computeapis.CLOUD_PROVIDER_UCLOUD,
	computeapis.HYPERVISOR_ZSTACK:    computeapis.CLOUD_PROVIDER_ZSTACK,
	computeapis.HYPERVISOR_GOOGLE:    computeapis.CLOUD_PROVIDER_GOOGLE,
	computeapis.HYPERVISOR_CTYUN:     computeapis.CLOUD_PROVIDER_CTYUN,
}

var BrandHypervisorMap = map[string]string{
	computeapis.CLOUD_PROVIDER_ONECLOUD:  computeapis.HYPERVISOR_KVM,
	computeapis.CLOUD_PROVIDER_VMWARE:    computeapis.HYPERVISOR_ESXI,
	computeapis.CLOUD_PROVIDER_ALIYUN:    computeapis.HYPERVISOR_ALIYUN,
	computeapis.CLOUD_PROVIDER_QCLOUD:    computeapis.HYPERVISOR_QCLOUD,
	computeapis.CLOUD_PROVIDER_AZURE:     computeapis.HYPERVISOR_AZURE,
	computeapis.CLOUD_PROVIDER_AWS:       computeapis.HYPERVISOR_AWS,
	computeapis.CLOUD_PROVIDER_HUAWEI:    computeapis.HYPERVISOR_HUAWEI,
	computeapis.CLOUD_PROVIDER_OPENSTACK: computeapis.HYPERVISOR_OPENSTACK,
	computeapis.CLOUD_PROVIDER_UCLOUD:    computeapis.HYPERVISOR_UCLOUD,
	computeapis.CLOUD_PROVIDER_ZSTACK:    computeapis.HYPERVISOR_ZSTACK,
	computeapis.CLOUD_PROVIDER_GOOGLE:    computeapis.HYPERVISOR_GOOGLE,
	computeapis.CLOUD_PROVIDER_CTYUN:     computeapis.HYPERVISOR_CTYUN,
}

func Hypervisor2Brand(hypervisor string) string {
	brand, ok := HypervisorBrandMap[hypervisor]
	if !ok {
		return "unkown"
	}
	return brand
}

func Brand2Hypervisor(brand string) string {
	hypervisor, ok := BrandHypervisorMap[brand]
	if !ok {
		return "unkown"
	}
	return hypervisor
}

func (gtm *SGuestTemplateManager) validateContent(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, content *jsonutils.JSONDict) (*computeapis.ServerCreateInput, error) {
	// hack
	if !content.Contains("name") && !content.Contains("generate_name") {
		content.Set("generate_name", jsonutils.NewString("fake_name"))
	}
	input, err := GuestManager.validateCreateData(ctx, userCred, ownerId, query, content)
	if err != nil {
		return nil, httperrors.NewInputParameterError("%v", err)
	}
	// check Image
	imageId := input.Disks[0].ImageId
	image, err := CachedimageManager.getImageInfo(ctx, userCred, imageId, false)
	if err != nil {
		return nil, errors.Wrapf(err, "getImageInfo of '%s'", imageId)
	}
	if image == nil {
		return nil, fmt.Errorf("no such image %s", imageId)
	}
	return input, nil
}

func (gtm *SGuestTemplateManager) validateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	cinput computeapis.GuestTemplateInput,
) (computeapis.GuestTemplateInput, error) {
	if cinput.Content == nil {
		return cinput, nil
	}
	content := cinput.Content
	// data := cinput.JSON(cinput)
	// not support guest image and guest snapshot for now
	if content.Contains("instance_snapshot_id") {
		return cinput, httperrors.NewInputParameterError(
			"no support for instance snapshot in guest template for now")
	}
	// I don't hope cinput.Content same with data["content"] will change in GuestManager.validateCreateData
	copy := jsonutils.DeepCopy(content).(*jsonutils.JSONDict)
	input, err := gtm.validateContent(ctx, userCred, ownerId, query, copy)
	if err != nil {
		return cinput, httperrors.NewInputParameterError("%v", err)
	}
	log.Debugf("data: %#v", input)
	// fill field
	cinput.VmemSize = input.VmemSize
	// data.Add(jsonutils.NewInt(int64(input.VmemSize)), "vmem_size")
	cinput.VcpuCount = input.VcpuCount
	// data.Add(jsonutils.NewInt(int64(input.VcpuCount)), "vcpu_count")
	cinput.OsType = input.OsType
	// data.Add(jsonutils.NewString(input.OsType), "os_type")
	cinput.Hypervisor = input.Hypervisor
	// data.Add(jsonutils.NewString(input.Hypervisor), "hypervisor")
	cinput.InstanceType = input.InstanceType
	cinput.CloudregionId = input.PreferRegion
	cinput.BillingType = input.BillingType

	// fill vpc
	if len(input.Networks) != 0 && len(input.Networks[0].Network) != 0 {
		model, err := NetworkManager.FetchById(input.Networks[0].Network)
		if err != nil {
			return cinput, errors.Wrap(err, "NetworkManager.FetchById")
		}
		net := model.(*SNetwork)
		vpc, _ := net.GetVpc()
		if vpc != nil {
			cinput.VpcId = vpc.Id
		}
	}

	if len(input.GuestImageID) > 0 {
		cinput.ImageType = IMAGE_TYPE_GUEST
		cinput.ImageId = input.GuestImageID
		// data.Add(jsonutils.NewString(IMAGE_TYPE_GUEST), "image_type")
		// data.Add(jsonutils.NewString(input.GuestImageID), "image_id")
	} else {
		cinput.ImageType = input.GuestImageID
		cinput.ImageId = input.Disks[0].ImageId // if input.Didks is empty???
		// data.Add(jsonutils.NewString(input.Disks[0].ImageId), "image_id")
		// data.Add(jsonutils.NewString(IMAGE_TYPE_NORMAL), "image_type")
	}

	// hide some properties
	content.Remove("name")
	content.Remove("generate_name")
	// "__count__" was converted to "count" by apigateway
	content.Remove("count")
	content.Remove("project_id")
	content.Remove("__count__")

	cinput.Content = content
	// data.Add(contentDict, "content")
	return cinput, nil
}

func (gt *SGuestTemplate) ValidateUpdateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input computeapis.GuestTemplateUpdateInput,
) (computeapis.GuestTemplateUpdateInput, error) {
	var err error
	input.GuestTemplateInput, err = GuestTemplateManager.validateData(ctx, userCred, gt.GetOwnerId(), query, input.GuestTemplateInput)
	if err != nil {
		return input, errors.Wrap(err, "GuestTemplateManager.validateData")
	}
	input.SharableVirtualResourceBaseUpdateInput, err = gt.SSharableVirtualResourceBase.ValidateUpdateData(ctx, userCred, query, input.SharableVirtualResourceBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "SSharableVirtualResourceBase.ValidateUpdateData")
	}
	return input, nil
}

func (manager *SGuestTemplateManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []computeapis.GuestTemplateDetails {
	rows := make([]computeapis.GuestTemplateDetails, len(objs))

	virtRows := manager.SSharableVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	crRows := manager.SCloudregionResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	vpcRows := manager.SVpcResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = computeapis.GuestTemplateDetails{
			SharableVirtualResourceDetails: virtRows[i],
			CloudregionResourceInfo:        crRows[i],
			VpcResourceInfo:                vpcRows[i],
		}
		rows[i], _ = objs[i].(*SGuestTemplate).getMoreDetails(ctx, userCred, rows[i])
	}

	return rows
}

func (gt *SGuestTemplate) getMoreDetails(ctx context.Context, userCred mcclient.TokenCredential,
	out computeapis.GuestTemplateDetails) (computeapis.GuestTemplateDetails, error) {

	input, err := cmdline.FetchServerCreateInputByJSON(gt.Content)
	if err != nil {
		return out, err
	}
	configInfo := computeapis.GuestTemplateConfigInfo{}
	if len(input.PreferZone) != 0 {
		zone := ZoneManager.FetchZoneById(input.PreferZone)
		if zone != nil {
			input.PreferZone = zone.GetName()
			out.ZoneId = zone.GetId()
		}
		out.Zone = input.PreferZone
	}
	out.Brand = Hypervisor2Brand(gt.Hypervisor)

	// metadata
	configInfo.Metadata = input.Metadata

	// sku deal
	if len(input.InstanceType) > 0 {
		skuOutput := computeapis.GuestTemplateSku{}
		sku, err := ServerSkuManager.FetchSkuByNameAndProvider(input.InstanceType, out.Provider, true)
		if err != nil {
			skuOutput.Name = input.InstanceType
			skuOutput.MemorySizeMb = gt.VmemSize
			skuOutput.CpuCoreCount = gt.VcpuCount
		} else {
			skuOutput.Name = sku.Name
			skuOutput.MemorySizeMb = sku.MemorySizeMB
			skuOutput.CpuCoreCount = sku.CpuCoreCount
			skuOutput.InstanceTypeCategory = sku.InstanceTypeCategory
			skuOutput.InstanceTypeFamily = sku.InstanceTypeFamily
		}
		configInfo.Sku = skuOutput
	}

	// disk deal
	disks := make([]computeapis.GuestTemplateDisk, len(input.Disks))
	for i := range input.Disks {
		disks[i] = computeapis.GuestTemplateDisk{
			Backend:  input.Disks[i].Backend,
			DiskType: input.Disks[i].DiskType,
			Index:    input.Disks[i].Index,
			SizeMb:   input.Disks[i].SizeMb,
		}
	}
	configInfo.Disks = disks

	// keypair
	if len(input.KeypairId) > 0 {
		model, err := KeypairManager.FetchByIdOrName(ctx, userCred, input.KeypairId)
		if err == nil {
			keypair := model.(*SKeypair)
			configInfo.Keypair = keypair.GetName()
		}
	}

	// network
	if len(input.Networks) > 0 {
		networkList := make([]computeapis.GuestTemplateNetwork, 0, len(input.Networks))
		networkIdList := make([]string, len(input.Networks))
		for i := range input.Networks {
			networkIdList[i] = input.Networks[i].Network
		}
		networkSet := sets.NewString(networkIdList...)

		wireQuery := WireManager.Query("id", "vpc_id").SubQuery()
		vpcQuery := VpcManager.Query("id", "name").SubQuery()
		q := NetworkManager.Query("id", "name", "wire_id", "guest_ip_start", "guest_ip_end", "vlan_id")
		if len(networkIdList) == 1 {
			q = q.Equals("id", networkIdList[0])
		} else {
			q = q.In("id", networkIdList)
		}
		q = q.LeftJoin(wireQuery, sqlchemy.Equals(q.Field("wire_id"), wireQuery.Field("id")))
		q = q.LeftJoin(vpcQuery, sqlchemy.Equals(wireQuery.Field("vpc_id"), vpcQuery.Field("id")))
		q = q.AppendField(vpcQuery.Field("id", "vpc_id"), vpcQuery.Field("name", "vpc_name"))
		q.All(&networkList)

		for _, p := range networkList {
			if networkSet.Has(p.ID) {
				networkSet.Delete(p.ID)
			}
		}

		// some specified network
		for _, id := range networkSet.UnsortedList() {
			networkList = append(networkList, computeapis.GuestTemplateNetwork{ID: id})
		}

		configInfo.Nets = networkList
	}

	if len(input.Secgroups) > 0 {
		q := SecurityGroupManager.Query("id", "name").In("id", input.Secgroups)
		rows, err := q.Rows()
		if err != nil {
			return out, errors.Wrap(err, "SQuery.Rows")
		}
		names := make([]string, 0, len(input.Secgroups))
		for rows.Next() {
			var id, name string
			rows.Scan(&id, &name)
			names = append(names, name)
		}
		rows.Close()
		out.Secgroups = names
	}

	// isolatedDevices
	if input.IsolatedDevices != nil && len(input.IsolatedDevices) != 0 {
		configInfo.IsolatedDeviceConfig = make([]computeapis.IsolatedDeviceConfig, len(input.IsolatedDevices))
		for i := range configInfo.IsolatedDeviceConfig {
			configInfo.IsolatedDeviceConfig[i] = *input.IsolatedDevices[i]
		}
	}

	// fill image info
	switch gt.ImageType {
	case IMAGE_TYPE_NORMAL:
		image, err := CachedimageManager.getImageInfo(ctx, userCred, gt.ImageId, false)
		if err == nil {
			configInfo.Image = image.Name
		} else {
			configInfo.Image = gt.ImageId
		}
	case IMAGE_TYPE_GUEST:
		s := auth.GetSession(ctx, userCred, options.Options.Region)
		ret, err := image.GuestImages.Get(s, gt.ImageId, jsonutils.JSONNull)
		if err != nil || !ret.Contains("id") {
			configInfo.Image = gt.ImageId
		} else {
			name, _ := ret.GetString("id")
			configInfo.Image = name
		}
	default:
		// no arrivals
	}

	// reset_password
	if input.ResetPassword == nil {
		configInfo.ResetPassword = true
	} else {
		configInfo.ResetPassword = *input.ResetPassword
	}

	out.ConfigInfo = configInfo
	return out, nil
}

func (gt *SGuestTemplate) PerformPublic(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data apis.PerformPublicProjectInput,
) (jsonutils.JSONObject, error) {

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

	targetScopeStr := data.Scope
	targetScope := rbacscope.String2ScopeDefault(targetScopeStr, rbacscope.ScopeSystem)

	// check if secgroup is public
	if len(input.SecgroupId) > 0 {
		model, err := SecurityGroupManager.FetchByIdOrName(ctx, userCred, input.SecgroupId)
		if err != nil {
			return nil, httperrors.NewResourceNotFoundError("there is no such secgroup %s descripted by guest template",
				input.SecgroupId)
		}
		secgroup := model.(*SSecurityGroup)
		sgScope := rbacscope.String2Scope(secgroup.PublicScope)
		if !secgroup.IsPublic || !sgScope.HigherEqual(targetScope) {
			return nil, gt.genForbiddenError("security group", input.SecgroupId, string(targetScope))
		}
	}

	// check if networks is public
	if len(input.Networks) > 0 {
		for i := range input.Networks {
			str := input.Networks[i].Network
			model, err := NetworkManager.FetchByIdOrName(ctx, userCred, str)
			if err != nil {
				return nil, httperrors.NewResourceNotFoundError(
					"there is no such secgroup %s descripted by guest template", str)
			}
			network := model.(*SNetwork)
			netScope := rbacscope.String2Scope(network.PublicScope)
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
		s := auth.GetSession(ctx, userCred, options.Options.Region)
		ret, err := image.GuestImages.Get(s, gt.ImageId, jsonutils.JSONNull)
		if err != nil {
			return nil, errors.Wrapf(err, "fail to fetch guest image %s descripted by guest template", gt.ImageId)
		}
		isPublic = jsonutils.QueryBoolean(ret, "is_public", false)
		publicScope, _ = ret.GetString("public_scope")
	default:
		//no arrivals
	}
	igScope := rbacscope.String2Scope(publicScope)
	if !isPublic || !igScope.HigherEqual(targetScope) {
		return nil, gt.genForbiddenError("image", "", string(targetScope))
	}

	return gt.SSharableVirtualResourceBase.PerformPublic(ctx, userCred, query, data)
}

func (gt *SGuestTemplate) genForbiddenError(resourceName, resourceStr, scope string) error {
	var (
		msgFmt  string
		msgArgs []interface{}
	)
	if resourceStr == "" {
		msgFmt = "the %s in guest template is not a public resource"
		msgArgs = []interface{}{resourceName}
	} else {
		msgFmt = "the %s %q in guest template is not a public resource"
		msgArgs = []interface{}{resourceName, resourceStr}
	}
	if scope != "" {
		msgFmt += " in %s scope"
		msgArgs = append(msgArgs, scope)
	}
	return httperrors.NewForbiddenError(msgFmt, msgArgs...)
}

func (gt *SGuestTemplate) ValidateDeleteCondition(ctx context.Context, info jsonutils.JSONObject) error {
	// check service catelog
	q := ServiceCatalogManager.Query("name").Equals("guest_template_id", gt.Id)
	names := make([]struct{ Name string }, 0, 1)
	err := q.All(&names)
	if err != nil {
		return errors.Wrap(err, "SQuery.All")
	}
	if len(names) > 0 {
		return httperrors.NewForbiddenError("guest template %s used by service catalog %s", gt.Id, names[0].Name)
	}
	// check scaling group
	q = ScalingGroupManager.Query("name").Equals("guest_template_id", gt.Id)
	names = make([]struct{ Name string }, 0, 1)
	err = q.All(&names)
	if err != nil {
		return errors.Wrap(err, "SQuery.All")
	}
	if len(names) > 0 {
		return httperrors.NewForbiddenError("guest template %s used by scalig group %s", gt.Id, names[0].Name)
	}
	return nil
}

// 主机模板列表
func (manager *SGuestTemplateManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input computeapis.GuestTemplateListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SSharableVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, input.SharableVirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SSharableVirtualResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SCloudregionResourceBaseManager.ListItemFilter(ctx, q, userCred, input.RegionalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.ListItemFilter")
	}
	if len(input.VpcId) > 0 {
		q, err = manager.SVpcResourceBaseManager.ListItemFilter(ctx, q, userCred, input.VpcFilterListInput)
		if err != nil {
			return nil, errors.Wrap(err, "SVpcResourceBaseManager.ListItemFilter")
		}
	}
	if len(input.CloudEnv) > 0 {
		cloudregions := CloudregionManager.Query().SubQuery()
		q = q.Join(cloudregions, sqlchemy.Equals(q.Field("cloudregion_id"), cloudregions.Field("id")))
		switch input.CloudEnv {
		case api.CLOUD_ENV_PUBLIC_CLOUD:
			q = q.Filter(sqlchemy.In(cloudregions.Field("provider"), CloudproviderManager.GetPublicProviderProvidersQuery()))
		case api.CLOUD_ENV_PRIVATE_CLOUD:
			q = q.Filter(sqlchemy.In(cloudregions.Field("provider"), CloudproviderManager.GetPrivateProviderProvidersQuery()))
		case api.CLOUD_ENV_ON_PREMISE:
			q = q.Filter(sqlchemy.Equals(cloudregions.Field("provider"), api.CLOUD_PROVIDER_ONECLOUD))
		case api.CLOUD_ENV_PRIVATE_ON_PREMISE:
			q = q.Filter(sqlchemy.OR(
				sqlchemy.Equals(cloudregions.Field("provider"), api.CLOUD_PROVIDER_ONECLOUD),
				sqlchemy.In(cloudregions.Field("provider"), CloudproviderManager.GetPrivateProviderProvidersQuery()),
			))
		}
	}
	if len(input.BillingType) > 0 {
		q = q.Equals("billing_type", input.BillingType)
	}
	if len(input.Brand) > 0 {
		q = q.Equals("hypervisor", Brand2Hypervisor(input.Brand))
	}
	return q, nil
}

func (manager *SGuestTemplateManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input computeapis.GuestTemplateListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SSharableVirtualResourceBaseManager.OrderByExtraFields(ctx, q, userCred, input.SharableVirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SSharableVirtualResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SCloudregionResourceBaseManager.OrderByExtraFields(ctx, q, userCred, input.RegionalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SVpcResourceBaseManager.OrderByExtraFields(ctx, q, userCred, input.VpcFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVpcResourceBaseManager.OrderByExtraFields")
	}
	return q, nil
}

func (manager *SGuestTemplateManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SSharableVirtualResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = manager.SCloudregionResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = manager.SVpcResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

type SGuestTemplateValidate struct {
	Hypervisor    string
	CloudregionId string
	VpcId         string
	NetworkIds    []string
}

func (gt *SGuestTemplate) Validate(ctx context.Context, userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider, stv SGuestTemplateValidate) (bool, string) {
	if stv.Hypervisor != "" && gt.Hypervisor != stv.Hypervisor {
		return false, fmt.Sprintf("GuestTemplate has mismatched hypervisor, need %s but %s", stv.Hypervisor, gt.Hypervisor)
	}
	if stv.CloudregionId != "" && gt.CloudregionId != stv.CloudregionId {
		return false, fmt.Sprintf("GuestTemplate has mismatched cloudregion, need %s but %s", stv.CloudregionId, gt.CloudregionId)
	}
	if stv.VpcId != "" && gt.VpcId != "" && stv.VpcId != gt.VpcId {
		return false, fmt.Sprintf("GuestTemplate has mismatched vpc, need %s bu %s", stv.VpcId, gt.VpcId)
	}

	// check networks
	input, err := GuestTemplateManager.validateContent(ctx, userCred, ownerId, jsonutils.NewDict(), gt.Content.(*jsonutils.JSONDict))
	if err != nil {
		return false, err.Error()
	}
	if len(input.Networks) != 0 && len(input.Networks[0].Network) != 0 {
		for i := range input.Networks {
			if !utils.IsInStringArray(input.Networks[i].Network, stv.NetworkIds) {
				return false, fmt.Sprintf("GuestTemplate's network '%s' not in networks '%s'", input.Networks[i].Network, stv.NetworkIds)
			}
		}
	}
	return true, ""
}

func (manager *SGuestTemplateManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SSharableVirtualResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SSharableVirtualResourceBaseManager.ListItemExportKeys")
	}
	if keys.ContainsAny(manager.SCloudregionResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SCloudregionResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.ListItemExportKeys")
		}
	}
	if keys.ContainsAny(manager.SVpcResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SVpcResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SVpcResourceBaseManager.ListItemExportKeys")
		}
	}

	return q, nil
}

func (g *SGuest) PerformSaveTemplate(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, input computeapis.GuestSaveToTemplateInput) (jsonutils.JSONObject, error) {
	g.SetStatus(ctx, userCred, computeapis.VM_TEMPLATE_SAVING, "save to template")

	if len(input.Name) == 0 && len(input.GenerateName) == 0 {
		input.GenerateName = fmt.Sprintf("%s-template", g.Name)
	}
	data := jsonutils.Marshal(input).(*jsonutils.JSONDict)
	if task, err := taskman.TaskManager.NewTask(ctx, "GuestSaveTemplateTask", g, userCred, data, "", "", nil); err != nil {
		return nil, errors.Wrap(err, "Unbale to init 'GuestSaveTemplateTask'")
	} else {
		task.ScheduleRun(nil)
	}
	return nil, nil
}

func (gt *SGuestTemplate) PerformInspect(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return nil, gt.inspect(ctx, userCred)
}

func (gt *SGuestTemplate) inspect(ctx context.Context, userCred mcclient.TokenCredential) error {
	_, err := GuestTemplateManager.validateContent(ctx, userCred, gt.GetOwnerId(), jsonutils.NewDict(), gt.Content.(*jsonutils.JSONDict))
	if err == nil {
		gt.updateCheckTime()
		gt.SetStatus(ctx, userCred, computeapis.GT_READY, "inspect successfully")
		logclient.AddSimpleActionLog(gt, logclient.ACT_HEALTH_CHECK, "", userCred, true)
		return nil
	}
	// invalid
	gt.updateCheckTime()
	reason := fmt.Sprintf("During the inspection, the guest template is not available: %s", err.Error())
	gt.SetStatus(ctx, userCred, computeapis.GT_INVALID, reason)
	logclient.AddSimpleActionLog(gt, logclient.ACT_HEALTH_CHECK, reason, userCred, false)
	return nil
}

func (gm *SGuestTemplateManager) InspectAllTemplate(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	lastCheckTime := time.Now().Add(time.Duration(-options.Options.GuestTemplateCheckInterval) * time.Hour)
	q := gm.Query()
	q = q.Filter(sqlchemy.OR(sqlchemy.IsNull(q.Field("last_check_time")), sqlchemy.LE(q.Field("last_check_time"),
		lastCheckTime)))
	gts := make([]SGuestTemplate, 0, 10)
	err := db.FetchModelObjects(gm, q, &gts)
	if err != nil {
		log.Errorf("Unable to fetch all guest templates that need to check: %s", err.Error())
		return
	}
	for i := range gts {
		gts[i].inspect(ctx, userCred)
	}
}
