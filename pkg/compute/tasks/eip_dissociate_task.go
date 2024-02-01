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

package tasks

import (
	"context"
	"fmt"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type EipDissociateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(EipDissociateTask{})
}

func (self *EipDissociateTask) TaskFail(ctx context.Context, eip *models.SElasticip, msg jsonutils.JSONObject, model db.IModel) {
	eip.SetStatus(ctx, self.UserCred, api.EIP_STATUS_READY, msg.String())
	self.SetStageFailed(ctx, msg)
	var logOp string
	if model != nil {
		switch srv := model.(type) {
		case *models.SGuest:
			srv.SetStatus(ctx, self.UserCred, api.VM_DISSOCIATE_EIP_FAILED, msg.String())
			logOp = logclient.ACT_VM_DISSOCIATE
		case *models.SNatGateway:
			srv.SetStatus(ctx, self.UserCred, api.VM_DISSOCIATE_EIP_FAILED, msg.String())
			logOp = logclient.ACT_NATGATEWAY_DISSOCIATE
		case *models.SLoadbalancer:
			srv.SetStatus(ctx, self.UserCred, api.VM_DISSOCIATE_EIP_FAILED, msg.String())
			logOp = logclient.ACT_LOADBALANCER_DISSOCIATE
		case *models.SGroup:
			srv.SetStatus(ctx, self.UserCred, api.VM_DISSOCIATE_EIP_FAILED, msg.String())
			logOp = logclient.ACT_INSTANCE_GROUP_DISSOCIATE
		}
		db.OpsLog.LogEvent(model, db.ACT_EIP_DETACH, msg, self.GetUserCred())
		logclient.AddActionLogWithStartable(self, model, logclient.ACT_EIP_DISSOCIATE, msg, self.UserCred, false)
		logclient.AddActionLogWithStartable(self, model, logOp, msg, self.UserCred, false)
	}
}

func (self *EipDissociateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	eip := obj.(*models.SElasticip)

	if eip.IsAssociated() {
		var (
			model db.IModel
			logOp string
		)

		if server := eip.GetAssociateVM(); server != nil {
			if server.Status != api.VM_DISSOCIATE_EIP {
				server.SetStatus(ctx, self.UserCred, api.VM_DISSOCIATE_EIP, "dissociate eip")
			}
			model = server
			logOp = logclient.ACT_VM_DISSOCIATE
		} else if lb := eip.GetAssociateLoadbalancer(); lb != nil {
			model = lb
			logOp = logclient.ACT_LOADBALANCER_DISSOCIATE
		} else if nat := eip.GetAssociateNatGateway(); nat != nil {
			model = nat
			logOp = logclient.ACT_NATGATEWAY_DISSOCIATE
		} else if grp := eip.GetAssociateInstanceGroup(); grp != nil {
			model = grp
			logOp = logclient.ACT_INSTANCE_GROUP_DISSOCIATE
		} else {
			self.TaskFail(ctx, eip, jsonutils.NewString("unsupported associate type"), nil)
			return
		}
		// lockman.LockObject(ctx, model)
		// defer lockman.ReleaseObject(ctx, model)

		if eip.IsManaged() {
			extEip, err := eip.GetIEip(ctx)
			if err != nil && errors.Cause(err) != cloudprovider.ErrNotFound {
				msg := fmt.Sprintf("fail to find iEIP for eip %s", err)
				self.TaskFail(ctx, eip, jsonutils.NewString(msg), model)
				return
			}
			if err == nil && len(extEip.GetAssociationExternalId()) > 0 {
				err = extEip.Dissociate()
				if err != nil {
					msg := fmt.Sprintf("fail to remote dissociate eip %s", err)
					self.TaskFail(ctx, eip, jsonutils.NewString(msg), model)
					return
				}
			}
		} else {
			var errs []error
			switch eip.AssociateType {
			case api.EIP_ASSOCIATE_TYPE_SERVER:
				var guestnics []models.SGuestnetwork
				q := models.GuestnetworkManager.Query().
					Equals("guest_id", model.GetId()).
					Equals("eip_id", eip.Id)
				if err := db.FetchModelObjects(models.GuestnetworkManager, q, &guestnics); err != nil {
					msg := errors.Wrapf(err, "fetch guest nic associated with eip %s(%s)", eip.Name, eip.Id).Error()
					self.TaskFail(ctx, eip, jsonutils.NewString(msg), model)
					return
				}
				for i := range guestnics {
					guestnic := &guestnics[i]
					if _, err := db.Update(guestnic, func() error {
						guestnic.EipId = ""
						return nil
					}); err != nil {
						errs = append(errs, errors.Wrapf(err, "nic %s", guestnic.Ifname))
					}
				}
			case api.EIP_ASSOCIATE_TYPE_INSTANCE_GROUP:
				var groupnics []models.SGroupnetwork
				q := models.GroupnetworkManager.Query().
					Equals("group_id", model.GetId()).
					Equals("eip_id", eip.Id)
				if err := db.FetchModelObjects(models.GroupnetworkManager, q, &groupnics); err != nil {
					msg := errors.Wrapf(err, "fetch group nic associated with eip %s(%s)", eip.Name, eip.Id).Error()
					self.TaskFail(ctx, eip, jsonutils.NewString(msg), model)
					return
				}
				for i := range groupnics {
					groupnic := &groupnics[i]
					if _, err := db.Update(groupnic, func() error {
						groupnic.EipId = ""
						return nil
					}); err != nil {
						errs = append(errs, errors.Wrapf(err, "nic %s/%s", groupnic.IpAddr, groupnic.Ip6Addr))
					}
				}
			case api.EIP_ASSOCIATE_TYPE_LOADBALANCER:
				lb := model.(*models.SLoadbalancer)
				lbnet, err := models.LoadbalancernetworkManager.FetchFirstByLbId(ctx, lb.Id)
				if err != nil {
					self.TaskFail(ctx, eip, jsonutils.NewString(err.Error()), model)
					return
				}
				if _, err := db.Update(lb, func() error {
					lb.Address = lbnet.IpAddr
					lb.AddressType = api.LB_ADDR_TYPE_INTRANET
					return nil
				}); err != nil {
					msg := errors.Wrap(err, "set loadbalancer address").Error()
					self.TaskFail(ctx, eip, jsonutils.NewString(msg), model)
					return
				}
			default:
				errs = append(errs, errors.Wrapf(httperrors.ErrNotSupported, "not supported type %s", eip.AssociateType))
			}
			if len(errs) > 0 {
				err := errors.NewAggregate(errs)
				msg := errors.Wrapf(err, "disassociate eip %s(%s)", eip.Name, eip.Id).Error()
				self.TaskFail(ctx, eip, jsonutils.NewString(msg), model)
				return
			}
		}

		if err := eip.Dissociate(ctx, self.UserCred); err != nil {
			msg := fmt.Sprintf("fail to local dissociate eip %s", err)
			self.TaskFail(ctx, eip, jsonutils.NewString(msg), model)
			return
		}

		eip.SetStatus(ctx, self.UserCred, api.EIP_STATUS_READY, "dissociate")

		logclient.AddActionLogWithStartable(self, model, logclient.ACT_EIP_DISSOCIATE, nil, self.UserCred, true)
		logclient.AddActionLogWithStartable(self, eip, logOp, nil, self.UserCred, true)

		if !self.IsSubtask() {
			switch srv := model.(type) {
			case *models.SGuest:
				srv.StartSyncstatus(ctx, self.UserCred, "")
			case *models.SGroup:
				srv.SetStatus(ctx, self.UserCred, "init", "success")
			}
		}
	}

	self.SetStageComplete(ctx, nil)

	autoDelete := jsonutils.QueryBoolean(self.GetParams(), "auto_delete", false)

	if eip.AutoDellocate.IsTrue() || autoDelete {
		eip.StartEipDeallocateTask(ctx, self.UserCred, "")
	}
}
