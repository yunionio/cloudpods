package models

import (
	"context"
	"database/sql"
	"strings"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log" //wjf
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

var (
	VpcPeerInwardManager *SVpcPeerInwardManager
)

func init() {
	//SVirtualResourceBaseManager NewVirtualResourceBaseManager
	VpcPeerInwardManager = &SVpcPeerInwardManager{
		SEnabledStatusInfrasResourceBaseManager: db.NewEnabledStatusInfrasResourceBaseManager(
			&SVpcPeerInward{},
			"vpc_peer_inward_tbl",
			"vpc_peer_inward",
			"vpc_peer_inwards",
		),
	}
	VpcPeerInwardManager.SetVirtualObject(VpcPeerInwardManager)
}

type SVpcPeerInwardManager struct {
	db.SEnabledStatusInfrasResourceBaseManager
	db.SExternalizedResourceBaseManager
	SVpcResourceBaseManager
}

type SVpcPeerInward struct {
	db.SEnabledStatusInfrasResourceBase
	db.SExternalizedResourceBase

	SVpcResourceBase

	PendingDeleted bool `nullable:"false" default:"false" index:"true" get:"user" list:"user" json:"pending_deleted"`

	VpcPeerId string `width:"128" charset:"ascii"  list:"user" json:"vpc_peer_id"`

	VpcLocalId string `width:"128" charset:"ascii"  list:"user" json:"vpc_local_id"`

	PeerVpcNetwork string `charset:"ascii" nullable:"true"  list:"user" json:"peer_vpc_network"`

	LocalVpcNetwork string `charset:"ascii" nullable:"true"  list:"user" json:"local_vpc_network"`
}

func (manager *SVpcPeerInwardManager) GetContextManagers() [][]db.IModelManager {
	return [][]db.IModelManager{
		{VpcManager},
	}
}

func (manager *SVpcPeerInwardManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.VpcPeeringConnectionInwardsListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SEnabledStatusInfrasResourceBaseManager.ListItemFilter(ctx, q, userCred, query.EnabledStatusInfrasResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledStatusInfrasResourceBaseManager.ListItemFilter")
	}

	q, err = manager.SExternalizedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ExternalizedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SExternalizedResourceBaseManager.ListItemFilter")
	}

	q, err = manager.SVpcResourceBaseManager.ListItemFilter(ctx, q, userCred, query.VpcFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVpcResourceBaseManager.ListItemFilter")
	}

	if len(query.VpcPeerId) > 0 {
		q = q.In("vpc_peer_id", query.VpcPeerId)
	}

	if len(query.VpcLocalId) > 0 {
		q = q.In("vpc_local_id", query.VpcLocalId)
	}

	if len(query.PeerVpcNetwork) > 0 {
		q = q.In("peer_vpc_network", query.PeerVpcNetwork)
	}

	if len(query.LocalVpcNetwork) > 0 {
		q = q.In("local_vpc_network", query.LocalVpcNetwork)
	}

	return q, nil
}

func (manager *SVpcPeerInwardManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input api.VpcPeeringConnectionInwardsCreateInput,
) (api.VpcPeeringConnectionInwardsCreateInput, error) {
	var err error
	input.EnabledStatusInfrasResourceBaseCreateInput, err = manager.SEnabledStatusInfrasResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.EnabledStatusInfrasResourceBaseCreateInput)
	if err != nil {
		return input, err
	}
	if len(input.VpcId) == 0 {
		return input, httperrors.NewMissingParameterError("vpc_id")
	}

	// get vpc ,peerVpc
	_vpc, err := VpcManager.FetchByIdOrName(userCred, input.VpcId)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return input, httperrors.NewResourceNotFoundError2("vpc", input.VpcId)
		}
		return input, httperrors.NewGeneralError(err)
	}
	vpc := _vpc.(*SVpc)

	_peerVpc, err := VpcManager.FetchByIdOrName(userCred, input.VpcPeerId)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return input, httperrors.NewResourceNotFoundError2("Peervpc", input.VpcPeerId)
		}
		return input, httperrors.NewGeneralError(err)
	}
	peerVpc := _peerVpc.(*SVpc)

	if len(vpc.ManagerId) == 0 || len(peerVpc.ManagerId) == 0 {
		return input, httperrors.NewInputParameterError("Only public cloud support vpcpeering")
	}

	// get account,providerFactory
	account := vpc.GetCloudaccount()
	peerAccount := peerVpc.GetCloudaccount()
	if account.Provider != peerAccount.Provider {
		return input, httperrors.NewNotSupportedError("vpc on different cloudprovider peering is not supported")
	}

	factory, err := cloudprovider.GetProviderFactory(account.Provider)
	if err != nil {
		return input, httperrors.NewGeneralError(errors.Wrapf(err, "cloudprovider.GetProviderFactory(%s)", account.Provider))
	}

	// check vpc ip range overlap
	if !factory.IsSupportVpcPeeringVpcCidrOverlap() {
		vpcIpv4Ranges := []netutils.IPV4AddrRange{}
		peervpcIpv4Ranges := []netutils.IPV4AddrRange{}
		vpcCidrBlocks := strings.Split(vpc.CidrBlock, ",")
		peervpcCidrBlocks := strings.Split(peerVpc.CidrBlock, ",")
		for i := range vpcCidrBlocks {
			vpcIpv4Range, err := netutils.NewIPV4Prefix(vpcCidrBlocks[i])
			if err != nil {
				return input, httperrors.NewGeneralError(errors.Wrapf(err, "convert vpc cidr %s to ipv4range error", vpcCidrBlocks[i]))
			}
			vpcIpv4Ranges = append(vpcIpv4Ranges, vpcIpv4Range.ToIPRange())
		}

		for i := range peervpcCidrBlocks {
			peervpcIpv4Range, err := netutils.NewIPV4Prefix(peervpcCidrBlocks[i])
			if err != nil {
				return input, httperrors.NewGeneralError(errors.Wrapf(err, "convert vpc cidr %s to ipv4range error", peervpcCidrBlocks[i]))
			}
			peervpcIpv4Ranges = append(peervpcIpv4Ranges, peervpcIpv4Range.ToIPRange())
		}
		for i := range vpcIpv4Ranges {
			for j := range peervpcIpv4Ranges {
				if vpcIpv4Ranges[i].IsOverlap(peervpcIpv4Ranges[j]) {
					return input, httperrors.NewNotSupportedError("ipv4 range overlap")
				}
			}
		}
	}
	/*
		CrossCloudEnv := account.AccessUrl != peerAccount.AccessUrl
		CrossRegion := vpc.CloudregionId != peerVpc.CloudregionId
		if CrossCloudEnv && !factory.IsSupportCrossCloudEnvVpcPeering() {
			return input, httperrors.NewNotSupportedError("cloudprovider %s not supported CrossCloud vpcpeering", account.Provider)
		}
		if CrossRegion && !factory.IsSupportCrossRegionVpcPeering() {
			return input, httperrors.NewNotSupportedError("cloudprovider %s not supported CrossRegion vpcpeering", account.Provider)
		}
		if CrossRegion {
			err := factory.ValidateCrossRegionVpcPeeringBandWidth(input.Bandwidth)
			if err != nil {
				return input, err
			}
		}
	*/
	// existed peer check
	vpcPC := SVpcPeerInward{}
	err = manager.Query().Equals("vpc_id", vpc.Id).Equals("peer_vpc_id", peerVpc.Id).First(&vpcPC)
	if err == nil {
		return input, httperrors.NewNotSupportedError("vpc %s and vpc %s have already connected", input.VpcId, input.VpcPeerId)
	} else {
		if errors.Cause(err) != sql.ErrNoRows {
			return input, httperrors.NewGeneralError(err)
		}
	}

	return input, nil
}

func (self *SVpcPeerInward) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {

	input := api.VpcPeeringConnectionInwardsCreateInput{}
	data.Unmarshal(&input)

	defer func() {
		self.SEnabledStatusInfrasResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	}()

	task, err := taskman.TaskManager.NewTask(ctx, "VpcPeeringConnectionInwardsCreateTask", self, userCred, nil, "", "", nil)
	if err != nil {
		self.SetStatus(userCred, api.VPC_PEERING_INWARDS_STATUS_CREATE_FAILED, errors.Wrapf(err, "NewTask").Error())
		return
	}
	task.ScheduleRun(nil)
	//params := jsonutils.NewDict()
	//task, err := taskman.TaskManager.NewTask(ctx, "VpcPeeringConnectionInwardsCreateTask", self, userCred, params, "", "", nil)
	//if err != nil {
	//	return
	//}
	//self.SetStatus(userCred, api.VPC_PEERING_INWARDS_STATUS_CREATING, "")
	//task.ScheduleRun(nil)
}

func (manager *SVpcPeerInwardManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SInfrasResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = manager.SVpcResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

func (manager *SVpcPeerInwardManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.VpcPeeringConnectionInwardsDetails {
	rows := make([]api.VpcPeeringConnectionInwardsDetails, len(objs))
	stdRows := manager.SEnabledStatusInfrasResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	vpcObjs := make([]interface{}, len(objs))

	for i := range rows {
		rows[i] = api.VpcPeeringConnectionInwardsDetails{
			EnabledStatusInfrasResourceBaseDetails: stdRows[i],
		}
		vpcPC := objs[i].(*SVpcPeerInward)
		vpcObj := &SVpcResourceBase{VpcId: vpcPC.VpcLocalId}
		vpcObjs[i] = vpcObj

		log.Infof("wjf test SVpcPeerInwardManager FetchCustomizeColumns  break0 echo vpc_local_id:%s,\nvpc_peer_id:%s,\nPeerVpcNetwork:%s,\nLocalVpcNetwork:%s!!!!!!\n", vpcPC.VpcLocalId, vpcPC.VpcPeerId, vpcPC.PeerVpcNetwork, vpcPC.LocalVpcNetwork)
		rows[i].VpcLocalId = vpcPC.VpcLocalId
		rows[i].VpcPeerId = vpcPC.VpcPeerId
		rows[i].PeerVpcNetwork = vpcPC.PeerVpcNetwork
		rows[i].LocalVpcNetwork = vpcPC.LocalVpcNetwork
	}

	vpcRows := manager.SVpcResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, vpcObjs, fields, isList)
	for i := range rows {
		rows[i].VpcResourceInfo = vpcRows[i]
	}

	return rows
}

func (self *SVpcPeerInward) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return self.StartDeleteVpcPeeringConnectionTask(ctx, userCred)
}

func (self *SVpcPeerInward) StartDeleteVpcPeeringConnectionTask(ctx context.Context, userCred mcclient.TokenCredential) error {
	self.SetStatus(userCred, api.VPC_PEERING_INWARDS_STATUS_DELETING, "")
	task, err := taskman.TaskManager.NewTask(ctx, "VpcPeeringConnectionInwardsDeleteTask", self, userCred, nil, "", "", nil)
	if err != nil {
		return errors.Wrap(err, "Start VpcPeeringConnectionInwardsDeleteTask fail")
	}
	task.ScheduleRun(nil)
	return nil
}

func (self *SVpcPeerInward) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (self *SVpcPeerInward) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return self.SEnabledStatusInfrasResourceBase.Delete(ctx, userCred)
}

func (self *SVpcPeerInward) PerformSyncstatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.VpcSyncstatusInput) (jsonutils.JSONObject, error) {
	return nil, StartResourceSyncStatusTask(ctx, userCred, self, "VpcPeeringConnectionInwardsSyncstatusTask", "")
}

func (manager *SVpcPeerInwardManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.VpcPeeringConnectionInwardsListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SStatusInfrasResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.StatusInfrasResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusInfrasResourceBaseManager.OrderByExtraFields")
	}

	q, err = manager.SVpcResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.VpcFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVpcResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (manager *SVpcPeerInwardManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SEnabledStatusInfrasResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledStatusInfrasResourceBaseManager.ListItemExportKeys")
	}

	return q, nil
}

func (self *SVpcPeerInward) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.VpcPeeringConnectionInwardsUpdateInput) (api.VpcPeeringConnectionInwardsUpdateInput, error) {
	var err error
	input.EnabledStatusInfrasResourceBaseUpdateInput, err = self.SEnabledStatusInfrasResourceBase.ValidateUpdateData(ctx, userCred, query, input.EnabledStatusInfrasResourceBaseUpdateInput)
	if err != nil {
		return input, err
	}
	return input, nil
}
