package shell

import "yunion.io/x/onecloud/pkg/util/printutils"

func printList(data interface{}, columns []string) {
	printutils.PrintGetterList(data, columns)
}

func printObject(obj interface{}) {
	printutils.PrintGetterObject(obj)
}
