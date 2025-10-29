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
	"yunion.io/x/pkg/errors"
)

func WaitStatusWithSync(res ICloudResource, expect string, sync func(status string), interval time.Duration, timeout time.Duration) error {
	startTime, status := time.Now(), res.GetStatus()
	for time.Since(startTime) < timeout {
		err := res.Refresh()
		if err != nil {
			return errors.Wrapf(err, "Refresh")
		}
		status = res.GetStatus()
		log.Infof("%s status %s expect %s", res.GetName(), status, expect)
		if sync != nil {
			sync(status)
		}
		if res.GetStatus() == expect {
			return nil
		}
		time.Sleep(interval)
	}
	return errors.Wrapf(errors.ErrTimeout, "status %s expect %s", status, expect)
}

func WaitStatus(res ICloudResource, expect string, interval time.Duration, timeout time.Duration) error {
	return WaitStatusWithSync(res, expect, nil, interval, timeout)
}

func WaitMultiStatusWithSync(res ICloudResource, expects []string, sync func(string), interval time.Duration, timeout time.Duration) error {
	startTime, status := time.Now(), res.GetStatus()
	for time.Since(startTime) < timeout {
		err := res.Refresh()
		if err != nil {
			return errors.Wrapf(err, "Refresh")
		}
		status = res.GetStatus()
		log.Infof("%s status %s expect %v", res.GetName(), status, expects)
		if sync != nil {
			sync(status)
		}
		for _, expect := range expects {
			if status == expect {
				return nil
			}
		}
		time.Sleep(interval)
	}
	return errors.Wrapf(errors.ErrTimeout, "status %s expect %v", status, expects)
}

func WaitMultiStatus(res ICloudResource, expects []string, interval time.Duration, timeout time.Duration) error {
	return WaitMultiStatusWithSync(res, expects, nil, interval, timeout)
}

func WaitStatusWithDelay(res ICloudResource, expect string, delay time.Duration, interval time.Duration, timeout time.Duration) error {
	time.Sleep(delay)
	return WaitStatus(res, expect, interval, timeout)
}

func WaitStatusWithInstanceErrorCheck(res ICloudResource, expect string, interval time.Duration, timeout time.Duration, errCheck func() error) error {
	startTime, status := time.Now(), res.GetStatus()
	for time.Since(startTime) < timeout {
		err := res.Refresh()
		if err != nil {
			return errors.Wrapf(err, "Refresh")
		}
		status = res.GetStatus()
		log.Infof("%s status %s expect %s", res.GetName(), status, expect)
		if status == expect {
			return nil
		}
		err = errCheck()
		if err != nil {
			return err
		}
		time.Sleep(interval)
	}
	return errors.Wrapf(errors.ErrTimeout, "status %s expect %s", status, expect)
}

func WaitDeletedWithDelay(res ICloudResource, delay time.Duration, interval time.Duration, timeout time.Duration) error {
	time.Sleep(delay)
	return WaitDeleted(res, interval, timeout)
}

func WaitDeleted(res ICloudResource, interval time.Duration, timeout time.Duration) error {
	startTime := time.Now()
	for time.Since(startTime) < timeout {
		err := res.Refresh()
		if err != nil {
			if errors.Cause(err) == ErrNotFound {
				return nil
			} else {
				return errors.Wrapf(err, "Refresh")
			}
		}
		time.Sleep(interval)
	}
	return ErrTimeout
}

func Wait(interval time.Duration, timeout time.Duration, callback func() (bool, error)) error {
	startTime := time.Now()
	for time.Since(startTime) < timeout {
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
	for time.Since(startTime) < timeout {
		ok := callback()
		if ok {
			return nil
		}
		time.Sleep(interval)
	}
	return ErrTimeout
}
