package models

import (
	"context"
	"database/sql"
	"time"

	"yunion.io/x/jsonutils"
	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/llm/options"
	cloudutil "yunion.io/x/onecloud/pkg/llm/utils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	computeoptions "yunion.io/x/onecloud/pkg/mcclient/options/compute"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"
)

func NewSLLMBaseManager(dt interface{}, tableName string, keyword string, keywordPlural string) SLLMBaseManager {
	return SLLMBaseManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			dt,
			tableName,
			keyword,
			keywordPlural,
		),
	}
}

type SLLMBaseManager struct {
	db.SVirtualResourceBaseManager
	db.SEnabledResourceBaseManager
}

type SLLMBase struct {
	db.SVirtualResourceBase
	db.SEnabledResourceBase

	SvrId string `width:"128" charset:"ascii" nullable:"true" list:"user"`
	LLMIp string `width:"20" charset:"ascii" nullable:"true" list:"user"`
	// Hypervisor     string `width:"128" charset:"ascii" nullable:"true" list:"user"`
	Priority    int `nullable:"false" default:"100" list:"user"`
	BandwidthMb int `nullable:"true" list:"user" create:"admin_optional"`

	LastAppProbe time.Time `nullable:"true" list:"user" create:"admin_optional"`

	// 是否请求同步更新镜像
	SyncImageRequest bool `default:"false" nullable:"false" list:"user" update:"user"`

	VolumeUsedMb int       `nullable:"true" list:"user"`
	VolumeUsedAt time.Time `nullable:"true" list:"user"`

	// 秒装应用配额（可安装的总容量限制）
	// InstantAppQuotaGb int `list:"user" update:"user" create:"optional" default:"0" nullable:"false"`

	DebugMode     bool `default:"false" nullable:"false" list:"user" update:"user"`
	RootfsUnlimit bool `default:"false" nullable:"false" list:"user" update:"user"`
}

func (man *SLLMBaseManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.LLMBaseCreateInput) (api.LLMBaseCreateInput, error) {
	var err error
	input.VirtualResourceCreateInput, err = man.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.VirtualResourceCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "validate VirtualResourceCreateInput")
	}

	if len(input.PreferHost) > 0 {
		s := auth.GetSession(ctx, userCred, "")
		hostJson, err := compute.Hosts.Get(s, input.PreferHost, nil)
		if err != nil {
			return input, errors.Wrap(err, "get host")
		}
		hostDetails := computeapi.HostDetails{}
		if err := hostJson.Unmarshal(&hostDetails); err != nil {
			return input, errors.Wrap(err, "unmarshal hostDetails")
		}
		if hostDetails.Enabled == nil || !*hostDetails.Enabled {
			return input, errors.Wrap(errors.ErrInvalidStatus, "not enabled")
		}
		if hostDetails.HostStatus != computeapi.HOST_ONLINE {
			return input, errors.Wrap(errors.ErrInvalidStatus, "not online")
		}
		if hostDetails.HostType != computeapi.HOST_TYPE_CONTAINER {
			return input, errors.Wrapf(httperrors.ErrNotAcceptable, "host_type %s not supported", hostDetails.HostType)
		}
		input.PreferHost = hostDetails.Id
	}

	return input, nil
}

func GetServerIdsByHost(ctx context.Context, userCred mcclient.TokenCredential, hostId string) ([]string, error) {
	s := auth.GetSession(ctx, userCred, options.Options.Region)
	params := computeoptions.ServerListOptions{}
	params.Scope = "maxallowed"
	params.Host = hostId
	params.Field = []string{"id"}
	limit := 1024
	params.Limit = &limit
	offset := 0
	total := -1
	idList := stringutils2.NewSortedStrings(nil)
	for total < 0 || offset < total {
		params.Offset = &offset
		results, err := compute.Servers.List(s, jsonutils.Marshal(params))
		if err != nil {
			return nil, errors.Wrap(err, "Servers.List")
		}
		total = results.Total
		for i := range results.Data {
			idStr, _ := results.Data[i].GetString("id")
			if len(idStr) > 0 {
				idList = idList.Append(idStr)
			}
		}
		offset += len(results.Data)
	}
	return idList, nil
}

func (man *SLLMBaseManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input api.LLMBaseListInput) (*sqlchemy.SQuery, error) {
	q, err := man.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, input.VirtualResourceListInput)
	if err != nil {
		return q, errors.Wrap(err, "VirtualResourceBaseManager.ListItemFilter")
	}
	q, err = man.SEnabledResourceBaseManager.ListItemFilter(ctx, q, userCred, input.EnabledResourceBaseListInput)
	if err != nil {
		return q, errors.Wrap(err, "SEnabledResourceBaseManager.ListItemFilter")
	}

	if len(input.Host) > 0 {
		serverIds, err := GetServerIdsByHost(ctx, userCred, input.Host)
		if err != nil {
			return nil, errors.Wrap(err, "GetServerIdsByHost")
		}
		q = q.In("svr_id", serverIds)
	}
	if len(input.Status) > 0 {
		s := auth.GetSession(ctx, userCred, options.Options.Region)
		params := computeoptions.ServerListOptions{}
		params.Scope = "maxallowed"
		params.Status = input.Status
		params.Field = []string{"guest_id"}
		limit := 1024
		params.Limit = &limit
		offset := 0
		total := -1
		idList := stringutils2.NewSortedStrings(nil)
		for total < 0 || offset < total {
			params.Offset = &offset
			results, err := compute.Containers.List(s, jsonutils.Marshal(params))
			if err != nil {
				return nil, errors.Wrap(err, "Containers.List")
			}
			total = results.Total
			for i := range results.Data {
				idStr, _ := results.Data[i].GetString("guest_id")
				if len(idStr) > 0 {
					idList = idList.Append(idStr)
				}
			}
			offset += len(results.Data)
		}
		q = q.In("svr_id", idList)
	}

	if input.NoVolume != nil {
		volumeQ := GetVolumeManager().Query("llm_id").SubQuery()
		q = q.LeftJoin(volumeQ, sqlchemy.Equals(q.Field("id"), volumeQ.Field("llm_id")))
		if *input.NoVolume {
			q = q.Filter(sqlchemy.IsNull(volumeQ.Field("llm_id")))
		} else {
			q = q.Filter(sqlchemy.IsNotNull(volumeQ.Field("llm_id")))
		}
	}
	if len(input.VolumeId) > 0 {
		volumeObj, err := GetVolumeManager().FetchByIdOrName(ctx, userCred, input.VolumeId)
		if err != nil {
			return nil, errors.Wrap(err, "VolumeManager.FetchByIdOrName")
		}
		vq := GetVolumeManager().Query().SubQuery()
		q = q.Join(vq, sqlchemy.Equals(q.Field("id"), vq.Field("llm_id")))
		q = q.Filter(sqlchemy.Equals(vq.Field("id"), volumeObj.GetId()))
	}

	accessQ := GetAccessInfoManager().Query().SubQuery()
	if input.ListenPort > 0 {
		q = q.Join(accessQ, sqlchemy.Equals(q.Field("id"), accessQ.Field("llm_id")))
		q = q.Filter(sqlchemy.Equals(accessQ.Field("listen_port"), input.ListenPort))
	}

	if len(input.PublicIp) > 0 {
		s := auth.GetSession(ctx, userCred, "")
		hostInput := computeapi.HostListInput{
			PublicIp: []string{input.PublicIp},
		}
		hostInput.Field = []string{"id"}
		hosts, err := compute.Hosts.List(s, jsonutils.Marshal(hostInput))
		if err != nil {
			return nil, errors.Wrap(err, "Hosts.List")
		}
		if len(hosts.Data) == 0 {
			return nil, httperrors.NewNotFoundError("Not found host by public_ip %s", input.PublicIp)
		}
		hostIds := []string{}
		for i := range hosts.Data {
			idStr, _ := hosts.Data[i].GetString("id")
			if len(idStr) > 0 {
				hostIds = append(hostIds, idStr)
			}
		}
		if len(hostIds) > 0 {
			serverIds, err := GetServerIdsByHost(ctx, userCred, hostIds[0])
			if err != nil {
				return nil, errors.Wrap(err, "GetServerIdsByHost")
			}
			q = q.In("svr_id", serverIds)
		}
	}
	// if input.Unused != nil {
	// 	instanceQ := GetDesktopInstanceManager().Query().SubQuery()
	// 	if *input.Unused {
	// 		q = q.NotEquals("id", instanceQ.Query(instanceQ.Field("desktop_id")).SubQuery())
	// 	} else {
	// 		q = q.Join(instanceQ, sqlchemy.Equals(q.Field("id"), instanceQ.Field("desktop_id")))
	// 	}
	// }

	return q, nil
}

func (llm *SLLMBase) GetServer(ctx context.Context) (*computeapi.ServerDetails, error) {
	return cloudutil.GetServer(ctx, llm.SvrId)
}

func (llm *SLLMBase) GetVolume() (*SVolume, error) {
	volume := &SVolume{}
	err := GetVolumeManager().Query().Equals("llm_id", llm.Id).First(volume)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, errors.Wrap(errors.ErrNotFound, "query volume")
		}
		return nil, errors.Wrap(err, "FetchVolume")
	}
	volume.SetModelManager(GetVolumeManager(), volume)
	return volume, nil
}

// 取消自动删除
func (llm *SLLMBase) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (llm *SLLMBase) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return llm.SVirtualResourceBase.Delete(ctx, userCred)
}

func (llm *SLLMBase) ServerDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	if len(llm.SvrId) == 0 {
		return nil
	}
	s := auth.GetSession(ctx, userCred, "")
	server, err := llm.GetServer(ctx)
	if err != nil {
		if errors.Cause(err) == errors.ErrNotFound {
			return nil
		} else {
			return errors.Wrap(err, "GetServer")
		}
	}
	if server.DisableDelete != nil && *server.DisableDelete {
		// update to allow delete
		_, err = compute.Servers.Update(s, llm.SvrId, jsonutils.Marshal(map[string]interface{}{"disable_delete": false}))
		if err != nil {
			return errors.Wrap(err, "update server to delete")
		}
	}
	_, err = compute.Servers.DeleteWithParam(s, llm.SvrId, jsonutils.Marshal(map[string]interface{}{
		"override_pending_delete": true,
	}), nil)
	if err != nil {
		return errors.Wrap(err, "delete server err:")
	}
	return nil
}

func (llm *SLLMBase) WaitDelete(ctx context.Context, userCred mcclient.TokenCredential, timeoutSecs int) error {
	return cloudutil.WaitDelete[computeapi.ServerDetails](ctx, &compute.Servers, llm.SvrId, timeoutSecs)
}

func (llm *SLLMBase) getImage(imageId string) (*SLLMImage, error) {
	image, err := GetLLMImageManager().FetchById(imageId)
	if err != nil {
		return nil, errors.Wrap(err, "fetch LLMImage")
	}
	return image.(*SLLMImage), nil
}

func (llm *SLLMBase) WaitServerStatus(ctx context.Context, userCred mcclient.TokenCredential, targetStatus []string, timeoutSecs int) (*computeapi.ServerDetails, error) {
	return cloudutil.WaitServerStatus(ctx, llm.SvrId, targetStatus, timeoutSecs)
}
