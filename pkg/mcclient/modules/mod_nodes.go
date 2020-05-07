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

package modules

import (
	"fmt"
	"net/url"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
)

type NodeManager struct {
	modulebase.ResourceManager
}

func _genSubtmitResults(idlist []string, status int, err error) []modulebase.SubmitResult {
	results := make([]modulebase.SubmitResult, len(idlist))
	var data jsonutils.JSONObject
	if err != nil {
		data = jsonutils.NewString(err.Error())
	} else {
		data = jsonutils.JSONNull
	}
	for i := 0; i < len(results); i++ {
		results[i] = modulebase.SubmitResult{Id: idlist[i], Status: status, Data: data}
	}
	return results
}

func (this *NodeManager) _batchPerformInContexts(s *mcclient.ClientSession, idlist []string, action string, params jsonutils.JSONObject, ctxs []modulebase.ManagerContext) []modulebase.SubmitResult {
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
	_, err = modulebase.Post(this.ResourceManager, s, path, body, this.KeywordPlural)
	if err != nil {
		return _genSubtmitResults(idlist, 400, err)
	}
	return _genSubtmitResults(idlist, 200, nil)
}

func (this *NodeManager) BatchPerformActionInContexts(s *mcclient.ClientSession, idlist []string, action string, params jsonutils.JSONObject, ctxs []modulebase.ManagerContext) []modulebase.SubmitResult {
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
	res, err := modulebase.Post(this.ResourceManager, s, "/nodes", params, "nodes")
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
	Nodes = NodeManager{NewServiceTreeManager("node", "nodes",
		[]string{"ID", "name", "status", "create_by", "update_by", "delete_by", "gmt_create", "gmt_modified", "gmt_delete", "is_deleted", "project_id", "remark"},
		[]string{})}

	register(&Nodes)
}
