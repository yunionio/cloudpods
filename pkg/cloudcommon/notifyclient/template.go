package notifyclient

import (
	"context"
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/i18n"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	npk "yunion.io/x/onecloud/pkg/mcclient/modules/notify"
)

var (
	templatesTable     map[string]*template.Template
	templatesTableLock *sync.Mutex

	notifyclientI18nTable                        = i18n.Table{}
	AdminSessionGenerator SAdminSessionGenerator = getAdminSesion
	UserLangFetcher       SUserLangFetcher       = getUserLang
	topicWithTemplateSet                         = &sync.Map{}
	checkTemplates        bool
)

type SAdminSessionGenerator func(ctx context.Context, region string, apiVersion string) (*mcclient.ClientSession, error)
type SUserLangFetcher func(uids []string) (map[string]string, error)

func getAdminSesion(ctx context.Context, region string, apiVersion string) (*mcclient.ClientSession, error) {
	return auth.GetAdminSession(ctx, region, apiVersion), nil
}

func getUserLang(uids []string) (map[string]string, error) {
	s, err := AdminSessionGenerator(context.Background(), consts.GetRegion(), "")
	if err != nil {
		return nil, err
	}
	uidLang := make(map[string]string)
	if len(uids) > 0 {
		params := jsonutils.NewDict()
		params.Set("filter", jsonutils.NewString(fmt.Sprintf("id.in(%s)", strings.Join(uids, ","))))
		params.Set("details", jsonutils.JSONFalse)
		params.Set("scope", jsonutils.NewString("system"))
		params.Set("system", jsonutils.JSONTrue)
		ret, err := modules.UsersV3.List(s, params)
		if err != nil {
			return nil, err
		}
		for i := range ret.Data {
			id, _ := ret.Data[i].GetString("id")
			langStr, _ := ret.Data[i].GetString("lang")
			uidLang[id] = langStr
		}
	}
	return uidLang, nil
}

func init() {
	templatesTableLock = &sync.Mutex{}
	templatesTable = make(map[string]*template.Template)
	notifyclientI18nTable.Set(suffix, i18n.NewTableEntry().EN("en").CN("cn"))
	templatesTableLock = &sync.Mutex{}
	templatesTable = make(map[string]*template.Template)
}

func hasTemplateOfTopic(topic string) bool {
	if checkTemplates {
		_, ok := topicWithTemplateSet.Load(topic)
		return ok
	}
	path := filepath.Join(consts.NotifyTemplateDir, consts.GetServiceType(), "content@cn")
	fileInfoList, err := ioutil.ReadDir(path)
	if err != nil {
		if os.IsNotExist(err) {
			checkTemplates = true
			return false
		}
		log.Errorf("unable to read dir %s", path)
		return false
	}
	for i := range fileInfoList {
		topicWithTemplateSet.Store(fileInfoList[i].Name(), nil)
	}
	checkTemplates = true
	_, ok := topicWithTemplateSet.Load(topic)
	return ok
}

func getTemplateString(suffix string, topic string, contType string, channel npk.TNotifyChannel) ([]byte, error) {
	contType = contType + "@" + suffix
	if len(channel) > 0 {
		path := filepath.Join(consts.NotifyTemplateDir, consts.GetServiceType(), contType, fmt.Sprintf("%s.%s", topic, string(channel)))
		cont, err := ioutil.ReadFile(path)
		if err == nil {
			return cont, nil
		}
	}
	path := filepath.Join(consts.NotifyTemplateDir, consts.GetServiceType(), contType, topic)
	return ioutil.ReadFile(path)
}

func getTemplate(suffix string, topic string, contType string, channel npk.TNotifyChannel) (*template.Template, error) {
	key := fmt.Sprintf("%s.%s.%s@%s", topic, contType, channel, suffix)
	templatesTableLock.Lock()
	defer templatesTableLock.Unlock()

	if _, ok := templatesTable[key]; !ok {
		cont, err := getTemplateString(suffix, topic, contType, channel)
		if err != nil {
			return nil, err
		}
		tmp := template.New(key)
		tmp.Funcs(template.FuncMap{"unescaped": unescaped})
		tmp, err = tmp.Parse(string(cont))
		if err != nil {
			return nil, err
		}
		templatesTable[key] = tmp
	}
	return templatesTable[key], nil
}

func getContent(suffix string, topic string, contType string, channel npk.TNotifyChannel, data jsonutils.JSONObject) (string, error) {
	if channel == npk.NotifyByWebhook {
		return "", nil
	}
	tmpl, err := getTemplate(suffix, topic, contType, channel)
	if err != nil {
		return "", err
	}
	buf := strings.Builder{}
	err = tmpl.Execute(&buf, data.Interface())
	if err != nil {
		return "", err
	}
	// log.Debugf("notify.getContent %s %s %s %s", topic, contType, data, buf.String())
	return buf.String(), nil
}

func unescaped(str string) template.HTML {
	return template.HTML(str)
}

const (
	suffix = "suffix"
)
