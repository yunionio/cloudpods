package guestdrivers

import (
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type SDiskInfo struct {
	DiskType    string
	Size        int
	Uuid        string
	BillingType string
	FsFromat    string
	AutoDelete  bool
	TemplateId  string
	DiskFormat  string
	Path        string
	Driver      string
	CacheMode   string
	ExpiredAt   time.Time

	Metadata map[string]string
}

func fetchIVMinfo(desc cloudprovider.SManagedVMCreateConfig, iVM cloudprovider.ICloudVM, guestId string, account, passwd string, action string) *jsonutils.JSONDict {
	data := jsonutils.NewDict()

	data.Add(jsonutils.NewString(iVM.GetOSType()), "os")

	if len(passwd) > 0 {
		encpasswd, err := utils.EncryptAESBase64(guestId, passwd)
		if err != nil {
			log.Errorf("encrypt password failed %s", err)
		}
		data.Add(jsonutils.NewString(account), "account")
		data.Add(jsonutils.NewString(encpasswd), "key")
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
			if metaData := idisks[i].GetMetadata(); metaData != nil {
				dinfo.Metadata = make(map[string]string, 0)
				metadata := map[string]string{}
				if err := metaData.Unmarshal(&metadata); err != nil {
					log.Errorf("Get disk %s metadata info error: %v", idisks[i].GetName(), err)
				} else {
					for k, v := range metadata {
						dinfo.Metadata["ext:"+k] = v
					}
				}
			}
			diskInfo[i] = dinfo
		}
		data.Add(jsonutils.Marshal(&diskInfo), "disks")
	}

	data.Add(jsonutils.NewString(iVM.GetGlobalId()), "uuid")
	metadata := map[string]string{}
	if _metadata := iVM.GetMetadata(); _metadata != nil {
		_metadata.Unmarshal(&metadata)
	}
	metadataDict := jsonutils.NewDict()
	for k, v := range metadata {
		metadataDict.Add(jsonutils.NewString(v), "ext:"+k)
	}
	data.Add(metadataDict, "metadata")

	if iVM.GetBillingType() == models.BILLING_TYPE_PREPAID {
		data.Add(jsonutils.NewTimeString(iVM.GetExpiredAt()), "expired_at")
	}

	return data
}
