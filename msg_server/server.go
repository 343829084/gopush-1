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
	"time"
	"flag"
	"sync"
	"encoding/json"
	"github.com/golang/glog"
	"github.com/funny/link"
	"github.com/oikomi/gopush/base"
	"github.com/oikomi/gopush/common"
	"github.com/oikomi/gopush/protocol"
	"github.com/oikomi/gopush/storage"
)

func init() {
	flag.Set("alsologtostderr", "true")
	flag.Set("log_dir", "false")
}

type MsgServer struct {
	cfg               *MsgServerConfig
	sessions          base.SessionMap
	channels          base.ChannelMap
	topics            protocol.TopicMap
	server            *link.Server
	sessionStore      *storage.SessionStore
	topicStore        *storage.TopicStore
	scanSessionMutex  sync.Mutex
}

func NewMsgServer(cfg *MsgServerConfig) *MsgServer {
	return &MsgServer {
		cfg                : cfg,
		sessions           : make(base.SessionMap),
		channels           : make(base.ChannelMap),
		topics             : make(protocol.TopicMap),
		server             : new(link.Server),
		sessionStore       : storage.NewSessionStore(storage.NewRedisStore(&storage.RedisStoreOptions{
			Network        : "tcp",
			Address        : cfg.Redis.Port,
			ConnectTimeout : time.Duration(cfg.Redis.ConnectTimeout)*time.Millisecond,
			ReadTimeout    : time.Duration(cfg.Redis.ReadTimeout)*time.Millisecond,
			WriteTimeout   : time.Duration(cfg.Redis.WriteTimeout)*time.Millisecond,
			Database       : 1,
			KeyPrefix      : "push",
		})),
		topicStore         : storage.NewTopicStore(storage.NewRedisStore(&storage.RedisStoreOptions {
			Network        : "tcp",
			Address        : cfg.Redis.Port,
			ConnectTimeout : time.Duration(cfg.Redis.ConnectTimeout)*time.Millisecond,
			ReadTimeout    : time.Duration(cfg.Redis.ReadTimeout)*time.Millisecond,
			WriteTimeout   : time.Duration(cfg.Redis.WriteTimeout)*time.Millisecond,
			Database       : 1,
			KeyPrefix      : "push",
		})),
	}
}

func (self *MsgServer)createChannels() {
	glog.Info("createChannels")
	for _, c := range base.ChannleList {
		glog.Info(c)
		channel := link.NewChannel(self.server.Protocol())
		self.channels[c] = base.NewChannelState(c, channel)
	}
}

func (self *MsgServer)scanDeadSession() {
	glog.Info("scanDeadSession")
	timer := time.NewTicker(self.cfg.ScanDeadSessionTimeout * time.Second)
	ttl := time.After(self.cfg.Expire * time.Second)
	for {
		select {
		case <-timer.C:
			//glog.Info("scanDeadSession timeout")
			go func() {
				for id, s := range self.sessions {
					self.scanSessionMutex.Lock()
					defer self.scanSessionMutex.Unlock()
					if (s.State).(*base.SessionState).Alive == false {
						glog.Info("delete" + id)
						delete(self.sessions, id)
						err := common.DelSessionFromCID(self.sessionStore, id)
						if err != nil {
							glog.Warningf("delete ID : %s failed!!", id)
						}
					} else {
						s.State.(*base.SessionState).Alive = false
					}
				}
				
			}()
		case <-ttl:
			break
		}
	}
}

func (self *MsgServer)parseProtocol(cmd []byte, session *link.Session) error {
	var c protocol.CmdSimple
	
	err := json.Unmarshal(cmd, &c)
	if err != nil {
		glog.Error("error:", err)
		return err
	}
	
	pp := NewProtoProc(self)
	
	glog.Info(c.CmdName)

	switch c.CmdName {
		case protocol.SEND_PING_CMD:
			pp.procPing(c, session)
		case protocol.SUBSCRIBE_CHANNEL_CMD:
			pp.procSubscribeChannel(c, session)
		case protocol.SEND_CLIENT_ID_CMD:
			err = pp.procClientID(c, session)
			if err != nil {
				glog.Error("error:", err)
				return err
			}
		case protocol.SEND_MESSAGE_P2P_CMD:
			pp.procSendMessageP2P(c, session)
			if err != nil {
				glog.Error("error:", err)
				return err
			}
		case protocol.ROUTE_MESSAGE_P2P_CMD:
			pp.procRouteMessageP2P(c, session)
			if err != nil {
				glog.Error("error:", err)
				return err
			}
		case protocol.CREATE_TOPIC_CMD:
			pp.procCreateTopic(c, session)
		case protocol.JOIN_TOPIC_CMD:
			pp.procJoinTopic(c, session)
		case protocol.SEND_MESSAGE_TOPIC_CMD:
			pp.procSendMessageTopic(c, session)
			if err != nil {
				glog.Error("error:", err)
				return err
			}
		}

	return err
}
