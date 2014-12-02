//
// Copyright 2014 Hong Miao. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"github.com/golang/glog"
	"encoding/json"
	"github.com/funny/link"
	"github.com/oikomi/gopush/session_manager/redis_store"
	"github.com/oikomi/gopush/protocol"
)

type SessionManager struct {
	cfg    Config
}   

func NewSessionManager(cfg Config) *SessionManager {
	return &SessionManager {
		cfg : cfg,
	}
}

func (self *SessionManager)connectMsgServer(ms string) (*link.Session, error) {
	p := link.PacketN(2, link.BigEndianBO, link.LittleEndianBF)
	client, err := link.Dial("tcp", ms, p)
	if err != nil {
		glog.Error(err.Error())
		panic(err)
	}

	return client, err
}

func (self *SessionManager)handleMsgServerClient(msc *link.Session, redisStore *redis_store.RedisStore) {
	msc.ReadLoop(func(msg link.InBuffer) {
		glog.Info("msg_server", msc.Conn().RemoteAddr().String(),"say:", string(msg.Get()))
		
		var ss redis_store.StoreSession
		
		glog.Info(string(msg.Get()))
		
		err := json.Unmarshal(msg.Get(), &ss)
		if err != nil {
			glog.Error("error:", err)
		}

		err = redisStore.Set(&ss)
		if err != nil {
			glog.Error("error:", err)
		}
		glog.Info("set sesion id success")
	})

	glog.Info("client", msc.Conn().RemoteAddr().String(), "close")
}

func (self *SessionManager)subscribeChannels(redisStore *redis_store.RedisStore) {
	glog.Info("subscribeChannels")
	var msgServerClientList []*link.Session
	for _, ms := range self.cfg.MsgServerList {
		msgServerClient, err := self.connectMsgServer(ms)
		if err != nil {
			glog.Error(err.Error())
			return
		}
		cmd := protocol.NewCmd()
		
		cmd.CmdName = protocol.SUBSCRIBE_CHANNEL_CMD
		cmd.Args = append(cmd.Args, SYSCTRL_CLIENT_STATUS)
		
		msgServerClient.Send(link.JSON {
			cmd,
		})
		
		msgServerClientList = append(msgServerClientList, msgServerClient)
	}

	for _, msc := range msgServerClientList {
		go self.handleMsgServerClient(msc, redisStore)
	}
}