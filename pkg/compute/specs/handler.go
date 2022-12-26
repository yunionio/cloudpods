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

package specs

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/appctx"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/sets"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
)

type specHandleFunc func(context context.Context, userCred mcclient.TokenCredential, query *jsonutils.JSONDict) (jsonutils.JSONObject, error)

func AddSpecHandler(prefix string, app *appsrv.Application) {
	// get models specs
	for key, handleF := range map[string]specHandleFunc{
		"":                 models.GetAllModelSpecs,
		"hosts":            models.GetHostSpecs,
		"isolated_devices": models.GetIsolatedDeviceSpecs,
		"servers":          models.GetServerSpecs,
	} {
		addModelSpecHandler(prefix, key, handleF, app)
	}

	// get model objects by spec key
	for key, handleF := range map[string]specQueryHandleFunc{
		"hosts":            queryHosts,
		"isolated_devices": queryIsolatedDevices,
	} {
		AddQuerySpecModelHandler(prefix, key, handleF, app)
	}
}

func processFilter(handleFunc specHandleFunc) appsrv.FilterHandler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		userCred := auth.FetchUserCredential(ctx, policy.FilterPolicyCredential)
		query, err := jsonutils.ParseQueryString(r.URL.RawQuery)
		if err != nil {
			httperrors.GeneralServerError(ctx, w, err)
			return
		}
		params := appctx.AppContextParams(ctx)
		for key, v := range params {
			query.(*jsonutils.JSONDict).Add(jsonutils.NewString(v), key)
		}
		spec, err := handleFunc(ctx, userCred, query.(*jsonutils.JSONDict))
		if err != nil {
			httperrors.GeneralServerError(ctx, w, err)
			return
		}
		ret := jsonutils.NewDict()
		ret.Add(spec, "spec")
		appsrv.SendJSON(w, ret)
	}
}

func addModelSpecHandler(prefix, managerPluralKey string, handleFunc specHandleFunc, app *appsrv.Application) {
	af := auth.Authenticate(processFilter(handleFunc))
	name := "get_spec"
	prefix = fmt.Sprintf("%s/specs", prefix)
	if len(managerPluralKey) != 0 {
		prefix = fmt.Sprintf("%s/%s", prefix, managerPluralKey)
		name = fmt.Sprintf("get_%s_spec", managerPluralKey)
	}
	app.AddHandler2("GET", prefix, af, nil, name, nil)
}

func AddQuerySpecModelHandler(prefix, managerPluralKey string, handleFunc specQueryHandleFunc, app *appsrv.Application) {
	af := auth.Authenticate(processFilter(queryModelHandle(handleFunc)))
	prefix = fmt.Sprintf("%s/specs/%s", prefix, managerPluralKey)
	name := fmt.Sprintf("get_%s_spec_query", managerPluralKey)
	app.AddHandler2("GET", fmt.Sprintf("%s/<spec_key>/resource", prefix), af, nil, name, nil)
}

func parseSpecKey(key string) ([]string, error) {
	unEscapeStr, err := url.PathUnescape(key)
	if err != nil {
		return nil, err
	}
	return strings.Split(unEscapeStr, "/"), nil
}

type specQueryHandleFunc func(context.Context, mcclient.TokenCredential, *jsonutils.JSONDict, []string) (jsonutils.JSONObject, error)

func queryModelHandle(queryF specQueryHandleFunc) specHandleFunc {
	return func(ctx context.Context, userCred mcclient.TokenCredential, query *jsonutils.JSONDict) (jsonutils.JSONObject, error) {
		specKey, _ := query.GetString("<spec_key>")
		if specKey == "" {
			return nil, httperrors.NewInputParameterError("Empty spec query key")
		}
		specKeys, err := parseSpecKey(specKey)
		if err != nil {
			return nil, httperrors.NewInputParameterError("Parse spec key %s error: %v", specKey, err)
		}
		return queryF(ctx, userCred, query, specKeys)
	}
}

func queryHosts(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query *jsonutils.JSONDict,
	specKeys []string,
) (jsonutils.JSONObject, error) {
	gpuModels := []string{}
	newSpecs := []string{}
	gpuHostIds := []string{}
	for _, specKey := range specKeys {
		if specKey == "gpu_model" {
			gpuModels = append(gpuModels, strings.Split(specKey, ":")[1])
		} else {
			newSpecs = append(newSpecs, specKey)
		}
	}

	if len(gpuModels) != 0 {
		devs, err := models.IsolatedDeviceManager.FindUnusedByModels(gpuModels)
		if err != nil {
			return nil, err
		}
		for _, dev := range devs {
			gpuHostIds = append(gpuHostIds, dev.HostId)
		}
	}

	isOk := func(obj models.ISpecModel) bool {
		if len(gpuHostIds) > 0 {
			if !sets.NewString(gpuHostIds...).Has(obj.GetId()) {
				return false
			}
			gpus, _ := models.IsolatedDeviceManager.FindUnusedGpusOnHost(obj.GetId())
			if len(gpus) == 0 {
				return false
			}
			hostGpuModels := []string{}
			for _, gpu := range gpus {
				hostGpuModels = append(hostGpuModels, gpu.Model)
			}
			if !sets.NewString(hostGpuModels...).IsSuperset(sets.NewString(gpuModels...)) {
				return false
			}
		}
		return true
	}
	return handleQueryModel(models.HostManager, ctx, userCred, query, specKeys, isOk)
}

func queryIsolatedDevices(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query *jsonutils.JSONDict,
	specKeys []string,
) (jsonutils.JSONObject, error) {
	return handleQueryModel(models.IsolatedDeviceManager, ctx, userCred, query, specKeys, nil)
}

func handleQueryModel(
	manager models.ISpecModelManager,
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query *jsonutils.JSONDict,
	specKeys []string,
	isOkF func(models.ISpecModel) bool,
) (jsonutils.JSONObject, error) {
	objects, err := models.ListItems(manager, ctx, userCred, query)
	if err != nil {
		return nil, httperrors.NewInternalServerError("Get object error: %v", err)
	}
	if len(objects) == 0 {
		ret := jsonutils.NewArray()
		return ret, nil
	}
	statusCheck, err := manager.GetSpecShouldCheckStatus(query)
	if err != nil {
		return nil, err
	}
	objs := QueryObjects(manager, objects, specKeys, isOkF, statusCheck)
	return QueryObjectsToJson(manager, objs, ctx, userCred, query)
}

func QueryObjects(
	manager models.ISpecModelManager,
	objs []models.ISpecModel,
	specKeys []string,
	isOkF func(models.ISpecModel) bool,
	statusCheck bool,
) []models.ISpecModel {
	selectedObjs := make([]models.ISpecModel, 0)
	for _, obj := range objs {
		specs := obj.GetSpec(statusCheck)
		if specs == nil {
			continue
		}
		if isOkF != nil && !isOkF(obj) {
			continue
		}
		specIdents := manager.GetSpecIdent(specs)
		if len(specIdents) == 0 {
			continue
		}
		if !sets.NewString(specIdents...).IsSuperset(sets.NewString(specKeys...)) {
			continue
		}
		selectedObjs = append(selectedObjs, obj)
	}
	return selectedObjs
}

func QueryObjectsToJson(manager models.ISpecModelManager, objs []models.ISpecModel, ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	iObjs := make([]interface{}, len(objs))
	for i := range objs {
		iObjs[i] = objs[i]
	}
	details, err := db.FetchCustomizeColumns(manager, ctx, userCred, query, iObjs, nil, false)
	if err != nil {
		return nil, errors.Wrapf(err, "get %s details", manager.KeywordPlural())
	}
	ret := jsonutils.NewArray()
	for idx, obj := range objs {
		jsonData := jsonutils.Marshal(obj)
		jsonDict, ok := jsonData.(*jsonutils.JSONDict)
		if !ok {
			return nil, fmt.Errorf("Invalid model data structure, not a dict")
		}
		extraDict := details[idx]
		if extraDict != nil {
			jsonDict.Update(extraDict)
		}
		ret.Add(jsonDict)
	}
	return ret, nil
}
