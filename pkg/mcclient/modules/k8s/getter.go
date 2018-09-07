package k8s

import (
	"fmt"
	"strings"
	"time"

	"github.com/hako/durafmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
)

var (
	getName      nameGetter
	getStatus    statusGetter
	getAge       ageGetter
	getNamespace namespaceGetter
	getLabel     labelGetter
)

type nameGetter struct{}

func (g nameGetter) GetName(obj jsonutils.JSONObject) interface{} {
	name, _ := obj.GetString("name")
	return name
}

type statusGetter struct{}

func (g statusGetter) GetStatus(obj jsonutils.JSONObject) interface{} {
	status, _ := obj.GetString("status")
	return status
}

type ageGetter struct{}

func (g ageGetter) GetAge(obj jsonutils.JSONObject) interface{} {
	creationTimestamp, err := obj.GetString("creationTimestamp")
	if err != nil {
		log.Errorf("Get creationTimestamp error: %v", err)
		return nil
	}
	t, _ := time.Parse(time.RFC3339, creationTimestamp)
	dur := time.Since(t)
	return durafmt.ParseShort(dur).String()
}

type namespaceGetter struct{}

func (g namespaceGetter) GetNamespace(obj jsonutils.JSONObject) interface{} {
	ns, _ := obj.GetString("namespace")
	return ns
}

type labelGetter struct{}

func (g labelGetter) GetLabels(obj jsonutils.JSONObject) interface{} {
	labels, _ := obj.GetMap("labels")
	str := ""
	ls := []string{}
	for k, v := range labels {
		vs, _ := v.GetString()
		ls = append(ls, fmt.Sprintf("%s=%s", k, vs))
	}
	if len(ls) != 0 {
		str = strings.Join(ls, ",")
	}
	return str
}
