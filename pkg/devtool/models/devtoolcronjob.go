package models

import (
	"context"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/cronman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

type SVSCronjob struct {
	Day      int   `json:"day" nullable:"true" create:"optional" list:"user" update:"user" default:"0"`
	Hour     int   `nullable:"true" create:"optional" list:"user" update:"user" default:"0"`
	Min      int   `nullable:"true" create:"optional" list:"user" update:"user" default:"0"`
	Sec      int   `nullable:"true" create:"optional" list:"user" update:"user" default:"0"`
	Interval int64 `nullable:"true" create:"optional" list:"user" update:"user" default:"0"`
	Start    bool  `nullable:"true" create:"optional" list:"user" update:"user" default:"false"`
	Enabled  bool  `nullable:"true" create:"optional" list:"user" update:"user" default:"false"`
}

// TODO: 为 AnsiblePlaybookID, TemplateID, ServerID 加索引
type SCronjob struct {
	SVSCronjob
	AnsiblePlaybookID string `width:"36" nullable:"false" create:"required" index:"true" list:"user" update:"user"`
	TemplateID        string `width:"36" nullable:"true" create:"optional" index:"true" list:"user" update:"user"`
	ServerID          string `width:"36" nullable:"true" create:"optional" index:"true" list:"user" update:"user"`
	db.SStandaloneResourceBase
}

type SCronjobManager struct {
	db.SStandaloneResourceBaseManager
}

var (
	CronjobManager     *SCronjobManager
	DevToolCronManager *cronman.SCronJobManager
	Session            *mcclient.ClientSession
)

func init() {
	CronjobManager = &SCronjobManager{
		SStandaloneResourceBaseManager: db.NewStandaloneResourceBaseManager(
			SCronjob{},
			"devtool_cronjobs_tbl",
			"devtool_cronjob",
			"devtool_cronjobs",
		),
	}
	CronjobManager.SetVirtualObject(CronjobManager)
	db.RegisterModelManager(CronjobManager)
	DevToolCronManager = cronman.InitCronJobManager(true, 8)
	DevToolCronManager.Start()
}

func RunAnsibleCronjob(id string, s *mcclient.ClientSession) cronman.TCronJobFunction {
	return func(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
		obj, err := CronjobManager.FetchById(id)
		if err != nil {
			log.Errorf("No cronjob with id: %s", id)
			return
		}
		log.Debugf("[RunAnsibleCronjob] %+v: ", obj)
		item := obj.(*SCronjob)

		log.Debugf("[RunAnsibleCronjob] perform ansible cronjob run: %s", item.AnsiblePlaybookID)
		ret, err := modules.AnsiblePlaybooks.PerformAction(s, item.AnsiblePlaybookID, "run", nil)
		if err != nil {
			log.Errorf("AnsiblePlaybooks.PerformAction error: %s", err)
		}
		log.Debugf("AnsiblePlaybooks.PerformAction ret: %+v", ret)
	}
}

func AddOneCronjob(item *SCronjob, s *mcclient.ClientSession) error {

	if !item.Enabled {
		log.Debugf("ansible cronjob %s (devtool item.Id: %s) is not enabled", item.Name, item.Id)
		return nil
	}
	if item.Interval > 0 {
		err := DevToolCronManager.AddJobAtIntervalsWithStartRun(item.Id, time.Duration(item.Interval)*time.Second, RunAnsibleCronjob(item.Id, s), item.Start)
		if err != nil {
			log.Errorf("ansible cronjob %s (devtool item.Id: %s) error! %s", item.Name, item.Id, err)
			return err
		}
		log.Infof("ansible cronjob %s (devtool item.Id: %s) registered at item.Interval: %ds", item.Name, item.Id, item.Interval)
	} else {
		err := DevToolCronManager.AddJobEveryFewDays(item.Id, int(item.Day), int(item.Hour), int(item.Min), int(item.Sec), RunAnsibleCronjob(item.Id, s), item.Start)
		if err != nil {
			log.Errorf("ansible cronjob %s (devtool item.Id: %s) registered at item.Interval: item.Day(%d) item.Hour(%d) item.Min(%d) item.Sec(%d) error: %s", item.Name, item.Id, int(item.Day), int(item.Hour), int(item.Min), int(item.Sec), err)
			return err
		}
		log.Infof("ansible cronjob %s (devtool item.Id: %s) registered at item.Interval: item.Day(%d) item.Hour(%d) item.Min(%d) item.Sec(%d)", item.Name, item.Id, int(item.Day), int(item.Hour), int(item.Min), int(item.Sec))
	}
	return nil
}

func (man *SCronjobManager) InitializeData() error {
	if Session == nil {
		Session = auth.GetAdminSession(nil, "", "")
	}

	go func() {
		items := make([]SCronjob, 0)
		q := CronjobManager.Query().Equals("enabled", true)
		err := q.All(&items)
		if err != nil {
			log.Errorf("query error: %s", err)
		}
		for _, item := range items {
			AddOneCronjob(&item, Session)
		}
	}()

	return nil
}

func (job *SCronjob) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerID mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	job.SStandaloneResourceBase.PostCreate(ctx, userCred, nil, query, data)
	AddOneCronjob(job, Session)
}

func (job *SCronjob) PostDelete(ctx context.Context, userCred mcclient.TokenCredential) {
	DevToolCronManager.Remove(job.Id)
}

func (job *SCronjob) PostUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	job.SStandaloneResourceBase.PostUpdate(ctx, userCred, query, data)
	DevToolCronManager.Remove(job.Id)
	AddOneCronjob(job, Session)
}
