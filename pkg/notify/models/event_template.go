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

package models

import (
	"context"
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/Masterminds/sprig"
	"golang.org/x/text/language"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	comapi "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/apis/notify"
	api "yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/i18n"
	notifyv2 "yunion.io/x/onecloud/pkg/notify"
	rpcapi "yunion.io/x/onecloud/pkg/notify/rpc/apis"
)

type SEventDisplay struct {
	sEvenWebhookMsg
	ResourceTypeDisplay string
	ActionDisplay       string
	AdvanceDays         int
}

type sEvenWebhookMsg struct {
	ResourceType    string                 `json:"resource_type"`
	Action          string                 `json:"action"`
	ResourceDetails map[string]interface{} `json:"resource_details"`
}

//templateDir = "/opt/yunion/share/local-templates"
type SLocalTemplateManager struct {
	templateDir    string
	templatesTable *sync.Map
}

var LocalTemplateManager *SLocalTemplateManager

func init() {
	LocalTemplateManager = &SLocalTemplateManager{
		templateDir:    "/opt/yunion/share/local-templates",
		templatesTable: &sync.Map{},
	}
}

func languageTag(lang string) language.Tag {
	var langStr string
	if lang == api.TEMPLATE_LANG_CN {
		langStr = "zh-CN"
	} else {
		langStr = "en"
	}
	t, _ := language.Parse(langStr)
	return t
}

func (lt *SLocalTemplateManager) detailsDisplay(resourceType string, details *jsonutils.JSONDict, tag language.Tag) {
	fields, ok := specFields[resourceType]
	if !ok {
		return
	}
	for _, field := range fields {
		if !details.Contains(field) {
			continue
		}
		v, _ := details.GetString(field)
		dv := specFieldTrans[resourceType].LookupByLang(tag, v)
		details.Set(field+"_display", jsonutils.NewString(dv))
	}
}

func (lt *SLocalTemplateManager) FillWithTemplate(ctx context.Context, lang string, no notifyv2.SNotification) (params rpcapi.SendParams, err error) {
	out, event := rpcapi.SendParams{}, no.Event
	rtStr, aStr, resultStr := event.ResourceType(), string(event.Action()), string(event.Result())
	dict, err := jsonutils.ParseString(no.Message)
	if err != nil {
		return out, errors.Wrapf(err, "unable to parse json from %q", no.Message)
	}
	webhookMsg := jsonutils.NewDict()
	webhookMsg.Set("resource_type", jsonutils.NewString(rtStr))
	webhookMsg.Set("action", jsonutils.NewString(aStr))
	webhookMsg.Set("result", jsonutils.NewString(resultStr))
	webhookMsg.Set("resource_details", dict)
	if no.ContactType == api.WEBHOOK {
		return rpcapi.SendParams{
			Title:   no.Event.StringWithDeli("_"),
			Message: webhookMsg.String(),
		}, nil
	}

	if lang == "" {
		lang = getLangSuffix(ctx)
	}

	tag := languageTag(lang)
	rtDis := notifyclientI18nTable.LookupByLang(tag, rtStr)
	if len(rtDis) == 0 {
		rtDis = rtStr
	}
	aDis := notifyclientI18nTable.LookupByLang(tag, aStr)
	if len(aDis) == 0 {
		aDis = aStr
	}
	resultDis := notifyclientI18nTable.LookupByLang(tag, resultStr)
	if len(resultDis) == 0 {
		resultDis = resultStr
	}

	lt.detailsDisplay(rtStr, dict.(*jsonutils.JSONDict), tag)

	templateParams := webhookMsg
	templateParams.Set("advance_days", jsonutils.NewInt(int64(no.AdvanceDays)))
	templateParams.Set("resource_type_display", jsonutils.NewString(rtDis))
	templateParams.Set("action_display", jsonutils.NewString(aDis))
	templateParams.Set("result_display", jsonutils.NewString(resultDis))

	// get title
	title, err := lt.fillWithTemplate(ctx, "title", no.ContactType, lang, event, templateParams)
	if err != nil {
		if errors.Cause(err) == errors.ErrNotFound {
			title = no.Topic
		} else {
			return out, err
		}
	}

	// get content
	content, err := lt.fillWithTemplate(ctx, "content", no.ContactType, lang, event, templateParams)
	if err != nil {
		if errors.Cause(err) == errors.ErrNotFound {
			content = no.Message
		} else {
			return out, err
		}
	}

	out.Title = title
	out.Message = content
	return out, nil
}

var action2Topic = make(map[string]string, 0)

func specTopic(event api.SEvent) string {
	switch event.Action() {
	case api.ActionRebuildRoot, api.ActionChangeIpaddr, api.ActionResetPassword:
		return string(api.ActionUpdate)
	case api.ActionDelete:
		switch event.ResourceType() {
		case api.TOPIC_RESOURCE_BAREMETAL, api.TOPIC_RESOURCE_SERVER, api.TOPIC_RESOURCE_LOADBALANCER, api.TOPIC_RESOURCE_DBINSTANCE, api.TOPIC_RESOURCE_ELASTICCACHE:
			return "DELETE_WITH_IP"
		}
	}
	return ""
}

func init() {
	action2Topic[string(api.ActionRebuildRoot)] = string(api.ActionUpdate)
	action2Topic[string(api.ActionResetPassword)] = string(api.ActionUpdate)
	action2Topic[string(api.ActionChangeIpaddr)] = string(api.ActionUpdate)
}

func (lt *SLocalTemplateManager) fillWithTemplate(ctx context.Context, titleOrContent string, contactType string, lang string, event api.SEvent, dis jsonutils.JSONObject) (string, error) {
	var (
		tmpl *template.Template
		err  error
	)
	actionResultStr := event.ActionWithResult("_")
	for _, topic := range []string{specTopic(event), event.StringWithDeli("_"), actionResultStr, "common"} {
		if topic == "" {
			continue
		}
		tmpl, err = lt.getTemplate(ctx, titleOrContent, contactType, topic, lang)
		if errors.Cause(err) == errors.ErrNotFound {
			continue
		}
		if err != nil {
			return "", errors.Wrap(err, "unable to getTemplate")
		}
		break
	}
	if tmpl == nil {
		return "", errors.ErrNotFound
	}

	buf := strings.Builder{}
	err = tmpl.Execute(&buf, dis.Interface())
	if err != nil {
		return "", errors.Wrap(err, "template.Execute")
	}
	return buf.String(), nil
}

var specFields = map[string][]string{
	notify.TOPIC_RESOURCE_SCALINGPOLICY: {
		"trigger_type",
		"action",
		"unit",
	},
	notify.TOPIC_RESOURCE_SCHEDULEDTASK: {
		"resource_type",
		"operation",
	},
}

var specFieldTrans = map[string]i18n.Table{}

func init() {
	var spI18nTable = i18n.Table{}
	spI18nTable.Set(comapi.TRIGGER_ALARM, i18n.NewTableEntry().EN("alarm").CN("告警"))
	spI18nTable.Set(comapi.TRIGGER_TIMING, i18n.NewTableEntry().EN("timing").CN("定时"))
	spI18nTable.Set(comapi.TRIGGER_CYCLE, i18n.NewTableEntry().EN("cycle").CN("周期"))
	spI18nTable.Set(comapi.ACTION_ADD, i18n.NewTableEntry().EN("add").CN("增加"))
	spI18nTable.Set(comapi.ACTION_REMOVE, i18n.NewTableEntry().EN("remove").CN("减少"))
	spI18nTable.Set(comapi.ACTION_SET, i18n.NewTableEntry().EN("set as").CN("设置为"))
	spI18nTable.Set(comapi.UNIT_ONE, i18n.NewTableEntry().EN("").CN("个"))
	spI18nTable.Set(comapi.UNIT_PERCENT, i18n.NewTableEntry().EN("%").CN("%"))

	var stI18nTable = i18n.Table{}
	stI18nTable.Set(comapi.ST_RESOURCE_SERVER, i18n.NewTableEntry().EN("virtual machine").CN("虚拟机"))
	stI18nTable.Set(comapi.ST_RESOURCE_OPERATION_RESTART, i18n.NewTableEntry().EN("restart").CN("重启"))
	stI18nTable.Set(comapi.ST_RESOURCE_OPERATION_STOP, i18n.NewTableEntry().EN("stop").CN("关机"))
	stI18nTable.Set(comapi.ST_RESOURCE_OPERATION_START, i18n.NewTableEntry().EN("start").CN("开机"))

	specFieldTrans[notify.TOPIC_RESOURCE_SCALINGPOLICY] = spI18nTable
	specFieldTrans[notify.TOPIC_RESOURCE_SCHEDULEDTASK] = stI18nTable
}

func (lt *SLocalTemplateManager) getTemplate(ctx context.Context, titleOrContent string, contactType string, topic string, lang string) (*template.Template, error) {
	key := fmt.Sprintf("%s.%s@%s", topic, titleOrContent, lang)

	obj, ok := lt.templatesTable.Load(key)
	var elem sTemplateElem
	if !ok {
		// read from file
		cont, err := lt.getTemplateString(ctx, titleOrContent, "", topic, lang)
		if err != nil {
			if err == errors.ErrNotFound {
				elem = sTemplateElem{
					template: nil,
				}
			}
			return nil, err
		} else {
			tmp := template.New(key)
			tmp.Funcs(sprig.FuncMap())
			tmp, err = tmp.Parse(string(cont))
			if err != nil {
				return nil, err
			}
			elem = sTemplateElem{
				template: tmp,
			}
		}
		lt.templatesTable.Store(key, elem)
	} else {
		elem = obj.(sTemplateElem)
	}
	if elem.template == nil {
		return nil, errors.ErrNotFound
	}
	return elem.template, nil
}

func (lt *SLocalTemplateManager) getTemplateString(ctx context.Context, titleOrContent string, contactType string, topic string, lang string) ([]byte, error) {
	topic = strings.ToUpper(topic)
	titleOrContent = titleOrContent + "@" + lang
	var path string
	if len(contactType) > 0 {
		path = filepath.Join(lt.templateDir, titleOrContent, contactType, fmt.Sprintf("%s.tmpl", topic))
	} else {
		path = filepath.Join(lt.templateDir, titleOrContent, fmt.Sprintf("%s.tmpl", topic))
	}
	content, err := ioutil.ReadFile(path)
	if err != nil {
		if _, ok := err.(*os.PathError); ok {
			return nil, errors.ErrNotFound
		}
		return nil, err
	}
	return content, nil
}

var (
	notifyclientI18nTable = i18n.Table{}
)

type sTemplateElem struct {
	template *template.Template
}

func setI18nTable(t i18n.Table, elems ...sI18nElme) {
	for i := range elems {
		t.Set(elems[i].k, i18n.NewTableEntry().EN(elems[i].en).CN(elems[i].cn))
	}
}

func getLangSuffix(ctx context.Context) string {
	return notifyclientI18nTable.Lookup(ctx, tempalteLang)
}

const (
	tempalteLang = "lang"
)

type sI18nElme struct {
	k  string
	en string
	cn string
}

func init() {
	setI18nTable(notifyclientI18nTable,
		sI18nElme{
			tempalteLang,
			api.TEMPLATE_LANG_EN,
			api.TEMPLATE_LANG_CN,
		},
		sI18nElme{
			api.TOPIC_RESOURCE_SERVER,
			"virtual machine",
			"虚拟机",
		},
		sI18nElme{
			api.TOPIC_RESOURCE_SCALINGGROUP,
			"scaling group",
			"弹性伸缩组",
		},
		sI18nElme{
			api.TOPIC_RESOURCE_SCALINGPOLICY,
			"scaling policy",
			"弹性伸缩策略",
		},
		sI18nElme{
			api.TOPIC_RESOURCE_IMAGE,
			"image",
			"系统镜像",
		},
		sI18nElme{
			api.TOPIC_RESOURCE_DISK,
			"disk",
			"硬盘",
		},
		sI18nElme{
			api.TOPIC_RESOURCE_SNAPSHOT,
			"snapshot",
			"硬盘快照",
		},
		sI18nElme{
			api.TOPIC_RESOURCE_INSTANCESNAPSHOT,
			"instance snapshot",
			"主机快照",
		},
		sI18nElme{
			api.TOPIC_RESOURCE_NETWORK,
			"network",
			"IP子网",
		},
		sI18nElme{
			api.TOPIC_RESOURCE_EIP,
			"EIP",
			"弹性公网IP",
		},
		sI18nElme{
			api.TOPIC_RESOURCE_SECGROUP,
			"security group",
			"安全组",
		},
		sI18nElme{
			api.TOPIC_RESOURCE_LOADBALANCER,
			"loadbalancer instance",
			"负载均衡实例",
		},
		sI18nElme{
			api.TOPIC_RESOURCE_LOADBALANCERACL,
			"loadbalancer ACL",
			"负载均衡访问控制",
		},
		sI18nElme{
			api.TOPIC_RESOURCE_LOADBALANCERCERTIFICATE,
			"loadbalancer certificate",
			"负载均衡证书",
		},
		sI18nElme{
			api.TOPIC_RESOURCE_BUCKET,
			"object storage bucket",
			"对象存储桶",
		},
		sI18nElme{
			api.TOPIC_RESOURCE_DBINSTANCE,
			"RDS instance",
			"RDS实例",
		},
		sI18nElme{
			api.TOPIC_RESOURCE_ELASTICCACHE,
			"Redis instance",
			"Redis实例",
		},
		sI18nElme{
			api.TOPIC_RESOURCE_SCHEDULEDTASK,
			"scheduled task",
			"定时任务",
		},
		sI18nElme{
			api.TOPIC_RESOURCE_BAREMETAL,
			"baremetal",
			"裸金属",
		},
		sI18nElme{
			string(api.ActionCreate),
			"created",
			"创建",
		},
		sI18nElme{
			string(api.ActionDelete),
			"deleted",
			"删除",
		},
		sI18nElme{
			string(api.ActionRebuildRoot),
			"rebuilded root",
			"重装系统",
		},
		sI18nElme{
			string(api.ActionResetPassword),
			"reseted password",
			"重置密码",
		},
		sI18nElme{
			string(api.ActionChangeConfig),
			"changed config",
			"更改配置",
		},
		sI18nElme{
			string(api.ActionResize),
			"resize",
			"扩容",
		},
		sI18nElme{
			string(api.ActionExpiredRelease),
			"expired and released",
			"到期释放",
		},
		sI18nElme{
			string(api.ActionExecute),
			"executed",
			"生效执行",
		},
		sI18nElme{
			string(api.ActionPendingDelete),
			"added to the recycle bin",
			"加入回收站",
		},
		sI18nElme{
			string(api.ResultFailed),
			"failed",
			"失败",
		},
		sI18nElme{
			string(api.ResultSucceed),
			"successfully",
			"成功",
		},
	)
}
