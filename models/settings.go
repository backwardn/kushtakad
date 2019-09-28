package models

import (
	"fmt"
	"net"
	"os"

	"github.com/asdine/storm"
	"github.com/gorilla/securecookie"
)

const SettingsID = 1

type Settings struct {
	ID           int64  `storm:"id,increment" json:"id"`
	SessionHash  []byte `json:"session_hash"`
	SessionBlock []byte `json:"session_block"`
	CsrfHash     []byte `json:"csrf_hash"`
	Host         string
	Scheme       string
	URI          string
}

func BuildURI(db *storm.DB) string {
	var scheme, host string
	st, err := FindSettings(db)
	if err != nil {
		log.Errorf("BuildURI failed %w", err)
	}

	if os.Getenv("KUSHTAKA_ENV") == "development" {
		scheme = "http"
		host = "localhost:3000"
	} else {
		scheme = st.Scheme
		host = st.Host
	}
	uri := fmt.Sprintf("%s://%s", scheme, host)
	log.Debug(uri)
	return uri
}

func InitSettings(db *storm.DB) (Settings, error) {
	var s Settings
	db.One("ID", SettingsID, &s)
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
			s.Host = "localhost:8080"
		} else {
			ip := GetOutboundIP().String()
			s.Host = fmt.Sprintf("%s:8080", ip)
		}
	}

	if s.Scheme != "http" || s.Scheme != "https" {
		s.Scheme = "http"
	}

	s.URI = BuildURI(db)
	log.Debug("InitSettings")
	err := db.Save(&s)
	if err != nil {
		return s, err
	}

	return s, nil
}

func FindSettings(db *storm.DB) (*Settings, error) {
	s := &Settings{}
	err := db.One("ID", SettingsID, s)
	if err != nil {
		log.Debugf("FindSettings error : %v", err)
	}
	return s, err
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
