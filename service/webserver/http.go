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
package webserver

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"strings"

	"github.com/asdine/storm"
)

// Http is a placeholder
func HTTP() (*HttpService, error) {
	s := &HttpService{}

	return s, nil
}

type HttpServiceConfig struct {
	Server string `toml:"server"`
}

type HttpService struct {
	SensorID             int64  `json:"sensor_id"`
	CloneID              int64  `json:"clone_id"`
	FQDN                 string `json:"fqdn"`
	Port                 int    `json:"port"`
	Type                 string `json:"type"`
	HostNameOrExternalIp string `json:"hostname_or_external_ip"`

	Host   string
	ApiKey string

	HttpServiceConfig
}

func (s HttpService) SetHost(h string) {
	s.Host = h
}

func (s HttpService) SetApiKey(k string) {
	s.ApiKey = k
}

func (s HttpService) HasDb() bool {
	return true
}

func (s *HttpService) CanHandle(payload []byte) bool {
	if bytes.HasPrefix(payload, []byte("GET")) {
		return true
	} else if bytes.HasPrefix(payload, []byte("HEAD")) {
		return true
	} else if bytes.HasPrefix(payload, []byte("POST")) {
		return true
	} else if bytes.HasPrefix(payload, []byte("PUT")) {
		return true
	} else if bytes.HasPrefix(payload, []byte("DELETE")) {
		return true
	} else if bytes.HasPrefix(payload, []byte("PATCH")) {
		return true
	} else if bytes.HasPrefix(payload, []byte("TRACE")) {
		return true
	} else if bytes.HasPrefix(payload, []byte("CONNECT")) {
		return true
	} else if bytes.HasPrefix(payload, []byte("OPTIONS")) {
		return true
	}

	return false
}

func (s HttpService) Handle(ctx context.Context, conn net.Conn, db *storm.DB) error {

	for {
		br := bufio.NewReader(conn)

		req, err := http.ReadRequest(br)
		if err == io.EOF {
			return nil
		} else if err != nil {
			return err
		}
		defer req.Body.Close()

		body := make([]byte, 1024)

		n, err := req.Body.Read(body)
		if err == io.EOF {
		} else if err != nil {
			return err
		}

		body = body[:n]
		io.Copy(ioutil.Discard, req.Body)

		//var redir Redirect

		/*
			db.One("URL", u, &redir)

			if redir.ID > 0 {
				for k, v := range redir.Headers {
					var s string
					for _, v1 := range v {
						v1 = strings.ReplaceAll(v1, "KUSHTAKA_URL_REPLACE", "localhost:3002")
						v1 = strings.ReplaceAll(v1, "https", "http")
						s = s + v1
					}
					w.Header().Set(k, s)
				}
				w.WriteHeader(redir.StatusCode)
				return
			}

		*/
		var res Res
		u := req.URL.RequestURI()
		log.Debug(u)
		err = db.One("URL", u, &res)
		if err != nil {
			log.Debug(err)
		}

		res.Body = replaceURL(res.Body)

		headers := http.Header{}
		var host string
		if s.Port == 80 || s.Port == 443 {
			host = fmt.Sprintf("%s", s.HostNameOrExternalIp)
		} else {
			host = fmt.Sprintf("%s:%d", s.HostNameOrExternalIp, s.Port)
		}

		for k, v := range res.Headers {
			var s string
			for _, v1 := range v {
				v1 = strings.ReplaceAll(v1, "KUSHTAKA_URL_REPLACE", host)
				v1 = strings.ReplaceAll(v1, "https", "http")
				s = s + v1
			}

			switch strings.TrimSpace(k) {
			case "Strict-Transport-Security":
			case "Content-Length":
			default:
				headers.Set(k, s)
			}
		}

		resp := http.Response{
			ContentLength: int64(len(res.Body)),
			Body:          ioutil.NopCloser(bytes.NewReader(res.Body)),
			StatusCode:    http.StatusOK,
			Status:        http.StatusText(http.StatusOK),
			Proto:         req.Proto,
			ProtoMajor:    req.ProtoMajor,
			ProtoMinor:    req.ProtoMinor,
			Request:       req,
			Header:        headers,
		}

		if err := resp.Write(conn); err != nil {
			return err
		}
	}
}

/*
func (s HttpService) Handle(ctx context.Context, conn net.Conn, db *storm.DB) error {

	for {
		br := bufio.NewReader(conn)

		req, err := http.ReadRequest(br)
		if err == io.EOF {
			return nil
		} else if err != nil {
			return err
		}

		defer req.Body.Close()

		body := make([]byte, 1024)

		n, err := req.Body.Read(body)
		if err == io.EOF {
		} else if err != nil {
			return err
		}

		body = body[:n]

		io.Copy(ioutil.Discard, req.Body)

		var res Res
		u := req.URL.RequestURI()
		log.Debug(u)
		err = db.One("URL", u, &res)
		if err != nil {
			log.Debug(err)
		}
		//msg := "<html><head></head><body><p>hello</p></body></html>"
		msg := "asdf"

		resp := http.Response{
			ContentLength: int64(len(msg)),
			Body:          ioutil.NopCloser(bytes.NewReader([]byte(msg))),
			StatusCode:    http.StatusOK,
			Status:        http.StatusText(http.StatusOK),
			Proto:         req.Proto,
			ProtoMajor:    req.ProtoMajor,
			ProtoMinor:    req.ProtoMinor,
			Request:       req,
			Header: http.Header{
				"Server": []string{s.Server},
			},
		}

		if err := resp.Write(conn); err != nil {
			return err
		}
	}
}

*/
