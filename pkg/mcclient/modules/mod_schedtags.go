package modules

import (
	"sync"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SchedtagManager struct {
	ResourceManager
}

var (
	Schedtags SchedtagManager
)

func (this *SchedtagManager) DoBatchSchedtagHostAddRemove(s *mcclient.ClientSession, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	var wg sync.WaitGroup
	ret := jsonutils.NewDict()
	hosts, e := params.GetArray("hosts")
	if e != nil {
		return ret, e
	}
	tags, e := params.GetArray("tags")
	if e != nil {
		return ret, e
	}
	action, e := params.GetString("action")
	if e != nil {
		return ret, e
	}

	wg.Add(len(tags) * len(hosts))

	for _, host := range hosts {
		for _, tag := range tags {
			go func(host, tag jsonutils.JSONObject) {
				defer wg.Done()
				_host, _ := host.GetString()
				_tag, _ := tag.GetString()
				if action == "remove" {
					Schedtaghosts.Detach(s, _tag, _host)
				} else if action == "add" {
					Schedtaghosts.Attach(s, _tag, _host, nil)
				}
			}(host, tag)
		}
	}

	wg.Wait()
	return ret, nil
}

func init() {
	Schedtags = SchedtagManager{NewComputeManager("schedtag", "schedtags",
		[]string{"ID", "Name", "Default_strategy"},
		[]string{})}

	registerCompute(&Schedtags)
}
