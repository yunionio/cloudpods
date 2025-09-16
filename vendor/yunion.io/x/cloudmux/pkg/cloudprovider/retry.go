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

package cloudprovider

import (
	"strings"
	"time"
)

func IsError(err error, errs []string) bool {
	for i := range errs {
		if strings.Index(err.Error(), errs[i]) >= 0 {
			return true
		}
	}
	return false
}

func RetryOnError(tryFunc func() error, errs []string, maxTries int) error {
	tried := 0
	for tried < maxTries {
		err := tryFunc()
		if err == nil {
			return nil
		}
		if err != nil && !IsError(err, errs) {
			return err
		}
		tried += 1
		time.Sleep(10 * time.Duration(tried) * time.Second)
	}
	return ErrTimeout
}

func RetryUntil(tryFunc func() (bool, error), maxTries int) error {
	tried := 0
	for tried < maxTries {
		stop, err := tryFunc()
		if stop {
			return nil
		}
		if err != nil {
			return err
		}
		tried += 1
		time.Sleep(10 * time.Duration(tried) * time.Second)
	}
	return ErrTimeout
}
