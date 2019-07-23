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

type Server struct {
	name string
}

func (s *Server) Send(args *SSendArgs, reply *SSendReply) error {
	if senderManager.msgChan == nil {
		reply.Success = false
		reply.Msg = NOTINIT
		return nil
	}
	senderManager.send(args, reply)
	return nil
}

func (s *Server) UpdateConfig(args *SUpdateConfigArgs, reply *SSendReply) error {
	if args.Config == nil {
		reply.Success = false
		reply.Msg = "Config shouldn't be nil."
		return nil
	}
	senderManager.configLock.Lock()
	for key, value := range args.Config {
		senderManager.configCache[key] = value
	}
	senderManager.configLock.Unlock()
	senderManager.restartSender()
	reply.Success = true
	return nil
}

type SSendArgs struct {
	Contact string
	Topic   string
	Message string
}

type SSendReply struct {
	Success bool
	Msg     string
}

type SUpdateConfigArgs struct {
	Config map[string]string
}
