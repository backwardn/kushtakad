package models

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"

	"github.com/gorilla/securecookie"
)

const SettingsID = 1

type Settings struct {
	SessionHash  []byte `json:"session_hash"`
	SessionBlock []byte `json:"session_block"`
	CsrfHash     []byte `json:"csrf_hash"`
	Host         string `json:"host"`
	Scheme       string `json:"scheme"`
	Port         string `json:"port"`
	URI          string `json:"uri"`
}

func (s *Settings) BuildURI() string {
	if os.Getenv("KUSHTAKA_ENV") == "development" {
		s.URI = fmt.Sprintf("%s://%s%s", "http", "localhost", ":3000")
	} else if len(s.URI) == 0 {
		s.URI = fmt.Sprintf("%s://%s%s", s.Scheme, s.Host, s.Port)
	}
	return s.URI
}

func InitSettings() (*Settings, error) {
	s, err := FindSettings()
	if len(s.SessionHash) != 32 {
		s.SessionHash = securecookie.GenerateRandomKey(32)
	}

	if len(s.SessionBlock) != 16 {
		s.SessionBlock = securecookie.GenerateRandomKey(16)
	}

	if len(s.CsrfHash) != 32 {
		s.CsrfHash = securecookie.GenerateRandomKey(32)
	}

	if len(s.Host) == 0 {
		if os.Getenv("KUSHTAKA_ENV") == "development" {
			s.Host = "localhost"
			s.Port = ":8080"
		} else {
			ip := GetOutboundIP().String()
			s.Host = fmt.Sprintf("%s", ip)
			s.Port = ":8080"
		}
	}

	if s.Scheme != "http" || s.Scheme != "https" {
		s.Scheme = "http"
	}
	s.BuildURI()

	b, err := json.MarshalIndent(s, "", " ")
	if err != nil {
		return s, err
	}

	err = ioutil.WriteFile("server.json", b, 0644)
	if err != nil {
		return s, err
	}

	return s, nil
}

func FindSettings() (*Settings, error) {
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
func GetOutboundIP() net.IP {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP
}
