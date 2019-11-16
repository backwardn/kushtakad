package models

import (
	"encoding/json"
	"io/ioutil"
	"net"
	"net/url"
	"os"
	"path"

	"github.com/gorilla/securecookie"
	"github.com/pkg/errors"
)

const (
	SettingsID   = 1
	ServerConfig = "server.json"
	DataDir      = "data"
)

type Settings struct {
	SessionHash  []byte `json:"session_hash"`
	SessionBlock []byte `json:"session_block"`
	CsrfHash     []byte `json:"csrf_hash"`
	BindAddr     string `json:"bind_addr"`
	URI          string `json:"base_uri"`
	LeEnabled    bool   `json:"lets_encrypt"`
	Host         string `json:"-"`
	Port         string `json:"-"`
	Scheme       string `json:"-"`
	FQDN         string `json:"-"`
}

const (
	StateProduction  = "production"
	StateTest        = "test"
	StateDevelopment = "development"
	prodDataDir      = "data"
	testDataDir      = "data_test"
	devDataDir       = "data_dev"
)

func (s *Settings) BuildBindAddr() {
	host, port, err := net.SplitHostPort(s.BindAddr)
	if err != nil {
		log.Error("Host %s, Port %s, Err %v", host, port, err)
	}
	s.Host = host
	s.Port = port
}

func (s *Settings) BuildBaseURI() string {
	u, err := url.Parse(s.URI)
	if err != nil {
		log.Fatal(err)
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
	env := os.Getenv("KUSHTAKA_ENV")
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
		if env == StateDevelopment && env == StateTest {
			s.URI = "http://localhost:3000"
		} else {
			s.URI = "http://localhost:8080"
		}
	}


	if len(s.BindAddr) < 4 {
		if env == StateDevelopment && env == StateTest {
			s.BindAddr = "localhost:8080"
		} else {
			s.BindAddr = "0.0.0.0:8080"
		}
	}
}

func InitSettings(dir string) (*Settings, error) {
	s, err := NewSettings(dir)
	s.CreateIfNew()
	s.BuildBindAddr()
	s.BuildBaseURI()

	err = s.WriteSettings(dir)
	if err != nil {
		return s, err
	}

	return s, nil
}

func (s *Settings) WriteSettings(dir string) error {
	b, err := json.MarshalIndent(s, "", " ")
	if err != nil {
		return err
	}

	fp := ServerCfgFile(dir)
	err = ioutil.WriteFile(fp, b, 0644)
	if err != nil {
		return err
	}
	return nil
}

func NewSettings(dir string) (*Settings, error) {
	log.Debug("start")
	settings := &Settings{}
	fp := ServerCfgFile(dir)
	jsonFile, err := os.Open(fp)
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

func ServerCfgFile(dir string) string {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(errors.Wrap(err, "ServerCfgFile() unable to detect current working directory"))
	}

	return path.Join(cwd, dir, ServerConfig)
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
