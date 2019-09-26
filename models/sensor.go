package models

import (
	"crypto/rand"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"

	"github.com/asdine/storm"
	validation "github.com/go-ozzo/ozzo-validation"
	"github.com/kushtaka/kushtakad/service"
	"github.com/kushtaka/kushtakad/service/telnet"
)

type Sensor struct {
	ID     int64        `storm:"id,increment,index"`
	TeamID int64        `storm:"id,index"`
	Name   string       `storm:"index,unique" json:"name"`
	Note   string       `storm:"index" json:"note"`
	ApiKey string       `storm:"index,unique" json:"api_key"`
	Cfgs   []ServiceCfg `storm:"index" json:"service_configs`
	mu     sync.Mutex
}

func NewSensor(name, note string, teamid int64) *Sensor {
	return &Sensor{Name: name, Note: note, TeamID: teamid, ApiKey: GenerateSecureKey()}
}

func GenerateSecureKey() string {
	k := make([]byte, 32)
	io.ReadFull(rand.Reader, k)
	return fmt.Sprintf("%x", k)
}

func (s *Sensor) Wash() {
	s.Name = strings.TrimSpace(s.Name)
	s.Name = Strip(s.Name)
}

func (s *Sensor) ValidateCreate() error {
	s.Wash()
	return validation.Errors{
		"Name": validation.Validate(
			&s.Name,
			validation.Required,
			validation.Length(4, 64).Error("must be between 4-64 characters")),
		"Note": validation.Validate(
			&s.Note,
			validation.Required,
			validation.Length(1, 3000).Error("must be between 1-3000 characters")),
		"TeamID": validation.Validate(
			&s.TeamID,
			validation.Required.Error("is required")),
	}.Filter()
}

func (s *Sensor) ServicesConfig(db *storm.DB) []*service.ServiceMap {
	var svm []*service.ServiceMap
	for _, v := range s.Cfgs {
		switch v.Type {
		case "telnet":
			var tel telnet.TelnetService
			db.One("ID", v.ServiceID, &tel)
			sm := &service.ServiceMap{
				Service:    tel,
				SensorName: s.Name,
				Type:       tel.Type,
				Port:       strconv.Itoa(tel.Port),
			}

			svm = append(svm, sm)
		}
	}

	return svm
}
