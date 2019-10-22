package webserver

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
	"github.com/kushtaka/kushtakad/events"
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
	HostNameOrExternalIp string `json:"http-hostname-or-external-ip"`

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

		u := req.URL.RequestURI()

		var redir Redirect
		err = db.One("URL", u, &redir)
		if err != nil {
			log.Debugf("Did not find %s for the Redirect > %s", u, err.Error())
		}

		// first we check to see if the URI is a redirect
		if redir.ID > 0 {
			headers := buildHeaders(redir.Headers)
			resp := http.Response{
				StatusCode: redir.StatusCode,
				Status:     http.StatusText(redir.StatusCode),
				Proto:      req.Proto,
				ProtoMajor: req.ProtoMajor,
				ProtoMinor: req.ProtoMinor,
				Request:    req,
				Header:     headers,
			}

			if err := resp.Write(conn); err != nil {
				log.Debug(err)
				return err
			}

			em := events.NewEventManager("http", s.Port, s.SensorID)
			err := em.SendEvent("new", s.Host, s.ApiKey, conn.RemoteAddr())
			if err != nil {
				log.Debug(err)
			}

		} else {

			// now we check to see if the URI is a page in the dataset
			var res Res
			err = db.One("URL", u, &res)
			if err != nil {

				// and this is the werid if statement
				// if the page doesn't exist, just grab the first page by ID 1
				// and redirect the attacker there
				// this could be much cleaner
				log.Debugf("Did not find %s for the Res > %s", u, err.Error())
				db.One("ID", 1, &res)
				headers := buildHeaders(redir.Headers)
				resp := http.Response{
					StatusCode: 301,
					Status:     http.StatusText(301),
					Proto:      req.Proto,
					ProtoMajor: req.ProtoMajor,
					ProtoMinor: req.ProtoMinor,
					Request:    req,
					Header:     headers,
				}

				if err := resp.Write(conn); err != nil {
					log.Debug(err)
					return err
				}

				em := events.NewEventManager("http", s.Port, s.SensorID)
				err := em.SendEvent("new", s.Host, s.ApiKey, conn.RemoteAddr())
				if err != nil {
					log.Debug(err)
				}
			} else {

				var host string
				if s.Port == 80 || s.Port == 443 {
					host = fmt.Sprintf("%s", s.HostNameOrExternalIp)
				} else {
					host = fmt.Sprintf("%s:%d", s.HostNameOrExternalIp, s.Port)
				}

				headers := buildHeaders(res.Headers)
				res.Body = replaceURL(host, res.Body)
				resp := http.Response{
					ContentLength: int64(len(res.Body)),
					Body:          ioutil.NopCloser(bytes.NewReader(res.Body)),
					StatusCode:    res.StatusCode,
					Status:        http.StatusText(res.StatusCode),
					Proto:         req.Proto,
					ProtoMajor:    req.ProtoMajor,
					ProtoMinor:    req.ProtoMinor,
					Request:       req,
					Header:        headers,
				}

				if err := resp.Write(conn); err != nil {
					log.Debug(err)
					return err
				}

				em := events.NewEventManager("http", s.Port, s.SensorID)
				err := em.SendEvent("new", s.Host, s.ApiKey, conn.RemoteAddr())
				if err != nil {
					log.Debug(err)
				}
			}
		}

	}
}

func buildHeaders(h http.Header) http.Header {
	headers := http.Header{}
	for k, v := range h {
		var s string
		for _, v1 := range v {
			v1 = strings.ReplaceAll(v1, "KUSHTAKA_URL_REPLACE", "localhost:3002")
			v1 = strings.ReplaceAll(v1, "https", "http")
			s = s + v1
		}
		headers.Set(k, s)
	}
	return headers
}
