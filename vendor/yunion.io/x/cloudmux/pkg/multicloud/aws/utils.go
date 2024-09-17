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

package aws

import (
	"fmt"
	"reflect"
	"strings"
)

func GetBucketName(regionId string, imageId string) string {
	return fmt.Sprintf("imgcache-%s-%s", strings.ToLower(regionId), imageId)
}

func StrVal(s *string) string {
	if s != nil {
		return *s
	}

	return ""
}

func IntVal(s *int64) int64 {
	if s != nil {
		return *s
	}

	return 0
}

// fill a pointer struct with zero value.
func FillZero(i interface{}) error {
	V := reflect.Indirect(reflect.ValueOf(i))

	if !V.CanSet() {
		return fmt.Errorf("input is not addressable: %#v", i)
	}

	if V.Kind() != reflect.Struct {
		return fmt.Errorf("only accept struct type")
	}

	for i := 0; i < V.NumField(); i++ {
		field := V.Field(i)

		if field.Kind() == reflect.Ptr && field.IsNil() {
			if field.CanSet() {
				field.Set(reflect.New(field.Type().Elem()))
			}
		}

		vField := reflect.Indirect(field)
		switch vField.Kind() {
		case reflect.Map:
			vField.Set(reflect.MakeMap(vField.Type()))
		case reflect.Struct:
			if field.CanInterface() {
				err := FillZero(field.Interface())
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func NextDeviceName(curDeviceNames []string) (string, error) {
	currents := []string{}
	for _, item := range curDeviceNames {
		currents = append(currents, strings.ToLower(item))
	}

	for _, prefix := range []string{"/dev/sd", "dev/vxd"} {
		for s := rune('a'); s < rune('z'); s++ {
			device := fmt.Sprintf("%s%c", prefix, s)
			found := false
			for _, item := range currents {
				if strings.HasPrefix(item, device) {
					found = true
				}
			}

			if !found {
				return device, nil
			}
		}
	}

	return "", fmt.Errorf("disk devicename out of index, current deivces: %s", currents)
}
