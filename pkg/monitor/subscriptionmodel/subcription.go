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

package subscriptionmodel

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/notify"
	"yunion.io/x/onecloud/pkg/monitor/alerting"
	cond "yunion.io/x/onecloud/pkg/monitor/alerting/conditions"
	"yunion.io/x/onecloud/pkg/monitor/alerting/notifiers"
	"yunion.io/x/onecloud/pkg/monitor/alerting/notifiers/templates"
	sub "yunion.io/x/onecloud/pkg/monitor/influxdbsubscribe"
	"yunion.io/x/onecloud/pkg/monitor/models"
	"yunion.io/x/onecloud/pkg/monitor/registry"
	"yunion.io/x/onecloud/pkg/monitor/tsdb"
)

var (
	SubscriptionManager *SSubscriptionManager
)

func init() {
	SubscriptionManager = &SSubscriptionManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			&SSubscriptionManager{},
			"",
			"subscription",
			"subscriptions",
		),
		systemAlerts: new(sync.Map),
	}
	SubscriptionManager.SetVirtualObject(SubscriptionManager)
}

type SSubscriptionManager struct {
	db.SVirtualResourceBaseManager
	systemAlerts *sync.Map
}

func (self *SSubscriptionManager) getThisFunctionUrl() string {
	return fmt.Sprintf("https://%s:%d/%s", monitor.MonitorComponentType, monitor.MonitorComponentPort, monitor.SubscribAPI)
}

func (self *SSubscriptionManager) AddSubscription() {
	sub := models.InfluxdbSubscription{
		SubName:  monitor.MonitorSubName,
		DataBase: monitor.MonitorSubDataBase,
		Rc:       monitor.MonitorDefaultRC,
		Url:      self.getThisFunctionUrl(),
	}
	err := models.DataSourceManager.DropSubscription(sub)
	if err != nil {
		log.Errorln("DropSubscription err:", err)
		return
	}
	log.Infof("drop success")
	err = models.DataSourceManager.AddSubscription(sub)
	if err != nil {
		log.Errorln("add subscription err:", err)
		return
	}
	log.Infof("add success")
	if err := self.LoadSystemAlerts(); err != nil {
		log.Errorf("load system alerts error: %v", err)
		return
	}
}

func (self *SSubscriptionManager) LoadSystemAlerts() error {
	alerts, err := models.CommonAlertManager.GetSystemAlerts()
	if err != nil {
		return errors.Wrap(err, "load system alerts")
	}
	for _, alert := range alerts {
		self.SetAlert(&alert)
	}
	return nil
}

func (self *SSubscriptionManager) AllowPerformWrite(ctx context.Context,
	userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return true
}

func (self *SSubscriptionManager) SetAlert(alert *models.SCommonAlert) {
	self.systemAlerts.Store(alert.GetId(), alert)
}

func (self *SSubscriptionManager) DeleteAlert(alert *models.SCommonAlert) {
	self.systemAlerts.Delete(alert.GetId())
}

func (self *SSubscriptionManager) GetSystemAlerts() []*models.SCommonAlert {
	ret := make([]*models.SCommonAlert, 0)
	self.systemAlerts.Range(func(key, val interface{}) bool {
		ret = append(ret, val.(*models.SCommonAlert))
		return true
	})
	return nil
}

func (self *SSubscriptionManager) PerformWrite(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, data []sub.Point) {
	sysAlerts := self.GetSystemAlerts()
	for _, sysalert := range sysAlerts {
		details := monitor.CommonAlertDetails{}
		details, err := sysalert.GetMoreDetails(ctx, details)
		if err != nil {
			log.Errorln("sysalert getMoreDetails err", err)
			continue
		}
		for _, metricDetails := range details.CommonAlertMetricDetails {
			evalMatch, match, err := self.Eval(*metricDetails, *sysalert, data)
			if err != nil {
				log.Errorln("SSubscriptionManager Eval error:", err)
				continue
			}
			if evalMatch {
				evalCtx := alerting.EvalContext{
					Firing:         true,
					IsTestRun:      false,
					IsDebug:        false,
					EvalMatches:    []*monitor.EvalMatch{match},
					Logs:           nil,
					Error:          nil,
					ConditionEvals: "",
					StartTime:      time.Now(),
					EndTime:        time.Now(),
					Rule:           nil,
					NoDataFound:    false,
					PrevAlertState: sysalert.GetState(),
					Ctx:            context.Background(),
					UserCred:       auth.AdminCredential(),
				}
				err := self.evalNotifyOfAlert(*sysalert, *metricDetails, evalCtx)
				if err != nil {
					log.Errorln(err)
				}
			}
		}
	}
}

func getPointsMeasurement(points []sub.Point) string {
	measurements := make(map[string]int)
	strBuff := new(strings.Builder)
	for _, point := range points {
		if val, ok := measurements[point.Name()]; ok {
			measurements[point.Name()] = val + 1
		}
		measurements[point.Name()] = 1
	}
	for key, count := range measurements {
		strBuff.WriteString(fmt.Sprintf("measurement:%s,count:%d", key, count))
		strBuff.WriteString("\n")
	}
	return strBuff.String()
}

func (self *SSubscriptionManager) isContainNotications(alert models.SCommonAlert) bool {
	alertNotis, err := alert.GetNotifications()
	if err != nil {
		log.Errorln(err)
		return false
	}
	if len(alertNotis) == 0 {
		return false
	}
	for _, an := range alertNotis {
		noti, err := an.GetNotification()
		if err != nil {
			return false
		}
		if !noti.IsDefault {
			return true
		}
	}
	return false
}

func (self *SSubscriptionManager) Eval(details monitor.CommonAlertMetricDetails, alert models.SCommonAlert, points []sub.Point) (bool,
	*monitor.EvalMatch, error) {
	serie := self.getPointsByAlertDetail(details, alert, points)
	reduceCondition := monitor.Condition{
		Type: details.Reduce,
	}
	if len(details.FieldOpt) != 0 {
		reduceCondition.Operators = []string{details.FieldOpt}
	}
	reducer, err := cond.NewAlertReducer(&reduceCondition)
	if err != nil {
		return false, nil, err
	}
	reduceValue, _ := reducer.Reduce(serie)

	evalCond := monitor.Condition{
		Type:   getQueryEvalType(details.Comparator),
		Params: []float64{details.Threshold},
	}
	evaluator, err := cond.NewAlertEvaluator(&evalCond)
	if err != nil {
		return false, nil, err
	}
	if reduceValue != nil {
		log.Printf("name:%s,reduceValue:%f", serie.Name, *reduceValue)
	}
	if evaluator.Eval(reduceValue) {
		match := monitor.EvalMatch{
			Condition: "",
			Value:     reduceValue,
			Metric:    serie.Name,
			Tags:      serie.Tags,
		}
		return true, &match, nil
	}
	return false, nil, nil
}

func getQueryEvalType(evalType string) string {
	typ := ""
	switch evalType {
	case ">=", ">":
		typ = "gt"
	case "<=", "<":
		typ = "lt"
	}
	return typ
}

func (self *SSubscriptionManager) getPointsByAlertDetail(details monitor.CommonAlertMetricDetails, alert models.SCommonAlert,
	points []sub.Point) *tsdb.TimeSeries {
	metricPoints := make([]sub.Point, 0)

	serie := tsdb.TimeSeries{
		RawName: "",
		Name:    "",
		Points:  make(tsdb.TimeSeriesPoints, 0),
		Tags:    nil,
	}

	if len(points) == 0 {
		return &serie
	}
	setting, _ := alert.GetSettings()
	model := setting.Conditions[0].Query.Model
	serie.Name = fmt.Sprintf("%s.%s", details.Measurement, details.Field)
	for _, point := range points {
		if point.Name() == model.Measurement {
			tagBool := true
			for _, tag := range model.Tags {
				if point.Tags().Map()[tag.Key] == tag.Value {
					tagBool = true
				} else {
					tagBool = false
				}

				if strings.ToUpper(tag.Condition) == "AND" && !tagBool {
					break
				}
			}
			if !tagBool {
				continue
			}
			if details.Groupby != "" && point.Tags().Map()[details.Groupby] == "" {
				continue
			}
			metricPoints = append(metricPoints, point)
		}
	}

	if len(metricPoints) == 0 {
		return &serie
	}
	serie.Tags = metricPoints[0].Tags().Map()
	for _, metricPoint := range metricPoints {
		if len(model.Selects) > 1 {
			fieldMap := metricPoint.Fields()
			point := make(tsdb.TimePoint, 0)
			for _, sel := range model.Selects {
				point = append(point, parseValue(fieldMap[sel[0].Params[0]]))
			}

			point = append(point, float64(metricPoint.UnixNano()))

			serie.Points = append(serie.Points, point)
			continue
		}
		fieldPoint := metricPoint.FieldIterator()
		for fieldPoint.Next() {
			if string(fieldPoint.FieldKey()) == details.Field && isValid(fieldPoint) {
				val := fieldPoint.FloatValue()
				timePoint := tsdb.NewTimePoint(&val, float64(metricPoint.UnixNano()))
				serie.Points = append(serie.Points, timePoint)
			}
		}
	}
	return &serie

}

func parseValue(value interface{}) *float64 {
	number, ok := value.(json.Number)
	if !ok {
		return nil
	}

	fvalue, err := number.Float64()
	if err == nil {
		return &fvalue
	}

	ivalue, err := number.Int64()
	if err == nil {
		ret := float64(ivalue)
		return &ret
	}

	return nil
}

func (self *SSubscriptionManager) evalNotifyOfAlert(alert models.SCommonAlert,
	metricDetails monitor.CommonAlertMetricDetails, evalContext alerting.EvalContext) error {
	rule, _ := alerting.NewRuleFromDBAlert(&alert.SAlert)
	rule.State = monitor.AlertStateAlerting
	evalContext.Rule = rule
	var err error
	if self.isContainNotications(alert) {
		switch metricDetails.Reduce {
		case "avg", "sum", "count", "median":
			self.updateAlertJob(alert)

		default:
			// alerting
			err = self.notifyByAlertNotis(alert, evalContext)
			if err != nil {
				log.Errorln("notifyByAlertNotis err:", err)
			}
		}
	} else {
		err = self.notifyBySysConfig(evalContext)
	}
	return err

}

func (self *SSubscriptionManager) updateAlertJob(alert models.SCommonAlert) {
	//upate alert value to dispatched immediately
	alert.Frequency = 1
	rule, err := alerting.NewRuleFromDBAlert(&alert.SAlert)
	if err != nil {
		log.Errorln("SSubscriptionManager updateAlertJob error:", err)
		return
	}
	services := registry.GetServices()
	for _, svc := range services {
		if svc.Name == "AlertEngine" {
			service := svc.Instance.(*alerting.AlertEngine)
			service.Scheduler.Update([]*alerting.Rule{rule})
		}
	}
}

func (self *SSubscriptionManager) notifyByAlertNotis(alert models.SCommonAlert, evalContext alerting.EvalContext) error {
	return self.doNotify(evalContext.Rule.Notifications, &evalContext)
}

func (n *SSubscriptionManager) doNotify(nIds []string, evalCtx *alerting.EvalContext) error {
	notis, err := models.NotificationManager.GetNotificationsWithDefault(nIds)
	if err != nil {
		return err
	}

	for _, obj := range notis {
		not, err := alerting.InitNotifier(alerting.NotificationConfig{
			Id:                    obj.GetId(),
			Name:                  obj.GetName(),
			Type:                  obj.Type,
			Frequency:             time.Duration(obj.Frequency),
			SendReminder:          obj.SendReminder,
			DisableResolveMessage: obj.DisableResolveMessage,
			Settings:              obj.Settings,
		})
		if err != nil {
			log.Errorf("Could not create notifier %s, error: %v", obj.GetId(), err)
			continue
		}
		state, err := models.AlertNotificationManager.Get(evalCtx.Rule.Id, obj.GetId())
		if err != nil {
			log.Errorf(" notification %s to alert %s error: %v", obj.GetName(), evalCtx.Rule.Id, err)
			continue
		}
		err = not.Notify(evalCtx, state.Params)
		if err != nil {
			log.Errorln("not Notify err:", err)
		}
	}
	return nil
}

func (self *SSubscriptionManager) notifyBySysConfig(evalContext alerting.EvalContext) error {
	config := notifiers.GetNotifyTemplateConfig(&evalContext)
	contentConfig := templates.NewTemplateConfig(config)
	content, err := contentConfig.GenerateMarkdown()
	if err != nil {
		return err
	}
	log.Printf("统一报警[alertId:%s,alertName:%s]发生告警", evalContext.Rule.Id, evalContext.Rule.Name)
	notifyclient.SystemNotify(notify.TNotifyPriority(config.Priority), config.Title, jsonutils.NewString(content))
	return nil
}

func isValid(iterator sub.FieldIterator) bool {
	switch iterator.Type() {
	case sub.Float:
		return true
	default:
		return false
	}
}
