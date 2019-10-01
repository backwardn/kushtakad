package models

import (
	"time"

	"github.com/asdine/storm"
	"github.com/mholt/certmagic"
)

type LE struct {
	Magic  *certmagic.Config
	Domain Domain
	DB     *storm.DB
}

type Domain struct {
	FQDN   string `json:"fqdn"`
	LETest *LETest
}

func NewStageLE(email string, domain Domain, db *storm.DB) LE {
	return LE{
		DB:     db,
		Magic:  leStageCfg(email),
		Domain: domain,
	}
}

func leStageCfg(email string) *certmagic.Config {
	cert := certmagic.NewDefault()
	cert.CA = certmagic.LetsEncryptStagingCA
	cert.Email = email
	cert.Agreed = true
	return cert
}

func NewLE(email string, domain Domain, db *storm.DB) LE {
	return LE{
		DB:     db,
		Magic:  leCfg(email),
		Domain: domain,
	}
}

func leCfg(email string) *certmagic.Config {
	cert := certmagic.NewDefault()
	cert.CA = certmagic.LetsEncryptProductionCA
	cert.Email = email
	cert.Agreed = true
	return cert
}

const LEFailed = "fail"
const LESuccess = "success"
const LEPending = "pending"

type LETest struct {
	ID           int64  `storm:"id,increment,index"`
	FQDN         string `storm:"index"`
	State        string `storm:"index"`
	StateMsg     string
	IsTestRecent bool // if not, don't allow
	Created      time.Time
}
