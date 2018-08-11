package notifyclient

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
)

const (
	PRIORITY_IMPORTANT = "important"
	PRIORITY_CRITICAL  = "fatal"
	PRIORITY_NORMAL    = "normal"

	SERVER_CREATED       = "SERVER_CREATED"
	SERVER_CREATED_ADMIN = "SERVER_CREATED_ADMIN"
	SERVER_DELETED       = "SERVER_DELETED"
	SERVER_DELETED_ADMIN = "SERVER_DELETED_ADMIN"
	SERVER_REBUILD_ROOT  = "SERVER_REBUILD_ROOT"
	SERVER_CHANGE_FLAVOR = "SERVER_CHANGE_FLAVOR"
)

var templateDir string

func SetTemplateDir(dir string) {
	templateDir = dir
}

func NotifySystemError(id string, name string, status string, reason string) error {
	log.Errorf("ID: %s Name %s Status %s REASON %s", id, name, status, reason)
	return nil
}

func Notify(to string, event string, priority string, data jsonutils.JSONObject) error {
	log.Infof("notify %s event %s priority %s data %s", to, event, priority, data)
	return nil
}
