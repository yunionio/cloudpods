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
	"time"

	"yunion.io/x/log"
)

func WaitStatus(res ICloudResource, expect string, interval time.Duration, timeout time.Duration) error {
	startTime := time.Now()
	for time.Now().Sub(startTime) < timeout {
		err := res.Refresh()
		if err != nil {
			return err
		}
		log.Debugf("status %s expect %s", res.GetStatus(), expect)
		if res.GetStatus() == expect {
			return nil
		}
		time.Sleep(interval)
	}
	return ErrTimeout
}

func WaitStatusWithDelay(res ICloudResource, expect string, delay time.Duration, interval time.Duration, timeout time.Duration) error {
	time.Sleep(delay)
	return WaitStatus(res, expect, interval, timeout)
}

func WaitStatusWithInstanceErrorCheck(res ICloudResource, expect string, interval time.Duration, timeout time.Duration, errCheck func() error) error {
	startTime := time.Now()
	for time.Now().Sub(startTime) < timeout {
		err := res.Refresh()
		if err != nil {
			return err
		}
		log.Debugf("status %s expect %s", res.GetStatus(), expect)
		if res.GetStatus() == expect {
			return nil
		}
		err = errCheck()
		if err != nil {
			return err
		}
		time.Sleep(interval)
	}
	return ErrTimeout
}

func WaitDeletedWithDelay(res ICloudResource, delay time.Duration, interval time.Duration, timeout time.Duration) error {
	time.Sleep(delay)
	return WaitDeleted(res, interval, timeout)
}

func WaitDeleted(res ICloudResource, interval time.Duration, timeout time.Duration) error {
	startTime := time.Now()
	for time.Now().Sub(startTime) < timeout {
		err := res.Refresh()
		if err != nil {
			if err == ErrNotFound {
				return nil
			} else {
				return err
			}
		}
		time.Sleep(interval)
	}
	return ErrTimeout
}

func Wait(interval time.Duration, timeout time.Duration, callback func() (bool, error)) error {
	startTime := time.Now()
	for time.Now().Sub(startTime) < timeout {
		ok, err := callback()
		if err != nil {
			return err
		}
		if ok {
			return nil
		}
		time.Sleep(interval)
	}
	return ErrTimeout
}

func WaitCreated(interval time.Duration, timeout time.Duration, callback func() bool) error {
	startTime := time.Now()
	for time.Now().Sub(startTime) < timeout {
		ok := callback()
		if ok {
			return nil
		}
		time.Sleep(interval)
	}
	return ErrTimeout
}
