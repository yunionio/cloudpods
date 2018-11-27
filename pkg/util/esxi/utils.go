package esxi

import (
	"context"
	"reflect"

	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/types"

	"yunion.io/x/log"
)

func Properties(c *vim25.Client, ctx context.Context, r types.ManagedObjectReference, ps []string, dst interface{}) error {
	return property.DefaultCollector(c).RetrieveOne(ctx, r, ps, dst)
}

func moRefId(ref types.ManagedObjectReference) string {
	return ref.Value
}

func moRefType(ref types.ManagedObjectReference) string {
	return ref.Type
}

func FetchAnonymousFieldValue(val interface{}, target interface{}) bool {
	return fetchAnonymousFieldValue(reflect.Indirect(reflect.ValueOf(val)),
		reflect.Indirect(reflect.ValueOf(target)))
}

func fetchAnonymousFieldValue(value reflect.Value, target reflect.Value) bool {
	for i := 0; i < value.NumField(); i += 1 {
		fieldValue := value.Field(i)
		fieldStruct := value.Type().Field(i)
		if fieldStruct.Anonymous && fieldStruct.Type.Kind() == reflect.Struct {
			if fieldStruct.Type == target.Type() {
				target.Set(fieldValue)
				return true
			}
			succ := fetchAnonymousFieldValue(fieldValue, target)
			if succ {
				return true
			}
		}
	}
	return false
}

func StructContains(type1 reflect.Type, type2 reflect.Type) bool {
	if type1.Kind() != reflect.Struct || type2.Kind() != reflect.Struct {
		log.Errorf("types should be struct!")
		return false
	}
	if type1 == type2 {
		return true
	}
	for i := 0; i < type1.NumField(); i += 1 {
		field := type1.Field(i)
		if field.Anonymous && field.Type.Kind() == reflect.Struct {
			contains := StructContains(field.Type, type2)
			if contains {
				return true
			}
		}
	}
	return false
}

func reverseArray(array interface{}) {
	arrayValue := reflect.Indirect(reflect.ValueOf(array))
	if arrayValue.Kind() != reflect.Slice && arrayValue.Kind() != reflect.Array {
		log.Errorf("reverse non array or slice")
		return
	}
	tmp := reflect.Indirect(reflect.New(arrayValue.Type().Elem()))
	for i, j := 0, arrayValue.Len()-1; i < j; i, j = i+1, j-1 {
		tmpi := arrayValue.Index(i)
		tmpj := arrayValue.Index(j)
		tmp.Set(tmpi)
		tmpi.Set(tmpj)
		tmpj.Set(tmp)
	}
}
