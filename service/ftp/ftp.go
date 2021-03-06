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
package ftp

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/asdine/storm"
	"github.com/kushtaka/kushtakad/events"
	"github.com/kushtaka/kushtakad/service/filesystem"
	"github.com/kushtaka/kushtakad/state"
)

func (s FtpService) SetHost(h string) {
	s.Host = h
}

func (s FtpService) SetApiKey(k string) {
	s.ApiKey = k
}

func (f *FtpService) ConfigureAndRun() {
	store, err := getStorage()
	if err != nil {
		log.Errorf("FTP: Could not initialize storage. %s", err.Error())
	}
	log.Debug("getStorage()")

	cert, err := store.Certificate()
	if err != nil {
		log.Errorf("TLS error: %s", err.Error())
	}
	log.Debug("Certificate()")

	f.recv = make(chan string)

	opts := &ServerOpts{
		Auth: &User{
			users: map[string]string{
				"anonymous": "anonymous",
			},
		},
		//Name: f.ServerName,
		Name: "ftp.bend.k12.or.us",
		//WelcomeMessage: f.Banner,
		WelcomeMessage: "DiskStation FTP server ready.\r\n",
		PassivePorts:   fmt.Sprintf("%d-%d", f.Port, f.Port),
	}

	f.server = NewServer(opts)

	f.server.tlsConfig = simpleTLSConfig(cert)
	if f.server.tlsConfig != nil {
		f.server.TLS = false
		f.server.ExplicitFTPS = false
	}

	fs, err := filesystem.New(state.TmpDirLocation(), "ftp", f.ServerName)
	if err != nil {
		log.Debugf("FTP Filesystem error: %s", err.Error())
	}

	log.Debugf("FsRoot : %s", f.FsRoot)
	log.Debugf("FileSystem rooted at %s", fs.RealPath("/"))

	f.driver = NewFileDriver(fs)
}

func FTP() *FtpService {

	store, err := getStorage()
	if err != nil {
		log.Errorf("FTP: Could not initialize storage. %s", err.Error())
	}

	cert, err := store.Certificate()
	if err != nil {
		log.Errorf("TLS error: %s", err.Error())
	}

	s := &FtpService{
		recv: make(chan string),
	}

	opts := &ServerOpts{
		Auth: &User{
			users: map[string]string{
				"anonymous": "anonymous",
			},
		},
		Name:           s.ServerName,
		WelcomeMessage: s.Banner,
		PassivePorts:   fmt.Sprintf("%d-%d", s.Port, s.Port),
	}

	s.server = NewServer(opts)

	s.server.tlsConfig = simpleTLSConfig(cert)
	if s.server.tlsConfig != nil {
		//s.server.TLS = true
		s.server.ExplicitFTPS = true
	}

	fs, err := filesystem.New(state.TmpDirLocation(), "ftp", s.ServerName)
	if err != nil {
		log.Debugf("FTP Filesystem error: %s", err.Error())
	}

	log.Debugf("FileSystem rooted at %s", fs.RealPath("/"))

	s.driver = NewFileDriver(fs)

	return s
}

type FtpService struct {
	SensorID     int64  `json:"sensor_id"`
	Banner       string `json:"banner"`
	Port         int    `json:"port"`
	PsvPortRange string `json:"passive-port-range"`
	ServerName   string `json:"server_name"`
	FsRoot       string `toml:"fs_base"`

	Host   string
	ApiKey string
	recv   chan string
	server *Server
	driver Driver
}

func (s FtpService) Handle(ctx context.Context, conn net.Conn, db *storm.DB) error {
	defer conn.Close()
	ftpConn := s.server.newConn(conn, s.driver, s.recv)

	split := strings.Split(conn.RemoteAddr().String(), ":")
	network := conn.RemoteAddr().Network()
	ip := split[0]
	attackerPort := ""
	if len(split) > 1 {
		attackerPort = split[1]
	}
	es := &events.EventSensor{
		SensorID:     s.SensorID,
		Type:         "ftp",
		Port:         s.Port,
		AttackerPort: attackerPort,
	}
	em := events.NewSensorEventManager(network, ip, es)
	err := em.SendEvent("new", s.Host, s.ApiKey)
	if err != nil {
		log.Debug(err)
	}

	ftpConn.Serve()

	return nil
}
