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

package guestdrivers

import (
	"time"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/utils"

	billing_api "yunion.io/x/onecloud/pkg/apis/billing"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/util/seclib2"
)

type SDiskInfo struct {
	DiskType          string
	Size              int
	Uuid              string
	BillingType       string
	FsFromat          string
	AutoDelete        bool
	TemplateId        string
	DiskFormat        string
	Path              string
	Driver            string
	CacheMode         string
	ExpiredAt         time.Time
	StorageExternalId string

	Metadata map[string]string
}

func fetchIVMinfo(desc cloudprovider.SManagedVMCreateConfig, iVM cloudprovider.ICloudVM, guestId string, account, passwd string, publicKey string, action string) *jsonutils.JSONDict {
	data := jsonutils.NewDict()

	data.Add(jsonutils.NewString(string(iVM.GetOsType())), "os")

	//避免在rebuild_root时绑定秘钥,没有account信息
	data.Add(jsonutils.NewString(account), "account")
	if len(passwd) > 0 || len(publicKey) > 0 {
		var encpasswd string
		var err error
		if len(publicKey) > 0 {
			encpasswd, err = seclib2.EncryptBase64(publicKey, passwd)
		} else {
			encpasswd, err = utils.EncryptAESBase64(guestId, passwd)
		}
		if err != nil {
			log.Errorf("encrypt password failed %s", err)
		} else {
			data.Add(jsonutils.NewString(encpasswd), "key")
		}
	}

	if len(desc.OsDistribution) > 0 {
		data.Add(jsonutils.NewString(desc.OsDistribution), "distro")
	}
	if len(desc.OsVersion) > 0 {
		data.Add(jsonutils.NewString(desc.OsVersion), "version")
	}

	idisks, err := iVM.GetIDisks()

	if err != nil {
		log.Errorf("GetiDisks error %s", err)
	} else {
		diskInfo := make([]SDiskInfo, len(idisks))
		for i := 0; i < len(idisks); i += 1 {
			dinfo := SDiskInfo{}
			dinfo.Uuid = idisks[i].GetGlobalId()
			dinfo.Size = idisks[i].GetDiskSizeMB()
			dinfo.DiskType = idisks[i].GetDiskType()
			dinfo.BillingType = idisks[i].GetBillingType()
			dinfo.DiskFormat = idisks[i].GetDiskFormat()
			dinfo.AutoDelete = idisks[i].GetIsAutoDelete()
			if action == "create" {
				dinfo.AutoDelete = true
			}
			dinfo.Path = idisks[i].GetAccessPath()
			dinfo.Driver = idisks[i].GetDriver()
			dinfo.CacheMode = idisks[i].GetCacheMode()
			dinfo.TemplateId = idisks[i].GetTemplateId()
			dinfo.FsFromat = idisks[i].GetFsFormat()
			dinfo.ExpiredAt = idisks[i].GetExpiredAt()
			dinfo.StorageExternalId = idisks[i].GetIStorageId()
			diskSysTags := idisks[i].GetSysTags()
			diskTags, _ := idisks[i].GetTags()
			if diskSysTags != nil || diskTags != nil {
				dinfo.Metadata = make(map[string]string, 0)
				for k, v := range diskSysTags {
					dinfo.Metadata[db.SYS_CLOUD_TAG_PREFIX+k] = v
				}
				for k, v := range diskTags {
					dinfo.Metadata[db.CLOUD_TAG_PREFIX+k] = v
				}
			}
			diskInfo[i] = dinfo
		}
		data.Add(jsonutils.Marshal(&diskInfo), "disks")
	}

	data.Add(jsonutils.NewString(iVM.GetGlobalId()), "uuid")
	sysTags := iVM.GetSysTags()
	tags, _ := iVM.GetTags()
	metadataDict := jsonutils.NewDict()
	for k, v := range sysTags {
		metadataDict.Add(jsonutils.NewString(v), db.SYS_CLOUD_TAG_PREFIX+k)
	}
	for k, v := range tags {
		metadataDict.Add(jsonutils.NewString(v), db.CLOUD_TAG_PREFIX+k)
	}
	data.Add(metadataDict, "metadata")

	if iVM.GetBillingType() == billing_api.BILLING_TYPE_PREPAID {
		data.Add(jsonutils.NewTimeString(iVM.GetExpiredAt()), "expired_at")
		data.Add(jsonutils.NewBool(iVM.IsAutoRenew()), "auto_renew")
	}

	return data
}
