package cloudprovider

import (
	"strings"
	"time"

	"yunion.io/x/log"
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

func Retry(tryFunc func() error, maxTries int) error {
	tried := 0
	for tried < maxTries {
		err := tryFunc()
		if err == nil {
			return nil
		}
		tried += 1
		log.Errorf("Tried %d fail %s", tried, err)
		time.Sleep(10 * time.Duration(tried) * time.Second)
	}
	return ErrTimeout
}
