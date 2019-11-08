package models

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/gorilla/securecookie"
)

const SettingsID = 1

type Settings struct {
	SessionHash  []byte `json:"session_hash"`
	SessionBlock []byte `json:"session_block"`
	CsrfHash     []byte `json:"csrf_hash"`
	LeEnabled    bool   `json:"lets_encrypt"`
	URI          string `json:"uri"`
	FQDN         string `json:"fqdn"`
	Host         string `json:"-"`
	Port         string `json:"-"`
	Scheme       string `json:"-"`
}

func (s *Settings) BuildURI() string {
	host := "0.0.0.0"
	port := "8080"
	scheme := "http"
	if os.Getenv("KUSHTAKA_ENV") == "development" {
		host = "localhost"
		port = "8080"
		scheme = "http"
	} else if s.LeEnabled {
		host = "0.0.0.0"
		port = "80"
		scheme = "https"
	}

	s.Host = host
	s.Port = port
	s.Scheme = scheme

	if len(s.FQDN) > 4 {
		s.URI = fmt.Sprintf("%s://%s:%s", scheme, s.FQDN, port)
	} else if os.Getenv("KUSHTAKA_ENV") == "development" {
		s.URI = fmt.Sprintf("%s://%s:%s", scheme, host, "3000")
	} else {
		s.URI = fmt.Sprintf("%s://%s:%s", scheme, host, port)
	}

	return s.URI
}

func InitSettings() (*Settings, error) {
	s, err := NewSettings()
	if len(s.SessionHash) != 32 {
		s.SessionHash = securecookie.GenerateRandomKey(32)
	}

	if len(s.SessionBlock) != 16 {
		s.SessionBlock = securecookie.GenerateRandomKey(16)
	}

	if len(s.CsrfHash) != 32 {
		s.CsrfHash = securecookie.GenerateRandomKey(32)
	}

	s.BuildURI()

	err = s.WriteSettings()
	if err != nil {
		return s, err
	}

	return s, nil
}

func (s *Settings) WriteSettings() error {
	b, err := json.MarshalIndent(s, "", " ")
	if err != nil {
		return err
	}

	err = ioutil.WriteFile("server.json", b, 0644)
	if err != nil {
		return err
	}
	return nil
}

func NewSettings() (*Settings, error) {
	log.Debug("start")
	settings := &Settings{}
	jsonFile, err := os.Open("server.json")
	if err != nil {
		return settings, err
	}
	defer jsonFile.Close()

	b, _ := ioutil.ReadAll(jsonFile)
	err = json.Unmarshal(b, &settings)
	if err != nil {
		return settings, err
	}

	log.Debug("end")
	return settings, nil
}

// Get preferred outbound ip of this machine
/*
func GetOutboundIP() net.IP {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP
}
*/
