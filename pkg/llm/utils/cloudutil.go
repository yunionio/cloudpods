package utils

import (
	"context"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"
	"yunion.io/x/pkg/utils"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules/compute"
)

type ResourceInfo struct {
	Id     string
	Name   string
	Status string
}

func NewResourceInfo(id, name, status string) ResourceInfo {
	return ResourceInfo{
		Id:     id,
		Name:   name,
		Status: status,
	}
}

func GetResource[R any](ctx context.Context, man modulebase.Manager, id string) (*R, error) {
	if len(id) == 0 {
		return nil, errors.Wrapf(httperrors.ErrInvalidStatus, "id is empty")
	}
	s := auth.GetAdminSession(ctx, "")
	resp, err := man.GetById(s, id, jsonutils.Marshal(map[string]interface{}{
		"scope": "max",
	}))
	if err != nil {
		if httputils.ErrorCode(err) == 404 {
			return nil, errors.Wrapf(errors.ErrNotFound, "GetById %s", id)
		}
		return nil, errors.Wrapf(err, "Get")
	}
	res := new(R)
	if err := resp.Unmarshal(res); err != nil {
		return nil, errors.Wrap(err, "Unmarshal")
	}
	return res, nil
}

func PerformResourceAction[R any](ctx context.Context, man modulebase.Manager, id string, action string) (*R, error) {
	s := auth.GetAdminSession(ctx, "")
	resp, err := man.PerformAction(s, id, action, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "PerformAction %s %s %s", man.GetKeyword(), id, action)
	}
	res := new(R)
	if err := resp.Unmarshal(res); err != nil {
		return nil, errors.Wrap(err, "Unmarshal")
	}
	return res, nil
}

func StopResource[R any](ctx context.Context, man modulebase.Manager, id string) (*R, error) {
	return PerformResourceAction[R](ctx, man, id, "stop")
}

func StartResource[R any](ctx context.Context, man modulebase.Manager, id string) (*R, error) {
	return PerformResourceAction[R](ctx, man, id, "start")
}

func WaitResourceStatus[R any](
	ctx context.Context,
	man modulebase.Manager,
	id string,
	getResInfo func(*R) ResourceInfo,
	targetStatus []string,
	timeoutSecs int,
	intervalSecs int) (*R, error) {
	expire := time.Now().Add(time.Second * time.Duration(timeoutSecs))
	for time.Now().Before(expire) {
		res, err := GetResource[R](ctx, man, id)
		if err != nil {
			return nil, errors.Wrapf(err, "GetResource")
		}
		resInfo := getResInfo(res)
		log.Debugf("Wait %s status %#v target status %v", man.GetKeyword(), resInfo.Status, targetStatus)
		if utils.IsInStringArray(resInfo.Status, targetStatus) {
			return res, nil
		}
		if strings.Contains(resInfo.Status, "fail") {
			return nil, errors.Wrapf(errors.ErrInvalidStatus, "resource %s status %s", resInfo.Name, resInfo.Status)
		}
		time.Sleep(time.Second * time.Duration(intervalSecs))
	}
	return nil, errors.Wrapf(httperrors.ErrTimeout, "wait %s status %s timeout", man.GetKeyword(), targetStatus)
}

func GetContainer(ctx context.Context, id string) (*computeapi.SContainer, error) {
	return GetResource[computeapi.SContainer](ctx, &compute.Containers, id)
}

func GetServer(ctx context.Context, id string) (*computeapi.ServerDetails, error) {
	return GetResource[computeapi.ServerDetails](ctx, &compute.Servers, id)
}

func WaitContainerStatus(ctx context.Context, id string, targetStatus []string, timeoutSecs int) (*computeapi.SContainer, error) {
	return WaitResourceStatus(ctx, &compute.Containers, id, func(ctr *computeapi.SContainer) ResourceInfo {
		return NewResourceInfo(ctr.Id, ctr.Name, ctr.Status)
	}, targetStatus, timeoutSecs, 1)
}

func WaitServerStatus(ctx context.Context, id string, targetStatus []string, timeoutSecs int) (*computeapi.ServerDetails, error) {
	return WaitResourceStatus(ctx, &compute.Servers, id, func(s *computeapi.ServerDetails) ResourceInfo {
		return NewResourceInfo(s.Id, s.Name, s.Status)
	}, targetStatus, timeoutSecs, 2)
}

func WaitDelete[R any](ctx context.Context, man modulebase.Manager, id string, timeoutSecs int) error {
	expire := time.Now().Add(time.Second * time.Duration(timeoutSecs))
	for time.Now().Before(expire) {
		_, err := GetResource[R](ctx, man, id)
		if err != nil {
			if errors.Cause(err) == errors.ErrNotFound {
				return nil
			}
			return errors.Wrapf(err, "Get %s %s", man.GetKeyword(), id)
		}
		time.Sleep(2 * time.Second)
	}
	return errors.Wrapf(httperrors.ErrTimeout, "wait %s %s deleted timeout", man.GetKeyword(), id)
}

func UpdateContainer(ctx context.Context, id string, getSpec func(*computeapi.SContainer) *computeapi.ContainerSpec) (*computeapi.SContainer, error) {
	s := auth.GetAdminSession(ctx, "")
	ctr, err := GetContainer(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "GetContainer %s", id)
	}
	curSpecStr := jsonutils.Marshal(ctr.Spec).String()
	newSpec := getSpec(ctr)
	newSpecStr := jsonutils.Marshal(newSpec).String()
	if curSpecStr != newSpecStr {
		ctr.Spec = newSpec
		resp, err := compute.Containers.Update(s, id, jsonutils.Marshal(ctr))
		if err != nil {
			return nil, errors.Wrapf(err, "UpdateContainer %s", id)
		}
		respCtr := new(computeapi.SContainer)
		if err := resp.Unmarshal(respCtr); err != nil {
			return nil, errors.Wrapf(err, "Unmarshal")
		}
		return respCtr, nil
	} else {
		log.Debugf("container spec not changed, skip update container %s spec", ctr.Name)
	}
	return ctr, nil
}

func ExecSyncContainer(ctx context.Context, containerId string, input *computeapi.ContainerExecSyncInput) (jsonutils.JSONObject, error) {
	session := auth.GetAdminSession(ctx, "")
	output, err := compute.Containers.PerformAction(session, containerId, "exec-sync", jsonutils.Marshal(input))
	if err != nil {
		return nil, errors.Wrap(err, "ExecSync")
	}

	return output, nil
}
