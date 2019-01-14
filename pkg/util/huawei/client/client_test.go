package client

import (
	"fmt"
	"testing"
)

func TestClient(t *testing.T) {
	client, err := NewClientWithAccessKey("cn-north-1", "41f6bfe48d7f4455b7754f7c1b11ae34", "XXXXXXXXXXX", "XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX")
	ret, err := client.Projects.List(nil)
	if err != nil {
		fmt.Println(err.Error())
	}

	fmt.Println(ret)

	r, err := client.Projects.Get("41f6bfe48d7f4455b7754f7c1b11ae34", nil)
	if err != nil {
		fmt.Println(err.Error())
	}

	fmt.Println(r)
}
