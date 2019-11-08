// Copyright 2016-2019 DutchSec (https://dutchsec.com/)
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
package telnet

import (
	"context"
	"fmt"
	"io"
	"net"
	"strings"

	"github.com/asdine/storm"
	"github.com/kushtaka/kushtakad/events"
	"github.com/op/go-logging"
	"github.com/rs/xid"
)

var (
	motd = `********************************************************************************
*             Copyright(C) 2008-2015 Huawei Technologies Co., Ltd.             *
*                             All rights reserved                              *
*                  Without the owner's prior written consent,                  *
*           no decompiling or reverse-engineering shall be allowed.            *
* Notice:                                                                      *
*                   This is a private communication system.                    *
*             Unauthorized access or use may lead to prosecution.              *
********************************************************************************

Warning: Telnet is not a secure protocol, and it is recommended to use STelnet. 

Login authentication


`
	prompt = `$ `
)

var log = logging.MustGetLogger("telnet")

// Telnet is a placeholder
func Telnet() *TelnetService {
	s := &TelnetService{
		Emulate: motd,
		Prompt:  prompt,
	}

	return s
}

type TelnetService struct {
	SensorID int64  `json:"sensor_id"`
	Port     int    `json:"port"`
	Prompt   string `json:"prompt"`
	Emulate  string `json:"emulate"`
	Type     string `json:"type"`

	Host   string
	ApiKey string
}

func (s TelnetService) SetHost(h string) {
	s.Host = h
}

func (s TelnetService) SetApiKey(k string) {
	s.ApiKey = k
}

func (s TelnetService) Handle(ctx context.Context, conn net.Conn, db *storm.DB) error {
	defer conn.Close()

	id := xid.New()

	split := strings.Split(conn.RemoteAddr().String(), ":")
	network := conn.RemoteAddr().Network()
	ip := split[0]
	es := &events.EventSensor{
		SensorID:     s.SensorID,
		Type:         s.Type,
		Port:         s.Port,
		AttackerPort: split[1],
	}
	em := events.NewSensorEventManager(network, ip, es)
	err := em.SendEvent("new", s.Host, s.ApiKey)
	if err != nil {
		log.Debug(err)
	}

	term := NewTerminal(conn, s.Prompt)

	term.Write([]byte(s.Emulate + "\n"))

	term.SetPrompt("Username: ")
	username, err := term.ReadLine()
	if err == io.EOF {
		return nil
	} else if err != nil {
		return err
	}

	password, err := term.ReadPassword("Password: ")
	if err == io.EOF {
		return nil
	} else if err != nil {
		return err
	}

	log.Debug(id, username, password)

	term.SetPrompt(s.Prompt)

	for {
		line, err := term.ReadLine()
		if err == io.EOF {
			return nil
		} else if err != nil {
			return err
		}

		if line == "" {
			continue
		}

		term.Write([]byte(fmt.Sprintf("sh: %s: command not found\n", line)))
	}
}
