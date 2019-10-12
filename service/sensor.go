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
package service

import (
	"context"
	"fmt"
	"net"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/asdine/storm"
	"github.com/fatih/color"
	"github.com/kushtaka/kushtakad/listener"
)

func configureServices(h *Hub, svm []*ServiceMap) listener.SocketConfig {

	sc := listener.SocketConfig{}
	for _, sm := range svm {
		log.Debugf("Configuring Service %s", sm.SensorName)
		addr, err := net.ResolveTCPAddr("tcp", net.JoinHostPort("localhost", sm.Port))
		if err != nil {
			log.Fatal(err)
		}
		h.ports[addr] = append(h.ports[addr], sm)
		sc.AddAddress(addr)
	}
	return sc

}

func startSensor(auth *Auth, ctx context.Context, svm []*ServiceMap) {

	h := &Hub{Auth: auth, ports: make(map[net.Addr][]*ServiceMap)}

	sc := configureServices(h, svm)

	incoming := make(chan net.Conn)

	l, err := listener.NewSocket(sc)
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		//TODO BenB: how to prevent this from eating so much CPU?
		for {
			conn, err := l.Accept()
			if err != nil {
				log.Fatal(err)
			}

			incoming <- conn

			runtime.Gosched() // in case of goroutine starvation // with many connection and single procs
		}
	}()

	err = l.Start(ctx)
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case conn := <-incoming:
				go h.handle(conn)
			}
		}
	}()

}

func (h *Hub) handle(c net.Conn) {

	sm, newConn, err := h.findService(c)
	if sm == nil {
		log.Debugf("No suitable handler for %s => %s: %s", c.RemoteAddr(), c.LocalAddr(), err.Error())
		return
	}

	log.Debugf("Handling connection for %s => %s %s(%s)", c.RemoteAddr(), c.LocalAddr(), sm.SensorName, sm.Type)

	newConn = TimeoutConn(newConn, time.Second*30)

	ctx := context.Background()

	if err := sm.Service.Handle(ctx, newConn, sm.DB); err != nil {
		log.Errorf(color.RedString("Error handling service: %s > %s", sm.Type, err.Error()))
	}
}

type Hub struct {
	mu *sync.Mutex

	// Maps a port and a protocol to an array of pointers to services
	ports map[net.Addr][]*ServiceMap

	Auth *Auth
}

// Wraps a Servicer, adding some metadata
type ServiceMap struct {
	Service Servicer `json:"service"`

	SensorName string `json:"sensor_name"`
	Type       string `json:"type"`
	Port       string `json:"port`
	DB         *storm.DB
}

type TmpMap struct {
	Service interface{} `json:"service"`

	SensorName string `json:"sensor_name"`
	Type       string `json:"type"`
	Port       string `json:"port`
}

type Servicer interface {
	Handle(context.Context, net.Conn, *storm.DB) error
	SetApiKey(k string)
	SetHost(h string)
}

type Service struct {
	Port int
}

type Listener interface {
	Start(ctx context.Context) error
	Accept() (net.Conn, error)
}

// Addr, proto, port, error
func ToAddr(input string) (net.Addr, string, int, error) {
	parts := strings.Split(input, "/")

	if len(parts) != 2 {
		return nil, "", 0, fmt.Errorf(`wrong format (needs to be "protocol/(host:)port")`)
	}

	proto := parts[0]

	host, port, err := net.SplitHostPort(parts[1])
	if err != nil {
		port = parts[1]
	}

	portUint16, err := strconv.ParseUint(port, 10, 16)
	if err != nil {
		return nil, "", 0, fmt.Errorf("error parsing port value: %s", err.Error())
	}

	switch proto {
	case "tcp":
		addr, err := net.ResolveTCPAddr("tcp", net.JoinHostPort(host, port))
		return addr, proto, int(portUint16), err
	case "udp":
		addr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(host, port))
		return addr, proto, int(portUint16), err
	default:
		return nil, "", 0, fmt.Errorf("unknown protocol %s", proto)
	}
}

type CanHandlerer interface {
	CanHandle([]byte) bool
}

func (hc *Hub) findService(conn net.Conn) (*ServiceMap, net.Conn, error) {
	localAddr := conn.LocalAddr()

	var serviceCandidates []*ServiceMap

	for k, sc := range hc.ports {
		if !compareAddr(k, localAddr) {
			continue
		}

		log.Debugf("findService %v", sc)
		serviceCandidates = sc
	}

	if len(serviceCandidates) == 0 {
		return nil, nil, fmt.Errorf("No service configured for the given port")
	} else if len(serviceCandidates) == 1 {
		return serviceCandidates[0], conn, nil
	}

	peekUninitialized := true
	var tConn net.Conn
	var pConn *peekConnection
	var n int
	buffer := make([]byte, 1024)
	for _, service := range serviceCandidates {
		ch, ok := service.Service.(CanHandlerer)
		if !ok {
			return service, conn, nil
		}
		if peekUninitialized {
			tConn = TimeoutConn(conn, time.Second*30)
			pConn = PeekConnection(tConn)
			log.Debug("Peeking connection %s => %s", conn.RemoteAddr(), conn.LocalAddr())
			_, err := pConn.Peek(buffer)
			if err != nil {
				return nil, nil, fmt.Errorf("could not peek bytes: %s", err.Error())
			}
			peekUninitialized = false
		}
		if ch.CanHandle(buffer[:n]) {
			return service, pConn, nil
		}
	}
	return nil, nil, fmt.Errorf("No suitable service for the given port")
}

func (h *Hub) heartbeat() {
	beat := time.Tick(30 * time.Second)
	count := 0
	for range beat {
		count++
	}
}

func compareAddr(addr1 net.Addr, addr2 net.Addr) bool {
	if ta1, ok := addr1.(*net.TCPAddr); ok {
		ta2, ok := addr2.(*net.TCPAddr)
		if !ok {
			return false
		}

		if ta1.Port != ta2.Port {
			return false
		}

		if ta1.IP == nil {
		} else if ta2.IP == nil {
		} else if !ta1.IP.Equal(ta2.IP) {
			return false
		}

		return true
	} else if ua1, ok := addr1.(*net.UDPAddr); ok {
		ua2, ok := addr2.(*net.UDPAddr)
		if !ok {
			return false
		}

		if ua1.Port != ua2.Port {
			return false
		}

		if ua1.IP == nil {
		} else if ua2.IP == nil {
		} else if !ua1.IP.Equal(ua2.IP) {
			return false
		}

		return true
	}

	return false
}
