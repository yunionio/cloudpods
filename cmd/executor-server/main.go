package main

import "yunion.io/x/onecloud/pkg/executor/executeserver"

func main() {
	executeserver.NewExecuteService().Run()
}
