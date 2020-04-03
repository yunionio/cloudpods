package models

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/cloudcommon/cronman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

const (
	EIPUsed = "EIPUsed"
)

var (
	SuggestSysRuleManager *SSuggestSysRuleManager
)

func init() {
	SuggestSysRuleManager = NewSuggestSysRuleManager(&DSuggestSysRuleConfig{}, "suggestrule", "suggestrules")
}

type SSuggestSysRuleManager struct {
	//TODO
	db.SVirtualResourceBaseManager
	db.SEnabledResourceBaseManager

	//存储初始化的内容，同时起到默认配置的作用。
	drivers map[string]IDriver

	//存储数据库中内容,init
	suggestSysAlartSettings map[string]*SSuggestSysAlartSetting
}

type DSuggestSysRuleConfig struct {
	db.SVirtualResourceBase
	db.SEnabledResourceBase

	Type    string               `width:"256" charset:"ascii" list:"user" update:"user"`
	Period  string               `width:"256" charset:"ascii" list:"user" update:"user"`
	Setting jsonutils.JSONObject ` list:"user" update:"user"`
	//执行时间
	ExecTime time.Time `json:"exec_time"`
}

type SSuggestSysAlartSetting struct {
	ID   string
	Name string

	EIPUnsed *EIPUnused `json:"eip_unsed"`
	//DiskUnsed *DiskUnsed `json:"eip_unsed"`
	//Driver2   interface{}
}

func NewSuggestSysRuleManager(dt interface{}, keyword, keywordPlural string) *SSuggestSysRuleManager {
	man := &SSuggestSysRuleManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			dt,
			"sugrule_tbl",
			keyword,
			keywordPlural,
		),
	}
	man.SetVirtualObject(man)

	man.registerDriver(man.newEIPUsedDriver())
	return man
}

func (man *SSuggestSysRuleManager) fetchSuggestSysAlartSettings(ruleType string) error {
	objs := make([]*DSuggestSysRuleConfig, 0)
	q := man.Query()
	if q == nil {
		fmt.Println(" query is nil")
	}
	if ruleType != "" {
		q.Equals("type", ruleType)
	}
	fmt.Println(">>>>>>>>>>>>>>>>>>>>>")
	err := db.FetchModelObjects(man, q, &objs)
	if err != nil && err != sql.ErrNoRows {
		return errors.Wrap(err, "db.FetchModelObjects")
	}
	for _, config := range objs {
		setting, err := config.unMarshal()
		if err != nil {
			continue
		}
		man.registerSetting(setting)
	}
	return nil
}

func (man *SSuggestSysRuleManager) registerDriver(drv IDriver) {
	if man.drivers == nil {
		man.drivers = make(map[string]IDriver, 0)
	}
	man.drivers[drv.GetType()] = drv
}

func (man *SSuggestSysRuleManager) registerSetting(setting *SSuggestSysAlartSetting) {
	if man.suggestSysAlartSettings == nil {
		man.suggestSysAlartSettings = make(map[string]*SSuggestSysAlartSetting, 0)
	}
	man.suggestSysAlartSettings[setting.Name] = setting
}

func (man *SSuggestSysRuleManager) GetDriver(name string) IDriver {
	return man.drivers[name]
}

func (man *SSuggestSysRuleManager) newEIPUsedDriver() IDriver {
	return NewEIPUsedDriver()
}

//根据数据库中查询得到的信息进行适配转换，同时更新drivers中的内容
func (dConfig *DSuggestSysRuleConfig) unMarshal() (*SSuggestSysAlartSetting, error) {
	setting := new(SSuggestSysAlartSetting)
	setting.ID = dConfig.Id
	setting.Name = dConfig.Name
	switch setting.Name {
	case EIPUsed:
		setting.EIPUnsed = new(EIPUnused)
		err := dConfig.Setting.Unmarshal(setting.EIPUnsed)
		if err != nil {
			log.Errorln(errors.Wrap(err, "DSuggestSysRuleConfig unMarshal error"))
			return nil, errors.Wrap(err, "DSuggestSysRuleConfig unMarshal error")
		}
	}
	return setting, nil
}

type IDriver interface {
	GetType() string
	Run(instance *SSuggestSysAlartSetting)
	ValidateSetting(input monitor.SuggestSysRuleCreateInput) error
	DoSuggestSysRule(ctx context.Context, userCred mcclient.TokenCredential, isStart bool)
}

type DiskUnsed struct {
	Status string
}

//get 筛选规则列表
func (manager *SSuggestSysRuleManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query monitor.SuggestSysRuleListInput) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, query.VirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SEnabledResourceBaseManager.ListItemFilter(ctx, q, userCred, query.EnabledResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledResourceBaseManager.ListItemFilter")
	}
	return q, nil
}

//post 调用方法
func (man *SSuggestSysRuleManager) ValidateCreateData(
	ctx context.Context, userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject,
	data monitor.SuggestSysRuleCreateInput) (*monitor.SuggestSysRuleCreateInput, error) {
	if data.Period == "" {
		// default 30 minutes
		data.Period = "30s"
	}
	if _, err := time.ParseDuration(data.Period); err != nil {
		return nil, httperrors.NewInputParameterError("Invalid period format: %s", data.Period)
	}
	drv := man.GetDriver(data.Name)
	if drv == nil {
		return nil, httperrors.NewInputParameterError("not support name %q", data.Name)
	}
	err := drv.ValidateSetting(data)
	if err != nil {
		return nil, errors.Wrap(err, "validate setting error")
	}
	return &data, nil
}

//get 返回数据时调用方法
func (man *SSuggestSysRuleManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []monitor.SuggestSysRuleDetails {
	rows := make([]monitor.SuggestSysRuleDetails, len(objs))
	virtRows := man.SVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = monitor.SuggestSysRuleDetails{
			VirtualResourceDetails: virtRows[i],
		}
		rows[i] = objs[i].(*DSuggestSysRuleConfig).getMoreDetails(rows[i])
	}
	return rows
}

func (self *DSuggestSysRuleConfig) getMoreDetails(out monitor.SuggestSysRuleDetails) monitor.SuggestSysRuleDetails {
	out.Name = self.Name
	out.Enabled = self.GetEnabled()
	//添加属性
	return out
}

//插入数据后，调用更新manager-map信息
func (self *DSuggestSysRuleConfig) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	self.SVirtualResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	//SuggestSysRuleManager.suggestSysAlartSettings[setting.Name] = setting
	//setting, err := self.unMarshal()
	//if err != nil {
	//	log.Errorln(errors.Wrap(err, "DSuggestSysRuleConfig PostCreate errorf!"))
	//}
	cronman.GetCronJobManager().Remove(self.Name)
	if self.Enabled.Bool() {
		dur, _ := time.ParseDuration(self.Period)
		cronman.GetCronJobManager().AddJobAtIntervalsWithStartRun(self.Name, dur,
			SuggestSysRuleManager.drivers[self.Name].DoSuggestSysRule, true)

	}
}

func (self *DSuggestSysRuleConfig) PostUpdate(
	ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, data jsonutils.JSONObject) {
	//setting, err := self.unMarshal()
	//if err != nil {
	//	log.Errorln(errors.Wrap(err, "DSuggestSysRuleConfig PostCreate errorf!"))
	//}
	//SuggestSysRuleManager.suggestSysAlartSettings[setting.Name] = setting
	cronman.GetCronJobManager().Remove(self.Name)
	if self.Enabled.Bool() {
		dur, _ := time.ParseDuration(self.Period)
		cronman.GetCronJobManager().AddJobAtIntervalsWithStartRun(self.Name, dur,
			SuggestSysRuleManager.drivers[self.Name].DoSuggestSysRule, true)

	}
}
