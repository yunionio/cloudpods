package db

var waitQueue []func()

func InitManager(initfunc func()) {
	waitQueue = append(waitQueue, initfunc)
}

func InitAllManagers() {
	for _, f := range waitQueue {
		f()
	}
}
