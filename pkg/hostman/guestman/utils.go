package guestman

import (
	"io/ioutil"
	"time"
)

// timer utils

func AddTimeout(second time.Duration, callback func()) {
	go func() {
		<-time.NewTimer(second).C
		callback()
	}()
}

/*
func PathNotExists(path string) bool {
    if _, err := os.Stat(path); os.IsNotExist(err) {
        return true
    }
    return false
}

func PathExists(path string) bool {
    if _, err := os.Stat(path); !os.IsNotExist(err) {
        return true
    }
    return false
}
*/

func FileGetContents(file string) (string, error) {
	content, err := ioutil.ReadFile(file)
	if err != nil {
		return "", err
	}
	return string(content), nil
}
