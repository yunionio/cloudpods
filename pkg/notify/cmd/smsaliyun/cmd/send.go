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
	"encoding/json"
	"log"
	"net/rpc"
	"sync"
)

const (
	accessKeyID  = ""
	accessSecret = ""
	signature    = ""
)

func main() {
	client, err := rpc.Dial("unix", "/home/zhengyu/etc/yunion/notify/mobile.sock")
	if err != nil {
		log.Fatal(err)
	}
	configArgs := SUpdateConfigArgs{
		Config: map[string]string{
			"access_key_id":     accessKeyID,
			"access_key_secret": accessSecret,
			"signature":         signature,
		},
	}
	var reply SSendReply
	err = client.Call("Server.UpdateConfig", &configArgs, &reply)
	if err != nil {
		log.Fatal(err)
	}
	map1 := map[string]interface{}{
		"code": "023456",
	}
	map2 := map[string]interface{}{
		"code": "000001",
	}
	json2, err := json.Marshal(&map2)
	json1, err := json.Marshal(&map1)
	args1 := SSendArgs{
		Contact:  "17150164281",
		Topic:    "verify",
		Messager: string(json2),
	}
	args2 := SSendArgs{
		Contact:  "17854290942",
		Topic:    "verify",
		Messager: string(json1),
	}
	if reply.Success {
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			var reply SSendReply
			err := client.Call("Server.Send", &args1, &reply)
			if err != nil {
				log.Print("no.1 send messager about ALARM failed because error")
				log.Print(err)
			} else if !reply.Success {
				log.Print("no.1 send messager about ALARM failed because reply.Success false")
				log.Print(reply.Msg)
			}
			wg.Done()
		}()
		go func() {
			var reply SSendReply
			err := client.Call("Server.Send", &args2, &reply)
			if err != nil {
				log.Print("no.2 send messager about VERIFY failed because error")
				log.Print(err)
			} else if !reply.Success {
				log.Print("no.2 send messager about VERIFY failed because reply.Success false")
				log.Print(reply.Msg)
			}
			wg.Done()
		}()
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
