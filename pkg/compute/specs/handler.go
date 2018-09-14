package specs

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"sort"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
)

type sModelManagersMap map[string]ISpecModelManager

func (m sModelManagersMap) Add(mans ...ISpecModelManager) sModelManagersMap {
	for _, man := range mans {
		m[man.KeywordPlural()] = man
	}
	return m
}

var modelManagerMap sModelManagersMap

func init() {
	modelManagerMap = make(map[string]ISpecModelManager)
	modelManagerMap.Add(models.HostManager, models.IsolatedDeviceManager, models.GuestManager)
}

type ISpecModelManager interface {
	db.IStandaloneModelManager
	GetSpecIdent(spec *jsonutils.JSONDict) []string
}

type ISpecModel interface {
	db.IStandaloneModel
	GetSpec(statusCheck bool) *jsonutils.JSONDict
}

type specHandleFunc func(context context.Context, userCred mcclient.TokenCredential, query *jsonutils.JSONDict) (*jsonutils.JSONDict, error)

func AddSpecHandler(prefix string, app *appsrv.Application) {
	for key, handleF := range map[string]specHandleFunc{
		"":                 AllModelSpecsHandler,
		"hosts":            GetHostSpecs,
		"isolated_devices": GetIsolatedDeviceSpecs,
		"servers":          GetServerSpecs,
	} {
		addHandler(prefix, key, handleF, app)
	}
}

func processFilter(handleFunc specHandleFunc) appsrv.FilterHandler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		userCred := auth.FetchUserCredential(ctx)
		query, err := jsonutils.ParseQueryString(r.URL.RawQuery)
		if err != nil {
			httperrors.GeneralServerError(w, err)
			return
		}
		spec, err := handleFunc(ctx, userCred, query.(*jsonutils.JSONDict))
		if err != nil {
			httperrors.GeneralServerError(w, err)
			return
		}
		ret := jsonutils.NewDict()
		ret.Add(spec, "spec")
		appsrv.SendJSON(w, ret)
	}
}

func addHandler(prefix, managerPluralKey string, handleFunc specHandleFunc, app *appsrv.Application) {
	af := auth.Authenticate(processFilter(handleFunc))
	name := "get_spec"
	prefix = fmt.Sprintf("%s/specs", prefix)
	if len(managerPluralKey) != 0 {
		prefix = fmt.Sprintf("%s/%s", prefix, managerPluralKey)
		name = fmt.Sprintf("get_%s_spec", managerPluralKey)
	}
	app.AddHandler2("GET", prefix, af, nil, name, nil)
}

func AllModelSpecsHandler(ctx context.Context, userCred mcclient.TokenCredential, query *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	ret := jsonutils.NewDict()
	for keyword, man := range modelManagerMap {
		spec, err := getModelSpecs(man, ctx, userCred, query)
		if err != nil {
			return nil, err
		}
		ret.Add(spec, keyword)
	}
	return ret, nil
}

func listItems(manager db.IModelManager, ctx context.Context, userCred mcclient.TokenCredential, queryDict *jsonutils.JSONDict) ([]db.IModel, error) {
	q := manager.Query()
	queryDict, err := manager.ValidateListConditions(ctx, userCred, queryDict)
	if err != nil {
		return nil, err
	}
	q, err = db.ListItemQueryFilters(manager, ctx, q, userCred, queryDict)
	if err != nil {
		return nil, err
	}
	rows, err := q.Rows()
	if err != nil {
		return nil, err
	}
	items := make([]db.IModel, 0)
	for rows.Next() {
		item, err := db.NewModelObject(manager)
		if err != nil {
			return nil, err
		}
		itemInitValue := reflect.Indirect(reflect.ValueOf(item))
		item, err = db.NewModelObject(manager)
		if err != nil {
			return nil, err
		}
		itemValue := reflect.Indirect(reflect.ValueOf(item))
		itemValue.Set(itemInitValue)
		err = q.Row2Struct(rows, item)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, err
}

func GetHostSpecs(ctx context.Context, userCred mcclient.TokenCredential, query *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return getModelSpecs(models.HostManager, ctx, userCred, query)
}

func GetIsolatedDeviceSpecs(ctx context.Context, userCred mcclient.TokenCredential, query *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return getModelSpecs(models.IsolatedDeviceManager, ctx, userCred, query)
}

func GetServerSpecs(ctx context.Context, userCred mcclient.TokenCredential, query *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	return getModelSpecs(models.GuestManager, ctx, userCred, query)
}

func getModelSpecs(manager ISpecModelManager, ctx context.Context, userCred mcclient.TokenCredential, query *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	items, err := listItems(manager, ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	retDict := jsonutils.NewDict()
	for _, obj := range items {
		specObj := obj.(ISpecModel)
		spec := specObj.GetSpec(true)
		if spec == nil {
			continue
		}
		log.Errorf("=========get %s spec: %s", specObj.GetShortDesc(), spec)
		specKeys := manager.GetSpecIdent(spec)
		sort.Strings(specKeys)
		specKey := strings.Join(specKeys, "/")
		if oldSpec, _ := retDict.Get(specKey); oldSpec == nil {
			spec.Add(jsonutils.NewInt(1), "count")
			retDict.Add(spec, specKey)
		} else {
			count, _ := oldSpec.Int("count")
			oldSpec.(*jsonutils.JSONDict).Set("count", jsonutils.NewInt(count+1))
			retDict.Set(specKey, oldSpec)
		}
	}
	log.Errorf("========ret specdict: %s", retDict)
	return retDict, nil
}
