package models

import (
	"encoding/json"
	"io/ioutil"
	"net"
	"net/url"
	"os"

	"github.com/gorilla/securecookie"
)

const SettingsID = 1

type Settings struct {
	SessionHash  []byte `json:"session_hash"`
	SessionBlock []byte `json:"session_block"`
	CsrfHash     []byte `json:"csrf_hash"`
	BindURI      string `json:"bind_uri"`
	URI          string `json:"base_uri"`
	LeEnabled    bool   `json:"lets_encrypt"`
	Host         string `json:"-"`
	Port         string `json:"-"`
	Scheme       string `json:"-"`
	FQDN         string `json:"-"`
}

func (s *Settings) BuildBindURI() {
	u, err := url.Parse(s.BindURI)
	if err != nil {
		panic(err)
	}

	host, port, err := net.SplitHostPort(u.Host)
	if err != nil {
		s.Host = u.Host
		s.Port = u.Port()
	} else {
		s.Host = host
		s.Port = port
	}
}

func (s *Settings) BuildBaseURI() string {
	u, err := url.Parse(s.URI)
	if err != nil {
		panic(err)
	}

	host, _, err := net.SplitHostPort(u.Host)
	if err != nil {
		s.FQDN = u.Host
	} else {
		s.FQDN = host
	}

	s.Scheme = u.Scheme
	return s.URI
}

func (s *Settings) CreateIfNew() {
	if len(s.SessionHash) != 32 {
		s.SessionHash = securecookie.GenerateRandomKey(32)
	}

	if len(s.SessionBlock) != 16 {
		s.SessionBlock = securecookie.GenerateRandomKey(16)
	}

	if len(s.CsrfHash) != 32 {
		s.CsrfHash = securecookie.GenerateRandomKey(32)
	}

	if len(s.URI) < 4 {
		if os.Getenv("KUSHTAKA_ENV") == "development" {
			s.URI = "http://localhost:8080"
		}
	}

	if len(s.BindURI) < 4 {
		if os.Getenv("KUSHTAKA_ENV") == "development" {
			s.BindURI = "http://localhost:8080"
		} else {
			s.BindURI = "http://0.0.0.0:8080"
		}
	}
}

func InitSettings() (*Settings, error) {
	s, err := NewSettings()
	s.CreateIfNew()
	s.BuildBindURI()
	s.BuildBaseURI()

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
