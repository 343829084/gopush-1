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
	"flag"
	"log"
	"encoding/binary"
	"github.com/funny/link"
	"math/rand"
)

var InputConfFile = flag.String("conf_file", "gateway.json", "input conf file name")   

func selectMsgServer(serverList []string, serverNum int) string{
	return serverList[rand.Intn(serverNum)]
}

func main() {
	flag.Parse()
	cfg, err := LoadConfig(*InputConfFile)
	if err != nil {
		log.Fatalln(err.Error())
		return
	}
	
	protocol := link.PacketN(2, binary.BigEndian)
	
	server, err := link.Listen(cfg.TransportProtocols, cfg.Listen, protocol)
	if err != nil {
		panic(err)
	}
	log.Println("server start:", server.Listener().Addr().String())
	log.Println(cfg.MsgServerList)

	server.AcceptLoop(func(session *link.Session) {
		log.Println("client", session.Conn().RemoteAddr().String(), "in")
		session.Send(link.Binary(selectMsgServer(cfg.MsgServerList, cfg.MsgServerNum)))
		session.Close(nil)
		log.Println("client", session.Conn().RemoteAddr().String(), "close")
	})
}
