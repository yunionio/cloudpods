package modules

import (
	"fmt"
	"net/url"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type NodeManager struct {
	ResourceManager
}

func _genSubtmitResults(idlist []string, status int, err error) []SubmitResult {
	results := make([]SubmitResult, len(idlist))
	var data jsonutils.JSONObject
	if err != nil {
		data = jsonutils.NewString(err.Error())
	} else {
		data = jsonutils.JSONNull
	}
	for i := 0; i < len(results); i++ {
		results[i] = SubmitResult{Id: idlist[i], Status: status, Data: data}
	}
	return results
}

func (this *NodeManager) _batchPerformInContexts(s *mcclient.ClientSession, idlist []string, action string, params jsonutils.JSONObject, ctxs []ManagerContext) []SubmitResult {
	labels, err := params.Get("labels")
	if err != nil {
		return _genSubtmitResults(idlist, 406, fmt.Errorf("Missing labels"))
	}

	node := jsonutils.NewDict()
	node.Add(jsonutils.NewStringArray(idlist), "nodes")
	node.Add(labels, "labels")

	body := jsonutils.NewDict()
	body.Add(node, "node")

	path := fmt.Sprintf("/%s/%s", this.ContextPath(ctxs), url.PathEscape(action))
	_, err = this._post(s, path, body, this.KeywordPlural)
	if err != nil {
		return _genSubtmitResults(idlist, 400, err)
	}
	return _genSubtmitResults(idlist, 200, nil)
}

func (this *NodeManager) BatchPerformActionInContexts(s *mcclient.ClientSession, idlist []string, action string, params jsonutils.JSONObject, ctxs []ManagerContext) []SubmitResult {
	switch action {
	case "remove-labels":
		return this._batchPerformInContexts(s, idlist, action, params, ctxs)
	case "add-labels":
		return this._batchPerformInContexts(s, idlist, action, params, ctxs)
	default:
		return this.ResourceManager.BatchPerformActionInContexts(s, idlist, action, params, ctxs)
	}
}

func (this *NodeManager) NewNode(s *mcclient.ClientSession, name string, ip string, res_type string) (jsonutils.JSONObject, error) {
	data := jsonutils.NewDict()
	data.Add(jsonutils.NewString(name), "name")
	data.Add(jsonutils.NewString(ip), "ip")
	if len(res_type) > 0 {
		data.Add(jsonutils.NewString(res_type), "res_type")
	}
	arr := jsonutils.NewArray()
	arr.Add(data)
	params := jsonutils.NewDict()
	params.Add(arr, "nodes")
	res, err := this._post(s, "/nodes", params, "nodes")
	if err != nil {
		return nil, err
	}
	fmt.Println(res)
	return res, nil
}

var (
	Nodes NodeManager
)

func init() {
	Nodes = NodeManager{NewMonitorManager("node", "nodes",
		[]string{"ID", "name", "status", "create_by", "update_by", "delete_by", "gmt_create", "gmt_modified", "gmt_delete", "is_deleted", "project_id", "remark"},
		[]string{})}

	register(&Nodes)
}
