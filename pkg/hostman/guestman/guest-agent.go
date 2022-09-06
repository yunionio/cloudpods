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

package guestman

import (
	"context"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/hostman/monitor"
	"yunion.io/x/onecloud/pkg/httperrors"
)

const (
	QGA_LOCK_TIMEOUT = time.Second * 5
	QGA_EXEC_TIMEOUT = time.Second * 10
)

func qgaExec(timeout time.Duration, qgaFunc func(chan error)) error {
	c := make(chan error, 1)
	go qgaFunc(c)
	select {
	case <-time.After(timeout):
		return errors.Errorf("qga command no resp after %fs", timeout.Seconds())
	case err := <-c:
		return err
	}
}

func (m *SGuestManager) checkAndInitGuestQga(sid string) (*SKVMGuestInstance, error) {
	guest, _ := m.GetServer(sid)
	if guest == nil {
		return nil, httperrors.NewNotFoundError("Not found guest by id %s", sid)
	}
	if !guest.IsRunning() {
		return nil, httperrors.NewBadRequestError("Guest %s is not in state running", sid)
	}
	if guest.guestAgent == nil {
		if err := guest.InitQga(); err != nil {
			return nil, errors.Wrap(err, "init qga")
		}
	}
	return guest, nil
}

func (m *SGuestManager) QgaGuestSetPassword(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	input := params.(*SQgaGuestSetPassword)
	guest, err := m.checkAndInitGuestQga(input.Sid)
	if err != nil {
		return nil, err
	}

	f := func(c chan error) {
		if guest.guestAgent.TryLock(QGA_LOCK_TIMEOUT) {
			defer guest.guestAgent.Unlock()
			c <- guest.guestAgent.GuestSetUserPassword(input.Username, input.Password, input.Crypted)
		} else {
			c <- errors.Errorf("qga unfinished last cmd, is qga unavailable?")
		}
	}
	err = qgaExec(QGA_EXEC_TIMEOUT, f)
	return nil, err
}

func (m *SGuestManager) QgaGuestPing(ctx context.Context, params interface{}) (jsonutils.JSONObject, error) {
	input := params.(*SBaseParms)
	guest, err := m.checkAndInitGuestQga(input.Sid)
	if err != nil {
		return nil, err
	}

	f := func(c chan error) {
		if guest.guestAgent.TryLock(QGA_LOCK_TIMEOUT) {
			defer guest.guestAgent.Unlock()
			c <- guest.guestAgent.GuestPing()
		} else {
			c <- errors.Errorf("qga unfinished last cmd, is qga unavailable?")
		}
	}
	err = qgaExec(QGA_EXEC_TIMEOUT, f)
	return nil, err
}

func (m *SGuestManager) QgaCommand(cmd *monitor.Command, sid string) (string, error) {
	guest, err := m.checkAndInitGuestQga(sid)
	if err != nil {
		return "", err
	}
	var res []byte
	f := func(c chan error) {
		if guest.guestAgent.TryLock(QGA_LOCK_TIMEOUT) {
			defer guest.guestAgent.Unlock()
			res, err = guest.guestAgent.QgaCommand(cmd)
			c <- err
		} else {
			c <- errors.Errorf("qga unfinished last cmd, is qga unavailable?")
		}
	}
	err = qgaExec(QGA_EXEC_TIMEOUT, f)
	return string(res), err
}
