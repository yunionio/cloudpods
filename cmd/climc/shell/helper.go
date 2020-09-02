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

package shell

import (
	"fmt"
	"reflect"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
)

type ResourceCmd struct {
	manager        modulebase.IBaseManager
	contextManager modulebase.IBaseManager

	keyword string
	prefix  string
}

func NewResourceCmd(manager modulebase.IBaseManager) *ResourceCmd {
	return &ResourceCmd{
		manager: manager,
		keyword: manager.GetKeyword(),
	}
}

func (cmd *ResourceCmd) SetPrefix(prefix string) *ResourceCmd {
	cmd.prefix = prefix
	return cmd
}

func (cmd *ResourceCmd) WithContextManager(manager modulebase.IBaseManager) *ResourceCmd {
	cmd.contextManager = manager
	return cmd
}

func (cmd *ResourceCmd) WithKeyword(keyword string) *ResourceCmd {
	return cmd.SetKeyword(keyword)
}

func (cmd *ResourceCmd) SetKeyword(keyword string) *ResourceCmd {
	if len(keyword) > 0 {
		cmd.keyword = keyword
	}
	return cmd
}

type IOpt interface {
	Params() (jsonutils.JSONObject, error)
}

type IListOpt interface {
	IOpt
	GetContextId() string
	GetExportFile() string
	GetExportKeys() string
	GetExportTexts() string
}

type ICreateOpt interface {
	IOpt
}

func (cmd ResourceCmd) runWithDesc(action, desc string, args interface{}, callback interface{}) {
	man := cmd.manager
	prefix := cmd.prefix
	if prefix != "" {
		prefix = cmd.prefix + "-"
	}
	descKeyword := man.GetKeyword()
	if _, ok := args.(IListOpt); ok {
		descKeyword = man.KeyString()
	}
	if desc == "" {
		desc = fmt.Sprintf("%s %s", strings.Title(action), descKeyword)
	}
	descArgs, ok := args.(IWithDescOpt)
	if ok {
		desc = descArgs.Description()
	}
	R(args, fmt.Sprintf("%s%s-%s", prefix, cmd.keyword, action), desc, callback)
}

func (cmd ResourceCmd) run(action string, args interface{}, callback interface{}) {
	cmd.runWithDesc(action, "", args, callback)
}

func (cmd ResourceCmd) List(args IListOpt) {
	man := cmd.manager
	callback := func(s *mcclient.ClientSession, args IListOpt) error {
		params, err := args.Params()
		if err != nil {
			return err
		}
		var result *modulebase.ListResult
		contextId := args.GetContextId()
		if cmd.contextManager != nil && len(contextId) > 0 {
			result, err = man.(modulebase.Manager).ListInContext(s, params, cmd.contextManager.(modulebase.Manager), contextId)
			if err != nil {
				return err
			}
		} else {
			result, err = man.List(s, params)
			if err != nil {
				return err
			}
		}
		exportFile := args.GetExportFile()
		if len(exportFile) > 0 {
			ExportList(result, exportFile, args.GetExportKeys(), args.GetExportTexts(), man.GetColumns(s))
			return nil
		}
		printList(result, man.GetColumns(s))
		return nil
	}
	cmd.run("list", args, callback)
}

func (cmd ResourceCmd) Create(args ICreateOpt) {
	man := cmd.manager
	callback := func(s *mcclient.ClientSession, args ICreateOpt) error {
		params, err := args.Params()
		if err != nil {
			return err
		}
		ret, err := man.(modulebase.Manager).Create(s, params)
		if err != nil {
			return err
		}
		printObject(ret)
		return nil
	}
	cmd.run("create", args, callback)
}

type IIdOpt interface {
	GetId() string
}

type IIdsOpt interface {
	GetIds() []string
}

type IShowOpt interface {
	IOpt
	IIdOpt
}

func (cmd ResourceCmd) Show(args IShowOpt) {
	man := cmd.manager
	callback := func(s *mcclient.ClientSession, args IShowOpt) error {
		params, err := args.Params()
		if err != nil {
			return err
		}
		ret, err := man.(modulebase.Manager).Get(s, args.GetId(), params)
		if err != nil {
			return err
		}
		printObject(ret)
		return nil
	}
	cmd.runWithDesc("show", fmt.Sprintf("Show details of a %s", man.GetKeyword()), args, callback)
}

func (cmd ResourceCmd) ClassShow(args IShowOpt) {
	man := cmd.manager
	callback := func(s *mcclient.ClientSession, args IShowOpt) error {
		params, err := args.Params()
		if err != nil {
			return err
		}
		ret, err := man.(modulebase.Manager).Get(s, args.GetId(), params)
		if err != nil {
			return err
		}
		printObject(ret)
		return nil
	}
	cmd.runWithDesc(args.GetId(), fmt.Sprintf("Show %s of a %s", args.GetId(), man.GetKeyword()), args, callback)
}

type IGetActionOpt interface {
	IOpt
	IIdOpt
}

type TCustomAction string

const (
	CustomActionGet = TCustomAction("Get")
	CustomActionDo  = TCustomAction("Do")
)

func (cmd ResourceCmd) Custom(action TCustomAction, funcname string, args IGetActionOpt) {
	man := cmd.manager
	callback := func(s *mcclient.ClientSession, args IGetActionOpt) error {
		params, err := args.Params()
		if err != nil {
			return err
		}
		mod := man.(modulebase.Manager)
		modvalue := reflect.ValueOf(mod)
		funcvalue := modvalue.MethodByName(string(action) + utils.Kebab2Camel(funcname, "-"))
		if !funcvalue.IsValid() || funcvalue.IsNil() {
			return fmt.Errorf("funcname %s not found", funcname)
		}
		callParams := make([]reflect.Value, 0)
		callParams = append(callParams, reflect.ValueOf(s))
		if len(args.GetId()) > 0 {
			callParams = append(callParams, reflect.ValueOf(args.GetId()))
		}
		if params == nil {
			params = jsonutils.NewDict()
		}
		callParams = append(callParams, reflect.ValueOf(params))
		retValue := funcvalue.Call(callParams)
		retobj := retValue[0]
		reterr := retValue[1]
		if reterr.IsNil() {
			v, ok := retobj.Interface().(jsonutils.JSONObject)
			if ok {
				printObject(v)
				return nil
			}
		}
		v, ok := reterr.Interface().(error)
		if ok {
			return v
		}
		return nil
	}
	cmd.runWithDesc(funcname, fmt.Sprintf("Get %s of a %s", funcname, man.GetKeyword()), args, callback)
}

type IDeleteOpt interface {
	IOpt
	IIdOpt
}

func (cmd ResourceCmd) Delete(args IDeleteOpt) {
	man := cmd.manager
	callback := func(s *mcclient.ClientSession, args IDeleteOpt) error {
		params, err := args.Params()
		if err != nil {
			return err
		}
		ret, err := man.(modulebase.Manager).Delete(s, args.GetId(), params)
		if err != nil {
			return err
		}
		printObject(ret)
		return nil
	}
	cmd.runWithDesc("delete", fmt.Sprintf("Delete %s", man.GetKeyword()), args, callback)
}

type IWithDescOpt interface {
	Description() string
}

type IPerformOpt interface {
	IOpt
	IIdOpt
}

func (cmd ResourceCmd) Perform(action string, args IPerformOpt) {
	man := cmd.manager
	callback := func(s *mcclient.ClientSession, args IPerformOpt) error {
		params, err := args.Params()
		if err != nil {
			return err
		}
		ret, err := man.(modulebase.Manager).PerformAction(s, args.GetId(), action, params)
		if err != nil {
			return err
		}
		printObject(ret)
		return nil
	}
	cmd.run(action, args, callback)
}

func (cmd ResourceCmd) PerformClass(action string, args IOpt) {
	man := cmd.manager
	callback := func(s *mcclient.ClientSession, args IOpt) error {
		params, err := args.Params()
		if err != nil {
			return err
		}
		ret, err := man.(modulebase.Manager).PerformClassAction(s, action, params)
		if err != nil {
			return err
		}
		printObject(ret)
		return nil
	}
	cmd.run(action, args, callback)
}

type IBatchPerformOpt interface {
	IIdsOpt
	IOpt
}

func (cmd ResourceCmd) BatchPerform(action string, args IBatchPerformOpt) {
	man := cmd.manager
	callback := func(s *mcclient.ClientSession, args IBatchPerformOpt) error {
		params, err := args.Params()
		if err != nil {
			return err
		}
		ret := man.(modulebase.Manager).BatchPerformAction(s, args.GetIds(), action, params)
		printBatchResults(ret, man.GetColumns(s))
		return nil
	}
	cmd.run(action, args, callback)
}

type IGetOpt interface {
	IIdOpt
	IOpt
}

func (cmd ResourceCmd) Get(specific string, args IGetOpt) {
	man := cmd.manager
	callback := func(s *mcclient.ClientSession, args IGetOpt) error {
		params, err := args.Params()
		if err != nil {
			return err
		}
		ret, err := man.(modulebase.Manager).GetSpecific(s, args.GetId(), specific, params)
		if err != nil {
			return err
		}
		printObject(ret)
		return nil
	}
	cmd.runWithDesc(specific, fmt.Sprintf("Get %s of a %s", specific, man.GetKeyword()), args, callback)
}

type IUpdateOpt interface {
	IOpt
	IIdOpt
}

func (cmd ResourceCmd) Update(args IUpdateOpt) {
	man := cmd.manager
	callback := func(s *mcclient.ClientSession, args IUpdateOpt) error {
		params, err := args.Params()
		if err != nil {
			return err
		}
		ret, err := man.(modulebase.Manager).Update(s, args.GetId(), params)
		if err != nil {
			return err
		}
		printObject(ret)
		return nil
	}
	cmd.run("update", args, callback)
}

type IMetadataOpt interface {
	IIdOpt
	IOpt
}

func (cmd ResourceCmd) GetMetadata(args IMetadataOpt) {
	man := cmd.manager
	callback := func(s *mcclient.ClientSession, args IMetadataOpt) error {
		params, err := args.Params()
		if err != nil {
			return err
		}
		ret, err := man.(modulebase.IResourceManager).GetMetadata(s, args.GetId(), params)
		if err != nil {
			return err
		}
		printObject(ret)
		return nil
	}
	cmd.runWithDesc("metadata", fmt.Sprintf("Show metadata of a %s", man.GetKeyword()), args, callback)
}

type IBatchDeleteOpt interface {
	IIdsOpt
	IOpt
}

func (cmd ResourceCmd) BatchDelete(args IBatchDeleteOpt) {
	man := cmd.manager
	callback := func(s *mcclient.ClientSession, args IBatchDeleteOpt) error {
		params, err := args.Params()
		if err != nil {
			return err
		}
		ret := man.(modulebase.Manager).BatchDelete(s, args.GetIds(), params)
		printBatchResults(ret, man.GetColumns(s))
		return nil
	}
	cmd.run("delete", args, callback)
}

type IBatchDeleteWithParamOpt interface {
	IBatchDeleteOpt
	QueryParams() (jsonutils.JSONObject, error)
}

func (cmd ResourceCmd) BatchDeleteWithParam(args IBatchDeleteWithParamOpt) {
	man := cmd.manager
	callback := func(s *mcclient.ClientSession, args IBatchDeleteWithParamOpt) error {
		queryParams, err := args.QueryParams()
		if err != nil {
			return err
		}
		params, err := args.Params()
		if err != nil {
			return err
		}
		ret := man.(modulebase.Manager).BatchDeleteWithParam(s, args.GetIds(), queryParams, params)
		printBatchResults(ret, man.GetColumns(s))
		return nil
	}
	cmd.run("delete", args, callback)
}

type IBatchPutOpt interface {
	IIdsOpt
	IOpt
}

func (cmd ResourceCmd) BatchPut(args IBatchPutOpt) {
	man := cmd.manager
	callback := func(s *mcclient.ClientSession, args IBatchPutOpt) error {
		params, err := args.Params()
		if err != nil {
			return err
		}
		ret := man.(modulebase.Manager).BatchPut(s, args.GetIds(), params)
		printBatchResults(ret, man.GetColumns(s))
		return nil
	}
	cmd.run("update", args, callback)
}

type JointCmd struct {
	*ResourceCmd
}

func NewJointCmd(manager modulebase.JointManager) *JointCmd {
	return &JointCmd{
		ResourceCmd: NewResourceCmd(manager),
	}
}

type IJointListOpt interface {
	IListOpt
	GetMasterOpt() string
	GetSlaveOpt() string
}

func (cmd JointCmd) List(args IJointListOpt) {
	man := cmd.manager.(modulebase.JointManager)
	callback := func(s *mcclient.ClientSession, args IJointListOpt) error {
		params, err := args.Params()
		if err != nil {
			return err
		}
		var result *modulebase.ListResult
		if len(args.GetMasterOpt()) > 0 {
			result, err = man.ListDescendent(s, args.GetMasterOpt(), params)
		} else if len(args.GetSlaveOpt()) > 0 {
			result, err = man.ListDescendent2(s, args.GetMasterOpt(), params)
		} else {
			result, err = man.List(s, params)
		}
		if err != nil {
			return err
		}
		printList(result, man.GetColumns(s))
		return nil
	}
	cmd.runWithDesc("list", fmt.Sprintf("list %s %s pairs", man.MasterManager().KeyString(), man.SlaveManager().KeyString()), args, callback)
}

type IJointShowOpt interface {
	IOpt
	GetMasterId() string
	GetSlaveId() string
}

func (cmd JointCmd) Show(args IJointShowOpt) {
	man := cmd.manager.(modulebase.JointManager)
	callback := func(s *mcclient.ClientSession, args IJointShowOpt) error {
		params, err := args.Params()
		if err != nil {
			return err
		}
		result, err := man.Get(s, args.GetMasterId(), args.GetSlaveId(), params)
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	}
	cmd.run("show", args, callback)
}
