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
	"yunion.io/x/pkg/util/printutils"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
)

type ResourceCmd struct {
	manager        modulebase.IBaseManager
	contextManager modulebase.IBaseManager

	keyword string
	prefix  string

	printObject func(jsonutils.JSONObject)
}

func NewResourceCmd(manager modulebase.IBaseManager) *ResourceCmd {
	return &ResourceCmd{
		manager:     manager,
		keyword:     manager.GetKeyword(),
		printObject: printObjectRecursive,
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

func (cmd *ResourceCmd) PrintObjectYAML() *ResourceCmd {
	cmd.printObject = func(obj jsonutils.JSONObject) {
		fmt.Print(obj.YAMLString())
	}
	return cmd
}

func (cmd *ResourceCmd) PrintObjectTable() *ResourceCmd {
	cmd.printObject = printutils.PrintJSONObject
	return cmd
}

func (cmd *ResourceCmd) PrintObjectKV() *ResourceCmd {
	cmd.printObject = printObjectFmtKv
	return cmd
}

func (cmd *ResourceCmd) PrintObjectFlattenKV() *ResourceCmd {
	cmd.printObject = func(obj jsonutils.JSONObject) {
		printObjectRecursiveEx(obj, printObjectFmtKv)
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

type IBatchCreateOpt interface {
	IOpt
	GetCountParam() int
}

func (cmd ResourceCmd) RunWithDesc(action, desc string, args interface{}, callback interface{}) {
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
	R(args, fmt.Sprintf("%s%s-%s", prefix, strings.ReplaceAll(cmd.keyword, "_", "-"), action), desc, callback)
}

func (cmd ResourceCmd) Run(action string, args interface{}, callback interface{}) {
	cmd.RunWithDesc(action, "", args, callback)
}

func (cmd ResourceCmd) List(args IListOpt) {
	man := cmd.manager
	callback := func(s *mcclient.ClientSession, args IListOpt) error {
		params, err := args.Params()
		if err != nil {
			return err
		}
		var result *printutils.ListResult
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
		PrintList(result, man.GetColumns(s))
		return nil
	}
	cmd.Run("list", args, callback)
}

func (cmd ResourceCmd) Create(args ICreateOpt) {
	cmd.CreateWithKeyword("create", args)
}

func (cmd ResourceCmd) CreateWithKeyword(keyword string, args ICreateOpt) {
	cmd.Run(keyword, args, cmd.create)
}

func (cmd ResourceCmd) create(s *mcclient.ClientSession, args ICreateOpt) error {
	man := cmd.manager
	params, err := args.Params()
	if err != nil {
		return err
	}
	ret, err := man.(modulebase.Manager).Create(s, params)
	if err != nil {
		return err
	}
	PrintObject(ret)
	return nil
}

func (cmd ResourceCmd) BatchCreate(args IBatchCreateOpt) {
	cmd.BatchCreateWithKeyword("create", args)
}

func (cmd ResourceCmd) BatchCreateWithKeyword(keyword string, args IBatchCreateOpt) {
	cmd.Run(keyword, args, cmd.batchCreate)
}

func (cmd ResourceCmd) batchCreate(s *mcclient.ClientSession, args IBatchCreateOpt) error {
	man := cmd.manager.(modulebase.Manager)
	count := args.GetCountParam()
	if count <= 1 {
		return cmd.create(s, args)
	}
	params, err := args.Params()
	if err != nil {
		return err
	}
	rets := man.BatchCreate(s, params, count)
	printBatchResults(rets, man.GetColumns(s))
	return nil
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

type IPropertyOpt interface {
	IOpt
	Property() string
}

func (cmd ResourceCmd) GetPropertyWithShowFunc(args IPropertyOpt, showFunc func(jsonutils.JSONObject) error) {
	man := cmd.manager
	callback := func(s *mcclient.ClientSession, args IPropertyOpt) error {
		params, err := args.Params()
		if err != nil {
			return err
		}
		ret, err := man.(modulebase.Manager).Get(s, args.Property(), params)
		if err != nil {
			return err
		}
		err = showFunc(ret)
		if err != nil {
			return err
		}
		return nil
	}
	cmd.RunWithDesc(args.Property(), fmt.Sprintf("Get property of a %s", man.GetKeyword()), args, callback)
}

func (cmd ResourceCmd) GetProperty(args IPropertyOpt) {
	cmd.GetPropertyWithShowFunc(args, func(ret jsonutils.JSONObject) error {
		if _, ok := ret.(*jsonutils.JSONArray); ok {
			data, _ := ret.GetArray()
			PrintList(&printutils.ListResult{
				Data: data,
			}, nil)
		} else {
			PrintObject(ret)
		}
		return nil
	})
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
		PrintObject(ret)
		return nil
	}
	cmd.RunWithDesc("show", fmt.Sprintf("Show details of a %s", man.GetKeyword()), args, callback)
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
		PrintObject(ret)
		return nil
	}
	cmd.RunWithDesc(args.GetId(), fmt.Sprintf("Show %s of a %s", args.GetId(), man.GetKeyword()), args, callback)
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
				PrintObject(v)
				return nil
			}
		}
		v, ok := reterr.Interface().(error)
		if ok {
			return v
		}
		return nil
	}
	cmd.RunWithDesc(funcname, fmt.Sprintf("Get %s of a %s", funcname, man.GetKeyword()), args, callback)
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
		PrintObject(ret)
		return nil
	}
	cmd.RunWithDesc("delete", fmt.Sprintf("Delete %s", man.GetKeyword()), args, callback)
}

type IDeleteWithParamOpt interface {
	IDeleteOpt
	QueryParams() (jsonutils.JSONObject, error)
}

func (cmd ResourceCmd) DeleteWithParam(args IDeleteWithParamOpt) {
	man := cmd.manager
	callback := func(s *mcclient.ClientSession, args IDeleteWithParamOpt) error {
		queryParams, err := args.QueryParams()
		if err != nil {
			return err
		}
		params, err := args.Params()
		if err != nil {
			return err
		}
		ret, err := man.(modulebase.Manager).DeleteWithParam(s, args.GetId(), queryParams, params)
		if err != nil {
			return err
		}
		PrintObject(ret)
		return nil
	}
	cmd.RunWithDesc("delete", fmt.Sprintf("Delete %s", man.GetKeyword()), args, callback)
}

type IWithDescOpt interface {
	Description() string
}

type IPerformOpt interface {
	IOpt
	IIdOpt
}

func (cmd ResourceCmd) PerformWithKeyword(keyword, action string, args IPerformOpt) {
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
		cmd.printObject(ret)
		return nil
	}
	cmd.Run(keyword, args, callback)
}

func (cmd ResourceCmd) PerformClassWithKeyword(keyword, action string, args IOpt) {
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
		cmd.printObject(ret)
		return nil
	}
	cmd.Run(keyword, args, callback)
}

func (cmd ResourceCmd) Perform(action string, args IPerformOpt) {
	cmd.PerformWithKeyword(action, action, args)
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
		cmd.printObject(ret)
		return nil
	}
	cmd.Run(action, args, callback)
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
	cmd.Run(action, args, callback)
}

type IGetOpt interface {
	IIdOpt
	IOpt
}

func (cmd ResourceCmd) GetWithCustomShow(specific string, show func(data jsonutils.JSONObject), args IGetOpt) {
	cmd.GetWithCustomOptionShow(specific, func(data jsonutils.JSONObject, _ IGetOpt) {
		show(data)
	}, args)
}

func (cmd ResourceCmd) GetWithCustomOptionShow(specific string, show func(data jsonutils.JSONObject, args IGetOpt), args IGetOpt) {
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
		show(ret, args)
		return nil
	}
	cmd.RunWithDesc(specific, fmt.Sprintf("Get %s of a %s", specific, man.GetKeyword()), args, callback)
}

func (cmd ResourceCmd) Get(specific string, args IGetOpt) {
	cmd.GetWithCustomShow(specific, PrintObject, args)
}

type IUpdateOpt interface {
	IOpt
	IIdOpt
}

func (cmd ResourceCmd) UpdateWithKeyword(keyword string, args IUpdateOpt) {
	man := cmd.manager
	callback := func(s *mcclient.ClientSession, args IUpdateOpt) error {
		_params, err := args.Params()
		if err != nil {
			return err
		}
		params := _params.(*jsonutils.JSONDict)
		params.Remove("id")
		if params.Length() == 0 {
			return InvalidUpdateError()
		}
		ret, err := man.(modulebase.Manager).Update(s, args.GetId(), params)
		if err != nil {
			return err
		}
		PrintObject(ret)
		return nil
	}
	cmd.Run(keyword, args, callback)
}

func (cmd ResourceCmd) Update(args IUpdateOpt) {
	cmd.UpdateWithKeyword("update", args)
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
		PrintObject(ret)
		return nil
	}
	cmd.RunWithDesc("metadata", fmt.Sprintf("Show metadata of a %s", man.GetKeyword()), args, callback)
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
	cmd.Run("delete", args, callback)
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
	cmd.Run("delete", args, callback)
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
	cmd.Run("update", args, callback)
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
		var result *printutils.ListResult
		if len(args.GetMasterOpt()) > 0 {
			result, err = man.ListDescendent(s, args.GetMasterOpt(), params)
		} else if len(args.GetSlaveOpt()) > 0 {
			result, err = man.ListDescendent2(s, args.GetSlaveOpt(), params)
		} else {
			result, err = man.List(s, params)
		}
		if err != nil {
			return err
		}
		PrintList(result, man.GetColumns(s))
		return nil
	}
	cmd.RunWithDesc("list", fmt.Sprintf("list %s %s pairs", man.MasterManager().KeyString(), man.SlaveManager().KeyString()), args, callback)
}

type IJointOpt interface {
	IOpt
	GetMasterId() string
	GetSlaveId() string
}

func (cmd JointCmd) Show(args IJointOpt) {
	man := cmd.manager.(modulebase.JointManager)
	callback := func(s *mcclient.ClientSession, args IJointOpt) error {
		params, err := args.Params()
		if err != nil {
			return err
		}
		result, err := man.Get(s, args.GetMasterId(), args.GetSlaveId(), params)
		if err != nil {
			return err
		}
		PrintObject(result)
		return nil
	}
	cmd.Run("show", args, callback)
}

func (cmd JointCmd) Attach(args IJointOpt) {
	man := cmd.manager.(modulebase.JointManager)
	callback := func(s *mcclient.ClientSession, args IJointOpt) error {
		params, err := args.Params()
		if err != nil {
			return err
		}
		result, err := man.Attach(s, args.GetMasterId(), args.GetSlaveId(), params)
		if err != nil {
			return err
		}
		PrintObject(result)
		return nil
	}
	cmd.Run("attach", args, callback)
}

func (cmd JointCmd) Detach(args IJointOpt) {
	man := cmd.manager.(modulebase.JointManager)
	callback := func(s *mcclient.ClientSession, args IJointOpt) error {
		params, err := args.Params()
		if err != nil {
			return err
		}
		result, err := man.Detach(s, args.GetMasterId(), args.GetSlaveId(), params)
		if err != nil {
			return err
		}
		PrintObject(result)
		return nil
	}
	cmd.Run("detach", args, callback)
}

func (cmd JointCmd) Update(args IJointOpt) {
	man := cmd.manager.(modulebase.JointManager)
	callback := func(s *mcclient.ClientSession, args IJointOpt) error {
		params, err := args.Params()
		if err != nil {
			return err
		}
		result, err := man.Update(s, args.GetMasterId(), args.GetSlaveId(), nil, params)
		if err != nil {
			return err
		}
		PrintObject(result)
		return nil
	}
	cmd.Run("update", args, callback)
}
