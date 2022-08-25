package guestman

import (
	"context"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/httperrors"
)

const (
	QGA_LOCK_TIMEOUT = time.Second * 10
	QGA_EXEC_TIMEOUT = time.Second * 5
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

	if guest.guestAgent.TryLock(QGA_LOCK_TIMEOUT) {
		defer guest.guestAgent.Unlock()
	} else {
		return nil, errors.Wrap(err, "qga unfinished last cmd, is qga unavailable?")
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
