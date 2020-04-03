package models

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

var (
	SuggestSysAlertManager *SSuggestSysAlertManager
)

const (
	DRIVER_ACTION = "delete"
)

type SSuggestSysAlertManager struct {
	db.SVirtualResourceBaseManager
	db.SEnabledResourceBaseManager
}

func init() {
	SuggestSysAlertManager = NewSuggestSysAlertManager(&DSuggestSysAlert{}, "sugggestalart", "sugggestalarts")
}

type DSuggestSysAlert struct {
	db.SVirtualResourceBase
	db.SEnabledResourceBase

	//监控规则对应的json对象
	MonitorConfig jsonutils.JSONObject `list:"user" create:"required" update:"user"`
	//监控规则type：Rule Type
	Type    string               `width:"256" charset:"ascii" list:"user" update:"user"`
	ResMeta jsonutils.JSONObject `charset:"ascii" list:"user" update:"user"`
	Problem jsonutils.JSONObject `charset:"ascii" list:"user" update:"user"`
	Suggest string               `width:"256" charset:"ascii" list:"user" update:"user"`
	Action  string               `width:"256" charset:"ascii" list:"user" update:"user"`
	//Description string `width:"256" charset:"ascii" list:"user"`
	ResId string `width:"256" charset:"ascii" list:"user" update:"user"`
	//// 根据规则定位时间
	//RuleAt time.Time `nullable:"false" created_at:"true" index:"true" get:"user" list:"user" json:"rule_at"`
}

func NewSuggestSysAlertManager(dt interface{}, keyword, keywordPlural string) *SSuggestSysAlertManager {
	man := &SSuggestSysAlertManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			dt,
			"sugalart_tbl",
			keyword,
			keywordPlural,
		),
	}
	man.SetVirtualObject(man)
	return man
}

//get 筛选规则列表
func (manager *SSuggestSysAlertManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query monitor.SuggestSysAlertListInput) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, query.VirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SEnabledResourceBaseManager.ListItemFilter(ctx, q, userCred, query.EnabledResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledResourceBaseManager.ListItemFilter")
	}
	if len(query.Type) > 0 {
		q = q.Equals("type", query.Type)
	}
	if len(query.ResId) > 0 {
		q = q.Equals("res_id", query.ResId)
	}
	return q, nil
}

func (man *SSuggestSysAlertManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input monitor.SuggestSysAlertListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = man.SVirtualResourceBaseManager.OrderByExtraFields(ctx, q, userCred, input.VirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.OrderByExtraFields")
	}
	return q, nil
}

func (man *SSuggestSysAlertManager) ValidateCreateData(
	ctx context.Context, userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject,
	data monitor.SuggestSysAlertCreateInput) (*monitor.SuggestSysAlertCreateInput, error) {
	//rule 查询到资源信息后没有将资源id，进行转换
	if len(data.ResID) == 0 {
		return nil, httperrors.NewInputParameterError("not found res_id %q", data.ResID)
	}
	name, err := db.GenerateName(man, ownerId, data.Name)
	if err != nil {
		return nil, err
	}
	data.Name = name
	if len(data.Type) == 0 {
		return nil, httperrors.NewInputParameterError("not found type %q", data.Type)
	}
	return &data, nil
}

//get 返回数据时调用方法
func (man *SSuggestSysAlertManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []monitor.SuggestSysAlertDetails {
	rows := make([]monitor.SuggestSysAlertDetails, len(objs))
	virtRows := man.SVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = monitor.SuggestSysAlertDetails{
			VirtualResourceDetails: virtRows[i],
		}
	}
	return rows
}

func (manager *SSuggestSysAlertManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SVirtualResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

//更新数据 validate
func (alert *DSuggestSysAlert) ValidateUpdateData(
	ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data monitor.SuggestSysAlertUpdateInput) (monitor.SuggestSysAlertUpdateInput, error) {
	//rule 查询到资源信息后没有将资源id，进行转换
	if len(data.ResID) == 0 {
		return data, httperrors.NewInputParameterError("not found res_id ")
	}
	if len(data.Type) == 0 {
		return data, httperrors.NewInputParameterError("not found type ")
	}
	var err error
	data.VirtualResourceBaseUpdateInput, err = alert.SVirtualResourceBase.ValidateUpdateData(ctx, userCred, query,
		data.VirtualResourceBaseUpdateInput)
	if err != nil {
		return data, errors.Wrap(err, "SVirtualResourceBase.ValidateUpdateData")
	}
	return data, nil
}

//删除delete时需要调用的方法
func (self *DSuggestSysAlert) GetExtraDetails(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	isList bool,
) (monitor.SuggestSysAlertDetails, error) {
	return monitor.SuggestSysAlertDetails{}, nil
}
