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

package main

import (
	"log"
	"net/rpc"
	"strconv"
	"sync"
)

const (
	AGENT_ID   = ""
	APP_KEY    = ""
	APP_SECRET = ""
	userID     = ""
)

func main() {
	client, err := rpc.Dial("unix", "/home/zhengyu/etc/yunion/notify/dingtalk.sock")
	if err != nil {
		log.Fatal(err)
	}
	configArgs := SUpdateConfigArgs{
		Config: map[string]string{
			"agent_id":   AGENT_ID,
			"app_key":    APP_KEY,
			"app_secret": APP_SECRET,
		},
	}
	var reply SSendReply
	err = client.Call("Server.UpdateConfig", &configArgs, &reply)
	if err != nil {
		log.Fatal(err)
	}
	baseMsg := "hello this is TEST"
	if reply.Success {
		var wg sync.WaitGroup

		send := func(n int) {
			var reply SSendReply
			args := SSendArgs{
				Contact:  userID,
				Topic:    "verify",
				Messager: baseMsg + strconv.Itoa(n),
			}
			err := client.Call("Server.Send", &args, &reply)
			if err != nil {
				log.Printf("no.%d send messager about alarm failed because error", n)
				log.Print(err)
			} else if !reply.Success {
				log.Printf("no.%d send messager about alarm failed because reply.Success false", n)
				log.Print(reply.Msg)
			}
			wg.Done()
		}
		for i := 1; i < 10; i++ {
			wg.Add(1)
			go send(i)
		}
		wg.Wait()
	}

}

type SSendArgs struct {
	Contact  string
	Topic    string
	Messager string
}

type SSendReply struct {
	Success bool
	Msg     string
}

type SUpdateConfigArgs struct {
	Config map[string]string
}
